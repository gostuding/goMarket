package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgerrcode"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gorm.io/gorm/logger"
)

type psqlStorage struct {
	con *gorm.DB
}

func NewPSQLStorage(connectionString string) (*psqlStorage, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection create error: %w", err)
	}
	storage := psqlStorage{
		con: db,
	}
	return &storage, structCheck(db)
}

func (s *psqlStorage) Registration(ctx context.Context, login, pwd, ua, ip string) (int, error) {
	user := Users{Login: login, Pwd: getMD5Hash(pwd), UserAgent: ua, IP: ip}
	result := s.con.WithContext(ctx).Create(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("sql error: %w", result.Error)
	}
	return int(user.ID), nil
}

func (s *psqlStorage) Login(ctx context.Context, login, pwd, ua, ip string) (int, error) {
	var user Users
	result := s.con.WithContext(ctx).Where("login = ? AND pwd = ?", login, getMD5Hash(pwd)).First(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("user error: %w", result.Error)
	}
	user.UserAgent = ua
	user.IP = ip
	result = s.con.WithContext(ctx).Save(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("update user in login error: %w", result.Error)
	}
	return int(user.ID), nil
}

func (s *psqlStorage) isOrderExist(ctx context.Context, order string) (*Orders, error) {
	var item Orders
	result := s.con.WithContext(ctx).Where("number = ? ", order).First(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	return &item, nil
}

func (s *psqlStorage) AddOrder(ctx context.Context, uid int, order string) (int, error) {
	item, err := s.isOrderExist(ctx, order)
	if item != nil {
		if item.UID == uid {
			return http.StatusOK, nil
		}
		return http.StatusConflict, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := s.con.WithContext(ctx).Create(&Orders{UID: uid, Number: order, Status: "NEW"})
		if result.Error != nil {
			return http.StatusInternalServerError, fmt.Errorf("create order error: %w", result.Error)
		}
		return http.StatusAccepted, nil
	}
	return http.StatusInternalServerError, fmt.Errorf("select order error: %w", err)
}

func (s *psqlStorage) getValues(ctx context.Context, uid int, values any) ([]byte, error) {
	result := s.con.WithContext(ctx).Order("id desc").Where("uid = ?", uid).Find(values)
	if result.Error != nil {
		return nil, fmt.Errorf("get values error: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("json convert error: %w", err)
	}
	return data, nil
}

func (s *psqlStorage) GetOrders(ctx context.Context, uid int) ([]byte, error) {
	var orders []Orders
	return s.getValues(ctx, uid, &orders)
}

func (s *psqlStorage) GetUserBalance(ctx context.Context, uid int) ([]byte, error) {
	var user Users
	type balance struct {
		Current   float32 `json:"current"`
		Withdrawn float32 `json:"withdrawn"`
	}
	result := s.con.WithContext(ctx).Where("id = ?", uid).First(&user) //nolint:all // more clearly
	if result.Error != nil {
		return nil, fmt.Errorf("get user balance error: %w", result.Error)
	}
	data, err := json.Marshal(balance{Current: user.Balance, Withdrawn: user.Withdrawn})
	if err != nil {
		return nil, fmt.Errorf("convert user balance to json error: %w", err)
	}
	return data, nil
}

func (s *psqlStorage) AddWithdraw(ctx context.Context, uid int, order string, sum float32) (int, error) {
	var user Users
	result := s.con.WithContext(ctx).Where("id = ?", uid).First(&user)
	if result.Error != nil {
		return http.StatusInternalServerError, fmt.Errorf("get user error: %w", result.Error)
	}
	if user.Balance < sum {
		return http.StatusPaymentRequired, nil
	}
	user.Balance -= sum
	user.Withdrawn += sum
	withdraw := Withdraws{Sum: sum, UID: int(user.ID), Number: order}
	err := s.con.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("update user balance error: %w", err)
		}
		if err := tx.Create(&withdraw).Error; err != nil {
			return fmt.Errorf("create withdraw error: %w", err)
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return http.StatusConflict, errors.New("order number repeat error")
		}
		return http.StatusInternalServerError, fmt.Errorf("transaction error: %w", err)
	}
	return http.StatusOK, nil
}

func (s *psqlStorage) GetWithdraws(ctx context.Context, uid int) ([]byte, error) {
	var withdraws []Withdraws
	return s.getValues(ctx, uid, &withdraws)
}

func (s *psqlStorage) GetAccrualOrders() []string {
	var orders []Orders
	result := s.con.Order("id").Where("status NOT IN ?", []string{"INVALID", "PROCESSED"}).Find(&orders)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil
	}
	numbers := make([]string, 0)
	for _, item := range orders {
		numbers = append(numbers, item.Number)
	}
	return numbers
}

func (s *psqlStorage) SetOrderData(number string, status string, balance float32) error {
	var order Orders
	var user Users
	result := s.con.Where("number = ?", number).First(&order)
	if result.Error != nil {
		return fmt.Errorf("update order status, get order (%s) error: %w", number, result.Error)
	}
	result = s.con.Where("id = ?", order.UID).First(&user)
	if result.Error != nil {
		return fmt.Errorf("update order status, get user (%d) error: %w", order.UID, result.Error)
	}
	user.Balance += balance
	order.Status = status
	order.Accrual = balance
	err := s.con.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("user balance update error: %w", err)
		}
		if err := tx.Save(&order).Error; err != nil {
			return fmt.Errorf("update order status and accural error: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update order status transaction error: %w", err)
	}
	return nil
}

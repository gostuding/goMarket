package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"

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
		return 0, fmt.Errorf("get user error: %w", result.Error)
	}
	user.UserAgent = ua
	user.IP = ip
	result = s.con.WithContext(ctx).Save(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("update user in login error: %w", result.Error)
	}
	return int(user.ID), nil
}

func (s *psqlStorage) isOrderExist(ctx context.Context, item *Orders) (*Orders, error) {
	result := s.con.WithContext(ctx).First(item)
	if result.Error != nil {
		return nil, result.Error
	}
	return item, nil
}

func (s *psqlStorage) AddOrder(ctx context.Context, order string, uid int) (int, error) {
	item, err := s.isOrderExist(ctx, &Orders{Order: order})
	if item != nil {
		if item.UID == uid {
			return http.StatusOK, nil
		}
		return http.StatusConflict, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := s.con.WithContext(ctx).Create(&Orders{UID: uid, Order: order, Status: "New"})
		if result.Error != nil {
			return http.StatusInternalServerError, fmt.Errorf("create order error: %w", result.Error)
		}
		return http.StatusAccepted, nil
	}
	return http.StatusInternalServerError, fmt.Errorf("select order error: %w", err)
}

func (s *psqlStorage) GetOrders(ctx context.Context, uid int) ([]byte, error) {
	var orders []Orders
	result := s.con.WithContext(ctx).Where("uid = ?", uid).Find(&orders)
	if result.Error != nil {
		return nil, fmt.Errorf("get orders error: %w", result.Error)
	}
	if len(orders) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(orders)
	if err != nil {
		return nil, fmt.Errorf("json convert error: %w", err)
	}
	return data, nil
}

func (s *psqlStorage) GetUserBalance(ctx context.Context, uid string) ([]byte, error) {
	var user Users
	result := s.con.WithContext(ctx).Where("id = ?", uid).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("get user balance error: %w", result.Error)
	}
	data, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("json convert error: %w", err)
	}
	return data, nil
}

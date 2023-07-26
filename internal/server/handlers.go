package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/gostuding/goMarket/internal/server/middlewares"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Storage interface {
	CheckOrdersStorage
	Registration(context.Context, string, string, string, string) (int, error)
	Login(context.Context, string, string, string, string) (int, error)
	AddOrder(context.Context, int, string) (int, error)
	GetOrders(context.Context, int) ([]byte, error)
	GetUserBalance(context.Context, int) ([]byte, error)
	AddWithdraw(context.Context, int, string, float32) (int, error)
	GetWithdraws(context.Context, int) ([]byte, error)
}

type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func isValidateLoginPassword(body []byte) (*LoginPassword, error) {
	var user LoginPassword
	err := json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("body convert to json error: %w", err)
	}
	if user.Login == "" || user.Password == "" {
		return nil, errors.New("empty values for registration error")
	}
	return &user, nil
}

func checkOrderNumber(order string) error {
	initPosition := 0
	if len(order)%2 > 0 {
		initPosition = 1
	}
	summ := 0
	for i := 0; i < len(order); i++ {
		value, err := strconv.Atoi(fmt.Sprintf("%c", order[i]))
		if err != nil {
			return fmt.Errorf("order error in position %d: %w", i, err)
		}
		if initPosition == i {
			initPosition += 2
			summ += (2 * value) % 9 //nolint:gomnd // <- algoritm constants
		} else {
			summ += value
		}
	}
	if summ%10 == 0 {
		return nil
	}
	return fmt.Errorf("order control summ error. Order: '%s'", order)
}

func Register(ctx context.Context, body, key []byte, remoteAddr, ua string,
	strg Storage, tokenLiveTime int) (string, int, error) {
	user, err := isValidateLoginPassword(body)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf(incorrectIPErroString, err)
	}
	uid, err := strg.Registration(ctx, user.Login, user.Password, ua, ip)
	if err != nil {
		var pgErr *pgconn.PgError
		status := http.StatusInternalServerError
		err = fmt.Errorf(gormError, err)
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			status = http.StatusConflict
			err = fmt.Errorf("user registrating duplicate error: '%s'", user.Login)
		}
		return "", status, err
	}
	token, err := middlewares.CreateToken(key, tokenLiveTime, uid, ua, user.Login, ip)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf(tokenGenerateError, err)
	}
	return token, http.StatusOK, nil
}

// Login godoc
// @Summary Авторизация пользователя в микросервисе
// @Accept json
// @Produce json
// @Param params body LoginPassword true "Логи и пароль пользователя в формате json"
// @Success 200
// @Router /api/user/login [post]
func LoginFunc(ctx context.Context, body, key []byte, remoteAddr, ua string,
	strg Storage, tokenLiveTime int) (string, int, error) {
	user, err := isValidateLoginPassword(body)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf(incorrectIPErroString, err)
	}
	uid, err := strg.Login(ctx, user.Login, user.Password, ua, ip)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", http.StatusUnauthorized, fmt.Errorf("user not found in system. Login: '%s'", user.Login)
		} else {
			return "", http.StatusInternalServerError, fmt.Errorf(gormError, err)
		}
	}
	token, err := middlewares.CreateToken(key, tokenLiveTime, uid, ua, user.Login, ip)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf(tokenGenerateError, err)
	}
	return token, http.StatusOK, nil
}

func OrdersAddFunc(ctx context.Context, order string, strg Storage) (int, error) {
	err := checkOrderNumber(order)
	if err != nil {
		return http.StatusUnprocessableEntity, fmt.Errorf("check order error: %w", err)
	}
	uid, ok := ctx.Value(middlewares.AuthUID).(int)
	if !ok {
		return http.StatusUnauthorized, errors.New(uidContextTypeError)
	}
	return strg.AddOrder(ctx, uid, order)
}

func getListCommon(args *RequestResponce, name string, f func(context.Context, int) ([]byte, error)) {
	args.logger.Debugln(name, "list request")
	args.w.Header().Add(contentTypeString, ctApplicationJSONString)
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	data, err := f(args.r.Context(), uid)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(name, "get list error", err)
		return
	}
	if data == nil {
		args.w.WriteHeader(http.StatusNoContent)
	} else {
		args.w.WriteHeader(http.StatusOK)
	}
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnln(writeResponceErrorString, err)
	}
}

func OrdersList(args RequestResponce) {
	getListCommon(&args, "orders", args.strg.GetOrders)
}

func UserBalance(args RequestResponce) {
	args.logger.Debug("user balance request")
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	data, err := args.strg.GetUserBalance(args.r.Context(), uid)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln("get user balance error", err)
		return
	}
	args.w.Header().Add(contentTypeString, ctApplicationJSONString)
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnln(writeResponceErrorString, err)
	}
}

func AddWithdraw(args RequestResponce) {
	body, err := io.ReadAll(args.r.Body)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(bodyReadError, err)
		return
	}
	var withdraw Withdraw
	err = json.Unmarshal(body, &withdraw)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnln(jsonConvertEerrorString, err)
		return
	}
	args.logger.Debugln("add withdraw request", withdraw.Order, withdraw.Sum)
	err = checkOrderNumber(withdraw.Order)
	if err != nil {
		args.w.WriteHeader(http.StatusUnprocessableEntity)
		args.logger.Warnln(checkOrderErrorString, err)
		return
	}
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	status, err := args.strg.AddWithdraw(args.r.Context(), uid, withdraw.Order, withdraw.Sum)
	if err != nil {
		args.logger.Warnln("add withdraw error", err)
	}
	args.logger.Debugln("add withdraw status", status)
	args.w.WriteHeader(status)
}

func WithdrawsList(args RequestResponce) {
	getListCommon(&args, "withdraws", args.strg.GetWithdraws)
}

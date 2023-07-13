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
	"time"

	"github.com/gostuding/goMarket/internal/server/middlewares"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	validateError = "request validate error: %w"
	gormError     = "gorm error: %w"
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

type RegisterStruct struct {
	RequestResponce
	key           []byte
	tokenLiveTime int
}

type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func validateLoginPassword(r *http.Request) (*LoginPassword, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body error: %w", err)
	}
	var user LoginPassword
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("body convert to json error: %w", err)
	}
	if user.Login == "" || user.Password == "" {
		return nil, errors.New("empty values for registration error")
	}
	return &user, nil
}

func addToken(args *RegisterStruct, uid int, login string) {
	ip, _, err := net.SplitHostPort(args.r.RemoteAddr)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("user ip addres error: %w", err)
		return
	}
	token, err := middlewares.CreateToken(args.key, args.tokenLiveTime, uid, args.r.UserAgent(), login, ip)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("token generation error: %w", err)
		return
	}
	args.w.Header().Add("Authorization", token)
	cookie := &http.Cookie{
		Name:    "token",
		Value:   token,
		MaxAge:  args.tokenLiveTime * int(time.Hour/time.Millisecond),
		Expires: time.Now().Add(time.Duration(args.tokenLiveTime) * time.Hour),
	}
	http.SetCookie(args.w, cookie)
	args.w.WriteHeader(http.StatusOK)
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
	return errors.New("order control summ not equal")
}

func Registration(args *RegisterStruct) {
	user, err := validateLoginPassword(args.r)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf(validateError, err)
		return
	}
	ip, _, err := net.SplitHostPort(args.r.RemoteAddr)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf(validateError, err)
		return
	}
	uid, err := args.strg.Registration(args.r.Context(), user.Login, user.Password, args.r.UserAgent(), ip)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			args.w.WriteHeader(http.StatusConflict)
			args.logger.Warnln("user duplicate error", user.Login)
		} else {
			args.w.WriteHeader(http.StatusInternalServerError)
			args.logger.Warnf(gormError, err)
		}
		return
	}
	args.logger.Debugf("new user success registrated: '%s'", user.Login)
	addToken(args, uid, user.Login)
}

func Login(args *RegisterStruct) {
	user, err := validateLoginPassword(args.r)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf(validateError, err)
		return
	}
	ip, _, err := net.SplitHostPort(args.r.RemoteAddr)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf(validateError, err)
		return
	}
	uid, err := args.strg.Login(args.r.Context(), user.Login,
		user.Password, args.r.UserAgent(), ip)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			args.w.WriteHeader(http.StatusUnauthorized)
			args.logger.Warnf("user not found in system. Login: '%s'", user.Login)
		} else {
			args.w.WriteHeader(http.StatusInternalServerError)
			args.logger.Warnf(gormError, err)
		}
		return
	}
	args.logger.Debugf("user login success: '%s'", user.Login)
	addToken(args, uid, user.Login)
}

func OrdersAdd(args RequestResponce) {
	body, err := io.ReadAll(args.r.Body)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(bodyReadError, err)
		return
	}
	if len(body) == 0 {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnln("empty orders body")
		return
	}
	order := string(body)
	err = checkOrderNumber(order)
	if err != nil {
		args.w.WriteHeader(http.StatusUnprocessableEntity)
		args.logger.Warnln(checkOrderErrorString, err)
		return
	}
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	status, err := args.strg.AddOrder(args.r.Context(), uid, order)
	if err != nil {
		args.logger.Warnln(err)
	}
	args.logger.Debugln("add order status", status)
	args.w.WriteHeader(status)
}

func getListCommon(args *RequestResponce, name string, f func(context.Context, int) ([]byte, error)) {
	args.logger.Debug(name, "list request")
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

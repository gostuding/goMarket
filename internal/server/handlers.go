package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	Registration(context.Context, string, string, string, string) (int, error)
	Login(context.Context, string, string, string, string) (int, error)
	AddOrder(context.Context, string, int) (int, error)
	GetOrders(context.Context, int) ([]byte, error)
	GetUserBalance(context.Context, string) ([]byte, error)
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
	token, err := middlewares.CreateToken(args.key, args.tokenLiveTime, uid, args.r.UserAgent(),
		login, strings.Split(args.r.RemoteAddr, ":")[0])
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
			summ += (2 * value) % 9
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
	uid, err := args.strg.Registration(args.r.Context(), user.Login, user.Password,
		args.r.UserAgent(), args.r.RemoteAddr)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			args.w.WriteHeader(http.StatusConflict)
			args.logger.Infoln("user duplicate error", user.Login)
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
	uid, err := args.strg.Login(args.r.Context(), user.Login, user.Password,
		args.r.UserAgent(), args.r.RemoteAddr)
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
		args.logger.Warnf("orders body read error: %w", err)
		return
	}
	if body == nil && string(body) == "" {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnln("empty orders body")
		return
	}
	order := string(body)
	err = checkOrderNumber(order)
	if err != nil {
		args.w.WriteHeader(http.StatusUnprocessableEntity)
		args.logger.Warnln(fmt.Sprintf("check order error: %v", err))
		return
	}
	uid, err := strconv.Atoi(args.r.Header.Get(authHeader))
	if err != nil {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(fmt.Sprintf("uid convert error: %v", err))
		return
	}
	status, err := args.strg.AddOrder(args.r.Context(), order, uid)
	if err != nil {
		args.logger.Warnln(err)
	}
	args.logger.Debugln("add order status", status)
	args.w.WriteHeader(status)
}

func OrdersList(args RequestResponce) {
	args.logger.Debug("orders list request")
	uid, err := strconv.Atoi(args.r.Header.Get(authHeader))
	if err != nil {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(fmt.Sprintf("uid convert error: %v", err))
		return
	}
	data, err := args.strg.GetOrders(args.r.Context(), uid)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(fmt.Sprintf("get orders list error: %v", err))
		return
	}
	if data == nil {
		args.w.WriteHeader(http.StatusNoContent)
	}
	args.w.Header().Add(contentTypeString, ctApplicationJsonString)
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnln(fmt.Sprintf(writeResponceErrorString, err))
	}
}

func UserBalance(args RequestResponce) {
	args.logger.Debug("user balance request")
	data, err := args.strg.GetUserBalance(args.r.Context(), args.r.Header.Get(authHeader))
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnln(fmt.Sprintf("get orders list error: %v", err))
		return
	}
	args.w.Header().Add(contentTypeString, ctApplicationJsonString)
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnln(fmt.Sprintf(writeResponceErrorString, err))
	}
}

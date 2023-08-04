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
	Close() error
	IsUniqueViolation(error) bool
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

// Register ...
// @Tags Авторизация
// @Summary Регистрация нового пользователя в микросервисе
// @Accept json
// @Param params body LoginPassword true "Логи и пароль пользователя в формате json"
// @Router /user/register [post]
// @Success 200 "Успешная регистрация пользователя"
// @Header 200 {string} Authorization "Токен авторизации"
// @failure 400 "Ошибка в теле запроса. Тело запроса не соответствует json формату"
// @failure 409 "Такой логин уже используется другим пользователем"
// @failure 500 "Внутренняя ошибка сервиса".
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
		status := http.StatusInternalServerError
		err = fmt.Errorf(gormError, err)
		if strg.IsUniqueViolation(err) {
			status = http.StatusConflict
			err = fmt.Errorf("user registrating duplicate error: '%s'", user.Login)
		}
		return "", status, err
	}
	token, err := middlewares.CreateToken(key, tokenLiveTime, uid, ua, ip)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf(tokenGenerateError, err)
	}
	return token, http.StatusOK, nil
}

// Login ...
// @Tags Авторизация
// @Summary Авторизация пользователя в микросервисе
// @Accept json
// @Param params body LoginPassword true "Логи и пароль пользователя в формате json"
// @Router /user/login [post]
// @Success 200 "Успешная авторизация"
// @Header 200 {string} Authorization "Токен авторизации"
// @failure 400 "Ошибка в теле запроса. Тело запроса не соответствует json формату"
// @failure 401 "Логин или пароль не найден"
// @failure 500 "Внутренняя ошибка сервиса".
func Login(ctx context.Context, body, key []byte, remoteAddr, ua string,
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
	token, err := middlewares.CreateToken(key, tokenLiveTime, uid, ua, ip)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf(tokenGenerateError, err)
	}
	return token, http.StatusOK, nil
}

// AddOrder ...
// @Tags Заказы
// @Summary Добавление номера заказа пользователя
// @Accept json
// @Param order body string true "Номер заказа"
// @Security ApiKeyAuth
// @Param Authorization header string false "Токен авторизации"
// @Router /user/orders [post]
// @Success 200 "Заказ уже был добавлен пользователем ранее"
// @Success 202 "Заказ успешно зарегистрирован за пользователем"
// @failure 400 "Ошибка в теле запроса. Тело запроса пустое"
// @failure 401 "Пользователь не авторизован"
// @failure 409 "Заказ зарегистрирован за другим пользователем"
// @failure 422 "Номер заказа не прошёл проверку подлинности"
// @failure 500 "Внутренняя ошибка сервиса".
func AddOrder(args requestResponce) {
	body, err := io.ReadAll(args.r.Body)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf(readRequestErrorString, err)
		return
	}
	defer args.r.Body.Close() //nolint:errcheck // <-senselessly
	if len(body) == 0 {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnln("empty add order request's body")
		return
	}
	err = checkOrderNumber(string(body))
	if err != nil {
		args.w.WriteHeader(http.StatusUnprocessableEntity)
		args.logger.Warnf("check order error: %w", err)
		return
	}
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	status, err := args.strg.AddOrder(args.r.Context(), uid, string(body))
	if err != nil {
		args.logger.Warnf("add order error: %w", err)
	}
	args.w.WriteHeader(status)
}

func getListCommon(args *requestResponce, name string, f func(context.Context, int) ([]byte, error)) {
	args.logger.Debugf("%s list request", name)
	args.w.Header().Add(contentTypeString, ctApplicationJSONString)
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	data, err := f(args.r.Context(), uid)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("%s get list error: %w", name, err)
		return
	}
	if data == nil {
		args.w.WriteHeader(http.StatusNoContent)
	} else {
		args.w.WriteHeader(http.StatusOK)
	}
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnf(writeResponceErrorString, err)
	}
}

// GetOrdersList ...
// @Tags Заказы
// @Summary Запрос списка заказов, зарегистрированных за пользователем
// @Accept json
// @Produce json
// @Router /user/orders [get]
// @Security ApiKeyAuth
// @Param Authorization header string false "Токен авторизации"
// @Success 200 {array} storage.Orders "Список зарегистрированных за пользователем заказов"
// @failure 204 "Нет данных для ответа"
// @failure 401 "Пользователь не авторизован"
// @failure 500 "Внутренняя ошибка сервиса".
func GetOrdersList(args requestResponce) {
	getListCommon(&args, "orders", args.strg.GetOrders)
}

// GetUserBalance ...
// @Tags Баланс пользователя
// @Summary Запрос баланса пользователя
// @Produce json
// @Security ApiKeyAuth
// @Param Authorization header string false "Токен авторизации"
// @Router /user/balance [get]
// @Success 200 {object} storage.BalanceStruct "Баланс пользователя"
// @failure 401 "Пользователь не авторизован"
// @failure 500 "Внутренняя ошибка сервиса".
func GetUserBalance(args requestResponce) {
	args.logger.Debug("user balance request")
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	data, err := args.strg.GetUserBalance(args.r.Context(), uid)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("get user balance error: %w", err)
		return
	}
	args.w.Header().Add(contentTypeString, ctApplicationJSONString)
	_, err = args.w.Write(data)
	if err != nil {
		args.logger.Warnf(writeResponceErrorString, err)
	}
}

// AddWithdraw ...
// @Tags Списание баллов
// @Summary Запрос на списание баллов в счёт другого заказа
// @Accept json
// @Param withdraw body Withdraw true "Номер заказа в счет которого списываются баллы"
// @Security ApiKeyAuth
// @Param Authorization header string false "Токен авторизации"
// @Router /user/balance/withdraw [post]
// @Success 200 "Списание успешно добавлено"
// @failure 400 "Ошибка в теле запроса. Тело запроса не соответствует формату json"
// @failure 401 "Пользователь не авторизован"
// @failure 402 "Недостаточно средств"
// @failure 409 "Заказ уже был зарегистрирован ранее"
// @failure 422 "Номер заказа не прошёл проверку подлинности"
// @failure 500 "Внутренняя ошибка сервиса".
func AddWithdraw(args requestResponce) {
	body, err := io.ReadAll(args.r.Body)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("body read error: %w", err)
		return
	}
	var withdraw Withdraw
	err = json.Unmarshal(body, &withdraw)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf("convert to json error: %w", err)
		return
	}
	args.logger.Debugf("add withdraw request %s: %f", withdraw.Order, withdraw.Sum)
	err = checkOrderNumber(withdraw.Order)
	if err != nil {
		args.w.WriteHeader(http.StatusUnprocessableEntity)
		args.logger.Warnf("check order error", err)
		return
	}
	uid, ok := args.r.Context().Value(middlewares.AuthUID).(int)
	if !ok {
		args.w.WriteHeader(http.StatusUnauthorized)
		args.logger.Warnln(uidContextTypeError)
		return
	}
	status, err := args.strg.AddWithdraw(args.r.Context(), uid, withdraw.Order, withdraw.Sum)
	if err != nil {
		args.logger.Warnf("add withdraw error: %w", err)
	}
	args.logger.Debugf("add withdraw status: %d \n", status)
	args.w.WriteHeader(status)
}

// GetWithdrawsList ...
// @Tags Списание баллов
// @Summary Запрос списка списаний баллов
// @Produce json
// @Router /user/withdrawals [get]
// @Security ApiKeyAuth
// @Param Authorization header string false "Токен авторизации"
// @Success 200 {array} storage.Withdraws "Список списаний"
// @failure 204 "Нет данных для ответа"
// @failure 401 "Пользователь не авторизован"
// @failure 500 "Внутренняя ошибка сервиса".
func GetWithdrawsList(args requestResponce) {
	getListCommon(&args, "withdraws", args.strg.GetWithdraws)
}

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gostuding/goMarket/internal/server/middlewares"
	"go.uber.org/zap"
)

type RequestResponce struct {
	r      *http.Request
	w      http.ResponseWriter
	strg   Storage
	logger *zap.SugaredLogger
}

type CheckOrdersStorage interface {
	GetAccrualOrders() []string
	SetOrderData(string, string, float32) error
}

type ordersStatus struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

func makeRouter(strg Storage, logger *zap.SugaredLogger, key []byte, tokenLiveTime int) http.Handler {
	exceptURLs := make([]string, 0)
	var registerURL = "/api/user/register"
	var loginURL = "/api/user/login"
	var userOrders = "/api/user/orders"
	exceptURLs = append(exceptURLs, registerURL, loginURL)

	router := chi.NewRouter()
	router.Use(middleware.RealIP, middlewares.AuthMiddleware(logger, exceptURLs, loginURL, key),
		middlewares.GzipMiddleware(logger), middleware.Recoverer)

	router.Post(registerURL, func(w http.ResponseWriter, r *http.Request) {
		rr := RequestResponce{r: r, w: w, strg: strg, logger: logger}
		Registration(&RegisterStruct{RequestResponce: rr, key: key, tokenLiveTime: tokenLiveTime})
	})
	router.Post(loginURL, func(w http.ResponseWriter, r *http.Request) {
		rr := RequestResponce{r: r, w: w, strg: strg, logger: logger}
		Login(&RegisterStruct{RequestResponce: rr, key: key, tokenLiveTime: tokenLiveTime})
	})
	router.Post(userOrders, func(w http.ResponseWriter, r *http.Request) {
		OrdersAdd(RequestResponce{r: r, w: w, strg: strg, logger: logger})
	})
	router.Get(userOrders, func(w http.ResponseWriter, r *http.Request) {
		OrdersList(RequestResponce{r: r, w: w, strg: strg, logger: logger})
	})
	router.Get("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
		UserBalance(RequestResponce{r: r, w: w, strg: strg, logger: logger})
	})
	router.Post("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
		AddWithdraw(RequestResponce{r: r, w: w, strg: strg, logger: logger})
	})
	router.Get("/api/user/withdrawals", func(w http.ResponseWriter, r *http.Request) {
		WithdrawsList(RequestResponce{r: r, w: w, strg: strg, logger: logger})
	})

	return router
}

func RunServer(cfg *Config, strg Storage, logger *zap.SugaredLogger) error {
	if cfg == nil {
		return fmt.Errorf("server options error")
	}
	logger.Infoln("Run server at adress: ", cfg.ServerAddress)
	handler := makeRouter(strg, logger, cfg.AuthSecretKey, cfg.AuthTokenLiveTime)
	go timeRequest(fmt.Sprintf("%s/api/orders", cfg.AccuralAddress), logger, strg)
	return http.ListenAndServe(cfg.ServerAddress, handler) //nolint:wrapcheck // <- senselessly
}

func timeRequest(url string, logger *zap.SugaredLogger, strg CheckOrdersStorage) {
	updateTicker := time.NewTicker(time.Second)
	defer updateTicker.Stop()
	for {
		<-updateTicker.C
		for _, order := range strg.GetAccrualOrders() {
			logger.Debugln("order accrual request", order)
			secs, err := accrualRequest(fmt.Sprintf("%s/%s", url, order), strg)
			if err != nil {
				logger.Warnln("accural request error", err)
			}
			if secs > 0 {
				logger.Debugln("wait accural system", secs, "seconds")
				time.Sleep(time.Duration(secs) * time.Second)
			}
		}
	}
}

func accrualRequest(url string, strg CheckOrdersStorage) (int, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request error: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		wait, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			return 60, fmt.Errorf("too many requests. Default wait time. error: %w", err)
		}
		return wait, errors.New("too many  requests")
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("responce (%s) status code incorrect: %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("responce body read error: %w", err)
	}
	var item ordersStatus
	err = json.Unmarshal(data, &item)
	if err != nil {
		return 0, fmt.Errorf("json conver error: %w", err)
	}
	return 0, strg.SetOrderData(item.Order, item.Status, item.Accrual)
}

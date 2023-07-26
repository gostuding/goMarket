package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gostuding/goMarket/internal/server/middlewares"
	"go.uber.org/zap"

	_ "github.com/gostuding/goMarket/docs"
	"github.com/swaggo/http-swagger/v2"
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
	exceptURLs = append(exceptURLs, registerURL, loginURL, "/swagger/", "/favicon.ico")

	router := chi.NewRouter()
	router.Use(middleware.RealIP, middlewares.AuthMiddleware(logger, exceptURLs, loginURL, key),
		middlewares.GzipMiddleware(logger), middleware.Recoverer)

	router.Post(registerURL, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Warnln(readRequestErrorString, err)
			return
		}
		token, status, err := Register(r.Context(), body, key, r.RemoteAddr, r.UserAgent(), strg, tokenLiveTime)
		if err != nil {
			logger.Warnln("registration error", err)
		}
		w.Header().Set(authHeader, token)
		w.WriteHeader(status)
	})

	router.Post(loginURL, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Warnln(readRequestErrorString, err)
			return
		}
		token, status, err := LoginFunc(r.Context(), body, key, r.RemoteAddr, r.UserAgent(), strg, tokenLiveTime)
		if err != nil {
			logger.Warnln("user login error", err)
		}
		w.Header().Set(authHeader, token)
		w.WriteHeader(status)
	})

	router.Post(userOrders, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Warnln(readRequestErrorString, err)
			return
		}
		if len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			logger.Warnln("empty add order request's body")
			return
		}
		status, err := OrdersAddFunc(r.Context(), string(body), strg)
		if err != nil {
			logger.Warnln("add order error", err)
		}
		w.WriteHeader(status)
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

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		fileBytes, err := ioutil.ReadFile("./static/icon.png")
		if err != nil {
			logger.Warnln("icon not found", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(fileBytes)
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
	defer resp.Body.Close() //nolint:errcheck // <- senselessly
	if resp.StatusCode == http.StatusTooManyRequests {
		wait, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			wait = 60 //nolit:gomnd // <- default value
			return wait, fmt.Errorf("too many requests. Default wait time. error: %w", err)
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
	return 0, strg.SetOrderData(item.Order, item.Status, item.Accrual) //nolint:wrapcheck // <- wrapped early
}

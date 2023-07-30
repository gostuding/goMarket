package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/gostuding/goMarket/docs"
	"github.com/gostuding/goMarket/internal/server/middlewares"
	"go.uber.org/zap"

	httpSwagger "github.com/swaggo/http-swagger/v2"
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

func LoginOrRegister(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, key []byte,
	strg Storage, tlt int,
	function func(context.Context, []byte, []byte, string, string, Storage, int) (string, int, error)) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Warnln(readRequestErrorString, err)
		return
	}
	token, status, err := function(r.Context(), body, key, r.RemoteAddr, r.UserAgent(), strg, tlt)
	if err != nil {
		logger.Warnln("error", err)
	}
	w.Header().Set(authHeader, token)
	w.WriteHeader(status)
}

func makeRouter(strg Storage, logger *zap.SugaredLogger, key []byte, tokenLiveTime int, address string) http.Handler {
	exceptURLs := make([]string, 0)
	var registerURL = "/api/user/register"
	var loginURL = "/api/user/login"
	var userOrders = "/api/user/orders"
	exceptURLs = append(exceptURLs, registerURL, loginURL, "/swagger/", iconPath)

	router := chi.NewRouter()

	docs.SwaggerInfo.Host = address

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
	}))

	router.Use(middleware.RealIP, middlewares.AuthMiddleware(logger, exceptURLs, loginURL, key),
		middlewares.GzipMiddleware(logger), middleware.Recoverer)

	router.Post(registerURL, func(w http.ResponseWriter, r *http.Request) {
		LoginOrRegister(w, r, logger, key, strg, tokenLiveTime, Register)
	})

	router.Post(loginURL, func(w http.ResponseWriter, r *http.Request) {
		LoginOrRegister(w, r, logger, key, strg, tokenLiveTime, LoginFunc)
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
		httpSwagger.URL(fmt.Sprintf("http://%s/swagger/doc.json", address)),
	))

	router.Get(iconPath, func(w http.ResponseWriter, r *http.Request) {
		fileBytes, err := os.ReadFile("./static/icon.png")
		if err != nil {
			logger.Warnln("icon not found", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write(fileBytes)
		if err != nil {
			logger.Warnln("write icon file error", err)
		}
	})

	return router
}

func RunServer(cfg *Config, strg Storage, logger *zap.SugaredLogger) error {
	if cfg == nil {
		return fmt.Errorf("server options error")
	}
	logger.Infoln("Run server at adress: ", cfg.ServerAddress)
	handler := makeRouter(strg, logger, cfg.AuthSecretKey, cfg.AuthTokenLiveTime, cfg.ServerAddress)
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunc()

	serverFinishError := make(chan error, 1)
	srv := http.Server{Addr: cfg.ServerAddress, Handler: handler}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go timeRequest(ctx, &wg, fmt.Sprintf("%s/api/orders", cfg.AccuralAddress), logger, strg)
	wg.Add(1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverFinishError <- nil
		} else {
			serverFinishError <- err
			logger.Warnf("server lister error: %w", err)
		}
		logger.Debugln("Server listen finished")
		wg.Done()
	}()

	go func(ctx context.Context) {
		<-ctx.Done()
		shtCtx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
		defer cancelFunc()
		if err := srv.Shutdown(shtCtx); err != nil {
			logger.Warnln("shutdown server erorr: ", err)
		}
	}(ctx)

	wg.Wait()
	return <-serverFinishError
}

func timeRequest(ctxStop context.Context, wg *sync.WaitGroup, url string,
	logger *zap.SugaredLogger, strg CheckOrdersStorage) {
	updateTicker := time.NewTicker(time.Second)
	defer updateTicker.Stop()
	sleepTime := time.Now()
	for {
		select {
		case <-updateTicker.C:
			if !time.Now().After(sleepTime) {
				break
			}
			for _, order := range strg.GetAccrualOrders() {
				secs, err := accrualRequest(fmt.Sprintf("%s/%s", url, order), strg)
				if err != nil {
					if errors.Is(err, syscall.ECONNREFUSED) {
						logger.Debugln("accureal system connection refised")
						break
					}
					logger.Warnln("accural request error", err)
				}
				if secs > 0 {
					logger.Debugln("wait accural system", secs, "seconds")
					sleepTime = time.Now().Add(time.Duration(time.Duration(secs).Seconds()))
				}
			}
		case <-ctxStop.Done():
			logger.Debugln("Accrual gorutine finished")
			wg.Done()
			return
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
		return 0, fmt.Errorf("do request error: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // <- senselessly
	if resp.StatusCode == http.StatusTooManyRequests {
		wait, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			return manyRequestsWaitTimeDef, fmt.Errorf("too many requests. Default wait time. error: %w", err)
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
	err = strg.SetOrderData(item.Order, item.Status, item.Accrual)
	if err != nil {
		return 0, fmt.Errorf("set order data error: %w", err)
	}
	return 0, nil
}

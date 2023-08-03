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

type ServerConfig struct {
	ServerAddress          string
	AccuralAddress         string
	AuthSecretKey          []byte
	AuthTokenLiveTime      int
	AccrualRequestInterval int
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerAddress:          "localhost:8080",
		AccuralAddress:         "http://localhost:8081",
		AccrualRequestInterval: defaultAccrualRequestInterval,
		AuthTokenLiveTime:      defaultAuthTokenLiveTime,
	}
}

type requestResponce struct {
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

func loginRegistrationCommon(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, key []byte,
	strg Storage, tlt int,
	mainFunc func(context.Context, []byte, []byte, string, string, Storage, int) (string, int, error)) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Warnf(readRequestErrorString, err)
		return
	}
	token, status, err := mainFunc(r.Context(), body, key, r.RemoteAddr, r.UserAgent(), strg, tlt)
	if err != nil {
		logger.Warnf("storage error: %w", err)
	}
	w.Header().Set("Authorization", token)
	w.WriteHeader(status)
}

func makeRouter(strg Storage, logger *zap.SugaredLogger, key []byte, tokenLiveTime int, address string) http.Handler {
	var loginURL = "/api/user/login"
	var ordersListURL = "/api/user/orders"
	router := chi.NewRouter()
	docs.SwaggerInfo.Host = address
	router.Use(middleware.RealIP, middlewares.GzipMiddleware(logger), middleware.Recoverer,
		cors.Handler(cors.Options{
			AllowedOrigins: []string{"https://*", "http://*"},
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		}),
	)

	router.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
		loginRegistrationCommon(w, r, logger, key, strg, tokenLiveTime, Register)
	})

	router.Post(loginURL, func(w http.ResponseWriter, r *http.Request) {
		loginRegistrationCommon(w, r, logger, key, strg, tokenLiveTime, Login)
	})

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://%s/swagger/doc.json", address)),
	))

	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		fileBytes, err := os.ReadFile("./static/icon.png")
		if err != nil {
			logger.Warnf("icon not found: %w", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write(fileBytes)
		if err != nil {
			logger.Warnf("write icon file error: %w", err)
		}
	})

	router.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware(logger, loginURL, key))

		r.Get(ordersListURL, func(w http.ResponseWriter, r *http.Request) {
			GetOrdersList(requestResponce{r: r, w: w, strg: strg, logger: logger})
		})

		r.Post(ordersListURL, func(w http.ResponseWriter, r *http.Request) {
			AddOrder(requestResponce{r: r, w: w, strg: strg, logger: logger})
		})

		r.Get("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
			GetUserBalance(requestResponce{r: r, w: w, strg: strg, logger: logger})
		})

		r.Post("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
			AddWithdraw(requestResponce{r: r, w: w, strg: strg, logger: logger})
		})

		r.Get("/api/user/withdrawals", func(w http.ResponseWriter, r *http.Request) {
			GetWithdrawsList(requestResponce{r: r, w: w, strg: strg, logger: logger})
		})
	})

	return router
}

func RunServer(cfg *ServerConfig, strg Storage, logger *zap.SugaredLogger) error {
	if cfg == nil {
		return errors.New("server options is nil")
	}
	logger.Infof("Run server at adress: %s", cfg.ServerAddress)
	handler := makeRouter(strg, logger, cfg.AuthSecretKey, cfg.AuthTokenLiveTime, cfg.ServerAddress)
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunc()

	serverFinishError := make(chan error, 1)
	srv := http.Server{Addr: cfg.ServerAddress, Handler: handler}
	go timeRequest(ctx, fmt.Sprintf("%s/api/orders", cfg.AccuralAddress),
		logger, strg, cfg.AccrualRequestInterval)

	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverFinishError <- nil
		} else {
			serverFinishError <- err
			logger.Warnf("server lister error: %w", err)
		}
		logger.Debugln("Server listen finished")
		if err := strg.Close(); err != nil {
			logger.Warnf("close storage connection error: %w", err)
		}
		close(serverFinishError)
	}()

	go func() {
		<-ctx.Done()
		shtCtx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
		defer cancelFunc()
		if err := srv.Shutdown(shtCtx); err != nil {
			logger.Warnf("shutdown server erorr: %w", err)
		}
	}()

	return <-serverFinishError
}

func timeRequest(ctxStop context.Context, url string,
	logger *zap.SugaredLogger, strg CheckOrdersStorage, interval int) {
	updateTicker := time.NewTicker(time.Duration(interval) * time.Second)
	defer updateTicker.Stop()
	sleepTime := time.Now()
	sleepChan := make(chan int, 1)
	errorChan := make(chan error, 1)

	go func() {
		for {
			select {
			case secs := <-sleepChan:
				logger.Debugf("wait accural system %d seconds", secs)
				sleepTime = time.Now().Add(time.Duration(time.Duration(secs).Seconds()))
			case err := <-errorChan:
				if errors.Is(err, syscall.ECONNREFUSED) {
					logger.Debugln("accureal system connection refised")
				} else {
					logger.Warnf("accural request error: %w", err)
				}
			case <-ctxStop.Done():
				return
			}
		}
	}()

	wg := sync.WaitGroup{}
	for {
		select {
		case <-updateTicker.C:
			if !time.Now().After(sleepTime) {
				break
			}
			for _, order := range strg.GetAccrualOrders() {
				wg.Add(1)
				go accrualRequest(fmt.Sprintf("%s/%s", url, order), strg, sleepChan, errorChan, &wg)
			}
			wg.Wait()
		case <-ctxStop.Done():
			close(sleepChan)
			close(errorChan)
			logger.Debugln("Accrual gorutine finished")
			return
		}
	}
}

func accrualRequest(url string, strg CheckOrdersStorage, sleepChan chan int,
	errorChan chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		errorChan <- fmt.Errorf("create request error: %w", err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		errorChan <- fmt.Errorf("do request error: %w", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck // <- senselessly
	if resp.StatusCode == http.StatusTooManyRequests {
		wait, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			sleepChan <- manyRequestsWaitTimeDef
			errorChan <- fmt.Errorf("too many requests. Default wait time. error: %w", err)
			return
		}
		sleepChan <- wait
		return
	}
	if resp.StatusCode != http.StatusOK {
		errorChan <- fmt.Errorf("responce (%s) status code incorrect: %d", url, resp.StatusCode)
		return
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		errorChan <- fmt.Errorf("responce body read error: %w", err)
		return
	}
	var item ordersStatus
	err = json.Unmarshal(data, &item)
	if err != nil {
		errorChan <- fmt.Errorf("json conver error: %w", err)
		return
	}
	err = strg.SetOrderData(item.Order, item.Status, item.Accrual)
	if err != nil {
		errorChan <- fmt.Errorf("set order data error: %w", err)
	}
}

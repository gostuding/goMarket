package server

import (
	"fmt"
	"net/http"

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

func makeRouter(strg Storage, logger *zap.SugaredLogger, key []byte, tokenLiveTime int) http.Handler {
	exceptURLs := make([]string, 0)
	var registerURL = "/api/user/register"
	var loginURL = "/api/user/login"
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
	router.Post("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
		// GetMetric(w, r, storage, getParams(r), logger, key)
	})
	router.Get("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
		// UpdateJSON(w, r, storage, logger, key)
	})
	router.Post("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
		// UpdateJSONSLice(w, r, storage, logger, key)
	})
	router.Get("/api/user/withdrawal", func(w http.ResponseWriter, r *http.Request) {
		// Ping(w, r, storage, logger)
	})

	return router
}

func RunServer(cfg *Config, strg Storage, logger *zap.SugaredLogger) error {
	if cfg == nil {
		return fmt.Errorf("server options error")
	}
	logger.Infoln("Run server at adress: ", cfg.ServerAddress)
	handler := makeRouter(strg, logger, cfg.AuthSecretKey, cfg.AuthTokenLiveTime)
	return http.ListenAndServe(cfg.ServerAddress, handler) //nolint:wrapcheck // <- senselessly
}

package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

type Storage interface {
	Ping(context.Context) error
}

func makeRouter(strg Storage, logger *zap.SugaredLogger) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, gzipMiddleware(logger), middleware.Recoverer)

	router.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
		// GetAllMetrics(w, r, storage, logger, key)
	})
	router.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
		// GetMetricJSON(w, r, storage, logger, key)
	})
	router.Post("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
		// GetMetric(w, r, storage, getParams(r), logger, key)
	})
	router.Get("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
		// Update(w, r, storage, updateParams(r), logger)
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
	return http.ListenAndServe(cfg.ServerAddress, makeRouter(strg, logger))
}

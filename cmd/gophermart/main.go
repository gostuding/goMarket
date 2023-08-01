package main

import (
	"log"

	"github.com/gostuding/goMarket/internal/logger"
	"github.com/gostuding/goMarket/internal/server"
	"github.com/gostuding/goMarket/internal/storage"
)

// @title Gophermart API
// @version 1.0
// @contact.name API Support
// @contact.email mag-nat1@yandex.ru
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description API для микросервиса накопительной системы лояльности «Гофермарт».

func main() {
	cfg := server.NewConfig()
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Init logger error: %v", err)
	}
	strg, err := storage.NewPSQLStorage(cfg.DBConnect, cfg.DBConnectionPull)
	if err != nil {
		logger.Fatalf("Create storage error: %v", err)
	}
	err = server.RunServer(cfg, strg, logger)
	if err != nil {
		logger.Fatalf("Run server error: %v", err)
	}
}

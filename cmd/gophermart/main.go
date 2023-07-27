package main

import (
	"log"

	"github.com/gostuding/goMarket/internal/logger"
	"github.com/gostuding/goMarket/internal/server"
	"github.com/gostuding/goMarket/internal/storage"
)

// @title Gophermart API
// @version 1.0
// @description API для микросервиса накопительной системы лояльности «Гофермарт»

// @contact.name API Support
// @contact.email mag-nat1@yandex.ru

// @host localhost:8080
// @BasePath /api

//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Authorization

func main() {
	cfg := server.NewConfig()
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	strg, err := storage.NewPSQLStorage(cfg.DBConnect)
	if err != nil {
		log.Fatal(err)
	}
	err = server.RunServer(cfg, strg, logger)
	if err != nil {
		logger.Fatalln(err)
	}
}

package main

import (
	"log"

	"github.com/gostuding/goMarket/internal/logger"
	"github.com/gostuding/goMarket/internal/server"
	"github.com/gostuding/goMarket/internal/storage"
)

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

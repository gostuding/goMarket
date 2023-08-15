package main

import (
	"flag"
	"log"
	"os"

	"github.com/gostuding/goMarket/internal/logger"
	"github.com/gostuding/goMarket/internal/server"
	"github.com/gostuding/goMarket/internal/storage"
)

type Config struct {
	ServerCfg  *server.ServerConfig
	StorageCfg *storage.StorageConfig
}

func envValue(value string, name string) string {
	env, ok := os.LookupEnv(name)
	if ok {
		return env
	}
	return value
}

func NewConfig() *Config {
	cfg := Config{
		ServerCfg:  server.NewServerConfig(),
		StorageCfg: storage.NewStorageConfig(),
	}
	key := "default"
	cfg.ServerCfg.ServerAddress = envValue(cfg.ServerCfg.ServerAddress, "RUN_ADDRESS")
	cfg.ServerCfg.AccuralAddress = envValue(cfg.ServerCfg.AccuralAddress, "ACCRUAL_SYSTEM_ADDRESS")
	key = envValue(key, "TOKEN_KEY")
	cfg.StorageCfg.DBConnect = envValue(cfg.StorageCfg.DBConnect, "DATABASE_URI")

	flag.StringVar(&cfg.ServerCfg.ServerAddress, "a", cfg.ServerCfg.ServerAddress,
		"адрес и порт запуска сервиса в формате ip:port")
	flag.StringVar(&cfg.ServerCfg.AccuralAddress, "r", cfg.ServerCfg.AccuralAddress,
		"адрес системы расчёта начислений")
	flag.IntVar(&cfg.ServerCfg.AccrualRequestInterval, "ri", cfg.ServerCfg.AccrualRequestInterval,
		"интервал запросов к системе расчета начислений (секунды)")
	flag.IntVar(&cfg.ServerCfg.AuthTokenLiveTime, "t", cfg.ServerCfg.AuthTokenLiveTime,
		"время жизни токена авторизации (секунды)")
	flag.StringVar(&key, "k", key, "ключ для формарования токена авторизации")
	flag.StringVar(&cfg.StorageCfg.DBConnect, "d", cfg.StorageCfg.DBConnect,
		"строка для подключения к базе данных")
	flag.IntVar(&cfg.StorageCfg.DBConnectionPull, "pc", cfg.StorageCfg.DBConnectionPull,
		"максимальное количество открытых соединений с БД")
	flag.Parse()
	cfg.ServerCfg.AuthSecretKey = []byte(key)
	return &cfg
}

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
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Init logger error: %v", err)
	}
	cfg := NewConfig()
	strg, err := storage.NewPSQLStorage(cfg.StorageCfg)
	if err != nil {
		logger.Fatalf("Create storage error: %v", err)
	}
	err = server.RunServer(cfg.ServerCfg, strg, logger)
	if err != nil {
		logger.Fatalf("Run server error: %v", err)
	}
}

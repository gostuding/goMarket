package server

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress  string
	DBConnect      string
	AccuralAddress string
}

func envValue(value string, name string) string {
	if osValue := os.Getenv(name); osValue != "" {
		return osValue
	}
	return value
}

func NewConfig() *Config {
	cfg := Config{
		ServerAddress:  "localhost:8080",
		DBConnect:      "host=localhost user=postgres database=market",
		AccuralAddress: "localhost:8081",
	}
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес и порт запуска сервиса в формате ip:port")
	flag.StringVar(&cfg.DBConnect, "d", cfg.DBConnect, "адрес подключения к базе данных")
	flag.StringVar(&cfg.AccuralAddress, "r", cfg.AccuralAddress, "адрес системы расчёта начислений")
	flag.Parse()

	cfg.ServerAddress = envValue(cfg.ServerAddress, "RUN_ADDRESS")
	cfg.DBConnect = envValue(cfg.DBConnect, "DATABASE_URI")
	cfg.AccuralAddress = envValue(cfg.AccuralAddress, "ACCRUAL_SYSTEM_ADDRESS")
	return &cfg
}

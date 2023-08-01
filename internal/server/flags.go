package server

import (
	"flag"
	"os"
)

const (
	defaultDBPullCount            = 100
	defaultAuthTokenLiveTime      = 3600
	defaultAccrualRequestInterval = 1
)

type Config struct {
	ServerAddress          string
	DBConnect              string
	AccuralAddress         string
	AuthSecretKey          []byte
	AuthTokenLiveTime      int
	DBConnectionPull       int
	AccrualRequestInterval int
}

func envValue(value string, name string) string {
	if osValue := os.Getenv(name); osValue != "" {
		return osValue
	}
	return value
}

func NewConfig() *Config {
	key := "default"
	cfg := Config{
		ServerAddress:          "localhost:8080",
		DBConnect:              "host=localhost user=postgres database=market",
		AccuralAddress:         "http://localhost:8081",
		AuthTokenLiveTime:      defaultAuthTokenLiveTime,
		DBConnectionPull:       defaultDBPullCount,
		AccrualRequestInterval: defaultAccrualRequestInterval,
	}

	cfg.ServerAddress = envValue(cfg.ServerAddress, "RUN_ADDRESS")
	cfg.DBConnect = envValue(cfg.DBConnect, "DATABASE_URI")
	cfg.AccuralAddress = envValue(cfg.AccuralAddress, "ACCRUAL_SYSTEM_ADDRESS")
	cfg.AuthSecretKey = []byte(envValue(key, "TOKEN_KEY"))

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес и порт запуска сервиса в формате ip:port")
	flag.StringVar(&cfg.DBConnect, "d", cfg.DBConnect, "адрес подключения к базе данных")
	flag.StringVar(&cfg.AccuralAddress, "r", cfg.AccuralAddress, "адрес системы расчёта начислений")
	flag.StringVar(&key, "k", key, "ключ для формарования токена авторизации")
	flag.IntVar(&cfg.AuthTokenLiveTime, "t", cfg.AuthTokenLiveTime, "время жизни токена авторизации (секунды)")
	flag.IntVar(&cfg.DBConnectionPull, "pc", cfg.DBConnectionPull, "количество открытых соединений с БД")
	flag.IntVar(&cfg.AccrualRequestInterval, "ri", cfg.AccrualRequestInterval, "интервал запросов к системе расчета начислений (секунды)")
	flag.Parse()
	return &cfg
}

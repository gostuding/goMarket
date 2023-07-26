# gopheramrt

Мини сервер по спринту 5-6

# Клонирование репозитория
```
git clone https://github.com/gostuding/goMarket
```
# Компиляция сервера

1. Перейти в папку `cmd/gophermart`
2. Выполнить команду `go build -ldflags "-s -w"`

# Запуск локальных тестов

`go test ./...`

# Swager

1. Запустить сервер командой `go run cmd/gophermart/.`
2. Открыть браузер по адресу: `http://localhost:8080/swagger/`
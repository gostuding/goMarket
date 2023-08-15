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

# Перед запуском сервера проверить наличие

1. СУБД postgres (см. документацию `https://postgrespro.ru/docs/postgresql/15/tutorial-install`)
2. База данный для сервера (по умолчанию используется название БД `market`) (см. документацию `https://postgrespro.ru/docs/postgresql/15/tutorial-createdb`)
3. Пользователь для работы с БД (по умолчанию используется `postgres`) (см. документацию `https://postgrespro.ru/docs/postgresql/15/app-createuser`)
4. В качестве параметра запуска сервиса передать соответствующую строку для подключения к базе данных (параметр -d)

# Параметры запуска

  -a string адрес и порт запуска сервиса в формате ip:port (default "localhost:8080")
  -d string строка подключения к базе данных (default "host=localhost user=postgres database=market")
  -k string ключ для формарования токена авторизации (default "default")
  -pc int максимальное количество соединений с БД (default 100)
  -r string адрес системы расчёта начислений (default "http://localhost:8081")
  -t int время жизни токена авторизации (секунды) (default 3600)


# Swager

1. Запустить сервер 
2. Открыть браузер по адресу: `http://$ADDRESS/swagger/` где `$ADDRESS` - адрес и порт сервера (default "localhost:8080")

# Запуск проверок golangci-lint в Linux (Local Installation)

1. Установить golangci-lint (`https://golangci-lint.run/usage/install/`)
2. Запустить в терминале файл `golint_run.sh` (./golint_run.sh)
3. Выявленные замечания будут записаны в файл `./golangci-lint/report.json`


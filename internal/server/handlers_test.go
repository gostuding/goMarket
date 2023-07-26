package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
	"github.com/gostuding/goMarket/internal/server/middlewares"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	uid := 1
	ctx := context.Background()
	unqueError := pgconn.PgError{Code: pgerrcode.UniqueViolation}
	m.EXPECT().Registration(ctx, "admin", "1", "ua", "127.0.0.1").Return(uid, nil)
	m.EXPECT().Registration(ctx, "repeat", "1", "ua", "127.0.0.1").Return(0, &unqueError)
	m.EXPECT().Registration(ctx, "user", "1", "ua", "127.0.0.1").Return(0, errors.New("database error"))

	type args struct {
		body          []byte
		key           []byte
		remoteAddr    string
		ua            string
		strg          Storage
		tokenLiveTime int
	}
	tests := []struct {
		name      string
		args      args
		want      string
		want1     int
		wantCheck bool
		wantErr   bool
	}{
		{
			name: "Успешная регистрация",
			args: args{
				body:       []byte(`{"login": "admin", "password": "1"}`),
				key:        []byte("default"),
				remoteAddr: "127.0.0.1:9000",
				ua:         "ua",
				strg:       m,
			},
			wantCheck: false,
			want:      "",
			want1:     http.StatusOK,
			wantErr:   false,
		},
		{
			name: "Повторная регистрация пользователя",
			args: args{
				body:       []byte(`{"login": "repeat", "password": "1"}`),
				key:        []byte("default"),
				remoteAddr: "127.0.0.1:9000",
				ua:         "ua",
				strg:       m,
			},
			wantCheck: true,
			want:      "",
			want1:     http.StatusConflict,
			wantErr:   true,
		},
		{
			name: "Пустой запрос на регистрацию пользователя",
			args: args{
				body:       nil,
				key:        []byte("default"),
				remoteAddr: "127.0.0.1:9000",
				ua:         "ua",
				strg:       m,
			},
			wantCheck: true,
			want:      "",
			want1:     http.StatusBadRequest,
			wantErr:   true,
		},
		{
			name: "Ошибка базы данных",
			args: args{
				body:       []byte(`{"login": "user", "password": "1"}`),
				key:        []byte("default"),
				remoteAddr: "127.0.0.1:9000",
				ua:         "ua",
				strg:       m,
			},
			wantCheck: true,
			want:      "",
			want1:     http.StatusInternalServerError,
			wantErr:   true,
		},
		{
			name: "Ошибка переданного ip",
			args: args{
				body:       []byte(`{"login": "user", "password": "1"}`),
				key:        []byte("default"),
				remoteAddr: "127.0.0.9000",
				ua:         "ua",
				strg:       m,
			},
			wantCheck: true,
			want:      "",
			want1:     http.StatusBadRequest,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := Register(ctx, tt.args.body, tt.args.key, tt.args.remoteAddr,
				tt.args.ua, tt.args.strg, tt.args.tokenLiveTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantCheck && got != tt.want {
				t.Errorf("Register() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Register() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestLoginFunc(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	uid := 1
	ctx := context.Background()
	m.EXPECT().Login(ctx, "admin", "1", "ua", "127.0.0.1").Return(uid, nil)
	m.EXPECT().Login(ctx, "noUser", "1", "ua", "127.0.0.1").Return(0, gorm.ErrRecordNotFound)
	m.EXPECT().Login(ctx, "user", "1", "ua", "127.0.0.1").Return(0, errors.New("internal error"))

	type args struct {
		body          []byte
		key           []byte
		remoteAddr    string
		ua            string
		strg          Storage
		tokenLiveTime int
	}
	tests := []struct {
		name      string
		args      args
		want      string
		want1     int
		checkWant bool
		wantErr   bool
	}{
		{
			name: "Успешная авторизация",
			args: args{
				body:          []byte(`{"login": "admin", "password": "1"}`),
				key:           []byte("default"),
				remoteAddr:    "127.0.0.1:9000",
				ua:            "ua",
				strg:          m,
				tokenLiveTime: 10,
			},
			checkWant: false,
			want:      "",
			want1:     http.StatusOK,
			wantErr:   false,
		},
		{
			name: "Пользователь не найден",
			args: args{
				body:          []byte(`{"login": "noUser", "password": "1"}`),
				key:           []byte("default"),
				remoteAddr:    "127.0.0.1:9000",
				ua:            "ua",
				strg:          m,
				tokenLiveTime: 10,
			},
			checkWant: true,
			want:      "",
			want1:     http.StatusUnauthorized,
			wantErr:   true,
		},
		{
			name: "Внутреняя ошибка БД",
			args: args{
				body:          []byte(`{"login": "user", "password": "1"}`),
				key:           []byte("default"),
				remoteAddr:    "127.0.0.1:9000",
				ua:            "ua",
				strg:          m,
				tokenLiveTime: 10,
			},
			checkWant: true,
			want:      "",
			want1:     http.StatusInternalServerError,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := LoginFunc(ctx, tt.args.body, tt.args.key, tt.args.remoteAddr,
				tt.args.ua, tt.args.strg, tt.args.tokenLiveTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoginFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkWant && got != tt.want {
				t.Errorf("LoginFunc() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("LoginFunc() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestOrdersAddFunc(t *testing.T) {
	type args struct {
		ctx   context.Context
		order string
		strg  Storage
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	ctx := context.Background()
	m.EXPECT().AddOrder(context.WithValue(ctx, middlewares.AuthUID, 1), 1, "55875248746").Return(http.StatusAccepted, nil)
	m.EXPECT().AddOrder(context.WithValue(ctx, middlewares.AuthUID, 2), 2, "55875248746").Return(http.StatusConflict, nil)
	m.EXPECT().AddOrder(context.WithValue(ctx, middlewares.AuthUID, 2), 2, "2377225624").Return(http.StatusOK, nil)

	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name:    "Успешное добавление",
			args:    args{ctx: context.WithValue(ctx, middlewares.AuthUID, 1), order: "55875248746", strg: m},
			want:    http.StatusAccepted,
			wantErr: false,
		},
		{
			name:    "Неправильный номер заказа",
			args:    args{ctx: ctx, order: "1", strg: m},
			want:    http.StatusUnprocessableEntity,
			wantErr: true,
		},
		{
			name:    "Пользователь не авторизован",
			args:    args{ctx: ctx, order: "55875248746", strg: m},
			want:    http.StatusUnauthorized,
			wantErr: true,
		},
		{
			name:    "Заказ добавлен другим пользователем",
			args:    args{ctx: context.WithValue(ctx, middlewares.AuthUID, 2), order: "55875248746", strg: m},
			want:    http.StatusConflict,
			wantErr: false,
		},
		{
			name:    "Заказ был добавлен ранее этим пользователем",
			args:    args{ctx: context.WithValue(ctx, middlewares.AuthUID, 2), order: "2377225624", strg: m},
			want:    http.StatusOK,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := OrdersAddFunc(tt.args.ctx, tt.args.order, tt.args.strg)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrdersAddFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OrdersAddFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

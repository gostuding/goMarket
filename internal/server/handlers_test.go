package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
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
		ctx           context.Context
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
		wantCheck bool
		want      string
		want1     int
		wantErr   bool
	}{
		{
			name: "Успешная регистрация",
			args: args{
				body:       []byte(`{"login": "admin", "password": "1"}`),
				ctx:        context.Background(),
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
				ctx:        context.Background(),
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
				ctx:        context.Background(),
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
				ctx:        context.Background(),
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
				ctx:        context.Background(),
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
			got, got1, err := Register(tt.args.ctx, tt.args.body, tt.args.key, tt.args.remoteAddr, tt.args.ua, tt.args.strg, tt.args.tokenLiveTime)
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
		ctx           context.Context
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
		checkWant bool
		want      string
		want1     int
		wantErr   bool
	}{
		{
			name: "Успешная авторизация",
			args: args{
				ctx:           ctx,
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
				ctx:           ctx,
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
				ctx:           ctx,
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
			got, got1, err := LoginFunc(tt.args.ctx, tt.args.body, tt.args.key, tt.args.remoteAddr, tt.args.ua, tt.args.strg, tt.args.tokenLiveTime)
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

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func testCommon(fName, got, want string, got1, want1 int, err error, wantError, wantCheck bool) error {
	if (err != nil) != wantError {
		return fmt.Errorf("%s error = %w, wantErr %v", fName, err, wantError)
	}
	if wantCheck && got != want {
		return fmt.Errorf("%s got = %v, want %v", fName, got, want)
	}
	if got1 != want1 {
		return fmt.Errorf("%s got1 = %v, want2 %v", fName, got1, want1)
	}
	return nil
}

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	uid := 1
	ctx := context.Background()
	unqueError := pgconn.PgError{Code: pgerrcode.UniqueViolation}
	errDB := errors.New("database error")
	m.EXPECT().Registration(ctx, "admin", gomock.Any(), "ua", "127.0.0.1").Return(uid, nil)
	m.EXPECT().Registration(ctx, "repeat", gomock.Any(), "ua", "127.0.0.1").Return(0, &unqueError)
	m.EXPECT().Registration(ctx, "user", gomock.Any(), "ua", "127.0.0.1").Return(0, errDB)
	m.EXPECT().IsUniqueViolation(fmt.Errorf("gorm error: %w", &unqueError)).Return(true)
	m.EXPECT().IsUniqueViolation(fmt.Errorf("gorm error: %w", errDB)).Return(false)

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
			if err = testCommon("Register()", got, tt.want, got1, tt.want1, err, tt.wantErr, tt.wantCheck); err != nil {
				t.Error(err.Error())
			}
		})
	}
}

func TestLoginFunc(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	uid := 1
	ctx := context.Background()
	m.EXPECT().Login(ctx, "admin", gomock.Any(), "ua", "127.0.0.1").Return(uid, nil)
	m.EXPECT().Login(ctx, "noUser", gomock.Any(), "ua", "127.0.0.1").Return(0, gorm.ErrRecordNotFound)
	m.EXPECT().Login(ctx, "user", gomock.Any(), "ua", "127.0.0.1").Return(0, errors.New("internal error"))

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
			got, got1, err := Login(ctx, tt.args.body, tt.args.key, tt.args.remoteAddr,
				tt.args.ua, tt.args.strg, tt.args.tokenLiveTime)
			if err = testCommon("Register()", got, tt.want, got1, tt.want1, err, tt.wantErr, tt.checkWant); err != nil {
				t.Error(err.Error())
			}
		})
	}
}

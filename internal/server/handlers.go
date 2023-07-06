package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gostuding/goMarket/internal/server/middlewares"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type Storage interface {
	Registration(context.Context, string, string, string, string) (int, error)
}

type RegisterStruct struct {
	RequestResponce
	key           []byte
	tokenLiveTime int
}

type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func validateLoginPassword(r *http.Request) (*LoginPassword, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body error: %w", err)
	}
	var user LoginPassword
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("body convert to json error: %w", err)
	}
	if user.Login == "" || user.Password == "" {
		return nil, errors.New("empty values for registration error")
	}
	return &user, nil
}

func addTokenInCookeis(token string, w http.ResponseWriter, liveTime int) {
	cookie := &http.Cookie{
		Name:    "token",
		Value:   token,
		MaxAge:  liveTime * int(time.Hour/time.Millisecond),
		Expires: time.Now().Add(time.Duration(liveTime) * time.Hour),
	}
	http.SetCookie(w, cookie)
}

func Registration(args *RegisterStruct) {
	user, err := validateLoginPassword(args.r)
	if err != nil {
		args.w.WriteHeader(http.StatusBadRequest)
		args.logger.Warnf("request validate error: %v", err)
		return
	}
	uid, err := args.strg.Registration(args.r.Context(), user.Login, user.Password,
		args.r.UserAgent(), args.r.RemoteAddr)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			args.w.WriteHeader(http.StatusConflict)
			args.logger.Infoln("user duplicate error", user.Login)
		} else {
			args.w.WriteHeader(http.StatusInternalServerError)
			args.logger.Warnf("gorm error: %w", err)
		}
		return
	}
	args.logger.Debugf("new user success registrated: '%s'", user.Login)
	token, err := middlewares.CreateToken(args.key, args.tokenLiveTime, uid, args.r.UserAgent(),
		user.Login, args.r.RemoteAddr)
	if err != nil {
		args.w.WriteHeader(http.StatusInternalServerError)
		args.logger.Warnf("token generation error: %w", err)
		return
	}
	args.w.Header().Add("Authorization", token)
	addTokenInCookeis(token, args.w, args.tokenLiveTime)
	args.w.WriteHeader(http.StatusOK)
}

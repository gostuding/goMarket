package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type uidstr string

const AuthUID uidstr = "uid"

type authJWTStruct struct {
	jwt.RegisteredClaims
	UserAgent string
	Login     string
	IP        string
	UID       int
}

func CreateToken(key []byte, liveTime, uid int, ua, login, ip string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authJWTStruct{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(liveTime) * time.Hour)),
		},
		UserAgent: ua,
		Login:     login,
		IP:        ip,
		UID:       uid,
	})
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign user token error: %w", err)
	}
	return tokenString, nil
}

func checkAuthToken(r *http.Request, key []byte) (int, error) {
	token := r.Header.Get(authString)
	if token == "" {
		for key, val := range r.Header {
			fmt.Println(key, val)
		}
		return 0, errors.New("token is empty")
	}
	claims := &authJWTStruct{}
	info, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return 0, fmt.Errorf("auth token parse error: %w", err)
	}
	if !info.Valid {
		return 0, errors.New("token is not valid")
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return 0, fmt.Errorf("user ip not equal to IP:port, error: %w", err)
	}
	if claims.UserAgent != r.UserAgent() || claims.IP != ip {
		return 0, errors.New("user data changed. Reauth requaged")
	}
	return claims.UID, nil
}

func AuthMiddleware(logger *zap.SugaredLogger, except []string,
	redirectURL string, key []byte) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, item := range except {
				if item == r.URL.Path {
					next.ServeHTTP(w, r)
					return
				}
			}
			uid, err := checkAuthToken(r, key)
			if err != nil {
				http.Redirect(w, r, redirectURL, http.StatusUnauthorized)
				logger.Warnf("%s authorization token error: %w", r.URL.Path, err)
				return
			}
			w.Header().Set(authString, r.Header.Get(authString))
			next.ServeHTTP(w, r.Clone(context.WithValue(r.Context(), AuthUID, uid)))
		}
		return http.HandlerFunc(fn)
	}
}

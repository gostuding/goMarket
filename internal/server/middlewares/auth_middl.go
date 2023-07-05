package middlewares

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type AuthJWTStruct struct {
	jwt.RegisteredClaims
	UserAgent string
	Login     string
	IP        string
	UID       int
}

func checkAuthToken(r *http.Request, key []byte) (string, error) {
	token := r.Header.Get(authString)
	if token == "" {
		return "", errors.New("token is empty")
	}
	claims := &AuthJWTStruct{}
	info, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return "", fmt.Errorf("auth token parse error: %w", err)
	}
	if !info.Valid {
		return "", errors.New("token is not valid")
	}
	if claims.UserAgent != r.UserAgent() || claims.IP != r.RemoteAddr {
		return "", errors.New("session ended. Login requaged")
	}
	r.Header.Set(authString, fmt.Sprintf("%d", claims.UID))
	return token, nil
}

type retFunc func(h http.Handler) http.Handler

func AuthMiddleware(logger *zap.SugaredLogger, except []string, redURL string, key []byte) retFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, item := range except {
				if item == r.URL.Path {
					next.ServeHTTP(w, r)
					return
				}
			}
			token, err := checkAuthToken(r, key)
			if err != nil {
				http.Redirect(w, r, redURL, http.StatusUnauthorized)
				logger.Warnf("user authorization token error: %w", err)
				return
			}
			w.Header().Set(authString, token)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

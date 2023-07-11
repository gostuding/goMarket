package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

func CreateToken(key []byte, liveTime, uid int, ua, login, ip string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthJWTStruct{
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

func checkAuthToken(r *http.Request, key []byte) (string, error) {
	token := r.Header.Get(authString)
	if token == "" {
		return "", errors.New("token is empty")
	}
	claims := &AuthJWTStruct{}
	info, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
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
	if claims.UserAgent != r.UserAgent() || claims.IP != strings.Split(r.RemoteAddr, ":")[0] {
		return "", errors.New("session ended. Login requaged")
	}
	r.Header.Set(authString, fmt.Sprintf("%d", claims.UID))
	return token, nil
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
			token, err := checkAuthToken(r, key)
			if err != nil {
				// http.Redirect(w, r, redirectURL, http.StatusUnauthorized)
				w.WriteHeader(http.StatusUnauthorized)
				logger.Warnf("%s authorization token error: %w", r.URL.Path, err)
				return
			}
			w.Header().Set(authString, token)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

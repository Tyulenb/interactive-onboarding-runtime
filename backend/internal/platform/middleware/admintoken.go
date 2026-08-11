package middleware

import (
	"net/http"
	"strings"
)

type AdminMiddleware struct {
	token string
}

func NewAdminMiddleware(token string) *AdminMiddleware {
	return &AdminMiddleware{
		token: token,
	}
}

func (a *AdminMiddleware) CheckAdminToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeProjectKeyError(w, http.StatusUnauthorized, "unauthorized", "admin token is not provided")
			return
		}
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != a.token {
			writeProjectKeyError(w, http.StatusUnauthorized, "unauthorized", "incorrect admin token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

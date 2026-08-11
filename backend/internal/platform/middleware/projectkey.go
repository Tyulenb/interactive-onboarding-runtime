package middleware

import (
	"net/http"
	"strings"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/httpserver"
	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
)

const (
	projectKeyHeader = "X-Project-Key"
)

func ExtractProjectKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectKey := strings.TrimSpace(r.Header.Get(projectKeyHeader))
		if projectKey == "" {
			writeProjectKeyError(w, http.StatusForbidden, "project_key_required", "X-Project-Key header is required")
			return
		}
		ctx := requestcontext.WithProjectKey(r.Context(), projectKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeProjectKeyError(w http.ResponseWriter, status int, code, message string) {
	httpserver.WriteJSON(w, status, httpserver.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
)

const (
	testTokenHeader = "X-Scenario-Test-Token"
)

func ExtractTestToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testToken := strings.TrimSpace(r.Header.Get(testTokenHeader))
		ctx := requestcontext.WithTestToken(r.Context(), testToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

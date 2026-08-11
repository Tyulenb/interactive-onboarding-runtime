package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
)

func TestExtractTestToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{name: "trimmed token", token: "  test-token  ", want: "test-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got, _ := requestcontext.TestToken(r.Context()); got != tt.want {
					t.Errorf("testToken = %q, want %q", got, tt.want)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(testTokenHeader, tt.token)
			ExtractTestToken(next).ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

func TestExtractTestTokenAllowsMissingToken(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if got, _ := requestcontext.TestToken(r.Context()); got != "" {
			t.Errorf("testToken = %q, want empty string", got)
		}
	})
	response := httptest.NewRecorder()

	ExtractTestToken(next).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Fatal("next handler was not called without a test token")
	}
}

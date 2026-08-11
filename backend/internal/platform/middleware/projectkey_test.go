package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
)

func TestExtractProjectKeySuccess(t *testing.T) {
	expectedToken := "keyheadertoken"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, _ := requestcontext.ProjectKey(r.Context()); got != expectedToken {
			t.Errorf("testToken = %q, want %q", got, expectedToken)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(projectKeyHeader, expectedToken)
	ExtractProjectKey(next).ServeHTTP(httptest.NewRecorder(), req)
}

func TestExtractProjectKeyEmpty(t *testing.T) {
	expectedToken := ""
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(projectKeyHeader, expectedToken)
	recorder := httptest.NewRecorder()
	ExtractProjectKey(next).ServeHTTP(recorder, req)

	if recorder.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("Expected status code %d, but got %d", http.StatusForbidden, recorder.Result().StatusCode)
	}
}

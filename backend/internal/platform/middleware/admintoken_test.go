package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrectAdminToken(t *testing.T) {
	adminMiddle := NewAdminMiddleware("admintoken")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer admintoken")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	adminMiddle.CheckAdminToken(next).ServeHTTP(recorder, req)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, but got %d", http.StatusOK, recorder.Result().StatusCode)
	}
}

func TestIncorrectAdminToken(t *testing.T) {
	adminMiddle := NewAdminMiddleware("admintoken")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer randomtoken")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	adminMiddle.CheckAdminToken(next).ServeHTTP(recorder, req)

	if recorder.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %d, but got %d", http.StatusUnauthorized, recorder.Result().StatusCode)
	}
}

func TestCheckAdminTokenRejectsMalformedAuthorizationHeader(t *testing.T) {
	adminMiddle := NewAdminMiddleware("admintoken")

	for _, authHeader := range []string{
		"admintoken",
		"Basic admintoken",
		"Bearer",
		"Bearer admintoken extra",
	} {
		t.Run(authHeader, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", authHeader)
			recorder := httptest.NewRecorder()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler must not be called")
			})
			adminMiddle.CheckAdminToken(next).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

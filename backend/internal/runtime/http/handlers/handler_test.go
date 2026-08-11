package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/httpserver"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	runtimeService "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/service"
)

type runtimeServiceMock struct {
	ctx         context.Context
	pagePattern string
	userID      string
	response    *runtimeModel.RuntimeScenarioResolveResponse
	err         error
}

func (m *runtimeServiceMock) FindScenarios(
	ctx context.Context,
	pagePattern, userID string,
) (*runtimeModel.RuntimeScenarioResolveResponse, error) {
	m.ctx = ctx
	m.pagePattern = pagePattern
	m.userID = userID
	return m.response, m.err
}

func TestGetScenarioRejectsMalformedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{`},
		{name: "unknown field", body: `{"page":"/home","user_id":"user-1","unknown":true}`},
		{name: "multiple JSON values", body: `{"page":"/home","user_id":"user-1"} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(tt.body))

			NewHandler(nil).GetScenario(response, req)

			assertErrorResponse(t, response, http.StatusBadRequest, "invalid_request")
		})
	}
}

func TestGetScenarioRejectsInvalidRequestFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing page", body: `{"user_id":"user-1"}`},
		{name: "empty page", body: `{"page":"","user_id":"user-1"}`},
		{name: "page too long", body: `{"page":"` + strings.Repeat("p", 2049) + `","user_id":"user-1"}`},
		{name: "missing user ID", body: `{"page":"/home"}`},
		{name: "empty user ID", body: `{"page":"/home","user_id":""}`},
		{name: "user ID too long", body: `{"page":"/home","user_id":"` + strings.Repeat("u", 256) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(tt.body))

			NewHandler(nil).GetScenario(response, req)

			assertErrorResponse(t, response, http.StatusUnprocessableEntity, "validation_error")
		})
	}
}

func TestGetScenarioReturnsServiceResponse(t *testing.T) {
	service := &runtimeServiceMock{response: &runtimeModel.RuntimeScenarioResolveResponse{
		IsTest: true,
		Scenarios: []runtimeModel.RuntimeScenario{{
			ID: "scenario-1",
			Steps: []runtimeModel.RuntimeStep{{
				ID:           "step-1",
				FrontendData: json.RawMessage(`{"placement":"bottom"}`),
			}},
		}},
	}}
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`))

	NewHandler(service).GetScenario(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if service.pagePattern != "/home" || service.userID != "user-1" {
		t.Errorf("service arguments = (%q, %q), want (/home, user-1)", service.pagePattern, service.userID)
	}

	var payload runtimeModel.RuntimeScenarioResolveResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.IsTest || len(payload.Scenarios) != 1 || payload.Scenarios[0].ID != "scenario-1" {
		t.Fatalf("unexpected response: %+v", payload)
	}
	if got := string(payload.Scenarios[0].Steps[0].FrontendData); got != `{"placement":"bottom"}` {
		t.Errorf("frontend_data = %s, want JSON object", got)
	}
}

func TestGetScenarioReturnsEmptyScenarioListForNilResponse(t *testing.T) {
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`))

	NewHandler(&runtimeServiceMock{}).GetScenario(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload runtimeModel.RuntimeScenarioResolveResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.IsTest || len(payload.Scenarios) != 0 || payload.Scenarios == nil {
		t.Fatalf("unexpected empty response: %+v", payload)
	}
}

func TestGetScenarioMapsTokenErrorsToForbidden(t *testing.T) {
	tests := []error{
		runtimeService.ErrTestTokenInvalid,
		runtimeService.ErrProjectTokenIsNotValid,
	}

	for _, serviceErr := range tests {
		t.Run(serviceErr.Error(), func(t *testing.T) {
			response := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`))

			NewHandler(&runtimeServiceMock{err: serviceErr}).GetScenario(response, req)

			assertErrorResponse(t, response, http.StatusForbidden, "forbidden")
		})
	}
}

func TestGetScenarioMapsPageMismatchToUnprocessableEntity(t *testing.T) {
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`))

	NewHandler(&runtimeServiceMock{err: runtimeService.ErrPageMismatch}).GetScenario(response, req)

	assertErrorResponse(t, response, http.StatusUnprocessableEntity, "unprocessable contend")
}

func TestGetScenarioMapsUnexpectedServiceErrorsToInternalServerError(t *testing.T) {
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`))

	NewHandler(&runtimeServiceMock{err: errors.New("database unavailable")}).GetScenario(response, req)

	assertErrorResponse(t, response, http.StatusInternalServerError, "internal_error")
}

func TestRegisterRoutesRegistersResolveEndpoint(t *testing.T) {
	router := http.NewServeMux()
	NewHandler(&runtimeServiceMock{response: &runtimeModel.RuntimeScenarioResolveResponse{Scenarios: []runtimeModel.RuntimeScenario{}}}).RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/sdk/scenarios/resolve", strings.NewReader(`{"page":"/home","user_id":"user-1"}`)))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}

	var payload httpserver.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Code != wantCode {
		t.Errorf("error code = %q, want %q", payload.Code, wantCode)
	}
}

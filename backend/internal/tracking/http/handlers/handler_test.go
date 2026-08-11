package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	trackingModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/model"
	trackingService "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/service"
)

type trackingServiceStub struct {
	startSession func(context.Context, *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error)
	createEvent  func(context.Context, *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error)
}

func (s trackingServiceStub) StartSession(ctx context.Context, request *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
	return s.startSession(ctx, request)
}

func (s trackingServiceStub) CreateEvent(ctx context.Context, request *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
	return s.createEvent(ctx, request)
}

func TestRegisterRoutes(t *testing.T) {
	service := trackingServiceStub{
		startSession: func(context.Context, *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
			return &trackingModel.OnboardingSession{ID: "session-1"}, nil
		},
		createEvent: func(context.Context, *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
			return &trackingModel.EventAcceptedResponse{}, nil
		},
	}
	router := http.NewServeMux()
	NewTrackingHandler(service).RegisterRoutes(router)

	for _, endpoint := range []string{"/api/v1/sdk/sessions", "/api/v1/sdk/events"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(`{}`)))
		if response.Code == http.StatusNotFound {
			t.Fatalf("POST %s was not registered", endpoint)
		}
	}
}

func TestCreateSession(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}
	startedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	service := trackingServiceStub{
		startSession: func(ctx context.Context, request *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
			if got := ctx.Value(key); got != "request-context" {
				t.Fatalf("context value = %v, want request-context", got)
			}
			if request.ScenarioID != "scenario-1" || request.UserID != "user-1" {
				t.Fatalf("unexpected request: %#v", request)
			}
			return &trackingModel.OnboardingSession{ID: "session-1", Status: trackingModel.SessionStatusActive, StartedAt: startedAt}, nil
		},
		createEvent: unexpectedCreateEvent(t),
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/sdk/sessions", strings.NewReader(`{"scenario_id":"scenario-1","user_id":"user-1"}`))
	request = request.WithContext(context.WithValue(request.Context(), key, "request-context"))
	response := httptest.NewRecorder()

	NewTrackingHandler(service).createSession(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var session trackingModel.OnboardingSession
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if session.ID != "session-1" || !session.StartedAt.Equal(startedAt) {
		t.Fatalf("unexpected response: %#v", session)
	}
}

func TestHandlersRejectInvalidJSONWithoutCallingService(t *testing.T) {
	service := trackingServiceStub{startSession: unexpectedStartSession(t), createEvent: unexpectedCreateEvent(t)}
	handler := NewTrackingHandler(service)

	for _, endpoint := range []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
	}{
		{name: "session", call: handler.createSession},
		{name: "event", call: handler.createEvent},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			endpoint.call(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unexpected":true}`)))

			assertErrorResponse(t, response, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		})
	}
}

func TestCreateEventReturnsAcceptedAndOKForDuplicate(t *testing.T) {
	for _, tt := range []struct {
		name      string
		duplicate bool
		want      int
	}{
		{name: "new event", want: http.StatusAccepted},
		{name: "duplicate event", duplicate: true, want: http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := trackingServiceStub{
				startSession: unexpectedStartSession(t),
				createEvent: func(_ context.Context, request *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
					if request.ID != "event-1" || request.SessionID != "session-1" || request.Type != trackingModel.EventTypeStepShown {
						t.Fatalf("unexpected request: %#v", request)
					}
					return &trackingModel.EventAcceptedResponse{Event: trackingModel.OnboardingEvent{ID: request.ID}, Duplicate: tt.duplicate}, nil
				},
			}
			response := httptest.NewRecorder()
			NewTrackingHandler(service).createEvent(response, httptest.NewRequest(http.MethodPost, "/api/v1/sdk/events", strings.NewReader(`{"id":"event-1","session_id":"session-1","type":"step_shown"}`)))

			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d", response.Code, tt.want)
			}
			var payload trackingModel.EventAcceptedResponse
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Event.ID != "event-1" || payload.Duplicate != tt.duplicate {
				t.Fatalf("unexpected response: %#v", payload)
			}
		})
	}
}

func TestServiceErrorsAreMapped(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{name: "validation", err: errors.Join(trackingService.ErrInvalidRequest, errors.New("id must be a UUID")), status: http.StatusUnprocessableEntity, code: "validation_error", message: "invalid tracking request\nid must be a UUID"},
		{name: "scenario missing", err: trackingService.ErrScenarioNotFound, status: http.StatusNotFound, code: "not_found", message: trackingService.ErrScenarioNotFound.Error()},
		{name: "session missing", err: trackingService.ErrSessionNotFound, status: http.StatusNotFound, code: "not_found", message: trackingService.ErrSessionNotFound.Error()},
		{name: "step missing", err: trackingService.ErrStepNotFound, status: http.StatusNotFound, code: "not_found", message: trackingService.ErrStepNotFound.Error()},
		{name: "session inactive", err: trackingService.ErrSessionNotActive, status: http.StatusConflict, code: "conflict", message: trackingService.ErrSessionNotActive.Error()},
		{name: "step mismatch", err: trackingService.ErrStepScenarioMismatch, status: http.StatusConflict, code: "conflict", message: trackingService.ErrStepScenarioMismatch.Error()},
		{name: "event ID conflict", err: trackingService.ErrEventIDConflict, status: http.StatusConflict, code: "conflict", message: trackingService.ErrEventIDConflict.Error()},
		{name: "invalid project key", err: trackingService.ErrProjectKeyInvalid, status: http.StatusForbidden, code: "forbidden", message: trackingService.ErrProjectKeyInvalid.Error()},
		{name: "unexpected", err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: "internal_error", message: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeServiceError(response, tt.err)
			assertErrorResponse(t, response, tt.status, tt.code, tt.message)
		})
	}
}

func unexpectedStartSession(t *testing.T) func(context.Context, *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
	t.Helper()
	return func(context.Context, *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
		t.Fatal("StartSession must not be called")
		return nil, nil
	}
}

func unexpectedCreateEvent(t *testing.T) func(context.Context, *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
	t.Helper()
	return func(context.Context, *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
		t.Fatal("CreateEvent must not be called")
		return nil, nil
	}
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode, wantMessage string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Code != wantCode || payload.Message != wantMessage {
		t.Fatalf("error response = %#v, want code=%q message=%q", payload, wantCode, wantMessage)
	}
}

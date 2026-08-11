package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/httpserver"
	trackingModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/model"
	trackingService "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/service"
)

type TrackingService interface {
	StartSession(context.Context, *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error)
	CreateEvent(context.Context, *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error)
}

type TrackingHandler struct {
	service TrackingService
}

func NewTrackingHandler(srvc TrackingService) *TrackingHandler {
	return &TrackingHandler{
		service: srvc,
	}
}

func (h *TrackingHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("POST /api/v1/sdk/sessions", h.createSession)
	router.HandleFunc("POST /api/v1/sdk/events", h.createEvent)
}

func (h *TrackingHandler) createSession(w http.ResponseWriter, r *http.Request) {
	startSessionReq := new(trackingModel.StartSessionRequest)
	if err := httpserver.ParseJSON(w, r, startSessionReq); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		return
	}

	onboardingSession, err := h.service.StartSession(r.Context(), startSessionReq)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, onboardingSession)
}

func (h *TrackingHandler) createEvent(w http.ResponseWriter, r *http.Request) {
	eventReq := new(trackingModel.CreateEventRequest)
	if err := httpserver.ParseJSON(w, r, eventReq); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		return
	}

	response, err := h.service.CreateEvent(r.Context(), eventReq)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	status := http.StatusAccepted
	if response.Duplicate {
		status = http.StatusOK
	}
	httpserver.WriteJSON(w, status, response)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, trackingService.ErrInvalidRequest):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
	case errors.Is(err, trackingService.ErrScenarioNotFound),
		errors.Is(err, trackingService.ErrSessionNotFound),
		errors.Is(err, trackingService.ErrStepNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, trackingService.ErrSessionNotActive),
		errors.Is(err, trackingService.ErrStepScenarioMismatch),
		errors.Is(err, trackingService.ErrEventIDConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, trackingService.ErrProjectKeyInvalid):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	httpserver.WriteJSON(w, status, httpserver.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

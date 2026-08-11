package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/httpserver"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	runtimeService "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/service"
	"github.com/go-playground/validator/v10"
)

type RuntimeService interface {
	FindScenarios(ctx context.Context, pagePattern, userID string) (*runtimeModel.RuntimeScenarioResolveResponse, error)
}

type RuntimeHandler struct {
	service RuntimeService
}

var requestValidator = validator.New()

func NewHandler(srvc RuntimeService) *RuntimeHandler {
	return &RuntimeHandler{
		service: srvc,
	}
}

func (h *RuntimeHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("POST /api/v1/sdk/scenarios/resolve", h.GetScenario)
}

func (h *RuntimeHandler) GetScenario(w http.ResponseWriter, r *http.Request) {
	scenarioRequest := new(runtimeModel.ResolveScenarioRequest)
	if err := httpserver.ParseJSON(w, r, scenarioRequest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		return
	}

	if err := requestValidator.Struct(scenarioRequest); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "ivalid request")
		return
	}
	scenarioResponse, err := h.service.FindScenarios(r.Context(), scenarioRequest.Page, scenarioRequest.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if scenarioResponse == nil {
		scenarioResponse = &runtimeModel.RuntimeScenarioResolveResponse{
			Scenarios: make([]runtimeModel.RuntimeScenario, 0),
		}
	}

	httpserver.WriteJSON(w, http.StatusOK, scenarioResponse)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runtimeService.ErrTestTokenInvalid),
		errors.Is(err, runtimeService.ErrProjectTokenIsNotValid):
		writeError(w, http.StatusForbidden, "forbidden", "invalid test or project token")
	case errors.Is(err, runtimeService.ErrScenarioNotFound):
		writeError(w, http.StatusNotFound, "not found", "Scenario was not found")
	case errors.Is(err, runtimeService.ErrPageMismatch),
		errors.Is(err, runtimeService.ErrInvalidScenarioConfiguration):
		writeError(w, http.StatusUnprocessableEntity, "unprocessable contend", "Project and scenario mismatch")
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

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
	trackingModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidRequest       = errors.New("invalid tracking request")
	ErrScenarioNotFound     = errors.New("scenario not found")
	ErrSessionNotFound      = errors.New("onboarding session not found")
	ErrSessionNotActive     = errors.New("onboarding session is not active")
	ErrStepNotFound         = errors.New("onboarding step not found")
	ErrStepScenarioMismatch = errors.New("onboarding step does not belong to session scenario")
	ErrEventIDConflict      = errors.New("event ID is already used by another event")
	ErrProjectKeyInvalid    = errors.New("project key is not valid")
)

type (
	SessionRepository interface {
		CreateOrGetActiveSession(context.Context, *trackingModel.OnboardingSession) (*trackingModel.OnboardingSession, error)
		UpdateSessionStatus(context.Context, string, trackingModel.SessionStatus, time.Time) (*trackingModel.OnboardingSession, error)
		GetSessionById(context.Context, string) (*trackingModel.OnboardingSession, error)
	}
	EventRepository interface {
		RecordEvent(context.Context, *trackingModel.OnboardingEvent) (*trackingModel.EventAcceptedResponse, error)
		GetEventByIdAndProjectKey(
			context.Context, *trackingModel.OnboardingEvent, string,
		) (*trackingModel.EventAcceptedResponse, bool, error)
	}
	Transactor interface {
		WithTx(context.Context, func(context.Context) error) error
	}
	ScenarioRepository interface {
		GetScenarioByIdAndProjectKey(ctx context.Context, scenarioId, projectKey string) (*trackingModel.Scenario, error)
	}
	StepRepository interface {
		GetStepById(ctx context.Context, stepId string) (*trackingModel.Step, error)
	}
)

type TrackingService struct {
	sessions   SessionRepository
	events     EventRepository
	scenarios  ScenarioRepository
	steps      StepRepository
	transactor Transactor
}

func NewTrackingService(s SessionRepository, e EventRepository, sc ScenarioRepository, st StepRepository, tx Transactor) *TrackingService {
	return &TrackingService{
		sessions:   s,
		events:     e,
		scenarios:  sc,
		steps:      st,
		transactor: tx,
	}
}

func (s *TrackingService) StartSession(ctx context.Context, session *trackingModel.StartSessionRequest) (*trackingModel.OnboardingSession, error) {
	if session == nil {
		return nil, invalid("request is required")
	}
	if _, err := uuid.Parse(session.ScenarioID); err != nil {
		return nil, invalid("scenario_id must be a UUID")
	}
	if length := utf8.RuneCountInString(session.UserID); length < 1 || length > 255 {
		return nil, invalid("user_id must contain from 1 to 255 characters")
	}

	err := s.validateProjectKey(ctx, session.ScenarioID, true)
	if err != nil {
		return nil, err
	}

	onboardingSession := &trackingModel.OnboardingSession{
		ID:         uuid.NewString(),
		ScenarioID: session.ScenarioID,
		UserID:     session.UserID,
		Status:     trackingModel.SessionStatusActive,
		StartedAt:  time.Now(),
		FinishedAt: nil,
	}
	return s.sessions.CreateOrGetActiveSession(ctx, onboardingSession)
}

func (s *TrackingService) CreateEvent(ctx context.Context, event *trackingModel.CreateEventRequest) (*trackingModel.EventAcceptedResponse, error) {
	occurredAt, err := validateEvent(event)
	if err != nil {
		return nil, err
	}
	projectKey, err := projectKeyFromContext(ctx)
	if err != nil {
		return nil, err
	}

	onboardingEvent := &trackingModel.OnboardingEvent{
		ID:         event.ID,
		SessionID:  event.SessionID,
		StepID:     event.StepID,
		Type:       event.Type,
		Data:       event.Data,
		OccurredAt: occurredAt,
		ReceivedAt: time.Now(),
	}
	if existing, found, lookUpErr := s.lookupEvent(ctx, onboardingEvent, projectKey); lookUpErr != nil || found {
		return existing, lookUpErr
	}

	session, err := s.sessions.GetSessionById(ctx, event.SessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	if session.Status != trackingModel.SessionStatusActive {
		return nil, ErrSessionNotActive
	}

	err = s.validateProjectKey(ctx, session.ScenarioID, false)
	if err != nil {
		return nil, err
	}

	if event.StepID != nil {
		step, getErr := s.steps.GetStepById(ctx, *event.StepID)
		if getErr != nil {
			if errors.Is(getErr, sql.ErrNoRows) {
				return nil, ErrStepNotFound
			} else {
				return nil, getErr
			}
		}

		if step.ScenarioID != session.ScenarioID {
			return nil, ErrStepScenarioMismatch
		}
	}

	var response *trackingModel.EventAcceptedResponse
	create := func(ctx context.Context) error {
		existing, found, lookUpErr := s.lookupEvent(ctx, onboardingEvent, projectKey)
		if lookUpErr != nil {
			return lookUpErr
		}
		if found {
			response = existing
			return nil
		}

		created, recordErr := s.events.RecordEvent(ctx, onboardingEvent)
		if recordErr != nil {
			if errors.Is(recordErr, sql.ErrNoRows) {
				existing, found, lookupErr := s.lookupEvent(ctx, onboardingEvent, projectKey)
				if lookupErr != nil {
					return lookupErr
				}
				if found {
					response = existing
					return nil
				}
				return ErrSessionNotActive
			}
			return fmt.Errorf("record event %q: %w", event.ID, recordErr)
		}
		if status, completesSession := completionStatus(event.Type); completesSession {
			if _, updateSessionErr := s.sessions.UpdateSessionStatus(ctx, event.SessionID, status, occurredAt); updateSessionErr != nil {
				return fmt.Errorf("complete session %q: %w", event.SessionID, updateSessionErr)
			}
		}
		response = created
		return nil
	}

	err = s.transactor.WithTx(ctx, create)
	if err != nil {
		if isUniqueViolation(err) {
			existing, found, lookupErr := s.lookupEvent(ctx, onboardingEvent, projectKey)
			if lookupErr == nil && found {
				return existing, nil
			}
			if errors.Is(lookupErr, sql.ErrNoRows) || lookupErr == nil {
				return nil, ErrEventIDConflict
			}
			return nil, fmt.Errorf("get concurrently created event %q: %w", event.ID, lookupErr)
		}
		return nil, err
	}
	return response, nil
}

func (s *TrackingService) lookupEvent(
	ctx context.Context, event *trackingModel.OnboardingEvent, projectKey string,
) (*trackingModel.EventAcceptedResponse, bool, error) {
	existing, matches, err := s.events.GetEventByIdAndProjectKey(ctx, event, projectKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get event %q: %w", event.ID, err)
	}
	if !matches {
		return nil, true, ErrEventIDConflict
	}
	return existing, true, nil
}

func (s *TrackingService) validateProjectKey(ctx context.Context, scenarioId string, enableCheck bool) error {
	projectKey, err := projectKeyFromContext(ctx)
	if err != nil {
		return err
	}
	scenario, err := s.scenarios.GetScenarioByIdAndProjectKey(ctx, scenarioId, projectKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrScenarioNotFound
		} else {
			return err
		}
	}
	if enableCheck && scenario.Status != trackingModel.ScenarioStatusEnabled {
		return invalid("scenario is not enabled")
	}
	return nil
}

func projectKeyFromContext(ctx context.Context) (string, error) {
	projectKey, ok := requestcontext.ProjectKey(ctx)
	if !ok || projectKey == "" {
		return "", ErrProjectKeyInvalid
	}
	return projectKey, nil
}

func validateEvent(event *trackingModel.CreateEventRequest) (time.Time, error) {
	if event == nil {
		return time.Time{}, invalid("request is required")
	}
	if _, err := uuid.Parse(event.ID); err != nil {
		return time.Time{}, invalid("id must be a UUID")
	}
	if _, err := uuid.Parse(event.SessionID); err != nil {
		return time.Time{}, invalid("session_id must be a UUID")
	}
	if event.StepID != nil {
		if event.Type == trackingModel.EventTypeOnboardingSkipped ||
			event.Type == trackingModel.EventTypeOnboardingCompleted {
			return time.Time{}, invalid("step_id is unnecessary")
		}
		if _, err := uuid.Parse(*event.StepID); err != nil {
			return time.Time{}, invalid("step_id must be a UUID")
		}
	} else if event.Type != trackingModel.EventTypeOnboardingSkipped &&
		event.Type != trackingModel.EventTypeOnboardingCompleted {
		return time.Time{}, invalid("step_id is required")
	}
	if !isEventType(event.Type) {
		return time.Time{}, invalid("type is not supported")
	}
	if len(event.Data) == 0 {
		event.Data = json.RawMessage(`{}`)
	} else {
		var data map[string]json.RawMessage
		if err := json.Unmarshal(event.Data, &data); err != nil || data == nil {
			return time.Time{}, invalid("data must be a JSON object")
		}
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
	if err != nil {
		return time.Time{}, invalid("occurred_at must be an RFC3339 timestamp")
	}
	return occurredAt, nil
}

func isEventType(eventType trackingModel.EventType) bool {
	switch eventType {
	case trackingModel.EventTypeStepShown,
		trackingModel.EventTypeStepCompleted,
		trackingModel.EventTypeStepSkipped,
		trackingModel.EventTypeOnboardingCompleted,
		trackingModel.EventTypeOnboardingSkipped:
		return true
	default:
		return false
	}
}

func completionStatus(eventType trackingModel.EventType) (trackingModel.SessionStatus, bool) {
	switch eventType {
	case trackingModel.EventTypeOnboardingCompleted:
		return trackingModel.SessionStatusCompleted, true
	case trackingModel.EventTypeOnboardingSkipped:
		return trackingModel.SessionStatusSkipped, true
	default:
		return "", false
	}
}

func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func invalid(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, message)
}

package event

import (
	"context"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	trackingModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/model"
	event "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/repository/event/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	queries *event.Queries
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{
		queries: event.New(db),
	}
}

func (e *EventRepository) getQueries(ctx context.Context) *event.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return e.queries.WithTx(tx)
	}

	return e.queries
}

func (e *EventRepository) RecordEvent(
	ctx context.Context, onboarding *trackingModel.OnboardingEvent,
) (*trackingModel.EventAcceptedResponse, error) {
	eventId, err := uuid.Parse(onboarding.ID)
	if err != nil {
		return nil, err
	}
	sessionId, err := uuid.Parse(onboarding.SessionID)
	if err != nil {
		return nil, err
	}
	var stepId pgtype.UUID
	if onboarding.StepID != nil {
		step, parseErr := uuid.Parse(*onboarding.StepID)
		if parseErr != nil {
			return nil, parseErr
		}
		stepId = pgtype.UUID{
			Bytes: step,
			Valid: true,
		}
	}
	createEvent := event.CreateEventParams{
		ID:         eventId,
		SessionID:  sessionId,
		StepID:     stepId,
		Type:       string(onboarding.Type),
		Data:       onboarding.Data,
		OccurredAt: pgtype.Timestamptz{Time: onboarding.OccurredAt, Valid: true},
		ReceivedAt: pgtype.Timestamptz{Time: onboarding.ReceivedAt, Valid: true},
	}

	createdEvent, err := e.getQueries(ctx).CreateEvent(ctx, createEvent)
	if err != nil {
		return nil, err
	}

	return adaptEvent(createdEvent, false), nil
}

func (e *EventRepository) GetEventByIdAndProjectKey(
	ctx context.Context, requested *trackingModel.OnboardingEvent, projectKey string,
) (*trackingModel.EventAcceptedResponse, bool, error) {
	parsedEventID, err := uuid.Parse(requested.ID)
	if err != nil {
		return nil, false, err
	}
	parsedSessionID, err := uuid.Parse(requested.SessionID)
	if err != nil {
		return nil, false, err
	}

	var stepID pgtype.UUID
	if requested.StepID != nil {
		parsedStepID, parseErr := uuid.Parse(*requested.StepID)
		if parseErr != nil {
			return nil, false, parseErr
		}
		stepID = pgtype.UUID{Bytes: parsedStepID, Valid: true}
	}

	found, err := e.getQueries(ctx).GetEventByIdAndProjectKey(ctx, event.GetEventByIdAndProjectKeyParams{
		SessionID:  parsedSessionID,
		StepID:     stepID,
		Type:       string(requested.Type),
		Data:       requested.Data,
		OccurredAt: pgtype.Timestamptz{Time: requested.OccurredAt, Valid: true},
		EventID:    parsedEventID,
		ProjectKey: projectKey,
	})
	if err != nil {
		return nil, false, err
	}

	return adaptEvent(event.OnboardingEvent{
		ID:         found.ID,
		SessionID:  found.SessionID,
		StepID:     found.StepID,
		Type:       found.Type,
		Data:       found.Data,
		OccurredAt: found.OccurredAt,
		ReceivedAt: found.ReceivedAt,
	}, true), found.RequestMatches.Bool, nil
}

func adaptEvent(source event.OnboardingEvent, duplicate bool) *trackingModel.EventAcceptedResponse {
	var stepID *string
	if source.StepID.Valid {
		value := uuid.UUID(source.StepID.Bytes).String()
		stepID = &value
	}

	return &trackingModel.EventAcceptedResponse{
		Event: trackingModel.OnboardingEvent{
			ID:         source.ID.String(),
			SessionID:  source.SessionID.String(),
			StepID:     stepID,
			Type:       trackingModel.EventType(source.Type),
			Data:       source.Data,
			OccurredAt: source.OccurredAt.Time,
			ReceivedAt: source.ReceivedAt.Time,
		},
		Duplicate: duplicate,
	}
}

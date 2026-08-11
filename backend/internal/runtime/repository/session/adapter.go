package session

import (
	"context"
	"fmt"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	session "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/repository/session/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SessionRepository struct {
	queries *session.Queries
}

func NewSessionRepository(db session.DBTX) *SessionRepository {
	q := session.New(db)
	return &SessionRepository{
		queries: q,
	}
}

func (r *SessionRepository) getQueries(ctx context.Context) *session.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *SessionRepository) GetSessionByScenarioAndUser(
	ctx context.Context, scenarioID, userID string,
) (*runtimeModel.Session, error) {
	parsedScenarioID, err := uuid.Parse(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("parse scenario ID: %w", err)
	}

	found, err := r.getQueries(ctx).GetSessionByScenarioAndUser(ctx, session.GetSessionByScenarioAndUserParams{
		ScenarioID: parsedScenarioID,
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}

	return adaptSession(found), nil
}

func adaptSession(source session.OnboardingSession) *runtimeModel.Session {
	return &runtimeModel.Session{
		ID:         source.ID.String(),
		ScenarioID: source.ScenarioID.String(),
		UserID:     source.UserID,
		Status:     runtimeModel.SessionStatus(source.Status),
		StartedAt:  source.StartedAt.Time,
		FinishedAt: nullableTime(source.FinishedAt),
	}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

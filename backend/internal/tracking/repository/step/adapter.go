package step

import (
	"context"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	trackingModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/model"
	stepSQLC "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/repository/step/sqlc"
	trackingService "github.com/DaniilSintsov/interactive-onboarding/backend/internal/tracking/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type StepRepository struct {
	queries *stepSQLC.Queries
}

var _ trackingService.StepRepository = (*StepRepository)(nil)

func NewStepRepository(db stepSQLC.DBTX) *StepRepository {
	return &StepRepository{queries: stepSQLC.New(db)}
}

func (r *StepRepository) getQueries(ctx context.Context) *stepSQLC.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *StepRepository) GetStepById(ctx context.Context, stepID string) (*trackingModel.Step, error) {
	id, err := uuid.Parse(stepID)
	if err != nil {
		return nil, err
	}

	found, err := r.getQueries(ctx).GetStepById(ctx, id)
	if err != nil {
		return nil, err
	}

	return adaptStep(found), nil
}

func adaptStep(source stepSQLC.OnboardingStep) *trackingModel.Step {
	return &trackingModel.Step{
		ID:           source.ID.String(),
		ScenarioID:   source.ScenarioID.String(),
		StepNum:      int(source.StepNum),
		Title:        source.Title,
		Description:  source.Description,
		ElementID:    source.ElementID.String(),
		FrontendData: source.FrontendData,
		CreatedAt:    source.CreatedAt.Time,
		UpdatedAt:    source.UpdatedAt.Time,
		DeletedAt:    nullableTime(source.DeletedAt),
	}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

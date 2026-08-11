package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	scenario "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/repository/scenario/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ScenarioRepository struct {
	queries *scenario.Queries
}

func NewScenarioRepository(db scenario.DBTX) *ScenarioRepository {
	return &ScenarioRepository{queries: scenario.New(db)}
}

func (r *ScenarioRepository) getQueries(ctx context.Context) *scenario.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *ScenarioRepository) GetScenarioById(ctx context.Context, scenarioID string) (*runtimeModel.Scenario, error) {
	parsedScenarioID, err := uuid.Parse(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("parse scenario ID: %w", err)
	}

	found, err := r.getQueries(ctx).GetScenarioById(ctx, parsedScenarioID)
	if err != nil {
		return nil, err
	}
	return adaptScenario(found), nil
}

func (r *ScenarioRepository) GetScenarioByIdAndProjectKey(
	ctx context.Context, scenarioID, projectKey string,
) (*runtimeModel.Scenario, error) {
	parsedScenarioID, err := uuid.Parse(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("parse scenario ID: %w", err)
	}

	found, err := r.queries.GetScenarioByIdAndProjectKey(ctx, scenario.GetScenarioByIdAndProjectKeyParams{
		ScenarioID: parsedScenarioID,
		ProjectKey: projectKey,
	})
	if err != nil {
		return nil, err
	}
	return adaptScenario(found), nil
}

func (r *ScenarioRepository) GetScenariosByPagePatternAndProjectkey(
	ctx context.Context, pagePattern, projectKey string,
) ([]runtimeModel.Scenario, error) {
	found, err := r.queries.GetScenariosByPagePatternAndProjectkey(ctx, scenario.GetScenariosByPagePatternAndProjectkeyParams{
		PagePattern: pagePattern,
		ProjectKey:  projectKey,
	})
	if err != nil {
		return nil, err
	}

	scenarios := make([]runtimeModel.Scenario, len(found))
	for i, item := range found {
		scenarios[i] = *adaptScenario(item)
	}
	return scenarios, nil
}

func adaptScenario(source scenario.OnboardingScenario) *runtimeModel.Scenario {
	return &runtimeModel.Scenario{
		ID:          source.ID.String(),
		ProjectID:   source.ProjectID.String(),
		Name:        source.Name,
		Description: source.Description,
		PagePattern: source.PagePattern,
		Status:      runtimeModel.ScenarioStatus(source.Status),
		PublishedAt: nullableTime(source.PublishedAt),
		CreatedAt:   source.CreatedAt.Time,
		UpdatedAt:   source.UpdatedAt.Time,
		DeletedAt:   nullableTime(source.DeletedAt),
	}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

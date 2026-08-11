package step

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	step "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/repository/step/sqlc"
	"github.com/google/uuid"
)

type StepRepository struct {
	queries *step.Queries
}

func NewStepRepository(db step.DBTX) *StepRepository {
	return &StepRepository{queries: step.New(db)}
}

func (r *StepRepository) getQueries(ctx context.Context) *step.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *StepRepository) GetStepsByScenarioId(ctx context.Context, scenarioID string) (*runtimeModel.RuntimeScenario, error) {
	parsedScenarioID, err := uuid.Parse(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("parse scenario ID: %w", err)
	}

	rows, err := r.getQueries(ctx).GetStepsByScenarioId(ctx, parsedScenarioID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}

	runtimeScenario := &runtimeModel.RuntimeScenario{
		ID:          rows[0].ScenarioID.String(),
		Name:        rows[0].ScenarioName,
		Description: rows[0].ScenarioDescription,
		PagePattern: rows[0].ScenarioPagePattern,
		Steps:       make([]runtimeModel.RuntimeStep, 0, len(rows)),
	}
	for _, row := range rows {
		if !row.StepID.Valid {
			continue
		}
		runtimeScenario.Steps = append(runtimeScenario.Steps, runtimeModel.RuntimeStep{
			ID:           uuid.UUID(row.StepID.Bytes).String(),
			StepNum:      int(row.StepNum.Int32),
			Title:        row.StepTitle.String,
			Description:  row.StepDescription.String,
			FrontendData: row.FrontendData,
			Element: runtimeModel.RuntimeElement{
				ID:          uuid.UUID(row.ElementID.Bytes).String(),
				Key:         row.ElementKey.String,
				Label:       row.ElementLabel.String,
				Description: row.ElementDescription.String,
			},
		})
	}
	return runtimeScenario, nil
}

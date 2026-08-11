package projects

import (
	"context"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/postgres/transactor"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	projects "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/repository/projects/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProjectRepository struct {
	queries *projects.Queries
}

func NewProjectRepository(db projects.DBTX) *ProjectRepository {
	return &ProjectRepository{queries: projects.New(db)}
}

func (r *ProjectRepository) getQueries(ctx context.Context) *projects.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *ProjectRepository) GetProjectByProjectKey(
	ctx context.Context, projectKey string,
) (*runtimeModel.Project, error) {
	found, err := r.getQueries(ctx).GetProjectByProjectKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	return &runtimeModel.Project{
		ID:         found.ID.String(),
		Name:       found.Name,
		ProjectKey: found.ProjectKey,
		CreatedAt:  found.CreatedAt.Time,
		UpdatedAt:  found.UpdatedAt.Time,
		DeletedAt:  nullableTime(found.DeletedAt),
	}, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

package token

import (
	"context"

	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
	token "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/repository/token/sqlc"
)

type TestTokensRepository struct {
	queries *token.Queries
}

func NewTestTokensRepository(db token.DBTX) *TestTokensRepository {
	return &TestTokensRepository{queries: token.New(db)}
}

func (r *TestTokensRepository) GetTokenByHash(ctx context.Context, hash []byte) (*runtimeModel.TestToken, error) {
	found, err := r.queries.GetTokenByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	return &runtimeModel.TestToken{
		ID:         found.ID.String(),
		ScenarioID: found.ScenarioID.String(),
		Hash:       found.Hash,
		CreatedAt:  found.CreatedAt.Time,
		ExpiresAt:  found.ExpiresAt.Time,
	}, nil
}

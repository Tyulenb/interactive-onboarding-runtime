package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
)

var (
	errTestTokenIsEmpty             = errors.New("test token is empty")
	ErrScenarioNotFound             = errors.New("scenario was not found")
	ErrTestTokenInvalid             = errors.New("test token is invalid")
	ErrProjectTokenIsNotValid       = errors.New("project token is not valid")
	ErrPageMismatch                 = errors.New("page belongs to another scenario")
	ErrInvalidScenarioConfiguration = errors.New("scenario does not contain any steps")
)

type (
	ScenarioRepository interface {
		GetScenarioById(ctx context.Context, scenarioId string) (*runtimeModel.Scenario, error)
		GetScenarioByIdAndProjectKey(ctx context.Context, scenarioId, projectKey string) (*runtimeModel.Scenario, error)
		GetScenariosByPagePatternAndProjectkey(ctx context.Context, pagePattern, projectKey string) ([]runtimeModel.Scenario, error)
	}
	SessionRepository interface {
		GetSessionByScenarioAndUser(ctx context.Context, scenarioId, userId string) (*runtimeModel.Session, error)
	}
	StepRepository interface {
		GetStepsByScenarioId(ctx context.Context, scenarioId string) (*runtimeModel.RuntimeScenario, error)
	}
	TestTokensRepository interface {
		GetTokenByHash(ctx context.Context, hash []byte) (*runtimeModel.TestToken, error)
	}
	TokenHasher interface {
		Hash(rawToken string) []byte
	}
	ProjectRepository interface {
		GetProjectByProjectKey(ctx context.Context, projectKey string) (*runtimeModel.Project, error)
	}
)

type RuntimeService struct {
	scenario    ScenarioRepository
	session     SessionRepository
	steps       StepRepository
	tokens      TestTokensRepository
	projects    ProjectRepository
	tokenHasher TokenHasher
}

func NewRuntimeService(
	rep ScenarioRepository, ses SessionRepository, st StepRepository,
	tok TestTokensRepository, hasher TokenHasher, pr ProjectRepository,
) *RuntimeService {
	return &RuntimeService{
		scenario:    rep,
		session:     ses,
		steps:       st,
		tokens:      tok,
		projects:    pr,
		tokenHasher: hasher,
	}
}

func (r *RuntimeService) FindScenarios(ctx context.Context, pagePattern, userId string) (*runtimeModel.RuntimeScenarioResolveResponse, error) {
	response := &runtimeModel.RuntimeScenarioResolveResponse{
		Scenarios: make([]runtimeModel.RuntimeScenario, 0),
	}
	testScenario, err := r.checkTestToken(ctx, pagePattern)
	if err != nil && !errors.Is(err, errTestTokenIsEmpty) {
		return nil, err
	} else if err == nil {
		if len(testScenario.Steps) < 1 {
			return nil, ErrInvalidScenarioConfiguration
		}
		response.IsTest = true
		response.Scenarios = []runtimeModel.RuntimeScenario{*testScenario}
		return response, nil
	}

	projectKey, ok := requestcontext.ProjectKey(ctx)
	if !ok {
		return nil, ErrProjectTokenIsNotValid
	}

	_, err = r.projects.GetProjectByProjectKey(ctx, projectKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectTokenIsNotValid
		}
		return nil, err
	}

	scenarios, err := r.scenario.GetScenariosByPagePatternAndProjectkey(ctx, pagePattern, projectKey)
	if err != nil {
		return nil, err
	}

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	filtered := scenarios[:0]
	for _, scenario := range scenarios {
		session, err := r.session.GetSessionByScenarioAndUser(ctx, scenario.ID, userId)
		if err == nil {
			if session.Status != runtimeModel.SessionStatusActive &&
				(session.FinishedAt == nil || session.FinishedAt.After(sixMonthsAgo)) {
				continue
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		filtered = append(filtered, scenario)
	}

	for i := range filtered {
		runtimeScenario, err := r.steps.GetStepsByScenarioId(ctx, filtered[i].ID)
		if err != nil {
			return nil, err
		}
		if len(runtimeScenario.Steps) >= 1 {
			response.Scenarios = append(response.Scenarios, *runtimeScenario)
		}
	}

	return response, nil
}

func (r *RuntimeService) checkTestToken(ctx context.Context, pagePattern string) (*runtimeModel.RuntimeScenario, error) {
	testToken, ok := requestcontext.TestToken(ctx)
	if !ok || testToken == "" {
		return nil, errTestTokenIsEmpty
	}
	hash := r.tokenHasher.Hash(testToken)
	token, err := r.tokens.GetTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTestTokenInvalid
		}
		return nil, err
	}
	if !token.ExpiresAt.After(time.Now()) {
		return nil, ErrTestTokenInvalid
	}
	err = r.validateProjectKey(ctx, token.ScenarioID, pagePattern)
	if err != nil {
		return nil, err
	}
	return r.steps.GetStepsByScenarioId(ctx, token.ScenarioID)
}

func (r *RuntimeService) validateProjectKey(ctx context.Context, scenarioId, pagePattern string) error {
	projectKey, ok := requestcontext.ProjectKey(ctx)
	if !ok {
		return ErrProjectTokenIsNotValid
	}
	scenario, err := r.scenario.GetScenarioByIdAndProjectKey(ctx, scenarioId, projectKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTestTokenInvalid
		} else {
			return err
		}
	}
	if scenario.PagePattern != pagePattern {
		return ErrPageMismatch
	}
	return nil
}

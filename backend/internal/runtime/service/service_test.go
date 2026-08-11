package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DaniilSintsov/interactive-onboarding/backend/internal/platform/requestcontext"
	runtimeModel "github.com/DaniilSintsov/interactive-onboarding/backend/internal/runtime/model"
)

func TestFindScenariosFiltersRecentlyFinishedScenarios(t *testing.T) {
	completedRecently := runtimeModel.Scenario{ID: "completed-recently"}
	active := runtimeModel.Scenario{ID: "active"}
	completedLongAgo := runtimeModel.Scenario{ID: "completed-long-ago"}
	unseen := runtimeModel.Scenario{ID: "unseen"}
	scenarios := &scenarioRepositoryFake{scenarios: []runtimeModel.Scenario{
		completedRecently, active, completedLongAgo, unseen,
	}}
	sessions := &sessionRepositoryFake{sessions: map[string]*runtimeModel.Session{
		completedRecently.ID: {Status: runtimeModel.SessionStatusCompleted, FinishedAt: timePtr(time.Now().AddDate(0, -5, 0))},
		active.ID:            {Status: runtimeModel.SessionStatusActive},
		completedLongAgo.ID:  {Status: runtimeModel.SessionStatusSkipped, FinishedAt: timePtr(time.Now().AddDate(0, -7, 0))},
	}}
	steps := &stepRepositoryFake{scenarios: map[string]*runtimeModel.RuntimeScenario{
		active.ID:           {ID: active.ID, Steps: []runtimeModel.RuntimeStep{{}}},
		completedLongAgo.ID: {ID: completedLongAgo.ID, Steps: []runtimeModel.RuntimeStep{{}}},
		unseen.ID:           {ID: unseen.ID, Steps: []runtimeModel.RuntimeStep{{}}},
	}}

	response, err := newRuntimeService(scenarios, sessions, steps, nil, nil).FindScenarios(projectContext(), "/home", "user-1")

	if err != nil {
		t.Fatalf("FindScenarios() error = %v", err)
	}
	if response.IsTest {
		t.Fatal("FindScenarios() unexpectedly returned test mode")
	}
	if len(response.Scenarios) != 3 {
		t.Fatalf("FindScenarios() returned %d scenarios, want 3", len(response.Scenarios))
	}
	got := map[string]bool{}
	for _, scenario := range response.Scenarios {
		got[scenario.ID] = true
	}
	if got[completedRecently.ID] || !got[active.ID] || !got[completedLongAgo.ID] || !got[unseen.ID] {
		t.Fatalf("FindScenarios() scenarios = %#v, want active, unseen, and long-ago completed scenarios", response.Scenarios)
	}
	if sessions.calls[completedLongAgo.ID] != 1 {
		t.Fatalf("session lookup for swapped scenario = %d, want 1", sessions.calls[completedLongAgo.ID])
	}
}

func TestFindScenariosReturnsTestScenarioWithoutNormalLookups(t *testing.T) {
	const token = "test-token"
	scenarios := &scenarioRepositoryFake{scenarioByProject: &runtimeModel.Scenario{
		ID: "test-scenario", PagePattern: "/any-page",
	}}
	sessions := &sessionRepositoryFake{}
	steps := &stepRepositoryFake{scenarios: map[string]*runtimeModel.RuntimeScenario{
		"test-scenario": {ID: "test-scenario", Name: "Test", Steps: []runtimeModel.RuntimeStep{{}}},
	}}
	tokens := &testTokensRepositoryFake{token: &runtimeModel.TestToken{
		ScenarioID: "test-scenario",
		ExpiresAt:  time.Now().Add(time.Hour),
	}}
	hasher := &tokenHasherFake{hash: []byte("hashed-token")}
	ctx := requestcontext.WithTestToken(projectContext(), token)

	response, err := newRuntimeService(scenarios, sessions, steps, tokens, hasher).FindScenarios(ctx, "/any-page", "user-1")

	if err != nil {
		t.Fatalf("FindScenarios() error = %v", err)
	}
	if !response.IsTest || len(response.Scenarios) != 1 || response.Scenarios[0].ID != "test-scenario" {
		t.Fatalf("FindScenarios() response = %#v, want one test scenario", response)
	}
	if hasher.rawToken != token {
		t.Fatalf("hasher raw token = %q, want %q", hasher.rawToken, token)
	}
	if scenarios.normalLookupCalls != 0 || sessions.totalCalls() != 0 {
		t.Fatalf("test mode made normal lookups: scenarios=%d sessions=%d", scenarios.normalLookupCalls, sessions.totalCalls())
	}
}

func TestFindScenariosReturnsExpectedErrors(t *testing.T) {
	projectLookupErr := errors.New("project lookup failed")
	tests := []struct {
		name string
		ctx  context.Context
		svc  *RuntimeService
		want error
	}{
		{
			name: "missing project key",
			ctx:  context.Background(),
			svc:  newRuntimeService(&scenarioRepositoryFake{}, &sessionRepositoryFake{}, &stepRepositoryFake{}, nil, nil),
			want: ErrProjectTokenIsNotValid,
		},
		{
			name: "scenario lookup failure",
			ctx:  projectContext(),
			svc:  newRuntimeService(&scenarioRepositoryFake{normalErr: errors.New("scenario lookup failed")}, &sessionRepositoryFake{}, &stepRepositoryFake{}, nil, nil),
			want: errors.New("scenario lookup failed"),
		},
		{
			name: "session lookup failure",
			ctx:  projectContext(),
			svc: newRuntimeService(&scenarioRepositoryFake{scenarios: []runtimeModel.Scenario{{ID: "scenario-1"}}},
				&sessionRepositoryFake{err: errors.New("session lookup failed")}, &stepRepositoryFake{}, nil, nil),
			want: errors.New("session lookup failed"),
		},
		{
			name: "project lookup failure",
			ctx:  projectContext(),
			svc: NewRuntimeService(
				&scenarioRepositoryFake{}, &sessionRepositoryFake{}, &stepRepositoryFake{}, nil, nil,
				&projectRepositoryFake{err: projectLookupErr},
			),
			want: projectLookupErr,
		},
		{
			name: "unknown project key",
			ctx:  projectContext(),
			svc: NewRuntimeService(
				&scenarioRepositoryFake{}, &sessionRepositoryFake{}, &stepRepositoryFake{}, nil, nil,
				&projectRepositoryFake{err: sql.ErrNoRows},
			),
			want: ErrProjectTokenIsNotValid,
		},
		{
			name: "unknown test token",
			ctx:  requestcontext.WithTestToken(projectContext(), "token"),
			svc: newRuntimeService(&scenarioRepositoryFake{}, &sessionRepositoryFake{}, &stepRepositoryFake{},
				&testTokensRepositoryFake{err: sql.ErrNoRows}, &tokenHasherFake{}),
			want: ErrTestTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.FindScenarios(tt.ctx, "/home", "user-1")
			if err == nil || err.Error() != tt.want.Error() {
				t.Fatalf("FindScenarios() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFindScenariosValidatesTestToken(t *testing.T) {
	tokenLookupErr := errors.New("token lookup failed")
	scenarioLookupErr := errors.New("test scenario lookup failed")
	validToken := func() *runtimeModel.TestToken {
		return &runtimeModel.TestToken{
			ScenarioID: "scenario-1",
			ExpiresAt:  time.Now().Add(time.Hour),
		}
	}
	tests := []struct {
		name      string
		scenarios *scenarioRepositoryFake
		tokens    *testTokensRepositoryFake
		want      error
	}{
		{
			name:      "expired token",
			scenarios: &scenarioRepositoryFake{},
			tokens: &testTokensRepositoryFake{token: &runtimeModel.TestToken{
				ScenarioID: "scenario-1",
				ExpiresAt:  time.Now().Add(-time.Minute),
			}},
			want: ErrTestTokenInvalid,
		},
		{
			name:      "cross-project token",
			scenarios: &scenarioRepositoryFake{scenarioByProjErr: sql.ErrNoRows},
			tokens:    &testTokensRepositoryFake{token: validToken()},
			want:      ErrTestTokenInvalid,
		},
		{
			name: "page mismatch",
			scenarios: &scenarioRepositoryFake{scenarioByProject: &runtimeModel.Scenario{
				ID: "scenario-1", PagePattern: "/other-page",
			}},
			tokens: &testTokensRepositoryFake{token: validToken()},
			want:   ErrPageMismatch,
		},
		{
			name:      "unexpected token lookup error",
			scenarios: &scenarioRepositoryFake{},
			tokens:    &testTokensRepositoryFake{err: tokenLookupErr},
			want:      tokenLookupErr,
		},
		{
			name:      "unexpected scenario lookup error",
			scenarios: &scenarioRepositoryFake{scenarioByProjErr: scenarioLookupErr},
			tokens:    &testTokensRepositoryFake{token: validToken()},
			want:      scenarioLookupErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := requestcontext.WithTestToken(projectContext(), "token")
			svc := newRuntimeService(tt.scenarios, &sessionRepositoryFake{}, &stepRepositoryFake{}, tt.tokens, &tokenHasherFake{})

			_, err := svc.FindScenarios(ctx, "/home", "user-1")
			if !errors.Is(err, tt.want) {
				t.Fatalf("FindScenarios() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func projectContext() context.Context {
	return requestcontext.WithProjectKey(context.Background(), "project-key")
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func newRuntimeService(scenario ScenarioRepository, session SessionRepository, steps StepRepository, tokens TestTokensRepository, hasher TokenHasher) *RuntimeService {
	return NewRuntimeService(scenario, session, steps, tokens, hasher, &projectRepositoryFake{
		project: &runtimeModel.Project{ProjectKey: "project-key"},
	})
}

type scenarioRepositoryFake struct {
	scenarios         []runtimeModel.Scenario
	normalErr         error
	scenarioByProject *runtimeModel.Scenario
	scenarioByProjErr error
	normalLookupCalls int
}

func (*scenarioRepositoryFake) GetScenarioById(context.Context, string) (*runtimeModel.Scenario, error) {
	return nil, sql.ErrNoRows
}

func (r *scenarioRepositoryFake) GetScenarioByIdAndProjectKey(context.Context, string, string) (*runtimeModel.Scenario, error) {
	return r.scenarioByProject, r.scenarioByProjErr
}

func (r *scenarioRepositoryFake) GetScenariosByPagePatternAndProjectkey(context.Context, string, string) ([]runtimeModel.Scenario, error) {
	r.normalLookupCalls++
	return r.scenarios, r.normalErr
}

type sessionRepositoryFake struct {
	sessions map[string]*runtimeModel.Session
	err      error
	calls    map[string]int
}

func (r *sessionRepositoryFake) GetSessionByScenarioAndUser(_ context.Context, scenarioID, _ string) (*runtimeModel.Session, error) {
	if r.calls == nil {
		r.calls = make(map[string]int)
	}
	r.calls[scenarioID]++
	if r.err != nil {
		return nil, r.err
	}
	if session, ok := r.sessions[scenarioID]; ok {
		return session, nil
	}
	return nil, sql.ErrNoRows
}

func (r *sessionRepositoryFake) totalCalls() int {
	total := 0
	for _, calls := range r.calls {
		total += calls
	}
	return total
}

type stepRepositoryFake struct {
	scenarios map[string]*runtimeModel.RuntimeScenario
	err       error
}

func (r *stepRepositoryFake) GetStepsByScenarioId(_ context.Context, scenarioID string) (*runtimeModel.RuntimeScenario, error) {
	if r.err != nil {
		return nil, r.err
	}
	if scenario, ok := r.scenarios[scenarioID]; ok {
		return scenario, nil
	}
	return nil, sql.ErrNoRows
}

type testTokensRepositoryFake struct {
	token *runtimeModel.TestToken
	err   error
}

func (r *testTokensRepositoryFake) GetTokenByHash(context.Context, []byte) (*runtimeModel.TestToken, error) {
	return r.token, r.err
}

type tokenHasherFake struct {
	hash     []byte
	rawToken string
}

func (h *tokenHasherFake) Hash(rawToken string) []byte {
	h.rawToken = rawToken
	return h.hash
}

type projectRepositoryFake struct {
	project *runtimeModel.Project
	err     error
	calls   int
}

func (r *projectRepositoryFake) GetProjectByProjectKey(context.Context, string) (*runtimeModel.Project, error) {
	r.calls++
	return r.project, r.err
}

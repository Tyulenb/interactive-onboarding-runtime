package model

import (
	"encoding/json"
	"time"
)

type ResolveScenarioRequest struct {
	Page   string `json:"page" validate:"required,min=1,max=2048"`
	UserID string `json:"user_id" validate:"required,min=1,max=255"`
}

type RuntimeScenarioResolveResponse struct {
	IsTest    bool              `json:"is_test"`
	Scenarios []RuntimeScenario `json:"scenarios"`
}

type RuntimeScenario struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	PagePattern string        `json:"page_pattern"`
	Steps       []RuntimeStep `json:"steps"`
}

type RuntimeStep struct {
	ID           string          `json:"id"`
	StepNum      int             `json:"step_num"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	FrontendData json.RawMessage `json:"frontend_data"`
	Element      RuntimeElement  `json:"element"`
}

type RuntimeElement struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type User struct {
	UserId    string `json:"user_id"`
	Onboarded bool   `json:"onboarded"`
}

type ScenarioStatus string

const (
	ScenarioStatusEnabled       ScenarioStatus = "enabled"
	ScenarioStatusDisabled      ScenarioStatus = "disabled"
	ScenarioStatusInDevelopment ScenarioStatus = "in_development"
)

type Scenario struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	PagePattern string         `json:"page_pattern"`
	Status      ScenarioStatus `json:"status"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   *time.Time     `json:"deleted_at"`
}

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusSkipped   SessionStatus = "skipped"
)

type Session struct {
	ID         string        `json:"id"`
	ScenarioID string        `json:"scenario_id"`
	UserID     string        `json:"user_id"`
	Status     SessionStatus `json:"status"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt *time.Time    `json:"finished_at"`
}

type TestToken struct {
	ID         string    `json:"id"`
	ScenarioID string    `json:"scenario_id"`
	Hash       []byte    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Project struct {
	ID         string
	Name       string
	ProjectKey string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

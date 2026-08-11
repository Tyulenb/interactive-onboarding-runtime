package model

import (
	"encoding/json"
	"time"
)

type StartSessionRequest struct {
	ScenarioID string `json:"scenario_id"`
	UserID     string `json:"user_id"`
}

type CreateEventRequest struct {
	ID         string          `json:"id"`
	SessionID  string          `json:"session_id"`
	StepID     *string         `json:"step_id"`
	Type       EventType       `json:"type"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurred_at"`
}

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusSkipped   SessionStatus = "skipped"
)

type OnboardingSession struct {
	ID         string        `json:"id"`
	ScenarioID string        `json:"scenario_id"`
	UserID     string        `json:"user_id"`
	Status     SessionStatus `json:"status"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt *time.Time    `json:"finished_at"`
}

type EventType string

const (
	EventTypeStepShown           EventType = "step_shown"
	EventTypeStepCompleted       EventType = "step_completed"
	EventTypeStepSkipped         EventType = "step_skipped"
	EventTypeOnboardingCompleted EventType = "onboarding_completed"
	EventTypeOnboardingSkipped   EventType = "onboarding_skipped"
)

type OnboardingEvent struct {
	ID         string          `json:"id"`
	SessionID  string          `json:"session_id"`
	StepID     *string         `json:"step_id"`
	Type       EventType       `json:"type"`
	Data       json.RawMessage `json:"data"`
	OccurredAt time.Time       `json:"occurred_at"`
	ReceivedAt time.Time       `json:"received_at"`
}

type EventAcceptedResponse struct {
	Event     OnboardingEvent `json:"event"`
	Duplicate bool            `json:"duplicate"`
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

type Step struct {
	ID           string          `json:"id"`
	ScenarioID   string          `json:"scenario_id"`
	StepNum      int             `json:"step_num"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	ElementID    string          `json:"element_id"`
	FrontendData json.RawMessage `json:"frontend_data"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at"`
}

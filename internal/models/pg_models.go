package models

import (
	"encoding/json"
	"time"
)

// Terminal status constants for the relational execution model.
const (
	StatusRunning    = "RUNNING"
	StatusInProgress = "IN_PROGRESS"
	StatusPassed     = "PASSED"
	StatusFailed     = "FAILED"
	StatusTimedOut   = "TIMED_OUT"
	StatusCancelled  = "CANCELLED"
	StatusAborted    = "ABORTED"
	StatusSkipped    = "SKIPPED"
)

// IsTerminalStatus returns true if the given status represents a completed
// (terminal) state. Active statuses are RUNNING and IN_PROGRESS.
func IsTerminalStatus(status string) bool {
	switch status {
	case StatusPassed, StatusFailed, StatusTimedOut,
		StatusCancelled, StatusAborted, StatusSkipped:
		return true
	default:
		return false
	}
}

// Run represents a top-level logical test run stored in PostgreSQL.
type Run struct {
	ID            string          `json:"id"`
	LogicalRunKey string          `json:"logicalRunKey"`
	Source        string          `json:"source"`
	Project       string          `json:"project"`
	Pipeline      string          `json:"pipeline"`
	Branch        string          `json:"branch"`
	CommitSHA     string          `json:"commitSha"`
	Status        string          `json:"status"`
	StartedAt     time.Time       `json:"startedAt"`
	FinishedAt    *time.Time      `json:"finishedAt,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// RunShard represents a shard within a logical run.
type RunShard struct {
	ID                 string     `json:"id"`
	RunID              string     `json:"runId"`
	ShardKey           string     `json:"shardKey"`
	ShardIndex         int        `json:"shardIndex"`
	ShardCountExpected *int       `json:"shardCountExpected,omitempty"`
	Status             string     `json:"status"`
	StartedAt          time.Time  `json:"startedAt"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// Suite represents a test suite belonging to a run.
type Suite struct {
	ID              string          `json:"id"`
	RunID           string          `json:"runId"`
	ExternalSuiteID string          `json:"externalSuiteId"`
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	StartedAt       time.Time       `json:"startedAt"`
	FinishedAt      *time.Time      `json:"finishedAt,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// Test represents an individual test case belonging to a suite.
type Test struct {
	ID             string          `json:"id"`
	SuiteID        string          `json:"suiteId"`
	ExternalTestID string          `json:"externalTestId"`
	Name           string          `json:"name"`
	Status         string          `json:"status"`
	AttemptCount   int             `json:"attemptCount"`
	StartedAt      time.Time       `json:"startedAt"`
	FinishedAt     *time.Time      `json:"finishedAt,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// TestAttempt represents a single attempt/retry of a test case.
type TestAttempt struct {
	ID            string          `json:"id"`
	TestID        string          `json:"testId"`
	AttemptIndex  int             `json:"attemptIndex"`
	Status        string          `json:"status"`
	StartedAt     time.Time       `json:"startedAt"`
	FinishedAt    *time.Time      `json:"finishedAt,omitempty"`
	Steps         json.RawMessage `json:"steps,omitempty"`
	StepsRef      *string         `json:"stepsRef,omitempty"`
	StepCount     int             `json:"stepCount"`
	DurationMs    int64           `json:"durationMs"`
	FailureReason *string         `json:"failureReason,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

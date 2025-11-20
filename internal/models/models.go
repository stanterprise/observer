package models

import (
	"time"

	"gorm.io/datatypes"
)

// TestSuiteRun corresponds to entities.TestSuiteRun in protobuf definitions.
// Primary key is the server-side "id" provided by the client (authoritative).
type TestSuiteRun struct {
	ID              string            `gorm:"column:id;primaryKey;type:text"`
	Name            string            `gorm:"column:name;type:text"`
	Description     string            `gorm:"column:description;type:text"`
	Status          string            `gorm:"column:status;type:text"`
	Metadata        datatypes.JSONMap `gorm:"column:metadata;type:jsonb"`
	Duration        *int64            `gorm:"column:duration"` // Duration in nanoseconds
	TestSuiteSpecID string            `gorm:"column:test_suite_spec_id;type:text"`
	InitiatedBy     string            `gorm:"column:initiated_by;type:text"`
	ProjectName     string            `gorm:"column:project_name;type:text"`
	StartTime       *time.Time        `gorm:"column:start_time"`
	EndTime         *time.Time        `gorm:"column:end_time"`
	CreatedAt       time.Time         `gorm:"column:created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at"`
}

func (TestSuiteRun) TableName() string { return "test_suite_runs" }

// TestCaseRun corresponds to entities.TestCaseRun in protobuf definitions.
// Primary key is the server-side "id" provided by the client (authoritative).
// RunID is a client-supplied external identifier and is NOT the primary key.
type TestCaseRun struct {
	ID         string            `gorm:"column:id;primaryKey;type:text"`
	RunID      string            `gorm:"column:run_id;type:text"`
	Title      string            `gorm:"column:title;type:text"`
	Status     string            `gorm:"column:status;type:text"`
	Metadata   datatypes.JSONMap `gorm:"column:metadata;type:jsonb"`
	Duration   *int64            `gorm:"column:duration"` // Duration in nanoseconds
	RetryCount *int32            `gorm:"column:retry_count"`
	RetryIndex *int32            `gorm:"column:retry_index"`
	Timeout    *int32            `gorm:"column:timeout"` // Timeout in milliseconds
	CreatedAt  time.Time         `gorm:"column:created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at"`
}

func (TestCaseRun) TableName() string { return "test_case_runs" }

// StepRun corresponds to entities.StepRun in protobuf definitions.
// We persist an auto-increment ID for ordering, and link to the parent test case via test_case_run_id.
type StepRun struct {
	ID            string    `gorm:"column:id;primaryKey"`
	RunID         string    `gorm:"column:run_id;type:text"`
	TestCaseRunID string    `gorm:"column:test_case_run_id;type:text"`
	Status        string    `gorm:"column:status;type:text"`
	Category      string    `gorm:"column:category;type:text"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (StepRun) TableName() string { return "step_runs" }

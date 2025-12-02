package models

import (
	"time"
)

// TestRunDocument represents a complete test run as a single MongoDB document.
// This document-based structure follows the issue requirements where:
// - Suite begin creates a new document (if root suite) or appends to parent
// - Suite end updates existing attributes
// - Test begin appends to parent suite
// - Test end updates test attributes
// - Step begin appends to parent (test or step)
// - Step end updates step attributes
type TestRunDocument struct {
	ID              string                 `bson:"_id"`
	Name            string                 `bson:"name,omitempty"`
	Description     string                 `bson:"description,omitempty"`
	Status          string                 `bson:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty"` // Duration in nanoseconds
	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty"`
	InitiatedBy     string                 `bson:"initiated_by,omitempty"`
	ProjectName     string                 `bson:"project_name,omitempty"`
	StartTime       *time.Time             `bson:"start_time,omitempty"`
	EndTime         *time.Time             `bson:"end_time,omitempty"`
	CreatedAt       time.Time              `bson:"created_at"`
	UpdatedAt       time.Time              `bson:"updated_at"`

	// Embedded child suites for nested suite structures
	Suites []*SuiteDocument `bson:"suites,omitempty"`
	// Embedded test cases
	Tests []*TestDocument `bson:"tests,omitempty"`
}

// SuiteDocument represents a test suite embedded within a test run document.
// Non-root suites are appended to their parent suite's Suites array.
type SuiteDocument struct {
	ID              string                 `bson:"id"`
	ParentSuiteID   string                 `bson:"parent_suite_id,omitempty"`
	Name            string                 `bson:"name,omitempty"`
	Description     string                 `bson:"description,omitempty"`
	Status          string                 `bson:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty"`
	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty"`
	InitiatedBy     string                 `bson:"initiated_by,omitempty"`
	ProjectName     string                 `bson:"project_name,omitempty"`
	StartTime       *time.Time             `bson:"start_time,omitempty"`
	EndTime         *time.Time             `bson:"end_time,omitempty"`
	CreatedAt       time.Time              `bson:"created_at"`
	UpdatedAt       time.Time              `bson:"updated_at"`

	// Nested child suites
	Suites []*SuiteDocument `bson:"suites,omitempty"`
	// Test cases within this suite
	Tests []*TestDocument `bson:"tests,omitempty"`
}

// TestDocument represents a test case embedded within a suite or run document.
type TestDocument struct {
	ID         string                 `bson:"id"`
	RunID      string                 `bson:"run_id,omitempty"`
	SuiteID    string                 `bson:"suite_id,omitempty"`
	Title      string                 `bson:"title,omitempty"`
	Status     string                 `bson:"status,omitempty"`
	Metadata   map[string]interface{} `bson:"metadata,omitempty"`
	Duration   *int64                 `bson:"duration,omitempty"`
	RetryCount *int32                 `bson:"retry_count,omitempty"`
	RetryIndex *int32                 `bson:"retry_index,omitempty"`
	Timeout    *int32                 `bson:"timeout,omitempty"`
	CreatedAt  time.Time              `bson:"created_at"`
	UpdatedAt  time.Time              `bson:"updated_at"`

	// Embedded steps for this test
	Steps []*StepDocument `bson:"steps,omitempty"`
}

// StepDocument represents a test step embedded within a test or parent step.
type StepDocument struct {
	ID            string    `bson:"id"`
	RunID         string    `bson:"run_id,omitempty"`
	TestCaseRunID string    `bson:"test_case_run_id,omitempty"`
	ParentStepID  string    `bson:"parent_step_id,omitempty"`
	Status        string    `bson:"status,omitempty"`
	Category      string    `bson:"category,omitempty"`
	Title         string    `bson:"title,omitempty"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	// Nested steps (for step hierarchies)
	Steps []*StepDocument `bson:"steps,omitempty"`
}

// ConvertTestSuiteRunToDocument converts a GORM TestSuiteRun to document format
func ConvertTestSuiteRunToDocument(suite *TestSuiteRun) *TestRunDocument {
	if suite == nil {
		return nil
	}

	md := make(map[string]interface{})
	if suite.Metadata != nil {
		md = suite.Metadata
	}

	return &TestRunDocument{
		ID:              suite.ID,
		Name:            suite.Name,
		Description:     suite.Description,
		Status:          suite.Status,
		Metadata:        md,
		Duration:        suite.Duration,
		TestSuiteSpecID: suite.TestSuiteSpecID,
		InitiatedBy:     suite.InitiatedBy,
		ProjectName:     suite.ProjectName,
		StartTime:       suite.StartTime,
		EndTime:         suite.EndTime,
		CreatedAt:       suite.CreatedAt,
		UpdatedAt:       suite.UpdatedAt,
		Suites:          []*SuiteDocument{},
		Tests:           []*TestDocument{},
	}
}

// ConvertTestCaseRunToDocument converts a GORM TestCaseRun to document format
func ConvertTestCaseRunToDocument(tc *TestCaseRun) *TestDocument {
	if tc == nil {
		return nil
	}

	md := make(map[string]interface{})
	if tc.Metadata != nil {
		md = tc.Metadata
	}

	return &TestDocument{
		ID:         tc.ID,
		RunID:      tc.RunID,
		Title:      tc.Title,
		Status:     tc.Status,
		Metadata:   md,
		Duration:   tc.Duration,
		RetryCount: tc.RetryCount,
		RetryIndex: tc.RetryIndex,
		Timeout:    tc.Timeout,
		CreatedAt:  tc.CreatedAt,
		UpdatedAt:  tc.UpdatedAt,
		Steps:      []*StepDocument{},
	}
}

// ConvertStepRunToDocument converts a GORM StepRun to document format
func ConvertStepRunToDocument(step *StepRun) *StepDocument {
	if step == nil {
		return nil
	}

	return &StepDocument{
		ID:            step.ID,
		RunID:         step.RunID,
		TestCaseRunID: step.TestCaseRunID,
		ParentStepID:  step.ParentStepID,
		Status:        step.Status,
		Category:      step.Category,
		Title:         step.Title,
		CreatedAt:     step.CreatedAt,
		UpdatedAt:     step.UpdatedAt,
		Steps:         []*StepDocument{},
	}
}

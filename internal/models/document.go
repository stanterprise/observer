package models

import (
	"time"
)

// TestRunDocument represents a complete test run as a single MongoDB document.
// This document-based structure follows the repository implementation where:
// - New runID creates a new test run document
// - Suite end updates existing suite attributes
// - Test begin appends to parent suite's Tests array
// - Test end updates test attributes
// - Step begin appends to parent (test or step) Steps array
// - Step end updates step attributes
type TestRunDocument struct {
	ID              string                 `bson:"_id"`
	Name            string                 `bson:"name,omitempty"`
	Description     string                 `bson:"description,omitempty"`
	Status          string                 `bson:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty"` // Duration in nanoseconds
	TotalTests      int32                  `bson:"total_tests,omitempty"`
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
	RunID           string                 `bson:"run_id,omitempty"`
	ParentSuiteID   string                 `bson:"parent_suite_id,omitempty"`
	Name            string                 `bson:"name,omitempty"`
	Description     string                 `bson:"description,omitempty"`
	Status          string                 `bson:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty"`
	Location        string                 `bson:"location,omitempty"`
	Type            string                 `bson:"type,omitempty"`
	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty"`
	InitiatedBy     string                 `bson:"initiated_by,omitempty"`
	ProjectName     string                 `bson:"project_name,omitempty"`
	Author          string                 `bson:"author,omitempty"`
	Owner           string                 `bson:"owner,omitempty"`
	TestCaseIds     []string               `bson:"test_case_ids,omitempty"`
	SubSuiteIds     []string               `bson:"sub_suite_ids,omitempty"`
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
	ID          string                 `bson:"id"`
	Name        string                 `bson:"name,omitempty"` // Same as Title, for protobuf compatibility
	Title       string                 `bson:"title,omitempty"`
	Description string                 `bson:"description,omitempty"`
	RunID       string                 `bson:"run_id,omitempty"`
	SuiteID     string                 `bson:"suite_id,omitempty"`
	Status      string                 `bson:"status,omitempty"`
	StartTime   *time.Time             `bson:"start_time,omitempty"`
	EndTime     *time.Time             `bson:"end_time,omitempty"`
	Duration    *int64                 `bson:"duration,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty"`
	Tags        []string               `bson:"tags,omitempty"`
	Location    string                 `bson:"location,omitempty"`
	RetryCount  *int32                 `bson:"retry_count,omitempty"`
	RetryIndex  *int32                 `bson:"retry_index,omitempty"`
	Timeout     *int32                 `bson:"timeout,omitempty"`

	// Legacy single error fields (for backward compatibility with older events)
	ErrorMessage string `bson:"error_message,omitempty"`
	StackTrace   string `bson:"stack_trace,omitempty"`

	// Attachments directly on the test (separate from failure/error attachments)
	Attachments []map[string]interface{} `bson:"attachments,omitempty"`

	// Test failures and errors (structured documents from new events)
	Failures []*TestFailureDocument `bson:"failures,omitempty"`
	Errors   []*TestErrorDocument   `bson:"errors,omitempty"`

	// Error list (simple string array, different from Errors documents)
	ErrorList []string `bson:"error_list,omitempty"`

	// Standard output and error streams
	StdOut []*OutputDocument `bson:"stdout,omitempty"`
	StdErr []*OutputDocument `bson:"stderr,omitempty"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`

	// Embedded steps for this test
	Steps []*StepDocument `bson:"steps,omitempty"`
}

// StepDocument represents a test step embedded within a test or parent step.
type StepDocument struct {
	ID            string                 `bson:"id"`
	RunID         string                 `bson:"run_id,omitempty"`
	TestCaseRunID string                 `bson:"test_case_run_id,omitempty"`
	ParentStepID  string                 `bson:"parent_step_id,omitempty"`
	Title         string                 `bson:"title,omitempty"`
	Description   string                 `bson:"description,omitempty"`
	StartTime     *time.Time             `bson:"start_time,omitempty"`
	Duration      *int64                 `bson:"duration,omitempty"`
	Type          string                 `bson:"type,omitempty"`
	Metadata      map[string]interface{} `bson:"metadata,omitempty"`
	WorkerIndex   string                 `bson:"worker_index,omitempty"`
	Status        string                 `bson:"status,omitempty"`
	Category      string                 `bson:"category,omitempty"`
	Location      string                 `bson:"location,omitempty"`

	// Error fields
	Error  string   `bson:"error,omitempty"`  // Single error message
	Errors []string `bson:"errors,omitempty"` // Array of error messages

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`

	// Nested steps (for step hierarchies)
	Steps []*StepDocument `bson:"steps,omitempty"`
}

// TestFailureDocument represents a test failure with details
type TestFailureDocument struct {
	FailureMessage string                   `bson:"failure_message,omitempty"`
	StackTrace     string                   `bson:"stack_trace,omitempty"`
	Timestamp      *time.Time               `bson:"timestamp,omitempty"`
	Attachments    []map[string]interface{} `bson:"attachments,omitempty"`
}

// TestErrorDocument represents a test error with details
type TestErrorDocument struct {
	ErrorMessage string                   `bson:"error_message,omitempty"`
	StackTrace   string                   `bson:"stack_trace,omitempty"`
	Timestamp    *time.Time               `bson:"timestamp,omitempty"`
	Attachments  []map[string]interface{} `bson:"attachments,omitempty"`
}

// OutputDocument represents stdout or stderr output
type OutputDocument struct {
	Message   string     `bson:"message,omitempty"`
	Timestamp *time.Time `bson:"timestamp,omitempty"`
}

// Type aliases for backward compatibility with GraphQL generated code
// These allow the existing GraphQL schema to work with the new MongoDB document models
type (
	TestCaseRun = TestDocument
	StepRun     = StepDocument
)

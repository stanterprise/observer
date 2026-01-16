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
	ID              string                 `bson:"_id" json:"id"`
	Name            string                 `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                 `bson:"description,omitempty" json:"description,omitempty"`
	Status          string                 `bson:"status,omitempty" json:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty" json:"duration,omitempty"` // Duration in nanoseconds
	TotalTests      int32                  `bson:"total_tests,omitempty" json:"totalTests,omitempty"`
	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty" json:"testSuiteSpecId,omitempty"`
	InitiatedBy     string                 `bson:"initiated_by,omitempty" json:"initiatedBy,omitempty"`
	ProjectName     string                 `bson:"project_name,omitempty" json:"projectName,omitempty"`
	StartTime       *time.Time             `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime         *time.Time             `bson:"end_time,omitempty" json:"endTime,omitempty"`
	CreatedAt       time.Time              `bson:"created_at" json:"createdAt"`
	UpdatedAt       time.Time              `bson:"updated_at" json:"updatedAt"`

	// Embedded child suites for nested suite structures
	Suites []*SuiteDocument `bson:"suites,omitempty" json:"suites,omitempty"`
	// Embedded test cases
	Tests []*TestDocument `bson:"tests,omitempty" json:"tests,omitempty"`
}

// SuiteDocument represents a test suite embedded within a test run document.
// Non-root suites are appended to their parent suite's Suites array.
type SuiteDocument struct {
	ID              string                 `bson:"id" json:"id"`
	RunID           string                 `bson:"run_id,omitempty" json:"runId,omitempty"`
	ParentSuiteID   string                 `bson:"parent_suite_id,omitempty" json:"parentSuiteId,omitempty"`
	Name            string                 `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                 `bson:"description,omitempty" json:"description,omitempty"`
	Status          string                 `bson:"status,omitempty" json:"status,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Duration        *int64                 `bson:"duration,omitempty" json:"duration,omitempty"`
	Location        string                 `bson:"location,omitempty" json:"location,omitempty"`
	Type            string                 `bson:"type,omitempty" json:"type,omitempty"`
	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty" json:"testSuiteSpecId,omitempty"`
	InitiatedBy     string                 `bson:"initiated_by,omitempty" json:"initiatedBy,omitempty"`
	ProjectName     string                 `bson:"project_name,omitempty" json:"projectName,omitempty"`
	Author          string                 `bson:"author,omitempty" json:"author,omitempty"`
	Owner           string                 `bson:"owner,omitempty" json:"owner,omitempty"`
	TestCaseIds     []string               `bson:"test_case_ids,omitempty" json:"testCaseIds,omitempty"`
	SubSuiteIds     []string               `bson:"sub_suite_ids,omitempty" json:"subSuiteIds,omitempty"`
	TestCases       []TestDocument         `bson:"test_cases,omitempty" json:"testCases,omitempty"`
	Tags            []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	StartTime       *time.Time             `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime         *time.Time             `bson:"end_time,omitempty" json:"endTime,omitempty"`
	CreatedAt       time.Time              `bson:"created_at" json:"createdAt"`
	UpdatedAt       time.Time              `bson:"updated_at" json:"updatedAt"`

	// Nested child suites
	Suites []*SuiteDocument `bson:"suites,omitempty" json:"suites,omitempty"`
	// Test cases within this suite
	Tests []*TestDocument `bson:"tests,omitempty" json:"tests,omitempty"`
}

// TestDocument represents a test case embedded within a suite or run document.
type TestDocument struct {
	ID          string                 `bson:"id" json:"id"`
	Name        string                 `bson:"name,omitempty" json:"name,omitempty"` // Same as Title, for protobuf compatibility
	Title       string                 `bson:"title,omitempty" json:"title,omitempty"`
	Description string                 `bson:"description,omitempty" json:"description,omitempty"`
	RunID       string                 `bson:"run_id,omitempty" json:"runId,omitempty"`
	SuiteID     string                 `bson:"suite_id,omitempty" json:"suiteId,omitempty"`
	Status      string                 `bson:"status,omitempty" json:"status,omitempty"`
	StartTime   *time.Time             `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime     *time.Time             `bson:"end_time,omitempty" json:"endTime,omitempty"`
	Duration    *int64                 `bson:"duration,omitempty" json:"duration,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Tags        []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	Location    string                 `bson:"location,omitempty" json:"location,omitempty"`
	RetryCount  *int32                 `bson:"retry_count,omitempty" json:"retryCount,omitempty"`
	RetryIndex  *int32                 `bson:"retry_index,omitempty" json:"retryIndex,omitempty"`
	Timeout     *int32                 `bson:"timeout,omitempty" json:"timeout,omitempty"`

	// Legacy single error fields (for backward compatibility with older events)
	ErrorMessage string `bson:"error_message,omitempty" json:"errorMessage,omitempty"`
	StackTrace   string `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`
	// Attachments directly on the test (separate from failure/error attachments)
	Attachments []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`

	// Test failures and errors (structured documents from new events)
	Failures []*TestFailureDocument `bson:"failures,omitempty" json:"failures,omitempty"`
	Errors   []*TestErrorDocument   `bson:"errors,omitempty" json:"errors,omitempty"`

	// Error list (simple string array, different from Errors documents)
	ErrorList []string `bson:"error_list,omitempty" json:"errorList,omitempty"`
	// Standard output and error streams
	StdOut []*OutputDocument `bson:"stdout,omitempty" json:"stdout,omitempty"`
	StdErr []*OutputDocument `bson:"stderr,omitempty" json:"stderr,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`

	// Embedded steps for this test
	Steps []*StepDocument `bson:"steps,omitempty" json:"steps,omitempty"`
}

// StepDocument represents a test step embedded within a test or parent step.
type StepDocument struct {
	ID            string                 `bson:"id" json:"id"`
	RunID         string                 `bson:"run_id,omitempty" json:"runId,omitempty"`
	TestCaseRunID string                 `bson:"test_case_run_id,omitempty" json:"testCaseRunId,omitempty"`
	ParentStepID  string                 `bson:"parent_step_id,omitempty" json:"parentStepId,omitempty"`
	Title         string                 `bson:"title,omitempty" json:"title,omitempty"`
	Description   string                 `bson:"description,omitempty" json:"description,omitempty"`
	StartTime     *time.Time             `bson:"start_time,omitempty" json:"startTime,omitempty"`
	Duration      *int64                 `bson:"duration,omitempty" json:"duration,omitempty"`
	Type          string                 `bson:"type,omitempty" json:"type,omitempty"`
	Metadata      map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Tags          []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	WorkerIndex   string                 `bson:"worker_index,omitempty" json:"workerIndex,omitempty"`
	Status        string                 `bson:"status,omitempty" json:"status,omitempty"`
	Category      string                 `bson:"category,omitempty" json:"category,omitempty"`
	Location      string                 `bson:"location,omitempty" json:"location,omitempty"`
	RetryIndex    int32                  `bson:"retry_index,omitempty" json:"retryIndex,omitempty"`

	// Error fields
	Error     string    `bson:"error,omitempty" json:"error,omitempty"`   // Single error message
	Errors    []string  `bson:"errors,omitempty" json:"errors,omitempty"` // Array of error messages
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`

	// Nested steps (for step hierarchies)
	Steps []*StepDocument `bson:"steps,omitempty" json:"steps,omitempty"`
}

// TestFailureDocument represents a test failure with details
type TestFailureDocument struct {
	FailureMessage string                   `bson:"failure_message,omitempty" json:"failureMessage,omitempty"`
	StackTrace     string                   `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`
	Timestamp      *time.Time               `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	Attachments    []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`
}

// TestErrorDocument represents a test error with details
type TestErrorDocument struct {
	ErrorMessage string                   `bson:"error_message,omitempty" json:"errorMessage,omitempty"`
	StackTrace   string                   `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`
	Timestamp    *time.Time               `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	Attachments  []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`
}

// OutputDocument represents stdout or stderr output
type OutputDocument struct {
	Message   string     `bson:"message,omitempty" json:"message,omitempty"`
	Timestamp *time.Time `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
}

// Type aliases for backward compatibility with GraphQL generated code
// These allow the existing GraphQL schema to work with the new MongoDB document models
type (
	TestCaseRun = TestDocument
	StepRun     = StepDocument
)

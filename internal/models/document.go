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
	// ActiveTestSteps buffers in-flight step trees by test id while the attempt is running.
	ActiveTestSteps map[string]*ActiveTestStepsDocument `bson:"active_test_steps,omitempty" json:"activeTestSteps,omitempty"`
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

// AttemptDocument represents a single test attempt/retry.
// Each test can have multiple attempts based on retry_count.
type AttemptDocument struct {
	RetryIndex   int32                    `bson:"retry_index" json:"retryIndex"`
	Steps        []*StepDocument          `bson:"steps,omitempty" json:"steps,omitempty"`
	Status       string                   `bson:"status,omitempty" json:"status,omitempty"`
	StartTime    *time.Time               `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime      *time.Time               `bson:"end_time,omitempty" json:"endTime,omitempty"`
	Duration     *int64                   `bson:"duration,omitempty" json:"duration,omitempty"`
	Attachments  []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`
	ErrorMessage string                   `bson:"error_message,omitempty" json:"errorMessage,omitempty"`
	StackTrace   string                   `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`
	ErrorList    []string                 `bson:"error_list,omitempty" json:"errorList,omitempty"`
	Failures     []*TestFailureDocument   `bson:"failures,omitempty" json:"failures,omitempty"`
	Errors       []*TestErrorDocument     `bson:"errors,omitempty" json:"errors,omitempty"`
	StdOut       []*OutputDocument        `bson:"stdout,omitempty" json:"stdout,omitempty"`
	StdErr       []*OutputDocument        `bson:"stderr,omitempty" json:"stderr,omitempty"`
	CreatedAt    time.Time                `bson:"created_at" json:"createdAt"`
	UpdatedAt    time.Time                `bson:"updated_at" json:"updatedAt"`
}

// ActiveTestStepsDocument stores transient in-flight steps for a single active test attempt.
// The owning TestRunDocument keys this object by test id in ActiveTestSteps.
type ActiveTestStepsDocument struct {
	TestID         string          `bson:"test_id" json:"testId"`
	RetryIndex     int32           `bson:"retry_index" json:"retryIndex"`
	Status         string          `bson:"status,omitempty" json:"status,omitempty"`
	Steps          []*StepDocument `bson:"steps,omitempty" json:"steps,omitempty"`
	FirstEventAt   *time.Time      `bson:"first_event_at,omitempty" json:"firstEventAt,omitempty"`
	LastEventAt    *time.Time      `bson:"last_event_at,omitempty" json:"lastEventAt,omitempty"`
	FlushStartedAt *time.Time      `bson:"flush_started_at,omitempty" json:"flushStartedAt,omitempty"`
	CreatedAt      time.Time       `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `bson:"updated_at" json:"updatedAt"`
}

// TestDocument represents a test case embedded within a suite or run document.
// With attempt-based retries: each test has an Attempts array containing per-attempt data.
// Test-level Status/StartTime/EndTime/Duration represent aggregated values across attempts.
type TestDocument struct {
	ID          string `bson:"id" json:"id"`
	Name        string `bson:"name,omitempty" json:"name,omitempty"` // Same as Title, for protobuf compatibility
	Title       string `bson:"title,omitempty" json:"title,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	RunID       string `bson:"run_id,omitempty" json:"runId,omitempty"`
	SuiteID     string `bson:"suite_id,omitempty" json:"suiteId,omitempty"`

	// Status mirrors the status of attempts[retry_index]
	Status string `bson:"status,omitempty" json:"status,omitempty"`

	// StartTime is the earliest (attempt[0].start_time)
	StartTime *time.Time `bson:"start_time,omitempty" json:"startTime,omitempty"`

	// EndTime is the latest (current attempt's end_time)
	EndTime *time.Time `bson:"end_time,omitempty" json:"endTime,omitempty"`

	// Duration is from the current attempt
	Duration *int64 `bson:"duration,omitempty" json:"duration,omitempty"`

	Metadata   map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Tags       []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	Location   string                 `bson:"location,omitempty" json:"location,omitempty"`
	RetryCount *int32                 `bson:"retry_count,omitempty" json:"retryCount,omitempty"`

	// RetryIndex indicates which attempt is currently active
	RetryIndex *int32 `bson:"retry_index,omitempty" json:"retryIndex,omitempty"`

	Timeout *int32 `bson:"timeout,omitempty" json:"timeout,omitempty"`

	// Attempts array: sized to retry_count+1, indexed by retry_index
	Attempts []*AttemptDocument `bson:"attempts,omitempty" json:"attempts,omitempty"`

	// DEPRECATED: Legacy single error fields (for backward compatibility with older events)
	// New code should use attempts[retry_index].error_message instead
	ErrorMessage string `bson:"error_message,omitempty" json:"errorMessage,omitempty"`

	// DEPRECATED: Use attempts[retry_index].stack_trace instead
	StackTrace string `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`

	// DEPRECATED: Use attempts[retry_index].attachments instead
	Attachments []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`

	// DEPRECATED: Use attempts[retry_index].failures instead
	Failures []*TestFailureDocument `bson:"failures,omitempty" json:"failures,omitempty"`

	// DEPRECATED: Use attempts[retry_index].errors instead
	Errors []*TestErrorDocument `bson:"errors,omitempty" json:"errors,omitempty"`

	// DEPRECATED: Use attempts[retry_index].error_list instead
	ErrorList []string `bson:"error_list,omitempty" json:"errorList,omitempty"`

	// DEPRECATED: Use attempts[retry_index].stdout instead
	StdOut []*OutputDocument `bson:"stdout,omitempty" json:"stdout,omitempty"`

	// DEPRECATED: Use attempts[retry_index].stderr instead
	StdErr []*OutputDocument `bson:"stderr,omitempty" json:"stderr,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`

	// DEPRECATED: Use attempts[retry_index].steps instead
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

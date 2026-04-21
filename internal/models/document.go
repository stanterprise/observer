package models

import (
	"time"
)

// TestRunDocument represents the MongoDB document structure for a test run, including active test steps buffering.
type TestRunDocument struct {
	ID string `bson:"_id" json:"id"`

	// ActiveTestSteps buffers in-flight step trees by test id while the attempt is running.
	ActiveTestSteps map[string]*ActiveTestStepsDocument `bson:"active_test_steps,omitempty" json:"activeTestSteps,omitempty"`
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
	TTLAt          *time.Time      `bson:"ttl_at,omitempty" json:"ttlAt,omitempty"`
	CreatedAt      time.Time       `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `bson:"updated_at" json:"updatedAt"`
}

// LiveStepBufferDocument is the standalone collection representation of an
// active step buffer. The collection is initialized separately so later phases
// can cut over from the embedded run-document buffer without another schema hop.
type LiveStepBufferDocument struct {
	ID             string          `bson:"_id" json:"id"`
	RunID          string          `bson:"run_id" json:"runId"`
	TestID         string          `bson:"test_id" json:"testId"`
	AttemptIndex   int32           `bson:"attempt_index" json:"attemptIndex"`
	Status         string          `bson:"status,omitempty" json:"status,omitempty"`
	Steps          []*StepDocument `bson:"steps,omitempty" json:"steps,omitempty"`
	FirstEventAt   *time.Time      `bson:"first_event_at,omitempty" json:"firstEventAt,omitempty"`
	LastEventAt    *time.Time      `bson:"last_event_at,omitempty" json:"lastEventAt,omitempty"`
	FlushStartedAt *time.Time      `bson:"flush_started_at,omitempty" json:"flushStartedAt,omitempty"`
	TTLAt          *time.Time      `bson:"ttl_at,omitempty" json:"ttlAt,omitempty"`
	CreatedAt      time.Time       `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `bson:"updated_at" json:"updatedAt"`
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

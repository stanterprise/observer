package models

import (
	"encoding/json"
	"time"
)

// TestRun maps to the PostgreSQL runs table.
// It intentionally mirrors TestRunDocument fields where practical.
type TestRun struct {
	ID          string                 `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name        string                 `gorm:"column:name;type:text" json:"name,omitempty"`
	Description string                 `gorm:"column:description;type:text" json:"description,omitempty"`
	Status      string                 `gorm:"column:status;type:text;index:idx_runs_status_started_at,priority:1" json:"status,omitempty"`
	Metadata    map[string]interface{} `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata,omitempty"`
	Duration    *int64                 `gorm:"column:duration" json:"duration,omitempty"`
	TotalTests  int32                  `gorm:"column:total_tests" json:"totalTests,omitempty"`
	InitiatedBy string                 `gorm:"column:initiated_by;type:text" json:"initiatedBy,omitempty"`
	ProjectName string                 `gorm:"column:project_name;type:text" json:"projectName,omitempty"`
	StartTime   *time.Time             `gorm:"column:started_at;index:idx_runs_status_started_at,priority:2;index:idx_runs_started_at" json:"startTime,omitempty"`
	EndTime     *time.Time             `gorm:"column:finished_at" json:"endTime,omitempty"`
	CreatedAt   time.Time              `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time              `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Shards []RunShard `gorm:"foreignKey:RunID;references:ID" json:"shards,omitempty"`
	Suites []Suite    `gorm:"foreignKey:RunID;references:ID" json:"suites,omitempty"`
	Tests  []Test     `gorm:"foreignKey:RunID;references:ID" json:"tests,omitempty"`
}

func (TestRun) TableName() string {
	return "runs"
}

// RunShard maps to the PostgreSQL run_shards table.
type RunShard struct {
	ID                 string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID              string     `gorm:"column:run_id;type:text;not null;index:idx_run_shards_run_status,priority:1;uniqueIndex:ux_run_shards_run_id_shard_key,priority:1" json:"runId"`
	ShardKey           string     `gorm:"column:shard_key;type:text;not null;uniqueIndex:ux_run_shards_run_id_shard_key,priority:2" json:"shardKey"`
	ShardIndex         *int32     `gorm:"column:shard_index" json:"shardIndex,omitempty"`
	ShardCountExpected *int32     `gorm:"column:shard_count_expected" json:"shardCountExpected,omitempty"`
	Status             string     `gorm:"column:status;type:text;index:idx_run_shards_run_status,priority:2" json:"status,omitempty"`
	StartTime          *time.Time `gorm:"column:started_at" json:"startTime,omitempty"`
	EndTime            *time.Time `gorm:"column:finished_at" json:"endTime,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (RunShard) TableName() string {
	return "run_shards"
}

// Suite maps to the PostgreSQL suites table.
type Suite struct {
	ID              string                 `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID           string                 `gorm:"column:run_id;type:text;not null;index:idx_suites_run_id;index:idx_suites_run_status,priority:1" json:"runId,omitempty"`
	ParentSuiteID   *string                `gorm:"column:parent_suite_id;type:text" json:"parentSuiteId,omitempty"`
	Name            string                 `gorm:"column:name;type:text" json:"name,omitempty"`
	Description     string                 `gorm:"column:description;type:text" json:"description,omitempty"`
	Status          string                 `gorm:"column:status;type:text;index:idx_suites_run_status,priority:2" json:"status,omitempty"`
	Metadata        map[string]interface{} `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata,omitempty"`
	Duration        *int64                 `gorm:"column:duration" json:"duration,omitempty"`
	Location        string                 `gorm:"column:location;type:text" json:"location,omitempty"`
	Type            string                 `gorm:"column:type;type:text" json:"type,omitempty"`
	TestSuiteSpecID string                 `gorm:"column:test_suite_spec_id;type:text" json:"testSuiteSpecId,omitempty"`
	InitiatedBy     string                 `gorm:"column:initiated_by;type:text" json:"initiatedBy,omitempty"`
	ProjectName     string                 `gorm:"column:project_name;type:text" json:"projectName,omitempty"`
	Author          string                 `gorm:"column:author;type:text" json:"author,omitempty"`
	Owner           string                 `gorm:"column:owner;type:text" json:"owner,omitempty"`
	TestCaseIDs     []string               `gorm:"column:test_case_ids;type:jsonb;serializer:json" json:"testCaseIds,omitempty"`
	SubSuiteIDs     []string               `gorm:"column:sub_suite_ids;type:jsonb;serializer:json" json:"subSuiteIds,omitempty"`
	Tags            []string               `gorm:"column:tags;type:jsonb;serializer:json" json:"tags,omitempty"`
	StartTime       *time.Time             `gorm:"column:started_at" json:"startTime,omitempty"`
	EndTime         *time.Time             `gorm:"column:finished_at" json:"endTime,omitempty"`
	CreatedAt       time.Time              `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time              `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Suites []Suite `gorm:"foreignKey:ParentSuiteID;references:ID" json:"suites,omitempty"`
	Tests  []Test  `gorm:"foreignKey:SuiteID;references:ID" json:"tests,omitempty"`
}

func (Suite) TableName() string {
	return "suites"
}

// Test maps to the PostgreSQL tests table.
type Test struct {
	ID          string                 `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID       string                 `gorm:"column:run_id;type:text;not null;index:idx_tests_run_id" json:"runId,omitempty"`
	SuiteID     *string                `gorm:"column:suite_id;type:text;not null;index:idx_tests_suite_status,priority:1" json:"suiteId,omitempty"`
	Name        string                 `gorm:"column:name;type:text" json:"name,omitempty"`
	Title       string                 `gorm:"column:title;type:text" json:"title,omitempty"`
	Description string                 `gorm:"column:description;type:text" json:"description,omitempty"`
	Status      string                 `gorm:"column:status;type:text;index:idx_tests_suite_status,priority:2" json:"status,omitempty"`
	StartTime   *time.Time             `gorm:"column:started_at" json:"startTime,omitempty"`
	EndTime     *time.Time             `gorm:"column:finished_at" json:"endTime,omitempty"`
	Duration    *int64                 `gorm:"column:duration" json:"duration,omitempty"`
	Metadata    map[string]interface{} `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata,omitempty"`
	Tags        []string               `gorm:"column:tags;type:jsonb;serializer:json" json:"tags,omitempty"`
	Location    string                 `gorm:"column:location;type:text" json:"location,omitempty"`
	RetryCount  *int32                 `gorm:"column:retry_count" json:"retryCount,omitempty"`
	RetryIndex  *int32                 `gorm:"column:retry_index" json:"retryIndex,omitempty"`
	Timeout     *int32                 `gorm:"column:timeout" json:"timeout,omitempty"`
	CreatedAt   time.Time              `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time              `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Attempts []TestAttempt `gorm:"foreignKey:TestID;references:ID" json:"attempts,omitempty"`
}

func (Test) TableName() string {
	return "tests"
}

// TestAttempt maps to PostgreSQL test_attempts.
// Steps are stored on the attempt row as requested.
type TestAttempt struct {
	ID           string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID        string     `gorm:"column:run_id;type:text;not null;index:idx_attempts_run_id" json:"runId,omitempty"`
	TestID       string     `gorm:"column:test_id;type:text;not null;index:idx_attempts_test_attempt,priority:1;uniqueIndex:ux_attempts_test_attempt_index,priority:1" json:"testId"`
	AttemptIndex int32      `gorm:"column:attempt_index;not null;index:idx_attempts_test_attempt,priority:2;uniqueIndex:ux_attempts_test_attempt_index,priority:2" json:"attemptIndex"`
	Status       string     `gorm:"column:status;type:text;index:idx_attempts_status_finished_at,priority:1" json:"status,omitempty"`
	StartTime    *time.Time `gorm:"column:started_at" json:"startTime,omitempty"`
	EndTime      *time.Time `gorm:"column:finished_at;index:idx_attempts_status_finished_at,priority:2" json:"endTime,omitempty"`
	Duration     *int64     `gorm:"column:duration" json:"duration,omitempty"`

	// Steps holds the step array containing step trees serialized as jsonb.
	// Go type is json.RawMessage (provisional — concrete typed decode at read time).
	Steps *json.RawMessage `gorm:"column:steps;type:jsonb" json:"steps,omitempty"`

	Attachments  []map[string]interface{} `gorm:"column:attachments;type:jsonb;serializer:json" json:"attachments,omitempty"`
	ErrorMessage string                   `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	StackTrace   string                   `gorm:"column:stack_trace;type:text" json:"stackTrace,omitempty"`
	ErrorList    []string                 `gorm:"column:error_list;type:jsonb;serializer:json" json:"errorList,omitempty"`
	Failures     []*TestFailureDocument   `gorm:"column:failures;type:jsonb;serializer:json" json:"failures,omitempty"`
	Errors       []*TestErrorDocument     `gorm:"column:errors;type:jsonb;serializer:json" json:"errors,omitempty"`
	StdOut       []*OutputDocument        `gorm:"column:stdout;type:jsonb;serializer:json" json:"stdout,omitempty"`
	StdErr       []*OutputDocument        `gorm:"column:stderr;type:jsonb;serializer:json" json:"stderr,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	AttachmentsRows []Attachment `gorm:"foreignKey:TestAttemptID;references:ID" json:"attachmentsRows,omitempty"`
}

func (TestAttempt) TableName() string {
	return "test_attempts"
}

// Attachment maps to PostgreSQL attachments metadata.
type Attachment struct {
	ID            string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID         string     `gorm:"column:run_id;type:text;not null;index:idx_attachments_run" json:"runId"`
	TestID        string     `gorm:"column:test_id;type:text;not null;index:idx_attachments_test" json:"testId"`
	TestAttemptID string     `gorm:"column:test_attempt_id;type:text;not null;index:idx_attachments_attempt" json:"testAttemptId"`
	StepID        *string    `gorm:"column:step_id;type:text" json:"stepId,omitempty"`
	Kind          string     `gorm:"column:kind;type:text" json:"kind,omitempty"`
	Name          string     `gorm:"column:name;type:text" json:"name,omitempty"`
	ContentType   string     `gorm:"column:content_type;type:text" json:"contentType,omitempty"`
	SizeBytes     int64      `gorm:"column:size_bytes" json:"sizeBytes,omitempty"`
	StorageKey    string     `gorm:"column:storage_key;type:text" json:"storageKey,omitempty"`
	Checksum      string     `gorm:"column:checksum;type:text" json:"checksum,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_attachments_created_at" json:"createdAt"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deletedAt,omitempty"`
}

func (Attachment) TableName() string {
	return "attachments"
}

// ModelsForMigration lists models in schema creation order.
func ModelsForMigration() []interface{} {
	return []interface{}{
		&TestRun{},
		&RunShard{},
		&Suite{},
		&Test{},
		&TestAttempt{},
		&Attachment{},
	}
}

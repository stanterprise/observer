package repository

import (
	"context"

	m "github.com/stanterprise/observer/internal/models"
)

// RunRepository defines persistence operations for runs.
type RunRepository interface {
	UpsertRun(ctx context.Context, run *m.Run) error
	GetRun(ctx context.Context, id string) (*m.Run, error)
	GetRunByLogicalKey(ctx context.Context, logicalRunKey string) (*m.Run, error)
	ListRuns(ctx context.Context, opts ListRunsOpts) ([]*m.Run, error)
	UpdateRunStatus(ctx context.Context, id, status string) error
}

// ListRunsOpts provides filtering for run listing queries.
type ListRunsOpts struct {
	Status string
	Limit  int
	Offset int
}

// RunShardRepository defines persistence operations for run shards.
type RunShardRepository interface {
	UpsertRunShard(ctx context.Context, shard *m.RunShard) error
	GetRunShard(ctx context.Context, id string) (*m.RunShard, error)
	ListRunShardsByRunID(ctx context.Context, runID string) ([]*m.RunShard, error)
	UpdateRunShardStatus(ctx context.Context, id, status string) error
}

// SuiteRepository defines persistence operations for suites.
type SuiteRepository interface {
	UpsertSuite(ctx context.Context, suite *m.Suite) error
	GetSuite(ctx context.Context, id string) (*m.Suite, error)
	ListSuitesByRunID(ctx context.Context, runID string) ([]*m.Suite, error)
	UpdateSuiteStatus(ctx context.Context, id, status string) error
}

// TestRepository defines persistence operations for tests.
type TestRepository interface {
	UpsertTest(ctx context.Context, test *m.Test) error
	GetTest(ctx context.Context, id string) (*m.Test, error)
	ListTestsBySuiteID(ctx context.Context, suiteID string) ([]*m.Test, error)
	UpdateTestStatus(ctx context.Context, id, status string) error
}

// TestAttemptRepository defines persistence operations for test attempts.
type TestAttemptRepository interface {
	UpsertTestAttempt(ctx context.Context, attempt *m.TestAttempt) error
	GetTestAttempt(ctx context.Context, id string) (*m.TestAttempt, error)
	GetTestAttemptByIndex(ctx context.Context, testID string, attemptIndex int) (*m.TestAttempt, error)
	ListAttemptsByTestID(ctx context.Context, testID string) ([]*m.TestAttempt, error)
	FinalizeAttempt(ctx context.Context, id string, status string, steps []byte, stepsRef *string, stepCount int, durationMs int64, failureReason *string) error
}

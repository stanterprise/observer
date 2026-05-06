package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestDeleteRuns_RemovesRunGraph(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	runID := "run-123"
	parentSuiteID := "run-123:suite:root"
	childSuiteID := "run-123:suite:child"
	testID := "run-123:test:test-1"
	attemptID := testID + ":0"

	seedRelationalDeleteData(t, repo,
		&m.TestRun{ID: runID, Name: "Run 123", CreatedAt: now, UpdatedAt: now},
		&m.RunExecution{ID: "exec-1", RunID: runID, Status: "PASSED", CreatedAt: now, UpdatedAt: now},
		&m.RunShard{ID: runID + ":1", RunID: runID, CreatedAt: now, UpdatedAt: now},
		&m.Suite{ID: parentSuiteID, RunID: runID, ExternalSuiteID: "root", Name: "Root", CreatedAt: now, UpdatedAt: now},
		&m.Suite{ID: childSuiteID, RunID: runID, ExternalSuiteID: "child", ParentSuiteID: &parentSuiteID, Name: "Child", CreatedAt: now, UpdatedAt: now},
		&m.Test{ID: testID, RunID: runID, ExternalTestID: "test-1", SuiteID: &childSuiteID, Name: "T1", Title: "T1", CreatedAt: now, UpdatedAt: now},
		&m.TestAttempt{ID: attemptID, RunID: runID, TestID: testID, AttemptIndex: 0, Status: "PASSED", CreatedAt: now, UpdatedAt: now},
		&m.Attachment{ID: "attachment-1", RunID: runID, TestID: testID, TestAttemptID: attemptID, Name: "trace.zip", CreatedAt: now},
	)

	seedRelationalDeleteData(t, repo,
		&m.TestRun{ID: "run-keep", Name: "Keep", CreatedAt: now, UpdatedAt: now},
		&m.RunExecution{ID: "exec-keep-1", RunID: "run-keep", Status: "PASSED", CreatedAt: now, UpdatedAt: now},
		&m.RunShard{ID: "run-keep:1", RunID: "run-keep", CreatedAt: now, UpdatedAt: now},
	)

	deleted, err := repo.DeleteRuns(ctx, []string{runID})
	if err != nil {
		t.Fatalf("DeleteRuns failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("DeleteRuns deleted = %d, want 1", deleted)
	}

	assertCountForRun(t, repo, &m.TestRun{}, "id", runID, 0)
	assertCountForRun(t, repo, &m.RunExecution{}, "run_id", runID, 0)
	assertCountForRun(t, repo, &m.RunShard{}, "run_id", runID, 0)
	assertCountForRun(t, repo, &m.Suite{}, "run_id", runID, 0)
	assertCountForRun(t, repo, &m.Test{}, "run_id", runID, 0)
	assertCountForRun(t, repo, &m.TestAttempt{}, "run_id", runID, 0)
	assertCountForRun(t, repo, &m.Attachment{}, "run_id", runID, 0)

	assertCountForRun(t, repo, &m.TestRun{}, "id", "run-keep", 1)
	assertCountForRun(t, repo, &m.RunExecution{}, "run_id", "run-keep", 1)
	assertCountForRun(t, repo, &m.RunShard{}, "run_id", "run-keep", 1)
}

func seedRelationalDeleteData(t *testing.T, repo *PostgresRepository, records ...interface{}) {
	t.Helper()
	for _, record := range records {
		if err := repo.db.WithContext(context.Background()).Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func assertCountForRun(t *testing.T, repo *PostgresRepository, model interface{}, column, value string, want int64) {
	t.Helper()
	var count int64
	if err := repo.db.WithContext(context.Background()).Model(model).Where(column+" = ?", value).Count(&count).Error; err != nil {
		t.Fatalf("count %T for %s=%s: %v", model, column, value, err)
	}
	if count != want {
		t.Fatalf("count %T for %s=%s = %d, want %d", model, column, value, count, want)
	}
}

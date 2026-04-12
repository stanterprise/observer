package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stanterprise/observer/internal/database"
	m "github.com/stanterprise/observer/internal/models"
)

// pgTestPool returns a *pgxpool.Pool connected to a test PostgreSQL instance.
// If POSTGRES_TEST_DSN is not set the test is skipped.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping PG integration test")
	}

	ctx := context.Background()
	conn, err := database.ConnectPostgres(dsn, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := conn.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	return conn.Pool
}

func TestPgRepository_RunLifecycle(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewPgRepository(pool, nil)
	ctx := context.Background()

	run := &m.Run{
		ID:            fmt.Sprintf("run-%d", time.Now().UnixNano()),
		LogicalRunKey: fmt.Sprintf("key-%d", time.Now().UnixNano()),
		Source:        "ci",
		Project:       "observer",
		Pipeline:      "main",
		Branch:        "feature/test",
		CommitSHA:     "abc123",
		Status:        m.StatusRunning,
		StartedAt:     time.Now().UTC(),
		Metadata:      json.RawMessage(`{"env":"test"}`),
	}

	// Insert
	if err := repo.UpsertRun(ctx, run); err != nil {
		t.Fatalf("UpsertRun: %v", err)
	}

	// Read back
	got, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got == nil {
		t.Fatal("GetRun returned nil")
	}
	if got.Project != "observer" {
		t.Errorf("Project = %q, want %q", got.Project, "observer")
	}

	// Read by logical key
	got2, err := repo.GetRunByLogicalKey(ctx, run.LogicalRunKey)
	if err != nil {
		t.Fatalf("GetRunByLogicalKey: %v", err)
	}
	if got2 == nil || got2.ID != run.ID {
		t.Errorf("GetRunByLogicalKey mismatch: got %+v", got2)
	}

	// Idempotent upsert (should not error)
	run.Branch = "feature/updated"
	if err := repo.UpsertRun(ctx, run); err != nil {
		t.Fatalf("Upsert (idempotent) failed: %v", err)
	}
	got3, _ := repo.GetRun(ctx, run.ID)
	if got3.Branch != "feature/updated" {
		t.Errorf("Branch after upsert = %q, want %q", got3.Branch, "feature/updated")
	}

	// Update status to terminal
	if err := repo.UpdateRunStatus(ctx, run.ID, m.StatusPassed); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	got4, _ := repo.GetRun(ctx, run.ID)
	if got4.Status != m.StatusPassed {
		t.Errorf("Status = %q, want %q", got4.Status, m.StatusPassed)
	}
	if got4.FinishedAt == nil {
		t.Error("FinishedAt should be set for terminal status")
	}

	// List
	list, err := repo.ListRuns(ctx, ListRunsOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	found := false
	for _, r := range list {
		if r.ID == run.ID {
			found = true
		}
	}
	if !found {
		t.Error("ListRuns did not include the created run")
	}
}

func TestPgRepository_SuiteLifecycle(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewPgRepository(pool, nil)
	ctx := context.Background()

	// Create parent run
	runID := fmt.Sprintf("run-suite-%d", time.Now().UnixNano())
	run := &m.Run{
		ID:            runID,
		LogicalRunKey: fmt.Sprintf("key-suite-%d", time.Now().UnixNano()),
		Status:        m.StatusRunning,
		StartedAt:     time.Now().UTC(),
	}
	if err := repo.UpsertRun(ctx, run); err != nil {
		t.Fatalf("UpsertRun: %v", err)
	}

	suite := &m.Suite{
		ID:              fmt.Sprintf("suite-%d", time.Now().UnixNano()),
		RunID:           runID,
		ExternalSuiteID: "ext-suite-1",
		Name:            "Login Tests",
		Status:          m.StatusRunning,
		StartedAt:       time.Now().UTC(),
		Metadata:        json.RawMessage(`{}`),
	}

	if err := repo.UpsertSuite(ctx, suite); err != nil {
		t.Fatalf("UpsertSuite: %v", err)
	}

	got, _ := repo.GetSuite(ctx, suite.ID)
	if got == nil || got.Name != "Login Tests" {
		t.Errorf("GetSuite mismatch: %+v", got)
	}

	list, _ := repo.ListSuitesByRunID(ctx, runID)
	if len(list) == 0 {
		t.Error("ListSuitesByRunID returned empty")
	}

	if err := repo.UpdateSuiteStatus(ctx, suite.ID, m.StatusPassed); err != nil {
		t.Fatalf("UpdateSuiteStatus: %v", err)
	}
}

func TestPgRepository_TestAndAttemptLifecycle(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewPgRepository(pool, nil)
	ctx := context.Background()

	// Create parent run + suite
	runID := fmt.Sprintf("run-ta-%d", time.Now().UnixNano())
	suiteID := fmt.Sprintf("suite-ta-%d", time.Now().UnixNano())
	testID := fmt.Sprintf("test-ta-%d", time.Now().UnixNano())
	attemptID := fmt.Sprintf("attempt-ta-%d", time.Now().UnixNano())

	run := &m.Run{ID: runID, LogicalRunKey: fmt.Sprintf("key-ta-%d", time.Now().UnixNano()), Status: m.StatusRunning, StartedAt: time.Now().UTC()}
	suite := &m.Suite{ID: suiteID, RunID: runID, Name: "Suite", Status: m.StatusRunning, StartedAt: time.Now().UTC()}
	test := &m.Test{ID: testID, SuiteID: suiteID, ExternalTestID: "ext-test-1", Name: "should login", Status: m.StatusRunning, AttemptCount: 1, StartedAt: time.Now().UTC()}
	attempt := &m.TestAttempt{ID: attemptID, TestID: testID, AttemptIndex: 0, Status: m.StatusRunning, StartedAt: time.Now().UTC()}

	for _, op := range []func() error{
		func() error { return repo.UpsertRun(ctx, run) },
		func() error { return repo.UpsertSuite(ctx, suite) },
		func() error { return repo.UpsertTest(ctx, test) },
		func() error { return repo.UpsertTestAttempt(ctx, attempt) },
	} {
		if err := op(); err != nil {
			t.Fatal(err)
		}
	}

	// Get attempt by index
	gotA, err := repo.GetTestAttemptByIndex(ctx, testID, 0)
	if err != nil {
		t.Fatalf("GetTestAttemptByIndex: %v", err)
	}
	if gotA == nil || gotA.Status != m.StatusRunning {
		t.Errorf("unexpected attempt: %+v", gotA)
	}

	// Finalize attempt
	stepsJSON := json.RawMessage(`[{"id":"s1","title":"click login"}]`)
	if err := repo.FinalizeAttempt(ctx, attemptID, m.StatusPassed, stepsJSON, nil, 1, 1234, nil); err != nil {
		t.Fatalf("FinalizeAttempt: %v", err)
	}

	gotA2, _ := repo.GetTestAttempt(ctx, attemptID)
	if gotA2.Status != m.StatusPassed {
		t.Errorf("Status = %q, want %q", gotA2.Status, m.StatusPassed)
	}
	if gotA2.StepCount != 1 {
		t.Errorf("StepCount = %d, want 1", gotA2.StepCount)
	}
	if gotA2.DurationMs != 1234 {
		t.Errorf("DurationMs = %d, want 1234", gotA2.DurationMs)
	}

	// List attempts
	attempts, _ := repo.ListAttemptsByTestID(ctx, testID)
	if len(attempts) != 1 {
		t.Errorf("ListAttemptsByTestID returned %d, want 1", len(attempts))
	}

	// Idempotent re-upsert
	if err := repo.UpsertTestAttempt(ctx, attempt); err != nil {
		t.Fatalf("Idempotent UpsertTestAttempt: %v", err)
	}
}

func TestPgRepository_GetNonExistent(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewPgRepository(pool, nil)
	ctx := context.Background()

	got, err := repo.GetRun(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("GetRun non-existent: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent run, got %+v", got)
	}

	gotS, err := repo.GetSuite(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("GetSuite non-existent: %v", err)
	}
	if gotS != nil {
		t.Errorf("expected nil for non-existent suite, got %+v", gotS)
	}

	gotT, err := repo.GetTest(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("GetTest non-existent: %v", err)
	}
	if gotT != nil {
		t.Errorf("expected nil for non-existent test, got %+v", gotT)
	}

	gotA, err := repo.GetTestAttempt(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("GetTestAttempt non-existent: %v", err)
	}
	if gotA != nil {
		t.Errorf("expected nil for non-existent attempt, got %+v", gotA)
	}
}

func TestPgRepository_RunShardLifecycle(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewPgRepository(pool, nil)
	ctx := context.Background()

	runID := fmt.Sprintf("run-shard-%d", time.Now().UnixNano())
	run := &m.Run{ID: runID, LogicalRunKey: fmt.Sprintf("key-shard-%d", time.Now().UnixNano()), Status: m.StatusRunning, StartedAt: time.Now().UTC()}
	if err := repo.UpsertRun(ctx, run); err != nil {
		t.Fatalf("UpsertRun: %v", err)
	}

	expected := 3
	shard := &m.RunShard{
		ID:                 fmt.Sprintf("shard-%d", time.Now().UnixNano()),
		RunID:              runID,
		ShardKey:           "shard-0",
		ShardIndex:         0,
		ShardCountExpected: &expected,
		Status:             m.StatusRunning,
		StartedAt:          time.Now().UTC(),
	}
	if err := repo.UpsertRunShard(ctx, shard); err != nil {
		t.Fatalf("UpsertRunShard: %v", err)
	}

	got, _ := repo.GetRunShard(ctx, shard.ID)
	if got == nil || got.ShardKey != "shard-0" {
		t.Errorf("unexpected shard: %+v", got)
	}

	list, _ := repo.ListRunShardsByRunID(ctx, runID)
	if len(list) != 1 {
		t.Errorf("ListRunShardsByRunID returned %d, want 1", len(list))
	}

	if err := repo.UpdateRunShardStatus(ctx, shard.ID, m.StatusPassed); err != nil {
		t.Fatalf("UpdateRunShardStatus: %v", err)
	}
}

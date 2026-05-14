package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestBuildAggregatedRunFromExecutions(t *testing.T) {
	start := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	finish := start.Add(5 * time.Minute)
	finishLater := finish.Add(2 * time.Minute)

	run, ok := buildAggregatedRunFromExecutions("run-123", []m.RunExecution{
		{RunID: "run-123", ID: "exec-a", Status: "PASSED", StartTime: &start, EndTime: &finish},
		{RunID: "run-123", ID: "exec-b", Status: "FAILED", StartTime: &finish, EndTime: &finishLater},
	}, time.Now())
	if !ok {
		t.Fatal("expected aggregated run")
	}
	if run.Status != "FAILED" {
		t.Fatalf("Status = %q, want FAILED", run.Status)
	}
	if run.StartTime == nil || !run.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", run.StartTime, start)
	}
	if run.EndTime == nil || !run.EndTime.Equal(finishLater) {
		t.Fatalf("EndTime = %v, want %v", run.EndTime, finishLater)
	}
}

func TestRefreshLogicalRunAggregate_UsesShardCompletionForLogicalRunStatus(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	finish := now.Add(3 * time.Minute)

	if err := repo.db.WithContext(context.Background()).Create(&m.TestRun{ID: "run-123", Name: "Logical Aggregate", Status: "RUNNING", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := repo.db.WithContext(context.Background()).Create([]m.RunExecution{
		{ID: "exec-a", RunID: "run-123", Status: "RUNNING", CreatedAt: now, UpdatedAt: now},
		{ID: "exec-b", RunID: "run-123", Status: "RUNNING", CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatalf("seed executions: %v", err)
	}

	if err := refreshLogicalRunAggregate(repo.db.WithContext(context.Background()), "run-123", finish); err != nil {
		t.Fatalf("refreshLogicalRunAggregate: %v", err)
	}

	var stored m.TestRun
	if err := repo.db.WithContext(context.Background()).First(&stored, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if stored.Status != "FAILED" {
		t.Fatalf("stored.Status = %q, want FAILED", stored.Status)
	}
	if stored.EndTime == nil || !stored.EndTime.Equal(finish) {
		t.Fatalf("stored.EndTime = %v, want %v", stored.EndTime, finish)
	}
}

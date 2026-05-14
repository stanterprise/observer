package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestRefreshLogicalRunAggregate_KeepsLogicalRunRunningWhileExecutionOpen(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	finish := now.Add(3 * time.Minute)

	if err := repo.db.WithContext(context.Background()).Create(&m.TestRun{ID: "run-123", Name: "Logical Aggregate", Status: "RUNNING", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := repo.db.WithContext(context.Background()).Create([]m.RunExecution{
		{ID: "exec-a", RunID: "run-123", Status: "FAILED", StartTime: &now, EndTime: &finish, CreatedAt: now, UpdatedAt: now},
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
	if stored.Status != "RUNNING" {
		t.Fatalf("stored.Status = %q, want RUNNING", stored.Status)
	}
}

func TestHandleRunEnd_FinalizesExecutionsAndAggregatesLogicalRun(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	startA := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	startB := startA.Add(30 * time.Second)

	if err := repo.HandleRunStart(ctx, &events.ReportRunStartEventRequest{RunId: "run-123", Name: "Logical Aggregate", ExecutionId: "exec-a"}); err != nil {
		t.Fatalf("HandleRunStart(exec-a) failed: %v", err)
	}
	if err := repo.HandleRunStart(ctx, &events.ReportRunStartEventRequest{RunId: "run-123", Name: "Logical Aggregate", ExecutionId: "exec-b"}); err != nil {
		t.Fatalf("HandleRunStart(exec-b) failed: %v", err)
	}

	if err := repo.HandleRunEnd(ctx, &events.TestRunEndEventRequest{
		RunId:       "run-123",
		ExecutionId: "exec-a",
		FinalStatus: common.TestStatus_FAILED,
		StartTime:   timestamppb.New(startA),
		Duration:    durationpb.New(2 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleRunEnd(exec-a) failed: %v", err)
	}

	var run m.TestRun
	if err := repo.db.WithContext(ctx).First(&run, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load run after first end: %v", err)
	}
	if run.Status != "RUNNING" {
		t.Fatalf("run.Status after first end = %q, want RUNNING", run.Status)
	}

	var executionA m.RunExecution
	if err := repo.db.WithContext(ctx).First(&executionA, "run_id = ? AND id = ?", "run-123", "exec-a").Error; err != nil {
		t.Fatalf("load exec-a after first end: %v", err)
	}
	if executionA.Status != "FAILED" {
		t.Fatalf("executionA.Status = %q, want FAILED", executionA.Status)
	}
	if executionA.EndTime == nil {
		t.Fatal("expected exec-a to be finalized with end time")
	}

	if err := repo.HandleRunEnd(ctx, &events.TestRunEndEventRequest{
		RunId:       "run-123",
		ExecutionId: "exec-b",
		FinalStatus: common.TestStatus_PASSED,
		StartTime:   timestamppb.New(startB),
		Duration:    durationpb.New(3 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleRunEnd(exec-b) failed: %v", err)
	}

	if err := repo.db.WithContext(ctx).First(&run, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load run after second end: %v", err)
	}
	if run.Status != "FAILED" {
		t.Fatalf("run.Status after second end = %q, want FAILED", run.Status)
	}

	var stats m.RunStat
	if err := repo.db.WithContext(ctx).First(&stats, "run_id = ?", "run-123").Error; err != nil {
		t.Fatalf("load run stats after end: %v", err)
	}
	if stats.Name != "Logical Aggregate" {
		t.Fatalf("stats.Name = %q, want Logical Aggregate", stats.Name)
	}
}

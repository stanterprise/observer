package postgres

import (
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestAllRunShardsFinished(t *testing.T) {
	now := time.Now()
	shardOne := int32(1)
	shardTwo := int32(2)
	shards := []m.RunShard{
		{ShardIndex: &shardOne, EndTime: &now},
		{ShardIndex: &shardTwo, EndTime: &now},
	}

	if !allRunShardsFinished(shards, 2) {
		t.Fatal("expected all shards to be finished")
	}
	if allRunShardsFinished(shards, 3) {
		t.Fatal("expected incomplete shard set when expected count is higher")
	}
}

func TestAggregateRunShardStatuses(t *testing.T) {
	statuses := []struct {
		name   string
		shards []m.RunShard
		want   string
	}{
		{
			name:   "failed wins",
			shards: []m.RunShard{{Status: "PASSED"}, {Status: "FAILED"}},
			want:   "FAILED",
		},
		{
			name:   "timedout beats passed",
			shards: []m.RunShard{{Status: "PASSED"}, {Status: "TIMEDOUT"}},
			want:   "TIMEDOUT",
		},
		{
			name:   "all passed required for passed",
			shards: []m.RunShard{{Status: "PASSED"}, {Status: "PASSED"}},
			want:   "PASSED",
		},
		{
			name:   "interrupted wins when no failures",
			shards: []m.RunShard{{Status: "PASSED"}, {Status: "INTERRUPTED"}},
			want:   "INTERRUPTED",
		},
		{
			name:   "unknown fallback",
			shards: []m.RunShard{{Status: ""}},
			want:   "UNKNOWN",
		},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			if got := aggregateRunShardStatuses(tt.shards); got != tt.want {
				t.Fatalf("aggregateRunShardStatuses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildAggregatedRunFromShards(t *testing.T) {
	start := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	finish := start.Add(5 * time.Minute)
	finishLater := finish.Add(2 * time.Minute)
	shardOne := int32(1)
	shardTwo := int32(2)

	run, ok := buildAggregatedRunFromShards("run-123", []m.RunShard{
		{ShardIndex: &shardOne, Status: "PASSED", StartTime: &start, EndTime: &finish},
		{ShardIndex: &shardTwo, Status: "INTERRUPTED", StartTime: &finish, EndTime: &finishLater},
	}, time.Now())
	if !ok {
		t.Fatal("expected aggregated run")
	}
	if run.ID != "run-123" {
		t.Fatalf("ID = %q, want run-123", run.ID)
	}
	if run.Status != "INTERRUPTED" {
		t.Fatalf("Status = %q, want INTERRUPTED", run.Status)
	}
	if run.StartTime == nil || !run.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", run.StartTime, start)
	}
	if run.EndTime == nil || !run.EndTime.Equal(finishLater) {
		t.Fatalf("EndTime = %v, want %v", run.EndTime, finishLater)
	}
	if run.Duration == nil || *run.Duration != finishLater.Sub(start).Nanoseconds() {
		t.Fatalf("Duration = %v, want %d", run.Duration, finishLater.Sub(start).Nanoseconds())
	}
}

func TestBuildAggregatedRunFromExecutions(t *testing.T) {
	start := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	finish := start.Add(5 * time.Minute)
	finishLater := finish.Add(2 * time.Minute)

	run, ok := buildAggregatedRunFromExecutions("run-123", []m.RunExecution{
		{RunID: "run-123", ExecutionID: "exec-a", Status: "PASSED", TotalTests: 3, StartTime: &start, EndTime: &finish},
		{RunID: "run-123", ExecutionID: "exec-b", Status: "FAILED", TotalTests: 5, StartTime: &finish, EndTime: &finishLater},
	}, time.Now())
	if !ok {
		t.Fatal("expected aggregated run")
	}
	if run.TotalTests != 8 {
		t.Fatalf("TotalTests = %d, want 8", run.TotalTests)
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

package models

import (
	"testing"
	"time"

	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRunStartEventToRunShard(t *testing.T) {
	req := &events.ReportRunStartEventRequest{
		RunId: "run-123",
		Metadata: map[string]string{
			"shard.current": "2",
			"shard.total":   "5",
		},
	}

	shard := RunStartEventToRunShard(req)
	if shard == nil {
		t.Fatal("expected run shard")
	}
	if shard.RunID != "run-123" {
		t.Fatalf("RunID = %q, want run-123", shard.RunID)
	}
	if shard.ShardIndex == nil || *shard.ShardIndex != 2 {
		t.Fatalf("ShardIndex = %v, want 2", shard.ShardIndex)
	}
	if shard.ID != "run-123:2" {
		t.Fatalf("ID = %q, want run-123:2", shard.ID)
	}
	if shard.ShardCountExpected == nil || *shard.ShardCountExpected != 5 {
		t.Fatalf("ShardCountExpected = %v, want 5", shard.ShardCountExpected)
	}
	if shard.Status != "RUNNING" {
		t.Fatalf("Status = %q, want RUNNING", shard.Status)
	}
	if shard.StartTime == nil {
		t.Fatal("expected StartTime to be set")
	}
}

func TestRunEndEventToRunShard(t *testing.T) {
	start := time.Date(2026, 4, 18, 9, 30, 0, 0, time.UTC)
	req := &events.TestRunEndEventRequest{
		RunId:       "run-123",
		FinalStatus: common.TestStatus_PASSED,
		StartTime:   timestamppb.New(start),
		Duration:    durationpb.New(5 * time.Second),
		Metadata: map[string]string{
			"shard_index": "3",
			"shard_count": "7",
		},
	}

	shard := RunEndEventToRunShard(req)
	if shard == nil {
		t.Fatal("expected run shard")
	}
	if shard.ShardIndex == nil || *shard.ShardIndex != 3 {
		t.Fatalf("ShardIndex = %v, want 3", shard.ShardIndex)
	}
	if shard.ID != "run-123:3" {
		t.Fatalf("ID = %q, want run-123:3", shard.ID)
	}
	if shard.ShardCountExpected == nil || *shard.ShardCountExpected != 7 {
		t.Fatalf("ShardCountExpected = %v, want 7", shard.ShardCountExpected)
	}
	if shard.Status != common.TestStatus_PASSED.String() {
		t.Fatalf("Status = %q, want %q", shard.Status, common.TestStatus_PASSED.String())
	}
	if shard.StartTime == nil || !shard.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", shard.StartTime, start)
	}
	if shard.EndTime == nil {
		t.Fatal("expected EndTime to be set")
	}
}

func TestRunStartEventToRunShardWithoutMetadata(t *testing.T) {
	if shard := RunStartEventToRunShard(&events.ReportRunStartEventRequest{RunId: "run-123"}); shard != nil {
		t.Fatalf("expected nil shard, got %+v", shard)
	}
}

func TestRunStartEventToRunShardRequiresShardCount(t *testing.T) {
	req := &events.ReportRunStartEventRequest{
		RunId: "run-123",
		Metadata: map[string]string{
			"shard.current": "2",
		},
	}

	if shard := RunStartEventToRunShard(req); shard != nil {
		t.Fatalf("expected nil shard when shard count is missing, got %+v", shard)
	}
}

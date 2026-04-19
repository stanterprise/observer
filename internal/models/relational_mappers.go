package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// RunStartEventToTestRun maps a ReportRunStartEventRequest to a TestRun row.
// The returned value is ready for an idempotent upsert into the runs table.
func RunStartEventToTestRun(req *events.ReportRunStartEventRequest) (*TestRun, []*Suite) {
	if req == nil {
		return nil, nil
	}

	now := time.Now()

	md := make(map[string]interface{})
	for k, v := range req.Metadata {
		md[k] = v
	}
	suites := make([]*Suite, 0, len(req.TestSuites))
	for _, protoSuite := range req.TestSuites {
		if protoSuite == nil {
			continue
		}
		suite := &Suite{
			ID:            protoSuite.Id,
			RunID:         protoSuite.RunId,
			ParentSuiteID: &protoSuite.ParentSuiteId,
			Name:          protoSuite.Name,
			Description:   protoSuite.Description,
			Metadata:      md,
			Location:      protoSuite.Location,
			Type:          protoSuite.Type.String(),
			InitiatedBy:   protoSuite.InitiatedBy,
			ProjectName:   protoSuite.Project,
		}
		suites = append(suites, suite)
	}

	return &TestRun{
		ID:         req.RunId,
		Name:       req.Name,
		Status:     "RUNNING",
		TotalTests: req.TotalTests,
		Metadata:   md,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, suites
}

// RunStartEventToRunShard maps run-level shard metadata to a RunShard row.
// Returns nil when the run is not sharded or shard metadata is incomplete.
func RunStartEventToRunShard(req *events.ReportRunStartEventRequest) *RunShard {
	if req == nil {
		return nil
	}

	shardIndex, shardCount := parseRunShardMetadata(req.Metadata)
	if shardIndex == nil || shardCount == nil {
		return nil
	}

	now := time.Now()

	return &RunShard{
		ID:                 buildRunShardID(req.RunId, shardIndex),
		RunID:              req.RunId,
		ShardIndex:         shardIndex,
		ShardCountExpected: shardCount,
		Status:             "RUNNING",
		StartTime:          &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// RunEndEventToTestRun maps a TestRunEndEventRequest to TestRun.
// The returned fields are intended for a partial update to finalize the run's terminal state.
func RunEndEventToTestRun(req *events.TestRunEndEventRequest) TestRun {
	if req == nil {
		return TestRun{}
	}

	now := time.Now()
	fields := TestRun{
		ID:        req.RunId,
		Status:    req.FinalStatus.String(),
		UpdatedAt: now,
	}

	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		fields.StartTime = &t
	}

	if req.Duration != nil {
		d := req.Duration.AsDuration().Nanoseconds()
		fields.Duration = &d
	}

	return fields
}

// RunEndEventToRunShard maps run-level shard metadata in a terminal event to a RunShard row.
// Returns nil when the run is not sharded or shard metadata is incomplete.
func RunEndEventToRunShard(req *events.TestRunEndEventRequest) *RunShard {
	if req == nil {
		return nil
	}

	shardIndex, shardCount := parseRunShardMetadata(req.Metadata)
	if shardIndex == nil || shardCount == nil {
		return nil
	}

	now := time.Now()
	shard := &RunShard{
		ID:                 buildRunShardID(req.RunId, shardIndex),
		RunID:              req.RunId,
		ShardIndex:         shardIndex,
		ShardCountExpected: shardCount,
		Status:             req.FinalStatus.String(),
		EndTime:            &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		shard.StartTime = &t
	}

	return shard
}

func parseRunShardMetadata(metadata map[string]string) (*int32, *int32) {
	shardIndex := firstInt32Metadata(metadata,
		"shard.current",
		"shard_current",
		"shard_index",
		"shardIndex",
	)
	shardCount := firstInt32Metadata(metadata,
		"shard.total",
		"shard_total",
		"shard_count",
		"shardCount",
		"shard_count_expected",
		"shardCountExpected",
	)

	return shardIndex, shardCount
}

func firstInt32Metadata(metadata map[string]string, keys ...string) *int32 {
	for _, key := range keys {
		value := strings.TrimSpace(metadata[key])
		if value == "" {
			continue
		}

		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			continue
		}

		converted := int32(parsed)
		return &converted
	}

	return nil
}

func buildRunShardID(runID string, shardIndex *int32) string {
	return runID + ":" + fmt.Sprintf("%d", *shardIndex)
}

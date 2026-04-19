package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// RunStartEventToTestRun maps a ReportRunStartEventRequest to a TestRun row.
// The returned value is ready for an idempotent upsert into the runs table.
func RunStartEventToTestRun(req *events.ReportRunStartEventRequest) (*TestRun, []*Suite) {
	if req == nil {
		return nil, nil
	}

	now := time.Now()

	md := stringMapToInterfaceMap(req.Metadata)
	suites := flattenSuiteRuns(req.TestSuites)

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

// RunStartEventToTests maps embedded test cases in a run-start payload to relational test rows.
func RunStartEventToTests(req *events.ReportRunStartEventRequest) []*Test {
	if req == nil {
		return nil
	}

	return flattenTestRuns(req.TestSuites)
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

// TestCaseRunToRelationalTest maps a TestCaseRun payload to the relational tests row.
func TestCaseRunToRelationalTest(protoTest *entities.TestCaseRun) *Test {
	if protoTest == nil {
		return nil
	}

	now := time.Now()
	metadata := stringMapToInterfaceMap(protoTest.Metadata)

	var suiteID *string
	if protoTest.TestSuiteId != "" {
		suiteID = &protoTest.TestSuiteId
	}

	var startTime *time.Time
	if protoTest.StartTime != nil {
		t := protoTest.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if protoTest.EndTime != nil {
		t := protoTest.EndTime.AsTime()
		endTime = &t
	}

	var duration *int64
	if protoTest.Duration != nil {
		d := protoTest.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	return &Test{
		ID:          protoTest.Id,
		RunID:       protoTest.RunId,
		SuiteID:     suiteID,
		Name:        protoTest.Name,
		Title:       protoTest.Name,
		Description: protoTest.Description,
		Status:      protoTest.Status.String(),
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    duration,
		Metadata:    metadata,
		Tags:        protoTest.Tags,
		Location:    protoTest.Location,
		RetryCount:  &protoTest.RetryCount,
		RetryIndex:  &protoTest.RetryIndex,
		Timeout:     &protoTest.Timeout,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// TestCaseRunToRelationalAttempt maps a TestCaseRun payload to the relational test_attempts row.
func TestCaseRunToRelationalAttempt(protoTest *entities.TestCaseRun, attachments []map[string]interface{}) *TestAttempt {
	if protoTest == nil {
		return nil
	}

	now := time.Now()
	attemptIndex := protoTest.RetryIndex

	var startTime *time.Time
	if protoTest.StartTime != nil {
		t := protoTest.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if protoTest.EndTime != nil {
		t := protoTest.EndTime.AsTime()
		endTime = &t
	}

	var duration *int64
	if protoTest.Duration != nil {
		d := protoTest.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	return &TestAttempt{
		ID:           buildTestAttemptID(protoTest.Id, attemptIndex),
		RunID:        protoTest.RunId,
		TestID:       protoTest.Id,
		AttemptIndex: attemptIndex,
		Status:       protoTest.Status.String(),
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     duration,
		Attachments:  attachments,
		ErrorMessage: protoTest.ErrorMessage,
		StackTrace:   protoTest.StackTrace,
		ErrorList:    protoTest.Errors,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
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

func flattenSuiteRuns(protoSuites []*entities.TestSuiteRun) []*Suite {
	suites := make([]*Suite, 0)
	for _, protoSuite := range protoSuites {
		suites = append(suites, flattenSingleSuite(protoSuite)...)
	}
	return suites
}

func flattenSingleSuite(protoSuite *entities.TestSuiteRun) []*Suite {
	if protoSuite == nil {
		return nil
	}

	now := time.Now()
	metadata := stringMapToInterfaceMap(protoSuite.Metadata)
	var parentSuiteID *string
	if protoSuite.ParentSuiteId != "" {
		parentSuiteID = &protoSuite.ParentSuiteId
	}

	var startTime *time.Time
	if protoSuite.StartTime != nil {
		t := protoSuite.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if protoSuite.EndTime != nil {
		t := protoSuite.EndTime.AsTime()
		endTime = &t
	}

	var duration *int64
	if protoSuite.Duration != nil {
		d := protoSuite.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	suite := &Suite{
		ID:            protoSuite.Id,
		RunID:         protoSuite.RunId,
		ParentSuiteID: parentSuiteID,
		Name:          protoSuite.Name,
		Description:   protoSuite.Description,
		Status:        protoSuite.Status.String(),
		Metadata:      metadata,
		Duration:      duration,
		Location:      protoSuite.Location,
		Type:          protoSuite.Type.String(),
		InitiatedBy:   protoSuite.InitiatedBy,
		ProjectName:   protoSuite.Project,
		Author:        protoSuite.Author,
		Owner:         protoSuite.Owner,
		TestCaseIDs:   protoSuite.TestCaseIds,
		SubSuiteIDs:   protoSuite.SubSuiteIds,
		StartTime:     startTime,
		EndTime:       endTime,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	suites := []*Suite{suite}
	for _, childSuite := range protoSuite.SubSuites {
		suites = append(suites, flattenSingleSuite(childSuite)...)
	}

	return suites
}

func flattenTestRuns(protoSuites []*entities.TestSuiteRun) []*Test {
	tests := make([]*Test, 0)
	for _, protoSuite := range protoSuites {
		tests = append(tests, flattenTestsForSuite(protoSuite)...)
	}
	return tests
}

func flattenTestsForSuite(protoSuite *entities.TestSuiteRun) []*Test {
	if protoSuite == nil {
		return nil
	}

	now := time.Now()
	tests := make([]*Test, 0, len(protoSuite.TestCases))
	for _, protoTest := range protoSuite.TestCases {
		if protoTest == nil {
			continue
		}

		metadata := stringMapToInterfaceMap(protoTest.Metadata)
		var suiteID *string
		if protoTest.TestSuiteId != "" {
			suiteID = &protoTest.TestSuiteId
		} else if protoSuite.Id != "" {
			suiteID = &protoSuite.Id
		}

		var startTime *time.Time
		if protoTest.StartTime != nil {
			t := protoTest.StartTime.AsTime()
			startTime = &t
		}

		var endTime *time.Time
		if protoTest.EndTime != nil {
			t := protoTest.EndTime.AsTime()
			endTime = &t
		}

		var duration *int64
		if protoTest.Duration != nil {
			d := protoTest.Duration.AsDuration().Nanoseconds()
			duration = &d
		}

		retryCount := &protoTest.RetryCount
		retryIndex := &protoTest.RetryIndex
		timeout := &protoTest.Timeout

		test := &Test{
			ID:          protoTest.Id,
			RunID:       protoTest.RunId,
			SuiteID:     suiteID,
			Name:        protoTest.Name,
			Title:       protoTest.Name,
			Description: protoTest.Description,
			Status:      protoTest.Status.String(),
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    duration,
			Metadata:    metadata,
			Tags:        protoTest.Tags,
			Location:    protoTest.Location,
			RetryCount:  retryCount,
			RetryIndex:  retryIndex,
			Timeout:     timeout,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		tests = append(tests, test)
	}

	for _, childSuite := range protoSuite.SubSuites {
		tests = append(tests, flattenTestsForSuite(childSuite)...)
	}

	return tests
}

func stringMapToInterfaceMap(metadata map[string]string) map[string]interface{} {
	converted := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		converted[k] = v
	}
	return converted
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

func buildTestAttemptID(testID string, attemptIndex int32) string {
	return fmt.Sprintf("%s:%d", testID, attemptIndex)
}

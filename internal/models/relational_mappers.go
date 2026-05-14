package models

import (
	"fmt"
	"strconv"
	"time"

	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

func RunStartEventToAllEntities(req *events.ReportRunStartEventRequest) (*TestRun, *RunExecution, []*Suite, []*Test) {
	if req == nil {
		return nil, nil, nil, nil
	}

	testRun := runStartEventToTestRun(req)
	relationalSuites := runStartEventToSuites(req)
	runExecution := runStartEventToRunExecution(req)
	relationalTests := runStartEventToTests(req)

	return testRun, runExecution, relationalSuites, relationalTests
}

func getShardInfoFromMetadata(metadata map[string]string) (total, current int32, ok bool) {
	if metadata == nil {
		return 0, 0, false
	}
	totalVal, hasTotal := metadata["shard.total"]
	currentVal, hasCurrent := metadata["shard.current"]
	if !hasTotal || !hasCurrent {
		return 0, 0, false
	}
	totalInt, errTotal := strconv.Atoi(totalVal)
	currentInt, errCurrent := strconv.Atoi(currentVal)
	if errTotal != nil || errCurrent != nil {
		return 0, 0, false
	}
	return int32(totalInt), int32(currentInt), true
}

// runStartEventToTestRun maps a ReportRunStartEventRequest to a TestRun row.
// The returned value is ready for an idempotent upsert into the runs table.
func runStartEventToTestRun(req *events.ReportRunStartEventRequest) *TestRun {
	if req == nil {
		return nil
	}

	now := time.Now()

	md := stringMapToInterfaceMap(req.Metadata)

	return &TestRun{
		ID:        req.RunId,
		Name:      req.Name,
		StartTime: &now,
		Status:    "RUNNING",
		Metadata:  md,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func runStartEventToSuites(req *events.ReportRunStartEventRequest) []*Suite {
	suites := flattenSuiteRuns(req.TestSuites)
	return suites
}

// runStartEventToRunExecution maps a run-start event to an execution-scoped row.
func runStartEventToRunExecution(req *events.ReportRunStartEventRequest) *RunExecution {
	if req == nil {
		return nil
	}

	now := time.Now()
	total, current, ok := getShardInfoFromMetadata(req.Metadata)
	var shardIndex *int32
	var shardCountExpected *int32
	if ok {
		shardIndex = &current
		shardCountExpected = &total
	}

	return &RunExecution{
		ID:                 req.ExecutionId,
		RunID:              req.RunId,
		IsShard:            ok,
		ShardIndex:         shardIndex,
		ShardCountExpected: shardCountExpected,
		StartTime:          &now,
		Name:               req.Name,
		Status:             "RUNNING",
		Metadata:           stringMapToInterfaceMap(req.Metadata),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// runStartEventToTests maps embedded test cases in a run-start payload to relational test rows.
func runStartEventToTests(req *events.ReportRunStartEventRequest) []*Test {
	if req == nil {
		return nil
	}

	return flattenTestRuns(req.TestSuites)
}

// runEndEventToTestRun maps a TestRunEndEventRequest to TestRun.
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

// RunEndEventToRunExecution maps a run-end event to an execution-scoped update.
func RunEndEventToRunExecution(req *events.TestRunEndEventRequest) *RunExecution {
	if req == nil {
		return nil
	}

	now := time.Now()

	execution := &RunExecution{
		ID:        req.ExecutionId,
		RunID:     req.RunId,
		Status:    req.FinalStatus.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		execution.StartTime = &t
	}
	if req.Duration != nil {
		d := req.Duration.AsDuration().Nanoseconds()
		execution.Duration = &d
	}
	end := now
	execution.EndTime = &end

	return execution
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
		internalSuiteID := protoTest.TestSuiteId
		suiteID = &internalSuiteID
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
		ID:             protoTest.Id,
		RunID:          protoTest.RunId,
		ExternalTestID: protoTest.Id,
		SuiteID:        suiteID,
		Name:           protoTest.Name,
		Title:          protoTest.Name,
		Description:    protoTest.Description,
		Status:         protoTest.Status.String(),
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       duration,
		Metadata:       metadata,
		Tags:           protoTest.Tags,
		Location:       protoTest.Location,
		RetryCount:     &protoTest.RetryCount,
		RetryIndex:     &protoTest.RetryIndex,
		Timeout:        &protoTest.Timeout,
		CreatedAt:      now,
		UpdatedAt:      now,
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
		ID:           BuildTestAttemptID(protoTest.RunId, protoTest.Id, protoTest.ExecutionId, attemptIndex),
		RunID:        protoTest.RunId,
		ExecutionID:  protoTest.ExecutionId,
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

func flattenSuiteRuns(protoSuites []*entities.TestSuiteRun) []*Suite {
	suites := make([]*Suite, 0)
	for _, protoSuite := range protoSuites {
		suites = append(suites, flattenSingleSuite(protoSuite)...)
	}
	return suites
}

func SuiteRunToRelationalSuite(protoSuite *entities.TestSuiteRun) *Suite {
	suites := flattenSingleSuite(protoSuite)
	if len(suites) == 0 {
		return nil
	}
	return suites[0]
}

func flattenSingleSuite(protoSuite *entities.TestSuiteRun) []*Suite {
	if protoSuite == nil {
		return nil
	}

	now := time.Now()
	metadata := stringMapToInterfaceMap(protoSuite.Metadata)
	var parentSuiteID *string
	if protoSuite.ParentSuiteId != "" {
		internalParentSuiteID := protoSuite.ParentSuiteId
		parentSuiteID = &internalParentSuiteID
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
		ID:              protoSuite.Id,
		RunID:           protoSuite.RunId,
		ExternalSuiteID: protoSuite.Id,
		ParentSuiteID:   parentSuiteID,
		Name:            protoSuite.Name,
		Description:     protoSuite.Description,
		Status:          protoSuite.Status.String(),
		Metadata:        metadata,
		Duration:        duration,
		Location:        protoSuite.Location,
		Type:            protoSuite.Type.String(),
		InitiatedBy:     protoSuite.InitiatedBy,
		ProjectName:     protoSuite.Project,
		Author:          protoSuite.Author,
		Owner:           protoSuite.Owner,
		TestCaseIDs:     protoSuite.TestCaseIds,
		SubSuiteIDs:     protoSuite.SubSuiteIds,
		StartTime:       startTime,
		EndTime:         endTime,
		CreatedAt:       now,
		UpdatedAt:       now,
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
			internalSuiteID := protoTest.TestSuiteId
			suiteID = &internalSuiteID
		} else if protoSuite.Id != "" {
			internalSuiteID := protoSuite.Id
			suiteID = &internalSuiteID
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
			ID:             protoTest.Id,
			RunID:          protoTest.RunId,
			ExternalTestID: protoTest.Id,
			SuiteID:        suiteID,
			Name:           protoTest.Name,
			Title:          protoTest.Name,
			Description:    protoTest.Description,
			Status:         protoTest.Status.String(),
			StartTime:      startTime,
			EndTime:        endTime,
			Duration:       duration,
			Metadata:       metadata,
			Tags:           protoTest.Tags,
			Location:       protoTest.Location,
			RetryCount:     retryCount,
			RetryIndex:     retryIndex,
			Timeout:        timeout,
			CreatedAt:      now,
			UpdatedAt:      now,
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

func BuildTestAttemptID(runID, testID, executionID string, attemptIndex int32) string {
	return fmt.Sprintf("%s:%s:%s:%d", runID, testID, executionID, attemptIndex)
}

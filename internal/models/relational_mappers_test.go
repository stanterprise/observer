package models

import (
	"testing"
	"time"

	"github.com/stanterprise/proto-go/testsystem/v1/common"
	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
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

func TestRunStartEventToRunExecution(t *testing.T) {
	req := &events.ReportRunStartEventRequest{
		RunId:       "run-123",
		ExecutionId: "exec-123",
		Name:        "Composite Run",
		TotalTests:  7,
		Metadata: map[string]string{
			"worker": "a",
		},
	}

	execution := RunStartEventToRunExecution(req)
	if execution == nil {
		t.Fatal("expected run execution")
	}
	if execution.ID != "run-123:execution:exec-123" {
		t.Fatalf("ID = %q, want run-123:execution:exec-123", execution.ID)
	}
	if execution.ExecutionID != "exec-123" {
		t.Fatalf("ExecutionID = %q, want exec-123", execution.ExecutionID)
	}
	if execution.TotalTests != 7 {
		t.Fatalf("TotalTests = %d, want 7", execution.TotalTests)
	}
	if execution.Status != "RUNNING" {
		t.Fatalf("Status = %q, want RUNNING", execution.Status)
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

func TestRunEndEventToRunExecution(t *testing.T) {
	start := time.Date(2026, 4, 18, 9, 30, 0, 0, time.UTC)
	req := &events.TestRunEndEventRequest{
		RunId:       "run-123",
		ExecutionId: "exec-123",
		FinalStatus: common.TestStatus_FAILED,
		StartTime:   timestamppb.New(start),
		Duration:    durationpb.New(5 * time.Second),
	}

	execution := RunEndEventToRunExecution(req)
	if execution == nil {
		t.Fatal("expected run execution")
	}
	if execution.ID != "run-123:execution:exec-123" {
		t.Fatalf("ID = %q, want run-123:execution:exec-123", execution.ID)
	}
	if execution.Status != common.TestStatus_FAILED.String() {
		t.Fatalf("Status = %q, want %q", execution.Status, common.TestStatus_FAILED.String())
	}
	if execution.StartTime == nil || !execution.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", execution.StartTime, start)
	}
	if execution.Duration == nil || *execution.Duration != int64((5*time.Second).Nanoseconds()) {
		t.Fatalf("Duration = %v, want %d", execution.Duration, int64((5 * time.Second).Nanoseconds()))
	}
	if execution.EndTime == nil {
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

func TestRunStartEventToTestRun_FlattensSuitesAndUsesSuiteMetadata(t *testing.T) {
	req := &events.ReportRunStartEventRequest{
		RunId:      "run-123",
		Name:       "Run",
		TotalTests: 3,
		Metadata: map[string]string{
			"run_level": "yes",
		},
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:      "suite-root",
				RunId:   "run-123",
				Name:    "Root Suite",
				Project: "chromium",
				Metadata: map[string]string{
					"suite_level": "root",
				},
				SubSuites: []*entities.TestSuiteRun{
					{
						Id:            "suite-child",
						RunId:         "run-123",
						ParentSuiteId: "suite-root",
						Name:          "Child Suite",
						Metadata: map[string]string{
							"suite_level": "child",
						},
					},
				},
			},
		},
	}

	run, suites := RunStartEventToTestRun(req)
	if run == nil {
		t.Fatal("expected run mapping")
	}
	if len(suites) != 2 {
		t.Fatalf("len(suites) = %d, want 2", len(suites))
	}
	if suites[0].Metadata["suite_level"] != "root" {
		t.Fatalf("root suite metadata = %+v, want suite-specific metadata", suites[0].Metadata)
	}
	if _, ok := suites[0].Metadata["run_level"]; ok {
		t.Fatalf("root suite should not inherit run metadata, got %+v", suites[0].Metadata)
	}
	if suites[0].ParentSuiteID != nil {
		t.Fatalf("root suite parent should be nil, got %v", *suites[0].ParentSuiteID)
	}
	if suites[1].ParentSuiteID == nil || *suites[1].ParentSuiteID != buildSuiteRowID("run-123", "suite-root") {
		t.Fatalf("child suite parent = %v, want %s", suites[1].ParentSuiteID, buildSuiteRowID("run-123", "suite-root"))
	}
	if suites[1].Metadata["suite_level"] != "child" {
		t.Fatalf("child suite metadata = %+v, want child metadata", suites[1].Metadata)
	}
}

func TestRunStartEventToTests_FlattensNestedTests(t *testing.T) {
	req := &events.ReportRunStartEventRequest{
		RunId: "run-123",
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:    "suite-root",
				RunId: "run-123",
				TestCases: []*entities.TestCaseRun{
					{
						Id:          "test-root",
						RunId:       "run-123",
						TestSuiteId: "suite-root",
						Name:        "Root Test",
						Metadata: map[string]string{
							"test_level": "root",
						},
						RetryCount: 1,
						RetryIndex: 0,
					},
				},
				SubSuites: []*entities.TestSuiteRun{
					{
						Id:            "suite-child",
						RunId:         "run-123",
						ParentSuiteId: "suite-root",
						TestCases: []*entities.TestCaseRun{
							{
								Id:    "test-child",
								RunId: "run-123",
								Name:  "Child Test",
							},
						},
					},
				},
			},
		},
	}

	tests := RunStartEventToTests(req)
	if len(tests) != 2 {
		t.Fatalf("len(tests) = %d, want 2", len(tests))
	}
	if tests[0].SuiteID == nil || *tests[0].SuiteID != buildSuiteRowID("run-123", "suite-root") {
		t.Fatalf("root test suiteID = %v, want %s", tests[0].SuiteID, buildSuiteRowID("run-123", "suite-root"))
	}
	if tests[0].Metadata["test_level"] != "root" {
		t.Fatalf("root test metadata = %+v, want test metadata", tests[0].Metadata)
	}
	if tests[0].RetryCount == nil || *tests[0].RetryCount != 1 {
		t.Fatalf("root test retryCount = %v, want 1", tests[0].RetryCount)
	}
	if tests[0].RetryIndex == nil || *tests[0].RetryIndex != 0 {
		t.Fatalf("root test retryIndex = %v, want 0", tests[0].RetryIndex)
	}
	if tests[1].SuiteID == nil || *tests[1].SuiteID != buildSuiteRowID("run-123", "suite-child") {
		t.Fatalf("child test suiteID = %v, want %s", tests[1].SuiteID, buildSuiteRowID("run-123", "suite-child"))
	}
}

func TestTestCaseRunToRelationalTest(t *testing.T) {
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)
	req := &entities.TestCaseRun{
		Id:          "test-123",
		RunId:       "run-123",
		TestSuiteId: "suite-123",
		Name:        "My Test",
		Description: "desc",
		Status:      common.TestStatus_RUNNING,
		StartTime:   timestamppb.New(start),
		EndTime:     timestamppb.New(end),
		Duration:    durationpb.New(2 * time.Second),
		Metadata: map[string]string{
			"browser": "chromium",
		},
		Tags:       []string{"smoke"},
		Location:   "spec.ts:10",
		RetryCount: 2,
		RetryIndex: 1,
		Timeout:    30000,
	}

	test := TestCaseRunToRelationalTest(req)
	if test == nil {
		t.Fatal("expected relational test mapping")
	}
	if test.ID != buildTestRowID("run-123", "test-123") || test.RunID != "run-123" {
		t.Fatalf("unexpected identity mapping: %+v", test)
	}
	if test.ExternalTestID != "test-123" {
		t.Fatalf("ExternalTestID = %q, want test-123", test.ExternalTestID)
	}
	if test.SuiteID == nil || *test.SuiteID != buildSuiteRowID("run-123", "suite-123") {
		t.Fatalf("SuiteID = %v, want %s", test.SuiteID, buildSuiteRowID("run-123", "suite-123"))
	}
	if test.Status != common.TestStatus_RUNNING.String() {
		t.Fatalf("Status = %q, want %q", test.Status, common.TestStatus_RUNNING.String())
	}
	if test.StartTime == nil || !test.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", test.StartTime, start)
	}
	if test.EndTime == nil || !test.EndTime.Equal(end) {
		t.Fatalf("EndTime = %v, want %v", test.EndTime, end)
	}
	if test.Duration == nil || *test.Duration != int64((2*time.Second).Nanoseconds()) {
		t.Fatalf("Duration = %v, want %d", test.Duration, (2 * time.Second).Nanoseconds())
	}
	if test.Metadata["browser"] != "chromium" {
		t.Fatalf("Metadata = %+v, want browser metadata", test.Metadata)
	}
	if test.RetryIndex == nil || *test.RetryIndex != 1 {
		t.Fatalf("RetryIndex = %v, want 1", test.RetryIndex)
	}
}

func TestTestCaseRunToRelationalAttempt_UsesExecutionAwareIdentity(t *testing.T) {
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	req := &entities.TestCaseRun{
		Id:          "test-123",
		RunId:       "run-123",
		ExecutionId: "exec-123",
		Status:      common.TestStatus_RUNNING,
		StartTime:   timestamppb.New(start),
		RetryIndex:  1,
	}

	attempt := TestCaseRunToRelationalAttempt(req, nil)
	if attempt == nil {
		t.Fatal("expected relational attempt mapping")
	}
	if attempt.ExecutionID != "exec-123" {
		t.Fatalf("ExecutionID = %q, want exec-123", attempt.ExecutionID)
	}
	if attempt.ID != "run-123:test:test-123:execution:exec-123:attempt:1" {
		t.Fatalf("ID = %q, want execution-aware attempt id", attempt.ID)
	}
	if attempt.TestID != buildTestRowID("run-123", "test-123") {
		t.Fatalf("TestID = %q, want %q", attempt.TestID, buildTestRowID("run-123", "test-123"))
	}
}

func TestTestCaseRunToRelationalAttempt(t *testing.T) {
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	req := &entities.TestCaseRun{
		Id:           "test-123",
		RunId:        "run-123",
		Status:       common.TestStatus_FAILED,
		StartTime:    timestamppb.New(start),
		ErrorMessage: "boom",
		StackTrace:   "trace",
		Errors:       []string{"boom", "stack"},
		RetryIndex:   2,
	}
	attachments := []map[string]interface{}{{"name": "trace.txt"}}

	attempt := TestCaseRunToRelationalAttempt(req, attachments)
	if attempt == nil {
		t.Fatal("expected relational attempt mapping")
	}
	if attempt.ID != buildTestAttemptID(buildTestRowID("run-123", "test-123"), "", 2) {
		t.Fatalf("ID = %q, want %s", attempt.ID, buildTestAttemptID(buildTestRowID("run-123", "test-123"), "", 2))
	}
	if attempt.TestID != buildTestRowID("run-123", "test-123") {
		t.Fatalf("TestID = %q, want %s", attempt.TestID, buildTestRowID("run-123", "test-123"))
	}
	if attempt.AttemptIndex != 2 {
		t.Fatalf("AttemptIndex = %d, want 2", attempt.AttemptIndex)
	}
	if attempt.Status != common.TestStatus_FAILED.String() {
		t.Fatalf("Status = %q, want %q", attempt.Status, common.TestStatus_FAILED.String())
	}
	if attempt.StartTime == nil || !attempt.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", attempt.StartTime, start)
	}
	if len(attempt.Attachments) != 1 || attempt.Attachments[0]["name"] != "trace.txt" {
		t.Fatalf("Attachments = %+v, want trace.txt attachment", attempt.Attachments)
	}
	if attempt.ErrorMessage != "boom" || attempt.StackTrace != "trace" {
		t.Fatalf("unexpected error mapping: %+v", attempt)
	}
}

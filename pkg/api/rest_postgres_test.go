package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	pgRepo "github.com/stanterprise/observer/internal/repository/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPostgresHandleMarkers(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	seedRuns(t, db,
		m.TestRun{ID: "run-1", Name: "Run 1", Metadata: map[string]interface{}{"MARKER": "release-1.0"}, CreatedAt: now, UpdatedAt: now},
		m.TestRun{ID: "run-2", Name: "Run 2", Metadata: map[string]interface{}{"MARKER": "release-1.0"}, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)},
		m.TestRun{ID: "run-3", Name: "Run 3", Metadata: map[string]interface{}{"MARKER": "nightly"}, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)},
		m.TestRun{ID: "run-4", Name: "Run 4", Metadata: map[string]interface{}{"environment": "staging"}, CreatedAt: now.Add(3 * time.Minute), UpdatedAt: now.Add(3 * time.Minute)},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/markers", nil)
	rec := httptest.NewRecorder()
	handler.handleMarkers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Markers []struct {
			Marker string `json:"marker"`
			Count  int64  `json:"count"`
		} `json:"markers"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Count != 2 {
		t.Fatalf("expected 2 unique markers, got %d", response.Count)
	}
	if len(response.Markers) != 2 {
		t.Fatalf("expected 2 marker rows, got %d", len(response.Markers))
	}
	if response.Markers[0].Marker != "release-1.0" || response.Markers[0].Count != 2 {
		t.Fatalf("unexpected first marker row: %+v", response.Markers[0])
	}
}

func TestPostgresHandleRuns(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC)
	suiteID := "suite-1"
	suiteID2 := "suite-2"
	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Release", Status: "RUNNING", CreatedAt: now, UpdatedAt: now})
	seedSuites(t, db,
		m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: now, UpdatedAt: now},
		m.Suite{ID: suiteID2, RunID: "run-1", Name: "Suite 2", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	)
	seedTests(t, db,
		m.Test{ID: "test-root", RunID: "run-1", SuiteID: &suiteID2, Name: "Root Test", Title: "Root Test", Status: "PASSED", CreatedAt: now, UpdatedAt: now},
		m.Test{ID: "test-suite", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", Status: "FAILED", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.handleRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Runs []struct {
			ID         string                 `json:"id"`
			Name       string                 `json:"name"`
			TotalTests int                    `json:"totalTests"`
			Statistics map[string]interface{} `json:"statistics"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(response.Runs))
	}
	if response.Runs[0].ID != "run-1" || response.Runs[0].TotalTests != 2 {
		t.Fatalf("unexpected run summary: %+v", response.Runs[0])
	}
	if got := int(response.Runs[0].Statistics["passed"].(float64)); got != 1 {
		t.Fatalf("passed count = %d, want 1", got)
	}
	if got := int(response.Runs[0].Statistics["failed"].(float64)); got != 1 {
		t.Fatalf("failed count = %d, want 1", got)
	}
}

func TestPostgresHandleRuns_LogicalRunWithMultipleExecutionsReturnsSingleRow(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 11, 30, 0, 0, time.UTC)

	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Logical Aggregate", Status: "RUNNING", CreatedAt: now, UpdatedAt: now})
	seedRunExecutions(t, db,
		m.RunExecution{ID: "exec-a", RunID: "run-1", Name: "Logical Aggregate", Status: "RUNNING", TotalTests: 3, CreatedAt: now, UpdatedAt: now},
		m.RunExecution{ID: "exec-b", RunID: "run-1", Name: "Logical Aggregate", Status: "RUNNING", TotalTests: 5, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.handleRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Runs []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("expected 1 logical run row, got %d: %s", len(response.Runs), rec.Body.String())
	}
	if response.Runs[0].ID != "run-1" {
		t.Fatalf("run id = %q, want run-1", response.Runs[0].ID)
	}
}

func TestPostgresHandleRuns_UsesShardCompletionForDisplayedRunStatus(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 11, 45, 0, 0, time.UTC)
	start := now.Add(-2 * time.Minute)
	finish := now.Add(3 * time.Minute)
	shardOne := int32(1)
	shardTwo := int32(2)

	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Logical Aggregate", Status: "RUNNING", CreatedAt: now, UpdatedAt: now})
	seedRunExecutions(t, db,
		m.RunExecution{ID: "exec-a", RunID: "run-1", Status: "RUNNING", TotalTests: 3, CreatedAt: now, UpdatedAt: now},
		m.RunExecution{ID: "exec-b", RunID: "run-1", Status: "RUNNING", TotalTests: 5, CreatedAt: now, UpdatedAt: now},
	)
	seedRunShards(t, db,
		m.RunShard{ID: "run-1:exec-a:1", RunID: "run-1", ExecutionID: "exec-a", ShardIndex: &shardOne, ShardCountExpected: &shardTwo, Status: "FAILED", StartTime: &start, EndTime: &finish, CreatedAt: now, UpdatedAt: now},
		m.RunShard{ID: "run-1:exec-b:2", RunID: "run-1", ExecutionID: "exec-b", ShardIndex: &shardTwo, ShardCountExpected: &shardTwo, Status: "PASSED", StartTime: &now, EndTime: &finish, CreatedAt: now, UpdatedAt: now},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.handleRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Runs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(response.Runs))
	}
	if response.Runs[0].Status != "FAILED" {
		t.Fatalf("run status = %q, want FAILED", response.Runs[0].Status)
	}
}

func TestPostgresHandleRunDetail(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	suiteID := "suite-1"
	retryIndex := int32(0)
	stepsPayload := stepPayload(t, []*m.StepDocument{{
		ID:        "step-1",
		Title:     "Step 1",
		Status:    "PASSED",
		CreatedAt: now,
		UpdatedAt: now,
	}})
	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Release", Status: "PASSED", CreatedAt: now, UpdatedAt: now})
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: now, UpdatedAt: now})
	seedTests(t, db, m.Test{ID: "test-1", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", Status: "PASSED", RetryIndex: &retryIndex, CreatedAt: now, UpdatedAt: now})
	seedAttempts(t, db, m.TestAttempt{ID: "test-1:0", RunID: "run-1", TestID: "test-1", AttemptIndex: 0, Status: "PASSED", Steps: stepsPayload, Attachments: []map[string]interface{}{{"name": "trace.txt", "storage": "inline", "content": "dGVzdA==", "content_encoding": "base64"}}, CreatedAt: now, UpdatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run-1", nil)
	rec := httptest.NewRecorder()
	handler.handleRunDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	suites, ok := raw["suites"].([]interface{})
	if !ok || len(suites) != 1 {
		t.Fatalf("raw suites = %+v, want 1 suite", raw["suites"])
	}
	suiteMap, ok := suites[0].(map[string]interface{})
	if !ok {
		t.Fatalf("raw suite payload = %+v, want object", suites[0])
	}
	tests, ok := suiteMap["tests"].([]interface{})
	if !ok || len(tests) != 1 {
		t.Fatalf("raw suite tests = %+v, want 1 test", suiteMap["tests"])
	}
	testMap, ok := tests[0].(map[string]interface{})
	if !ok {
		t.Fatalf("raw test payload = %+v, want object", tests[0])
	}
	attempts, ok := testMap["attempts"].([]interface{})
	if !ok || len(attempts) != 1 {
		t.Fatalf("raw attempts = %+v, want 1 attempt", testMap["attempts"])
	}
	_, ok = attempts[0].(map[string]interface{})
	if !ok {
		t.Fatalf("raw attempt payload = %+v, want object", attempts[0])
	}

	var response m.TestRun
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != "run-1" {
		t.Fatalf("run id = %q, want run-1", response.ID)
	}
	if response.Status != "PASSED" {
		t.Fatalf("run status = %q, want PASSED", response.Status)
	}
}

func TestPostgresHandleRunDetail_RecomputesInflatedTotalTestsFromAttachedTests(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 18, 12, 30, 0, 0, time.UTC)
	suiteID := "suite-1"
	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Release", Status: "FAILED", TotalTests: 13062, CreatedAt: now, UpdatedAt: now})
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: now, UpdatedAt: now})
	seedTests(t, db,
		m.Test{ID: "test-1", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test 1", Title: "Suite Test 1", Status: "PASSED", CreatedAt: now, UpdatedAt: now},
		m.Test{ID: "test-2", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test 2", Title: "Suite Test 2", Status: "FAILED", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run-1", nil)
	rec := httptest.NewRecorder()
	handler.handleRunDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response m.TestRun
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TotalTests != 2 {
		t.Fatalf("run totalTests = %d, want 2", response.TotalTests)
	}
}

func TestPostgresHandleRuns_DerivesStatisticsFromAttemptStatusWhenTestWasResetToNotRun(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 29, 3, 35, 1, 0, time.UTC)
	suiteID := "suite-1"
	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Logical Aggregate", Status: "PASSED", CreatedAt: now, UpdatedAt: now})
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: now, UpdatedAt: now})
	seedTests(t, db, m.Test{ID: "test-1", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", Status: "NOT_RUN", CreatedAt: now, UpdatedAt: now})
	seedAttempts(t, db, m.TestAttempt{ID: "test-1:execution:exec-a:attempt:0", RunID: "run-1", ExecutionID: "exec-a", TestID: "test-1", AttemptIndex: 0, Status: "PASSED", CreatedAt: now, UpdatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.handleRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Runs []struct {
			Status     string                 `json:"status"`
			Statistics map[string]interface{} `json:"statistics"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(response.Runs))
	}
	if got := int(response.Runs[0].Statistics["passed"].(float64)); got != 1 {
		t.Fatalf("passed count = %d, want 1", got)
	}
	if got := int(response.Runs[0].Statistics["unknown"].(float64)); got != 0 {
		t.Fatalf("unknown count = %d, want 0", got)
	}
}

func TestPostgresHandleRunDetail_DerivesNestedTestStatusFromAttemptStatusWhenTestWasResetToNotRun(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 29, 3, 35, 1, 0, time.UTC)
	suiteID := "suite-1"
	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Logical Aggregate", Status: "PASSED", CreatedAt: now, UpdatedAt: now})
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: now, UpdatedAt: now})
	seedTests(t, db, m.Test{ID: "test-1", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", Status: "NOT_RUN", CreatedAt: now, UpdatedAt: now})
	seedAttempts(t, db, m.TestAttempt{ID: "test-1:execution:exec-a:attempt:0", RunID: "run-1", ExecutionID: "exec-a", TestID: "test-1", AttemptIndex: 0, Status: "PASSED", CreatedAt: now, UpdatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run-1", nil)
	rec := httptest.NewRecorder()
	handler.handleRunDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response m.TestRun
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Suites) != 1 || len(response.Suites[0].Tests) != 1 {
		t.Fatalf("unexpected suites payload: %+v", response.Suites)
	}
	if response.Suites[0].Tests[0].Status != "PASSED" {
		t.Fatalf("nested test status = %q, want PASSED", response.Suites[0].Tests[0].Status)
	}
}

func TestPostgresHandleRuns_UsesLatestExecutionStatusForRepeatedLogicalTest(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	firstStart := time.Date(2026, 4, 29, 4, 30, 0, 0, time.UTC)
	firstEnd := firstStart.Add(2 * time.Second)
	secondStart := firstStart.Add(10 * time.Second)
	secondEnd := secondStart.Add(2 * time.Second)
	suiteID := "suite-1"

	seedRuns(t, db, m.TestRun{ID: "run-1", Name: "Logical Aggregate", Status: "FAILED", CreatedAt: firstStart, UpdatedAt: secondEnd})
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", Name: "Suite", CreatedAt: firstStart, UpdatedAt: secondEnd})
	seedTests(t, db, m.Test{ID: "test-1", RunID: "run-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", Status: "PASSED", CreatedAt: firstStart, UpdatedAt: secondEnd})
	seedAttempts(t, db,
		m.TestAttempt{ID: "test-1:execution:exec-a:attempt:0", RunID: "run-1", ExecutionID: "exec-a", TestID: "test-1", AttemptIndex: 0, Status: "PASSED", StartTime: &firstStart, EndTime: &firstEnd, CreatedAt: firstStart, UpdatedAt: firstEnd},
		m.TestAttempt{ID: "test-1:execution:exec-b:attempt:0", RunID: "run-1", ExecutionID: "exec-b", TestID: "test-1", AttemptIndex: 0, Status: "FAILED", StartTime: &secondStart, EndTime: &secondEnd, CreatedAt: secondStart, UpdatedAt: secondEnd},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.handleRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Runs []struct {
			Statistics map[string]interface{} `json:"statistics"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(response.Runs))
	}
	if got := int(response.Runs[0].Statistics["failed"].(float64)); got != 1 {
		t.Fatalf("failed count = %d, want 1", got)
	}
	if got := int(response.Runs[0].Statistics["passed"].(float64)); got != 0 {
		t.Fatalf("passed count = %d, want 0", got)
	}
}

func TestPostgresLoadLiveRunningTestDetails_UsesLiveRunningDetailRepo(t *testing.T) {
	handler := NewPostgresHandler(nil, nil)
	now := time.Date(2026, 4, 19, 15, 0, 0, 0, time.UTC)
	retryIndex := int32(0)
	stepStart := now.Add(5 * time.Second)
	stepDuration := int64(3 * time.Second)
	stepPayload := stepPayload(t, []*m.StepDocument{{
		ID:        "step-1",
		Title:     "Live step",
		Status:    "RUNNING",
		StartTime: &stepStart,
		Duration:  &stepDuration,
	}})

	handler.SetLiveRunRepo(fakeLiveRunRepo{doc: &m.TestRun{
		ID: "run-1",
		Tests: []*m.Test{{
			ID:         "test-1",
			Status:     "RUNNING",
			RetryIndex: &retryIndex,
			Attempts: []m.TestAttempt{{
				ID:           "test-1:0",
				AttemptIndex: 0,
				Status:       "RUNNING",
				StartTime:    &stepStart,
				Duration:     &stepDuration,
				Steps:        stepPayload,
			}},
		}},
	}})

	baseTests := []*m.Test{{
		ID:         "test-1",
		Status:     "RUNNING",
		RetryIndex: &retryIndex,
	}}

	liveTests, err := handler.loadLiveRunningTestDetails(context.Background(), "run-1", baseTests)
	if err != nil {
		t.Fatalf("loadLiveRunningTestDetails: %v", err)
	}
	if len(liveTests) != 1 {
		t.Fatalf("expected 1 live test, got %d", len(liveTests))
	}
	if len(liveTests[0].Attempts) != 1 {
		t.Fatalf("expected 1 live attempt, got %+v", liveTests[0].Attempts)
	}
	if liveTests[0].Attempts[0].Status != "RUNNING" {
		t.Fatalf("attempt status = %q, want RUNNING", liveTests[0].Attempts[0].Status)
	}

	var steps []*m.StepDocument
	if liveTests[0].Attempts[0].Steps == nil {
		t.Fatal("expected live attempt steps payload")
	}
	if err := json.Unmarshal(*liveTests[0].Attempts[0].Steps, &steps); err != nil {
		t.Fatalf("decode live attempt steps: %v", err)
	}
	if len(steps) != 1 || steps[0].Title != "Live step" {
		t.Fatalf("expected live step details, got %+v", steps)
	}
}

func TestPostgresHandleDeleteRuns(t *testing.T) {
	handler, db := setupPostgresHandler(t)
	now := time.Date(2026, 4, 19, 14, 0, 0, 0, time.UTC)
	suiteID := "run-1:suite:root"
	testID := "run-1:test:test-1"
	attemptID := testID + ":0"

	seedRuns(t, db,
		m.TestRun{ID: "run-1", Name: "Delete Me", CreatedAt: now, UpdatedAt: now},
		m.TestRun{ID: "run-keep", Name: "Keep Me", CreatedAt: now, UpdatedAt: now},
	)
	seedSuites(t, db, m.Suite{ID: suiteID, RunID: "run-1", ExternalSuiteID: "root", Name: "Suite", CreatedAt: now, UpdatedAt: now})
	seedTests(t, db, m.Test{ID: testID, RunID: "run-1", ExternalTestID: "test-1", SuiteID: &suiteID, Name: "Suite Test", Title: "Suite Test", CreatedAt: now, UpdatedAt: now})
	seedAttempts(t, db, m.TestAttempt{ID: attemptID, RunID: "run-1", TestID: testID, AttemptIndex: 0, Status: "PASSED", CreatedAt: now, UpdatedAt: now})
	seedAttachments(t, db, m.Attachment{ID: "attachment-1", RunID: "run-1", TestID: testID, TestAttemptID: attemptID, Name: "trace.zip", CreatedAt: now})

	body := bytes.NewBufferString(`{"runIds":["run-1"]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/runs/delete", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleDeleteRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Deleted   int64 `json:"deleted"`
		Requested int   `json:"requested"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Deleted != 1 || response.Requested != 1 {
		t.Fatalf("unexpected delete response: %+v", response)
	}

	assertRecordCount(t, db, &m.TestRun{}, "id", "run-1", 0)
	assertRecordCount(t, db, &m.Suite{}, "run_id", "run-1", 0)
	assertRecordCount(t, db, &m.Test{}, "run_id", "run-1", 0)
	assertRecordCount(t, db, &m.TestAttempt{}, "run_id", "run-1", 0)
	assertRecordCount(t, db, &m.Attachment{}, "run_id", "run-1", 0)
	assertRecordCount(t, db, &m.TestRun{}, "id", "run-keep", 1)
}

func setupPostgresHandler(t *testing.T) (*PostgresHandler, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(modelsForPostgresHandlerTests()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := pgRepo.NewPostgresRepository(db, nil)
	return NewPostgresHandler(repo, nil), db
}

func modelsForPostgresHandlerTests() []interface{} {
	return []interface{}{
		&m.TestRun{},
		&m.RunExecution{},
		&m.RunShard{},
		&m.Suite{},
		&m.Test{},
		&m.TestAttempt{},
		&m.Attachment{},
	}
}

func seedRuns(t *testing.T, db *gorm.DB, runs ...m.TestRun) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&runs).Error; err != nil {
		t.Fatalf("seed runs: %v", err)
	}
}

func seedRunExecutions(t *testing.T, db *gorm.DB, executions ...m.RunExecution) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&executions).Error; err != nil {
		t.Fatalf("seed run executions: %v", err)
	}
}

func seedRunShards(t *testing.T, db *gorm.DB, shards ...m.RunShard) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&shards).Error; err != nil {
		t.Fatalf("seed run shards: %v", err)
	}
}

func seedSuites(t *testing.T, db *gorm.DB, suites ...m.Suite) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&suites).Error; err != nil {
		t.Fatalf("seed suites: %v", err)
	}
}

func seedTests(t *testing.T, db *gorm.DB, tests ...m.Test) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&tests).Error; err != nil {
		t.Fatalf("seed tests: %v", err)
	}
}

func seedAttempts(t *testing.T, db *gorm.DB, attempts ...m.TestAttempt) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&attempts).Error; err != nil {
		t.Fatalf("seed attempts: %v", err)
	}
}

func seedAttachments(t *testing.T, db *gorm.DB, attachments ...m.Attachment) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&attachments).Error; err != nil {
		t.Fatalf("seed attachments: %v", err)
	}
}

func assertRecordCount(t *testing.T, db *gorm.DB, model interface{}, column, value string, want int64) {
	t.Helper()
	var count int64
	if err := db.WithContext(context.Background()).Model(model).Where(column+" = ?", value).Count(&count).Error; err != nil {
		t.Fatalf("count %T for %s=%s: %v", model, column, value, err)
	}
	if count != want {
		t.Fatalf("count %T for %s=%s = %d, want %d", model, column, value, count, want)
	}
}

type fakeLiveRunRepo struct {
	doc *m.TestRun
	err error
}

func (f fakeLiveRunRepo) GetTestRun(_ context.Context, _ string) (*m.TestRun, error) {
	return f.doc, f.err
}

func stepPayload(t *testing.T, steps []*m.StepDocument) *m.Step {
	t.Helper()
	raw, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal steps: %v", err)
	}
	payload := m.Step(raw)
	return &payload
}

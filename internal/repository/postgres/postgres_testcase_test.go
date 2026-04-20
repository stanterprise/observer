package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestUpsertTestBeginCreatesTestAndAttempt(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	test := &m.Test{
		ID:             "run-123:test:test-123",
		RunID:          "run-123",
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "RUNNING",
		StartTime:      &start,
		Metadata:       map[string]interface{}{"browser": "chromium"},
		RetryCount:     int32Ptr(2),
		RetryIndex:     int32Ptr(0),
		Timeout:        int32Ptr(30000),
	}
	attempt := &m.TestAttempt{
		ID:           "run-123:test:test-123:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-123",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
		Attachments:  []map[string]interface{}{{"name": "stdout.txt"}},
	}

	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "run-123:test:test-123").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.Status != "RUNNING" {
		t.Fatalf("stored test status = %q, want RUNNING", storedTest.Status)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("stored suite id = %v, want %s", storedTest.SuiteID, suiteID)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", "run-123:test:test-123", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load stored attempt: %v", err)
	}
	if storedAttempt.Status != "RUNNING" {
		t.Fatalf("stored attempt status = %q, want RUNNING", storedAttempt.Status)
	}
	if len(storedAttempt.Attachments) != 1 || storedAttempt.Attachments[0]["name"] != "stdout.txt" {
		t.Fatalf("stored attempt attachments = %+v, want stdout.txt", storedAttempt.Attachments)
	}
}

func TestUpsertTestBeginAndFinalizeUpdateSeededPlaceholderTest(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)

	seeded := &m.Test{
		ID:             "run-123:test:test-seeded",
		RunID:          "run-123",
		ExternalTestID: "test-seeded",
		SuiteID:        &suiteID,
		Name:           "Seeded Placeholder",
		Title:          "Seeded Placeholder",
		Status:         "NOT_RUN",
		CreatedAt:      start,
		UpdatedAt:      start,
	}
	if err := repo.db.WithContext(ctx).Create(seeded).Error; err != nil {
		t.Fatalf("seed test placeholder: %v", err)
	}

	beginTest := &m.Test{
		ID:             seeded.ID,
		RunID:          seeded.RunID,
		ExternalTestID: seeded.ExternalTestID,
		SuiteID:        &suiteID,
		Name:           "Seeded Placeholder",
		Title:          "Seeded Placeholder",
		Status:         "RUNNING",
		StartTime:      &start,
		Metadata:       map[string]interface{}{"annotation_0_type": "tag", "annotation_0_description": "regression"},
		Tags:           []string{"@sample"},
		RetryCount:     int32Ptr(3),
		RetryIndex:     int32Ptr(0),
		Timeout:        int32Ptr(0),
	}
	beginAttempt := &m.TestAttempt{
		ID:           seeded.ID + ":0",
		RunID:        seeded.RunID,
		TestID:       seeded.ID,
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
		Attachments:  []map[string]interface{}{{"name": "stdout.txt"}},
	}
	if err := repo.UpsertTestBegin(ctx, beginTest, beginAttempt); err != nil {
		t.Fatalf("UpsertTestBegin on seeded placeholder failed: %v", err)
	}

	endTest := &m.Test{
		ID:             seeded.ID,
		RunID:          seeded.RunID,
		ExternalTestID: seeded.ExternalTestID,
		SuiteID:        &suiteID,
		Name:           "Seeded Placeholder",
		Title:          "Seeded Placeholder",
		Status:         "PASSED",
		StartTime:      &start,
		EndTime:        &end,
		Duration:       int64Ptr(int64((2 * time.Second).Nanoseconds())),
		Metadata:       map[string]interface{}{"annotation_0_type": "tag", "annotation_0_description": "regression"},
		Tags:           []string{"@sample"},
		RetryCount:     int32Ptr(3),
		RetryIndex:     int32Ptr(0),
		Timeout:        int32Ptr(0),
	}
	endAttempt := &m.TestAttempt{
		ID:           seeded.ID + ":0",
		RunID:        seeded.RunID,
		TestID:       seeded.ID,
		AttemptIndex: 0,
		Status:       "PASSED",
		StartTime:    &start,
		EndTime:      &end,
		Duration:     int64Ptr(int64((2 * time.Second).Nanoseconds())),
		Attachments:  []map[string]interface{}{{"name": "result.json"}},
	}
	if err := repo.FinalizeTestEnd(ctx, endTest, endAttempt); err != nil {
		t.Fatalf("FinalizeTestEnd on seeded placeholder failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", seeded.ID).Error; err != nil {
		t.Fatalf("load updated test: %v", err)
	}
	if storedTest.Status != "PASSED" {
		t.Fatalf("stored test status = %q, want PASSED", storedTest.Status)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("stored suite id = %v, want %s", storedTest.SuiteID, suiteID)
	}
	if got := storedTest.Metadata["annotation_0_description"]; got != "regression" {
		t.Fatalf("stored metadata annotation_0_description = %v, want regression", got)
	}
	if len(storedTest.Tags) != 1 || storedTest.Tags[0] != "@sample" {
		t.Fatalf("stored tags = %+v, want @sample", storedTest.Tags)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", seeded.ID, 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load updated attempt: %v", err)
	}
	if storedAttempt.Status != "PASSED" {
		t.Fatalf("stored attempt status = %q, want PASSED", storedAttempt.Status)
	}
	if len(storedAttempt.Attachments) != 1 || storedAttempt.Attachments[0]["name"] != "result.json" {
		t.Fatalf("stored attempt attachments = %+v, want result.json", storedAttempt.Attachments)
	}
}

func TestFinalizeTestEndAggregatesPassingRetries(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	firstEnd := start.Add(2 * time.Second)
	secondStart := start.Add(3 * time.Second)
	secondEnd := start.Add(5 * time.Second)

	firstTest := &m.Test{
		ID:             "run-123:test:test-123",
		RunID:          "run-123",
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "FAILED",
		StartTime:      &start,
		EndTime:        &firstEnd,
		Duration:       int64Ptr(int64((2 * time.Second).Nanoseconds())),
		RetryCount:     int32Ptr(2),
		RetryIndex:     int32Ptr(0),
	}
	firstAttempt := &m.TestAttempt{
		ID:           "run-123:test:test-123:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-123",
		AttemptIndex: 0,
		Status:       "FAILED",
		StartTime:    &start,
		EndTime:      &firstEnd,
		Duration:     int64Ptr(int64((2 * time.Second).Nanoseconds())),
		ErrorMessage: "boom",
	}
	if err := repo.UpsertTestBegin(ctx, firstTest, firstAttempt); err != nil {
		t.Fatalf("seed first attempt: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, firstTest, firstAttempt); err != nil {
		t.Fatalf("finalize first attempt: %v", err)
	}

	secondTest := &m.Test{
		ID:             "run-123:test:test-123",
		RunID:          "run-123",
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "PASSED",
		StartTime:      &secondStart,
		EndTime:        &secondEnd,
		Duration:       int64Ptr(int64((2 * time.Second).Nanoseconds())),
		RetryCount:     int32Ptr(2),
		RetryIndex:     int32Ptr(1),
	}
	secondAttempt := &m.TestAttempt{
		ID:           "run-123:test:test-123:1",
		RunID:        "run-123",
		TestID:       "run-123:test:test-123",
		AttemptIndex: 1,
		Status:       "PASSED",
		StartTime:    &secondStart,
		EndTime:      &secondEnd,
		Duration:     int64Ptr(int64((2 * time.Second).Nanoseconds())),
		Attachments:  []map[string]interface{}{{"name": "result.json"}},
	}
	if err := repo.UpsertTestBegin(ctx, secondTest, secondAttempt); err != nil {
		t.Fatalf("seed second attempt: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, secondTest, secondAttempt); err != nil {
		t.Fatalf("finalize second attempt: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "run-123:test:test-123").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.Status != "PASSED" {
		t.Fatalf("stored test status = %q, want PASSED", storedTest.Status)
	}
	if storedTest.RetryIndex == nil || *storedTest.RetryIndex != 1 {
		t.Fatalf("stored retry index = %v, want 1", storedTest.RetryIndex)
	}
	if storedTest.EndTime == nil || !storedTest.EndTime.Equal(secondEnd) {
		t.Fatalf("stored end time = %v, want %v", storedTest.EndTime, secondEnd)
	}

	var storedAttempts []m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ?", "run-123:test:test-123").Order("attempt_index asc").Find(&storedAttempts).Error; err != nil {
		t.Fatalf("load stored attempts: %v", err)
	}
	if len(storedAttempts) != 2 {
		t.Fatalf("len(storedAttempts) = %d, want 2", len(storedAttempts))
	}
	if storedAttempts[1].Status != "PASSED" {
		t.Fatalf("second attempt status = %q, want PASSED", storedAttempts[1].Status)
	}
	if len(storedAttempts[1].Attachments) != 1 || storedAttempts[1].Attachments[0]["name"] != "result.json" {
		t.Fatalf("second attempt attachments = %+v, want result.json", storedAttempts[1].Attachments)
	}
}

func TestFinalizeTestEndPersistsAttemptStepsWithoutClearingOnLaterRetry(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)

	test := &m.Test{
		ID:             "run-123:test:test-steps",
		RunID:          "run-123",
		ExternalTestID: "test-steps",
		SuiteID:        &suiteID,
		Name:           "Step Test",
		Title:          "Step Test",
		Status:         "PASSED",
		StartTime:      &start,
		EndTime:        &end,
		RetryCount:     int32Ptr(0),
		RetryIndex:     int32Ptr(0),
	}
	stepsRaw := jsonRawMessage(t, []*m.StepDocument{{ID: "step-1", Title: "Click", Status: "PASSED"}})
	attempt := &m.TestAttempt{
		ID:           "run-123:test:test-steps:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-steps",
		AttemptIndex: 0,
		Status:       "PASSED",
		StartTime:    &start,
		EndTime:      &end,
		Steps:        stepsRaw,
	}

	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, test, attempt); err != nil {
		t.Fatalf("FinalizeTestEnd failed: %v", err)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", "run-123:test:test-steps", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load stored attempt: %v", err)
	}
	if storedAttempt.Steps == nil {
		t.Fatal("expected stored steps to be persisted")
	}
	decoded := decodeAttemptSteps(storedAttempt.Steps)
	if len(decoded) != 1 || decoded[0].ID != "step-1" {
		t.Fatalf("decoded steps = %+v, want step-1", decoded)
	}

	secondEnd := end.Add(2 * time.Second)
	retryTest := &m.Test{
		ID:             "run-123:test:test-steps",
		RunID:          "run-123",
		ExternalTestID: "test-steps",
		SuiteID:        &suiteID,
		Name:           "Step Test",
		Title:          "Step Test",
		Status:         "PASSED",
		StartTime:      &end,
		EndTime:        &secondEnd,
		RetryCount:     int32Ptr(1),
		RetryIndex:     int32Ptr(1),
	}
	retryAttempt := &m.TestAttempt{
		ID:           "run-123:test:test-steps:1",
		RunID:        "run-123",
		TestID:       "run-123:test:test-steps",
		AttemptIndex: 1,
		Status:       "PASSED",
		StartTime:    &end,
		EndTime:      &secondEnd,
	}
	if err := repo.UpsertTestBegin(ctx, retryTest, retryAttempt); err != nil {
		t.Fatalf("UpsertTestBegin retry failed: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, retryTest, retryAttempt); err != nil {
		t.Fatalf("FinalizeTestEnd retry failed: %v", err)
	}

	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", "run-123:test:test-steps", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("reload stored attempt: %v", err)
	}
	decoded = decodeAttemptSteps(storedAttempt.Steps)
	if len(decoded) != 1 || decoded[0].ID != "step-1" {
		t.Fatalf("decoded steps after retry = %+v, want preserved step-1", decoded)
	}
}

func TestFinalizeTestEndPreservesSuiteIDForSparseTerminalPayload(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)

	beginTest := &m.Test{
		ID:             "run-123:test:test-sparse",
		RunID:          "run-123",
		ExternalTestID: "test-sparse",
		SuiteID:        &suiteID,
		Name:           "Sparse Test",
		Title:          "Sparse Test",
		Status:         "RUNNING",
		StartTime:      &start,
		RetryCount:     int32Ptr(0),
		RetryIndex:     int32Ptr(0),
	}
	beginAttempt := &m.TestAttempt{
		ID:           "run-123:test:test-sparse:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-sparse",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
	}
	if err := repo.UpsertTestBegin(ctx, beginTest, beginAttempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	endTest := &m.Test{
		ID:             "run-123:test:test-sparse",
		RunID:          "run-123",
		ExternalTestID: "test-sparse",
		Status:         "PASSED",
		EndTime:        &end,
		RetryCount:     int32Ptr(0),
		RetryIndex:     int32Ptr(0),
	}
	endAttempt := &m.TestAttempt{
		ID:           "run-123:test:test-sparse:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-sparse",
		AttemptIndex: 0,
		Status:       "PASSED",
		EndTime:      &end,
	}
	if err := repo.FinalizeTestEnd(ctx, endTest, endAttempt); err != nil {
		t.Fatalf("FinalizeTestEnd failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "run-123:test:test-sparse").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("stored suite id = %v, want %s", storedTest.SuiteID, suiteID)
	}
	if storedTest.Name != "Sparse Test" {
		t.Fatalf("stored name = %q, want Sparse Test", storedTest.Name)
	}
	if storedTest.Status != "PASSED" {
		t.Fatalf("stored status = %q, want PASSED", storedTest.Status)
	}
}

func TestAggregateTestAttemptStatuses(t *testing.T) {
	attempts := []m.TestAttempt{{AttemptIndex: 0, Status: "FAILED"}, {AttemptIndex: 1, Status: "PASSED"}}
	if got := aggregateTestAttemptStatuses(attempts, "FAILED"); got != "PASSED" {
		t.Fatalf("aggregateTestAttemptStatuses() = %q, want PASSED", got)
	}
	if got := aggregateTestAttemptStatuses([]m.TestAttempt{{AttemptIndex: 0, Status: "FAILED"}}, "FAILED"); got != "FAILED" {
		t.Fatalf("aggregateTestAttemptStatuses(single failure) = %q, want FAILED", got)
	}
}

func TestAppendTestFailureAndError(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	test := &m.Test{
		ID:             "run-123:test:test-123",
		RunID:          "run-123",
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "FAILED",
		StartTime:      &start,
		RetryCount:     int32Ptr(1),
		RetryIndex:     int32Ptr(0),
	}
	attempt := &m.TestAttempt{
		ID:           "run-123:test:test-123:0",
		RunID:        "run-123",
		TestID:       "run-123:test:test-123",
		AttemptIndex: 0,
		Status:       "FAILED",
		StartTime:    &start,
	}
	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("seed attempt: %v", err)
	}

	failureTime := start.Add(time.Second)
	failure := &m.TestFailureDocument{
		FailureMessage: "assertion failed",
		StackTrace:     "stack trace",
		Timestamp:      &failureTime,
		Attachments:    []map[string]interface{}{{"name": "failure.txt"}},
	}
	if err := repo.AppendTestFailure(ctx, "run-123", "test-123", 0, failure); err != nil {
		t.Fatalf("AppendTestFailure failed: %v", err)
	}

	errorTime := start.Add(2 * time.Second)
	errorDoc := &m.TestErrorDocument{
		ErrorMessage: "stderr line",
		StackTrace:   "error stack",
		Timestamp:    &errorTime,
		Attachments:  []map[string]interface{}{{"name": "error.txt"}},
	}
	if err := repo.AppendTestError(ctx, "run-123", "test-123", 0, errorDoc); err != nil {
		t.Fatalf("AppendTestError failed: %v", err)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", "run-123:test:test-123", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load stored attempt: %v", err)
	}
	if len(storedAttempt.Failures) != 1 || storedAttempt.Failures[0].FailureMessage != "assertion failed" {
		t.Fatalf("stored failures = %+v, want assertion failed", storedAttempt.Failures)
	}
	if len(storedAttempt.Failures[0].Attachments) != 1 || storedAttempt.Failures[0].Attachments[0]["name"] != "failure.txt" {
		t.Fatalf("stored failure attachments = %+v, want failure.txt", storedAttempt.Failures[0].Attachments)
	}
	if len(storedAttempt.Errors) != 1 || storedAttempt.Errors[0].ErrorMessage != "stderr line" {
		t.Fatalf("stored errors = %+v, want stderr line", storedAttempt.Errors)
	}
	if len(storedAttempt.Errors[0].Attachments) != 1 || storedAttempt.Errors[0].Attachments[0]["name"] != "error.txt" {
		t.Fatalf("stored error attachments = %+v, want error.txt", storedAttempt.Errors[0].Attachments)
	}
}

func int64Ptr(value int64) *int64 {
	converted := value
	return &converted
}

func jsonRawMessage(t *testing.T, steps []*m.StepDocument) *json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal raw message: %v", err)
	}
	message := json.RawMessage(raw)
	return &message
}

func TestGetRunDocuments_PreservesHistoricalRunsForRepeatedExternalTestID(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteRun1 := "run-1:suite:suite-123"
	suiteRun2 := "run-2:suite:suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	for _, run := range []m.TestRun{{ID: "run-1", Name: "Run 1", Status: "PASSED", CreatedAt: start, UpdatedAt: start}, {ID: "run-2", Name: "Run 2", Status: "FAILED", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)}} {
		if err := repo.db.WithContext(ctx).Create(&run).Error; err != nil {
			t.Fatalf("seed run %s: %v", run.ID, err)
		}
	}
	for _, suite := range []m.Suite{{ID: suiteRun1, RunID: "run-1", ExternalSuiteID: "suite-123", Name: "Suite", CreatedAt: start, UpdatedAt: start}, {ID: suiteRun2, RunID: "run-2", ExternalSuiteID: "suite-123", Name: "Suite", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)}} {
		if err := repo.db.WithContext(ctx).Create(&suite).Error; err != nil {
			t.Fatalf("seed suite %s: %v", suite.ID, err)
		}
	}

	testRun1 := &m.Test{ID: "run-1:test:test-123", RunID: "run-1", ExternalTestID: "test-123", SuiteID: &suiteRun1, Name: "Test", Title: "Test", Status: "PASSED", CreatedAt: start, UpdatedAt: start}
	testRun2 := &m.Test{ID: "run-2:test:test-123", RunID: "run-2", ExternalTestID: "test-123", SuiteID: &suiteRun2, Name: "Test", Title: "Test", Status: "FAILED", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)}
	for _, test := range []*m.Test{testRun1, testRun2} {
		if err := repo.db.WithContext(ctx).Create(test).Error; err != nil {
			t.Fatalf("seed test %s: %v", test.ID, err)
		}
	}

	docs, _, err := repo.GetRunDocuments(ctx, ListRunsFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetRunDocuments failed: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 run documents, got %d", len(docs))
	}
	if len(docs[0].Suites) == 0 || len(docs[0].Suites[0].Tests) != 1 {
		t.Fatalf("latest run missing test payload: %+v", docs[0].Suites)
	}
	if len(docs[1].Suites) == 0 || len(docs[1].Suites[0].Tests) != 1 {
		t.Fatalf("historical run missing test payload: %+v", docs[1].Suites)
	}
	if docs[0].Suites[0].Tests[0].ID != "test-123" || docs[1].Suites[0].Tests[0].ID != "test-123" {
		t.Fatalf("expected external test IDs in API payloads, got %+v and %+v", docs[0].Suites[0].Tests[0], docs[1].Suites[0].Tests[0])
	}

	trends, err := repo.GetTestTrends(ctx, "test-123", 10)
	if err != nil {
		t.Fatalf("GetTestTrends failed: %v", err)
	}
	if len(trends) != 2 {
		t.Fatalf("expected 2 trend rows, got %d", len(trends))
	}
}

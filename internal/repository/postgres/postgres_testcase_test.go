package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestUpsertTestBeginUsesRawProtobufIDs(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	test := &m.Test{
		ID:             "test-123",
		RunID:          "run-123",
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "RUNNING",
		StartTime:      &start,
	}
	attempt := &m.TestAttempt{
		RunID:        "run-123",
		ExecutionID:  "exec-123",
		TestID:       "test-123",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
	}

	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "test-123").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.ExternalTestID != "test-123" {
		t.Fatalf("stored external test id = %q, want test-123", storedTest.ExternalTestID)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("stored suite id = %v, want %s", storedTest.SuiteID, suiteID)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND execution_id = ? AND attempt_index = ?", "test-123", "exec-123", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load stored attempt: %v", err)
	}
	if storedAttempt.ID != "run-123:test-123:exec-123:0" {
		t.Fatalf("stored attempt id = %q, want run-123:test-123:exec-123:0", storedAttempt.ID)
	}
}

func TestUpsertTestBeginKeepsSameRawTestIDInSeparateRuns(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	firstSuiteID := "suite-run-1"
	secondSuiteID := "suite-run-2"
	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	firstTest := &m.Test{
		ID:             "test-123",
		RunID:          "run-1",
		ExternalTestID: "test-123",
		SuiteID:        &firstSuiteID,
		Name:           "Run 1 Test",
		Title:          "Run 1 Test",
		Status:         "RUNNING",
		StartTime:      &start,
	}
	firstAttempt := &m.TestAttempt{
		RunID:        "run-1",
		ExecutionID:  "exec-1",
		TestID:       "test-123",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
	}

	secondStart := start.Add(time.Minute)
	secondTest := &m.Test{
		ID:             "test-123",
		RunID:          "run-2",
		ExternalTestID: "test-123",
		SuiteID:        &secondSuiteID,
		Name:           "Run 2 Test",
		Title:          "Run 2 Test",
		Status:         "RUNNING",
		StartTime:      &secondStart,
	}
	secondAttempt := &m.TestAttempt{
		RunID:        "run-2",
		ExecutionID:  "exec-2",
		TestID:       "test-123",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &secondStart,
	}

	if err := repo.UpsertTestBegin(ctx, firstTest, firstAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(first) failed: %v", err)
	}
	if err := repo.UpsertTestBegin(ctx, secondTest, secondAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(second) failed: %v", err)
	}

	var tests []m.Test
	if err := repo.db.WithContext(ctx).Order("run_id asc").Find(&tests).Error; err != nil {
		t.Fatalf("list tests: %v", err)
	}
	if len(tests) != 2 {
		t.Fatalf("len(tests) = %d, want 2", len(tests))
	}
	if tests[0].RunID != "run-1" || tests[1].RunID != "run-2" {
		t.Fatalf("stored run ids = [%s %s], want [run-1 run-2]", tests[0].RunID, tests[1].RunID)
	}

	var attempts []m.TestAttempt
	if err := repo.db.WithContext(ctx).Order("run_id asc").Find(&attempts).Error; err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("len(attempts) = %d, want 2", len(attempts))
	}
	if attempts[0].RunID != "run-1" || attempts[1].RunID != "run-2" {
		t.Fatalf("stored attempt run ids = [%s %s], want [run-1 run-2]", attempts[0].RunID, attempts[1].RunID)
	}
}

func TestUpsertTestBeginKeepsSameRawTestIDInSeparateRunsWithoutExecutionID(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	firstSuiteID := "suite-run-1"
	secondSuiteID := "suite-run-2"
	start := time.Date(2026, 5, 5, 12, 30, 0, 0, time.UTC)

	for _, test := range []*m.Test{
		{ID: "test-123", RunID: "run-1", ExternalTestID: "test-123", SuiteID: &firstSuiteID, Name: "Run 1 Test", Title: "Run 1 Test", Status: "RUNNING", StartTime: &start},
		{ID: "test-123", RunID: "run-2", ExternalTestID: "test-123", SuiteID: &secondSuiteID, Name: "Run 2 Test", Title: "Run 2 Test", Status: "RUNNING", StartTime: &start},
	} {
		attempt := &m.TestAttempt{
			RunID:        test.RunID,
			ExecutionID:  "",
			TestID:       test.ID,
			AttemptIndex: 0,
			Status:       "RUNNING",
			StartTime:    test.StartTime,
		}
		if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
			t.Fatalf("UpsertTestBegin(%s) failed: %v", test.RunID, err)
		}
	}

	var attempts []m.TestAttempt
	if err := repo.db.WithContext(ctx).Order("run_id asc").Find(&attempts).Error; err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("len(attempts) = %d, want 2", len(attempts))
	}
}

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

func TestUpsertTestBeginCreatesPlaceholderSuiteWhenMissing(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	runID := "run-123"
	suiteID := "suite-missing"
	start := time.Date(2026, 5, 6, 3, 0, 0, 0, time.UTC)

	test := &m.Test{
		ID:             "test-123",
		RunID:          runID,
		ExternalTestID: "test-123",
		SuiteID:        &suiteID,
		Name:           "My Test",
		Title:          "My Test",
		Status:         "RUNNING",
		StartTime:      &start,
	}
	attempt := &m.TestAttempt{RunID: runID, TestID: "test-123", AttemptIndex: 0, Status: "RUNNING", StartTime: &start}

	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	var suite m.Suite
	if err := repo.db.WithContext(ctx).Where("run_id = ? AND id = ?", runID, suiteID).First(&suite).Error; err != nil {
		t.Fatalf("load placeholder suite: %v", err)
	}
	if suite.ExternalSuiteID != suiteID {
		t.Fatalf("suite.ExternalSuiteID = %q, want %q", suite.ExternalSuiteID, suiteID)
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
	if storedTest.Status != "FLAKY" {
		t.Fatalf("stored test status = %q, want FLAKY", storedTest.Status)
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
	stepsRaw := stepPayload(t, []*m.StepDocument{{ID: "step-1", Title: "Click", Status: "PASSED"}})
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
	attempts := []m.TestAttempt{
		{AttemptIndex: 0, CreatedAt: time.Now(), Status: "FAILED"},
		{AttemptIndex: 1, CreatedAt: time.Now().Add(time.Second * 100), Status: "PASSED"},
	}
	if got := aggregateTestAttemptStatuses(attempts); got != "FLAKY" {
		t.Fatalf("aggregateTestAttemptStatuses() = %q, want FLAKY", got)
	}
	if got := aggregateTestAttemptStatuses([]m.TestAttempt{{AttemptIndex: 0, CreatedAt: time.Now(), Status: "FAILED"}}); got != "FAILED" {
		t.Fatalf("aggregateTestAttemptStatuses(single failure) = %q, want FAILED", got)
	}
}

func TestLatestExecutionAttemptSet_SelectsMostRecentExecution(t *testing.T) {
	earlier := time.Date(2026, 4, 29, 4, 0, 0, 0, time.UTC)
	later := earlier.Add(10 * time.Second)
	attempts := []m.TestAttempt{
		{ExecutionID: "exec-a", AttemptIndex: 0, Status: "PASSED", UpdatedAt: earlier},
		{ExecutionID: "exec-b", AttemptIndex: 0, Status: "FAILED", UpdatedAt: later},
	}

	selected, executionID := latestExecutionAttemptSet(attempts)
	if executionID != "exec-b" {
		t.Fatalf("executionID = %q, want exec-b", executionID)
	}
	if len(selected) != 1 || selected[0].ExecutionID != "exec-b" {
		t.Fatalf("selected attempts = %+v, want exec-b only", selected)
	}
}

func TestFinalizeTestEnd_LaterExecutionOverridesEarlierExecutionStatus(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "run-123:suite:suite-123"
	firstStart := time.Date(2026, 4, 29, 4, 10, 0, 0, time.UTC)
	firstEnd := firstStart.Add(2 * time.Second)
	secondStart := firstStart.Add(10 * time.Second)
	secondEnd := secondStart.Add(2 * time.Second)

	firstTest := &m.Test{
		ID:             "run-123:test:test-execution-aware",
		RunID:          "run-123",
		ExternalTestID: "test-execution-aware",
		SuiteID:        &suiteID,
		Name:           "Execution Aware",
		Title:          "Execution Aware",
		Status:         "PASSED",
		StartTime:      &firstStart,
		EndTime:        &firstEnd,
		RetryCount:     int32Ptr(0),
		RetryIndex:     int32Ptr(0),
	}
	firstAttempt := &m.TestAttempt{
		ID:           firstTest.ID + ":execution:exec-a:attempt:0",
		RunID:        firstTest.RunID,
		ExecutionID:  "exec-a",
		TestID:       firstTest.ID,
		AttemptIndex: 0,
		Status:       "PASSED",
		StartTime:    &firstStart,
		EndTime:      &firstEnd,
	}
	if err := repo.UpsertTestBegin(ctx, firstTest, firstAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(exec-a) failed: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, firstTest, firstAttempt); err != nil {
		t.Fatalf("FinalizeTestEnd(exec-a) failed: %v", err)
	}

	secondTest := &m.Test{
		ID:             firstTest.ID,
		RunID:          firstTest.RunID,
		ExternalTestID: firstTest.ExternalTestID,
		SuiteID:        &suiteID,
		Name:           firstTest.Name,
		Title:          firstTest.Title,
		Status:         "FAILED",
		StartTime:      &secondStart,
		EndTime:        &secondEnd,
		RetryCount:     int32Ptr(0),
		RetryIndex:     int32Ptr(0),
	}
	secondAttempt := &m.TestAttempt{
		ID:           secondTest.ID + ":execution:exec-b:attempt:0",
		RunID:        secondTest.RunID,
		ExecutionID:  "exec-b",
		TestID:       secondTest.ID,
		AttemptIndex: 0,
		Status:       "FAILED",
		StartTime:    &secondStart,
		EndTime:      &secondEnd,
	}
	if err := repo.UpsertTestBegin(ctx, secondTest, secondAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(exec-b) failed: %v", err)
	}
	if err := repo.FinalizeTestEnd(ctx, secondTest, secondAttempt); err != nil {
		t.Fatalf("FinalizeTestEnd(exec-b) failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", firstTest.ID).Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.Status != "FAILED" {
		t.Fatalf("storedTest.Status = %q, want FAILED", storedTest.Status)
	}
	if storedTest.EndTime == nil || !storedTest.EndTime.Equal(secondEnd) {
		t.Fatalf("storedTest.EndTime = %v, want %v", storedTest.EndTime, secondEnd)
	}
}

func TestGetRun_HydratesLatestExecutionStatusFromAttempts(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	start := time.Date(2026, 4, 29, 4, 20, 0, 0, time.UTC)
	firstEnd := start.Add(2 * time.Second)
	secondStart := start.Add(10 * time.Second)
	secondEnd := secondStart.Add(2 * time.Second)
	rootSuiteID := "run-123:suite:root"
	testID := "run-123:test:test-hydrated"

	if err := repo.db.WithContext(ctx).Create(&m.TestRun{ID: "run-123", Name: "Run 123", Status: "FAILED", CreatedAt: start, UpdatedAt: secondEnd}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := repo.db.WithContext(ctx).Create(&m.Suite{ID: rootSuiteID, RunID: "run-123", Name: "Root", CreatedAt: start, UpdatedAt: secondEnd}).Error; err != nil {
		t.Fatalf("seed suite: %v", err)
	}
	if err := repo.db.WithContext(ctx).Create(&m.Test{ID: testID, RunID: "run-123", ExternalTestID: "test-hydrated", SuiteID: &rootSuiteID, Name: "Hydrated", Title: "Hydrated", Status: "PASSED", CreatedAt: start, UpdatedAt: secondEnd}).Error; err != nil {
		t.Fatalf("seed test: %v", err)
	}
	for _, attempt := range []m.TestAttempt{
		{ID: testID + ":execution:exec-a:attempt:0", RunID: "run-123", ExecutionID: "exec-a", TestID: testID, AttemptIndex: 0, Status: "PASSED", StartTime: &start, EndTime: &firstEnd, CreatedAt: start, UpdatedAt: firstEnd},
		{ID: testID + ":execution:exec-b:attempt:0", RunID: "run-123", ExecutionID: "exec-b", TestID: testID, AttemptIndex: 0, Status: "FAILED", StartTime: &secondStart, EndTime: &secondEnd, CreatedAt: secondStart, UpdatedAt: secondEnd},
	} {
		if err := repo.db.WithContext(ctx).Create(&attempt).Error; err != nil {
			t.Fatalf("seed attempt %s: %v", attempt.ID, err)
		}
	}

	doc, err := repo.GetRun(ctx, "run-123", false)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if doc == nil || len(doc.Suites) != 1 || len(doc.Suites[0].Tests) != 1 {
		t.Fatalf("unexpected run payload: %+v", doc)
	}
	if got := doc.Suites[0].Tests[0].Status; got != "FAILED" {
		t.Fatalf("hydrated test status = %q, want FAILED", got)
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
		ExecutionID:  "",
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
	if err := repo.AppendTestFailure(ctx, "run-123", "", "test-123", 0, failure); err != nil {
		t.Fatalf("AppendTestFailure failed: %v", err)
	}

	errorTime := start.Add(2 * time.Second)
	errorDoc := &m.TestErrorDocument{
		ErrorMessage: "stderr line",
		StackTrace:   "error stack",
		Timestamp:    &errorTime,
		Attachments:  []map[string]interface{}{{"name": "error.txt"}},
	}
	if err := repo.AppendTestError(ctx, "run-123", "", "test-123", 0, errorDoc); err != nil {
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

func TestUpsertTestBeginSeparatesAttemptsByExecutionID(t *testing.T) {
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
	}
	firstAttempt := &m.TestAttempt{ID: "run-123:test:test-123:execution:exec-a:attempt:0", RunID: "run-123", ExecutionID: "exec-a", TestID: "run-123:test:test-123", AttemptIndex: 0, Status: "RUNNING", StartTime: &start}
	secondAttempt := &m.TestAttempt{ID: "run-123:test:test-123:execution:exec-b:attempt:0", RunID: "run-123", ExecutionID: "exec-b", TestID: "run-123:test:test-123", AttemptIndex: 0, Status: "RUNNING", StartTime: &start}

	if err := repo.UpsertTestBegin(ctx, test, firstAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(firstAttempt) failed: %v", err)
	}
	if err := repo.UpsertTestBegin(ctx, test, secondAttempt); err != nil {
		t.Fatalf("UpsertTestBegin(secondAttempt) failed: %v", err)
	}

	var count int64
	if err := repo.db.WithContext(ctx).Model(&m.TestAttempt{}).Where("test_id = ?", test.ID).Count(&count).Error; err != nil {
		t.Fatalf("count execution-scoped attempts: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func int64Ptr(value int64) *int64 {
	converted := value
	return &converted
}

func strPtr(value string) *string {
	converted := value
	return &converted
}

func stepPayload(t *testing.T, steps []*m.StepDocument) *m.Step {
	t.Helper()
	payload, err := m.StepFromDocuments(steps)
	if err != nil {
		t.Fatalf("marshal step payload: %v", err)
	}
	return payload
}

func TestGetRuns_PreservesHistoricalRunsForRepeatedExternalTestID(t *testing.T) {
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

	docs, _, err := repo.GetRuns(ctx, ListRunsFilter{}, 10, 0, true)
	if err != nil {
		t.Fatalf("GetRuns failed: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 run documents, got %d", len(docs))
	}
	if docs[0].ID != "run-2" || docs[1].ID != "run-1" {
		t.Fatalf("expected runs ordered newest-first, got %q then %q", docs[0].ID, docs[1].ID)
	}
	if docs[0].Name != "Run 2" || docs[1].Name != "Run 1" {
		t.Fatalf("expected run names to be preserved, got %q and %q", docs[0].Name, docs[1].Name)
	}
	if docs[0].Status != "FAILED" || docs[1].Status != "PASSED" {
		t.Fatalf("expected run statuses to be preserved, got %q and %q", docs[0].Status, docs[1].Status)
	}

	trends, err := repo.GetTestTrends(ctx, "test-123", 10)
	if err != nil {
		t.Fatalf("GetTestTrends failed: %v", err)
	}
	if len(trends) != 2 {
		t.Fatalf("expected 2 trend rows, got %d", len(trends))
	}
	if trends[0].RunID != "run-2" || trends[1].RunID != "run-1" {
		t.Fatalf("expected historical trend rows for both runs, got %+v", trends)
	}
}

func TestGetRuns_AssociatesRawSuiteAndTestIDsPerRun(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	start := time.Date(2026, 5, 5, 13, 0, 0, 0, time.UTC)

	for _, run := range []m.TestRun{
		{ID: "run-1", Name: "Run 1", Status: "PASSED", CreatedAt: start, UpdatedAt: start},
		{ID: "run-2", Name: "Run 2", Status: "FAILED", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)},
	} {
		if err := repo.db.WithContext(ctx).Create(&run).Error; err != nil {
			t.Fatalf("seed run %s: %v", run.ID, err)
		}
	}

	for _, suite := range []m.Suite{
		{ID: "suite-123", RunID: "run-1", ExternalSuiteID: "suite-123", Name: "Suite 1", CreatedAt: start, UpdatedAt: start},
		{ID: "suite-123", RunID: "run-2", ExternalSuiteID: "suite-123", Name: "Suite 2", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)},
	} {
		if err := repo.db.WithContext(ctx).Create(&suite).Error; err != nil {
			t.Fatalf("seed suite %s/%s: %v", suite.RunID, suite.ID, err)
		}
	}

	for _, test := range []m.Test{
		{ID: "test-123", RunID: "run-1", ExternalTestID: "test-123", SuiteID: strPtr("suite-123"), Name: "Test 1", Title: "Test 1", Status: "PASSED", CreatedAt: start, UpdatedAt: start},
		{ID: "test-123", RunID: "run-2", ExternalTestID: "test-123", SuiteID: strPtr("suite-123"), Name: "Test 2", Title: "Test 2", Status: "FAILED", CreatedAt: start.Add(time.Minute), UpdatedAt: start.Add(time.Minute)},
	} {
		if err := repo.db.WithContext(ctx).Create(&test).Error; err != nil {
			t.Fatalf("seed test %s/%s: %v", test.RunID, test.ID, err)
		}
	}

	docs, _, err := repo.GetRuns(ctx, ListRunsFilter{}, 10, 0, false)
	if err != nil {
		t.Fatalf("GetRuns failed: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("len(docs) = %d, want 2", len(docs))
	}

	for _, doc := range docs {
		if len(doc.Suites) != 1 {
			t.Fatalf("run %s suites = %+v, want exactly 1 suite", doc.ID, doc.Suites)
		}
		if len(doc.Suites[0].Tests) != 1 {
			t.Fatalf("run %s suite tests = %+v, want exactly 1 test", doc.ID, doc.Suites[0].Tests)
		}
		if doc.Suites[0].RunID != doc.ID {
			t.Fatalf("run %s suite run_id = %s, want %s", doc.ID, doc.Suites[0].RunID, doc.ID)
		}
		if doc.Suites[0].Tests[0].RunID != doc.ID {
			t.Fatalf("run %s test run_id = %s, want %s", doc.ID, doc.Suites[0].Tests[0].RunID, doc.ID)
		}
	}
}

func TestGetRun_PopulatesNestedSuitesTestsAndAttempts(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	childSuiteID := "run-123:suite:child"
	rootSuiteID := "run-123:suite:root"

	run := &m.TestRun{ID: "run-123", Name: "Run 123", Status: "FAILED", CreatedAt: start, UpdatedAt: start}
	if err := repo.db.WithContext(ctx).Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}

	rootSuite := &m.Suite{ID: rootSuiteID, RunID: run.ID, Name: "Root Suite", CreatedAt: start, UpdatedAt: start}
	childSuite := &m.Suite{ID: childSuiteID, RunID: run.ID, ParentSuiteID: &rootSuiteID, Name: "Child Suite", CreatedAt: start.Add(time.Second), UpdatedAt: start.Add(time.Second)}
	for _, suite := range []*m.Suite{rootSuite, childSuite} {
		if err := repo.db.WithContext(ctx).Create(suite).Error; err != nil {
			t.Fatalf("seed suite %s: %v", suite.ID, err)
		}
	}

	rootTest := &m.Test{ID: "run-123:test:root", RunID: run.ID, SuiteID: &rootSuiteID, Name: "Root Test", Title: "Root Test", Status: "PASSED", CreatedAt: start, UpdatedAt: start}
	nestedTest := &m.Test{ID: "run-123:test:nested", RunID: run.ID, SuiteID: &childSuiteID, Name: "Nested Test", Title: "Nested Test", Status: "FAILED", CreatedAt: start.Add(2 * time.Second), UpdatedAt: start.Add(2 * time.Second)}
	for _, test := range []*m.Test{rootTest, nestedTest} {
		if err := repo.db.WithContext(ctx).Create(test).Error; err != nil {
			t.Fatalf("seed test %s: %v", test.ID, err)
		}
	}

	attemptSteps := []*m.StepDocument{{ID: "step-1", Title: "Step 1", Status: "FAILED", CreatedAt: start, UpdatedAt: start}}
	attempt := m.TestAttempt{
		ID:           nestedTest.ID + ":0",
		RunID:        run.ID,
		TestID:       nestedTest.ID,
		AttemptIndex: 0,
		Status:       "FAILED",
		Steps:        stepPayload(t, attemptSteps),
		CreatedAt:    start,
		UpdatedAt:    start,
	}
	if err := repo.db.WithContext(ctx).Create(&attempt).Error; err != nil {
		t.Fatalf("seed attempt: %v", err)
	}

	doc, err := repo.GetRun(ctx, run.ID, true)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if doc == nil {
		t.Fatal("expected run document, got nil")
	}
	if len(doc.Suites) != 1 {
		t.Fatalf("len(doc.Suites) = %d, want 1", len(doc.Suites))
	}
	if len(doc.Tests) != 0 {
		t.Fatalf("doc.Tests = %+v, want no run-level tests", doc.Tests)
	}
	if len(doc.Suites[0].Tests) != 1 || doc.Suites[0].Tests[0].ID != rootTest.ID {
		t.Fatalf("root suite tests = %+v, want root test %s", doc.Suites[0].Tests, rootTest.ID)
	}
	if len(doc.Suites[0].Suites) != 1 || doc.Suites[0].Suites[0].ID != childSuiteID {
		t.Fatalf("root suite children = %+v, want child suite %s", doc.Suites[0].Suites, childSuiteID)
	}
	if len(doc.Suites[0].Suites[0].Tests) != 1 || doc.Suites[0].Suites[0].Tests[0].ID != nestedTest.ID {
		t.Fatalf("child suite tests = %+v, want nested test %s", doc.Suites[0].Suites[0].Tests, nestedTest.ID)
	}
	if len(doc.Suites[0].Suites[0].Tests[0].Attempts) != 1 {
		t.Fatalf("nested test attempts = %+v, want 1 attempt", doc.Suites[0].Suites[0].Tests[0].Attempts)
	}
}

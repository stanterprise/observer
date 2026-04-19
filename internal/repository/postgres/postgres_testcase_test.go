package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func TestUpsertTestBeginCreatesTestAndAttempt(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	test := &m.Test{
		ID:         "test-123",
		RunID:      "run-123",
		SuiteID:    &suiteID,
		Name:       "My Test",
		Title:      "My Test",
		Status:     "RUNNING",
		StartTime:  &start,
		Metadata:   map[string]interface{}{"browser": "chromium"},
		RetryCount: int32Ptr(2),
		RetryIndex: int32Ptr(0),
		Timeout:    int32Ptr(30000),
	}
	attempt := &m.TestAttempt{
		ID:           "test-123:0",
		RunID:        "run-123",
		TestID:       "test-123",
		AttemptIndex: 0,
		Status:       "RUNNING",
		StartTime:    &start,
		Attachments:  []map[string]interface{}{{"name": "stdout.txt"}},
	}

	if err := repo.UpsertTestBegin(ctx, test, attempt); err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "test-123").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.Status != "RUNNING" {
		t.Fatalf("stored test status = %q, want RUNNING", storedTest.Status)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("stored suite id = %v, want %s", storedTest.SuiteID, suiteID)
	}

	var storedAttempt m.TestAttempt
	if err := repo.db.WithContext(ctx).Where("test_id = ? AND attempt_index = ?", "test-123", 0).First(&storedAttempt).Error; err != nil {
		t.Fatalf("load stored attempt: %v", err)
	}
	if storedAttempt.Status != "RUNNING" {
		t.Fatalf("stored attempt status = %q, want RUNNING", storedAttempt.Status)
	}
	if len(storedAttempt.Attachments) != 1 || storedAttempt.Attachments[0]["name"] != "stdout.txt" {
		t.Fatalf("stored attempt attachments = %+v, want stdout.txt", storedAttempt.Attachments)
	}
}

func TestFinalizeTestEndAggregatesPassingRetries(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	suiteID := "suite-123"
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	firstEnd := start.Add(2 * time.Second)
	secondStart := start.Add(3 * time.Second)
	secondEnd := start.Add(5 * time.Second)

	firstTest := &m.Test{
		ID:         "test-123",
		RunID:      "run-123",
		SuiteID:    &suiteID,
		Name:       "My Test",
		Title:      "My Test",
		Status:     "FAILED",
		StartTime:  &start,
		EndTime:    &firstEnd,
		Duration:   int64Ptr(int64((2 * time.Second).Nanoseconds())),
		RetryCount: int32Ptr(2),
		RetryIndex: int32Ptr(0),
	}
	firstAttempt := &m.TestAttempt{
		ID:           "test-123:0",
		RunID:        "run-123",
		TestID:       "test-123",
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
		ID:         "test-123",
		RunID:      "run-123",
		SuiteID:    &suiteID,
		Name:       "My Test",
		Title:      "My Test",
		Status:     "PASSED",
		StartTime:  &secondStart,
		EndTime:    &secondEnd,
		Duration:   int64Ptr(int64((2 * time.Second).Nanoseconds())),
		RetryCount: int32Ptr(2),
		RetryIndex: int32Ptr(1),
	}
	secondAttempt := &m.TestAttempt{
		ID:           "test-123:1",
		RunID:        "run-123",
		TestID:       "test-123",
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
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "test-123").Error; err != nil {
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
	if err := repo.db.WithContext(ctx).Where("test_id = ?", "test-123").Order("attempt_index asc").Find(&storedAttempts).Error; err != nil {
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

func TestAggregateTestAttemptStatuses(t *testing.T) {
	attempts := []m.TestAttempt{{AttemptIndex: 0, Status: "FAILED"}, {AttemptIndex: 1, Status: "PASSED"}}
	if got := aggregateTestAttemptStatuses(attempts, "FAILED"); got != "PASSED" {
		t.Fatalf("aggregateTestAttemptStatuses() = %q, want PASSED", got)
	}
	if got := aggregateTestAttemptStatuses([]m.TestAttempt{{AttemptIndex: 0, Status: "FAILED"}}, "FAILED"); got != "FAILED" {
		t.Fatalf("aggregateTestAttemptStatuses(single failure) = %q, want FAILED", got)
	}
}

func int64Ptr(value int64) *int64 {
	converted := value
	return &converted
}

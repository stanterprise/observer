package mongodb

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func insertRunDocument(t *testing.T, repo *MongoRepository, runID string, tests []*m.TestDocument) {
	t.Helper()
	ctx := context.Background()
	doc := &m.TestRunDocument{
		ID:        runID,
		Name:      "Test Run",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests:     tests,
		Suites:    []*m.SuiteDocument{},
	}
	if _, err := repo.collection.InsertOne(ctx, doc); err != nil {
		t.Fatalf("insertRunDocument failed: %v", err)
	}
}

func int32Ptr(value int32) *int32 {
	v := value
	return &v
}

func fetchActiveTestStepBuffer(t *testing.T, repo *MongoRepository, runID, testID string) bson.M {
	t.Helper()
	ctx := context.Background()

	var doc bson.M
	if err := repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc); err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	active, ok := doc["active_test_steps"].(bson.M)
	if !ok {
		return nil
	}
	buffer, _ := active[stepBufferKey(testID)].(bson.M)
	return buffer
}

func fetchActiveTestSteps(t *testing.T, repo *MongoRepository, runID, testID string) []bson.M {
	t.Helper()
	buffer := fetchActiveTestStepBuffer(t, repo, runID, testID)
	if buffer == nil {
		return nil
	}
	rawSteps, ok := buffer["steps"].(bson.A)
	if !ok {
		return nil
	}
	steps := make([]bson.M, 0, len(rawSteps))
	for _, raw := range rawSteps {
		step, ok := raw.(bson.M)
		if ok {
			steps = append(steps, step)
		}
	}
	return steps
}

func TestUpsertStepBegin_BuffersStepInActiveRunDocument(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-buffered"
	testID := "test-step-buffered"

	insertRunDocument(t, repo, runID, []*m.TestDocument{{
		ID:         testID,
		Title:      "Buffered Test",
		Status:     "RUNNING",
		RetryIndex: int32Ptr(0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}})

	if err := repo.SyncActiveTestSteps(ctx, runID, testID, 0, nil); err != nil {
		t.Fatalf("SyncActiveTestSteps failed: %v", err)
	}

	startTime := time.Now()
	step := &m.StepDocument{
		ID:        "step-001",
		Title:     "Click button",
		Status:    "RUNNING",
		StartTime: &startTime,
	}

	if err := repo.UpsertStepBegin(ctx, runID, step, testID, 0); err != nil {
		t.Fatalf("UpsertStepBegin failed: %v", err)
	}

	buffer := fetchActiveTestStepBuffer(t, repo, runID, testID)
	if buffer == nil {
		t.Fatal("expected active step buffer to exist")
	}
	if buffer["retry_index"] != int32(0) {
		t.Fatalf("expected retry_index=0, got %v", buffer["retry_index"])
	}
	if buffer["status"] != activeStepBufferStatusActive {
		t.Fatalf("expected status=%q, got %v", activeStepBufferStatusActive, buffer["status"])
	}
	if buffer["first_event_at"] == nil {
		t.Fatal("expected first_event_at to be set")
	}
	if buffer["last_event_at"] == nil {
		t.Fatal("expected last_event_at to be set")
	}
	if buffer["ttl_at"] == nil {
		t.Fatal("expected ttl_at to be set")
	}
	steps := fetchActiveTestSteps(t, repo, runID, testID)
	if len(steps) != 1 {
		t.Fatalf("expected 1 buffered step, got %d", len(steps))
	}
	if steps[0]["id"] != "step-001" {
		t.Errorf("expected step id 'step-001', got %v", steps[0]["id"])
	}
}

func TestSyncActiveTestSteps_DuplicateBeginPreservesBufferedSteps(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-duplicate-begin"
	testID := "test-step-duplicate-begin"

	insertRunDocument(t, repo, runID, []*m.TestDocument{{
		ID:         testID,
		Title:      "Buffered Test",
		Status:     "RUNNING",
		RetryIndex: int32Ptr(0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}})

	if err := repo.SyncActiveTestSteps(ctx, runID, testID, 0, nil); err != nil {
		t.Fatalf("first SyncActiveTestSteps failed: %v", err)
	}

	step := &m.StepDocument{ID: "step-001", Title: "Click", Status: "RUNNING", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.UpsertStepBegin(ctx, runID, step, testID, 0); err != nil {
		t.Fatalf("UpsertStepBegin failed: %v", err)
	}

	if err := repo.SyncActiveTestSteps(ctx, runID, testID, 0, nil); err != nil {
		t.Fatalf("duplicate SyncActiveTestSteps failed: %v", err)
	}

	steps := fetchActiveTestSteps(t, repo, runID, testID)
	if len(steps) != 1 {
		t.Fatalf("expected duplicate begin to preserve 1 buffered step, got %d", len(steps))
	}
}

func TestSyncActiveTestSteps_NewRetryReplacesBufferedSteps(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-new-retry"
	testID := "test-step-new-retry"

	insertRunDocument(t, repo, runID, []*m.TestDocument{{
		ID:         testID,
		Title:      "Buffered Test",
		Status:     "RUNNING",
		RetryIndex: int32Ptr(1),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}})

	if err := repo.SyncActiveTestSteps(ctx, runID, testID, 0, nil); err != nil {
		t.Fatalf("SyncActiveTestSteps retry 0 failed: %v", err)
	}
	if err := repo.UpsertStepBegin(ctx, runID, &m.StepDocument{ID: "step-retry-0", Title: "First", Status: "RUNNING", CreatedAt: time.Now(), UpdatedAt: time.Now()}, testID, 0); err != nil {
		t.Fatalf("UpsertStepBegin retry 0 failed: %v", err)
	}

	if err := repo.SyncActiveTestSteps(ctx, runID, testID, 1, nil); err != nil {
		t.Fatalf("SyncActiveTestSteps retry 1 failed: %v", err)
	}

	buffer := fetchActiveTestStepBuffer(t, repo, runID, testID)
	if buffer == nil {
		t.Fatal("expected active step buffer to exist for retry 1")
	}
	if buffer["retry_index"] != int32(1) {
		t.Fatalf("expected retry_index=1 after reset, got %v", buffer["retry_index"])
	}
	if buffer["ttl_at"] == nil {
		t.Fatal("expected ttl_at to be set after retry reset")
	}
	steps := fetchActiveTestSteps(t, repo, runID, testID)
	if len(steps) != 0 {
		t.Fatalf("expected retry 1 buffer to start empty, got %d steps", len(steps))
	}
}

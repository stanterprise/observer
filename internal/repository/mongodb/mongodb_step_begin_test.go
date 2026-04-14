package mongodb

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// insertRunDocument inserts a minimal run document with the provided tests.
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

// insertRunDocumentRawEmptyAttempts inserts a run document using raw BSON so that the
// test's attempts field is explicitly stored as an empty array [].
// The Go model would omit it via omitempty, which would cause a MongoDB "path must exist"
// error instead of the expected ModifiedCount==0 when attempting a nested update.
func insertRunDocumentRawEmptyAttempts(t *testing.T, repo *MongoRepository, runID, testID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	doc := bson.D{
		{Key: "_id", Value: runID},
		{Key: "name", Value: "Test Run"},
		{Key: "status", Value: "running"},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "suites", Value: bson.A{}},
		{Key: "tests", Value: bson.A{
			bson.D{
				{Key: "id", Value: testID},
				{Key: "title", Value: "Test Without Attempts"},
				{Key: "status", Value: "RUNNING"},
				{Key: "created_at", Value: now},
				{Key: "updated_at", Value: now},
				{Key: "attempts", Value: bson.A{}}, // present but empty
			},
		}},
	}
	if _, err := repo.collection.InsertOne(ctx, doc); err != nil {
		t.Fatalf("insertRunDocumentRawEmptyAttempts failed: %v", err)
	}
}

// fetchTestAttempts retrieves the attempts array for a specific test in a run document.
func fetchTestAttempts(t *testing.T, repo *MongoRepository, runID, testID string) []bson.M {
	t.Helper()
	ctx := context.Background()

	var doc bson.M
	if err := repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc); err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	tests, ok := doc["tests"].(bson.A)
	if !ok {
		return nil
	}
	for _, rawTest := range tests {
		test, ok := rawTest.(bson.M)
		if !ok {
			continue
		}
		if test["id"] == testID {
			if attempts, ok := test["attempts"].(bson.A); ok {
				result := make([]bson.M, 0, len(attempts))
				for _, a := range attempts {
					if attempt, ok := a.(bson.M); ok {
						result = append(result, attempt)
					}
				}
				return result
			}
			return nil
		}
	}
	return nil
}

// fetchAttemptSteps retrieves the steps array for a specific attempt in a test.
func fetchAttemptSteps(t *testing.T, repo *MongoRepository, runID, testID string, retryIndex int32) []bson.M {
	t.Helper()
	ctx := context.Background()

	var doc bson.M
	if err := repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc); err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	tests, ok := doc["tests"].(bson.A)
	if !ok {
		return nil
	}
	for _, rawTest := range tests {
		test, ok := rawTest.(bson.M)
		if !ok {
			continue
		}
		if test["id"] != testID {
			continue
		}
		attempts, ok := test["attempts"].(bson.A)
		if !ok {
			return nil
		}
		for _, rawAttempt := range attempts {
			attempt, ok := rawAttempt.(bson.M)
			if !ok {
				continue
			}
			if ri, ok := attempt["retry_index"].(int32); ok && ri == retryIndex {
				if steps, ok := attempt["steps"].(bson.A); ok {
					result := make([]bson.M, 0, len(steps))
					for _, s := range steps {
						if step, ok := s.(bson.M); ok {
							result = append(result, step)
						}
					}
					return result
				}
				return nil
			}
		}
	}
	return nil
}

// TestUpsertStepBegin_MissingAttempt verifies that UpsertStepBegin creates the missing
// attempt via ensureAttemptExists when the attempts array exists but has no entry for
// the given retry_index, then successfully inserts the step.
// The document is inserted via raw BSON to keep attempts: [] explicit in the document
// (the Go model's omitempty would suppress it, causing a "path must exist" error instead).
func TestUpsertStepBegin_MissingAttempt(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-missing-attempt"
	testID := "test-step-missing-attempt"

	// Insert with attempts: [] stored explicitly so the nested $[attempt] positional
	// filter returns ModifiedCount==0 (no matching attempt) rather than an error.
	insertRunDocumentRawEmptyAttempts(t, repo, runID, testID)

	startTime := time.Now()
	step := &m.StepDocument{
		ID:        "step-001",
		Title:     "Click button",
		Status:    "RUNNING",
		StartTime: &startTime,
	}

	// UpsertStepBegin should detect ModifiedCount==0 (no matching attempt[0]),
	// call ensureAttemptExists to create it, and then insert the step successfully.
	err := repo.UpsertStepBegin(ctx, runID, step, testID, 0)
	if err != nil {
		t.Fatalf("UpsertStepBegin failed: %v", err)
	}

	// Verify exactly one attempt was created (no duplicates).
	attempts := fetchTestAttempts(t, repo, runID, testID)
	if len(attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(attempts))
	}

	// Verify the step was inserted into the attempt.
	steps := fetchAttemptSteps(t, repo, runID, testID, 0)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step in attempt, got %d", len(steps))
	}
	if steps[0]["id"] != "step-001" {
		t.Errorf("expected step id 'step-001', got %v", steps[0]["id"])
	}
}

// TestUpsertStepBegin_MissingAttempt_NoDuplicates verifies that calling
// ensureAttemptExists multiple times for the same retry_index is idempotent
// and does not create duplicate attempts.
func TestUpsertStepBegin_MissingAttempt_NoDuplicates(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-no-dup-attempt"
	testID := "test-step-no-dup-attempt"

	insertRunDocumentRawEmptyAttempts(t, repo, runID, testID)

	now := time.Now()

	// Call ensureAttemptExists multiple times for the same retry_index.
	// The atomic pipeline update guarantees idempotency: each call is a no-op once
	// retry_index=0 already exists.
	for i := 0; i < 5; i++ {
		if err := repo.ensureAttemptExists(ctx, runID, testID, 0, &now, now); err != nil {
			t.Fatalf("ensureAttemptExists call %d failed: %v", i, err)
		}
	}

	// Verify exactly one attempt was created (no duplicates).
	attempts := fetchTestAttempts(t, repo, runID, testID)
	if len(attempts) != 1 {
		t.Errorf("expected 1 attempt after %d calls, got %d", 5, len(attempts))
	}
	if ri, ok := attempts[0]["retry_index"].(int32); !ok || ri != 0 {
		t.Errorf("expected retry_index=0, got %v", attempts[0]["retry_index"])
	}
}

// TestUpsertStepBegin_MissingAttempt_RetryScenario verifies the realistic retry scenario:
// a test has attempt[0] from the first run; UpsertStepBegin is called for retry_index=1
// (the second retry), which does not yet have an attempt entry.
// ensureAttemptExists should create attempt[1] atomically without duplicating attempt[0].
func TestUpsertStepBegin_MissingAttempt_RetryScenario(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-step-multi-retry"
	testID := "test-step-multi-retry"

	// Pre-populate test with attempt[0] already present (from the first run).
	retryIndex := int32(1)
	existingAttempt := &m.AttemptDocument{
		RetryIndex: 0,
		Steps:      []*m.StepDocument{},
		Status:     "FAILED",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	testDoc := &m.TestDocument{
		ID:         testID,
		Title:      "Test With Existing Attempt",
		Status:     "RUNNING",
		RetryIndex: &retryIndex,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Steps:      []*m.StepDocument{},
		Attempts:   []*m.AttemptDocument{existingAttempt},
	}
	insertRunDocument(t, repo, runID, []*m.TestDocument{testDoc})

	startTime := time.Now()
	step := &m.StepDocument{
		ID:        "step-retry-001",
		Title:     "Navigate to page",
		Status:    "RUNNING",
		StartTime: &startTime,
	}

	// UpsertStepBegin for retry_index=1 should detect ModifiedCount==0
	// (attempt[0] exists but attempt[1] does not), call ensureAttemptExists to create
	// attempt[1], and then insert the step successfully.
	err := repo.UpsertStepBegin(ctx, runID, step, testID, 1)
	if err != nil {
		t.Fatalf("UpsertStepBegin failed: %v", err)
	}

	// Verify exactly 2 attempts exist (attempt[0] unchanged, attempt[1] created).
	attempts := fetchTestAttempts(t, repo, runID, testID)
	if len(attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(attempts))
	}

	// Verify attempt[0] is intact and attempt[1] was created correctly.
	var found0, found1 bool
	for _, a := range attempts {
		switch a["retry_index"].(int32) {
		case 0:
			found0 = true
			if status, ok := a["status"].(string); !ok || status != "FAILED" {
				t.Errorf("attempt[0] status should be FAILED, got %v", a["status"])
			}
		case 1:
			found1 = true
			if status, ok := a["status"].(string); !ok || status != "RUNNING" {
				t.Errorf("attempt[1] status should be RUNNING, got %v", a["status"])
			}
		}
	}
	if !found0 {
		t.Error("attempt with retry_index=0 not found")
	}
	if !found1 {
		t.Error("attempt with retry_index=1 not found")
	}

	// Verify the step was inserted into attempt[1] only.
	steps1 := fetchAttemptSteps(t, repo, runID, testID, 1)
	if len(steps1) != 1 {
		t.Fatalf("expected 1 step in attempt[1], got %d", len(steps1))
	}
	if steps1[0]["id"] != "step-retry-001" {
		t.Errorf("expected step id 'step-retry-001', got %v", steps1[0]["id"])
	}

	// Verify attempt[0] has no steps.
	steps0 := fetchAttemptSteps(t, repo, runID, testID, 0)
	if len(steps0) != 0 {
		t.Errorf("expected 0 steps in attempt[0], got %d", len(steps0))
	}

	// Calling ensureAttemptExists for retry_index=0 again should be idempotent
	// (attempt[0] already exists; no duplicate should be created).
	now := time.Now()
	if err := repo.ensureAttemptExists(ctx, runID, testID, 0, &now, now); err != nil {
		t.Fatalf("ensureAttemptExists (idempotent) failed: %v", err)
	}
	attempts = fetchTestAttempts(t, repo, runID, testID)
	if len(attempts) != 2 {
		t.Errorf("expected still 2 attempts after idempotent call, got %d", len(attempts))
	}
}

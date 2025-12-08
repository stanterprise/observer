package repository

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// TestUpsertSuiteBegin_UpdateExisting verifies that calling UpsertSuiteBegin
// with the same ID updates the existing suite instead of creating duplicates
func TestUpsertSuiteBegin_UpdateExisting(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite first time
	rootSuite := &m.SuiteDocument{
		ID:          "suite-upsert-1",
		Name:        "Original Name",
		Description: "Original Description",
		Status:      "RUNNING",
		ProjectName: "test-project",
	}
	err := repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("First UpsertSuiteBegin failed: %v", err)
	}

	// Call UpsertSuiteBegin again with updated data
	rootSuite.Name = "Updated Name"
	rootSuite.Description = "Updated Description"
	rootSuite.Status = "PASSED"
	err = repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("Second UpsertSuiteBegin failed: %v", err)
	}

	// Verify document was updated, not duplicated
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-upsert-1"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if doc.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", doc.Name)
	}
	if doc.Description != "Updated Description" {
		t.Errorf("Description = %v, want Updated Description", doc.Description)
	}
	if doc.Status != "PASSED" {
		t.Errorf("Status = %v, want PASSED", doc.Status)
	}

	// Verify only one document exists
	count, err := testCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count != 1 {
		t.Errorf("Document count = %v, want 1", count)
	}
}

// TestUpsertSuiteBegin_NestedSuiteUpdateExisting verifies that calling
// UpsertSuiteBegin for a nested suite updates it instead of appending duplicates
func TestUpsertSuiteBegin_NestedSuiteUpdateExisting(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	rootSuite := &m.SuiteDocument{
		ID:     "suite-root-upsert-2",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Create nested suite first time
	nestedSuite := &m.SuiteDocument{
		ID:          "suite-nested-upsert-1",
		Name:        "Original Nested",
		Description: "Original",
		Status:      "RUNNING",
	}
	err = repo.UpsertSuiteBegin(ctx, nestedSuite, "suite-root-upsert-2")
	if err != nil {
		t.Fatalf("First UpsertSuiteBegin for nested suite failed: %v", err)
	}

	// Update nested suite
	nestedSuite.Name = "Updated Nested"
	nestedSuite.Description = "Updated"
	nestedSuite.Status = "PASSED"
	err = repo.UpsertSuiteBegin(ctx, nestedSuite, "suite-root-upsert-2")
	if err != nil {
		t.Fatalf("Second UpsertSuiteBegin for nested suite failed: %v", err)
	}

	// Verify nested suite was updated, not duplicated
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-upsert-2"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Suites) != 1 {
		t.Fatalf("Suites length = %v, want 1 (no duplicates)", len(doc.Suites))
	}

	nested := doc.Suites[0]
	if nested.Name != "Updated Nested" {
		t.Errorf("Nested suite Name = %v, want Updated Nested", nested.Name)
	}
	if nested.Description != "Updated" {
		t.Errorf("Nested suite Description = %v, want Updated", nested.Description)
	}
	if nested.Status != "PASSED" {
		t.Errorf("Nested suite Status = %v, want PASSED", nested.Status)
	}
}

// TestUpsertTestBegin_UpdateExisting verifies that calling UpsertTestBegin
// with the same ID updates the existing test instead of creating duplicates
func TestUpsertTestBegin_UpdateExisting(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	suite := &m.SuiteDocument{
		ID:     "suite-root-upsert-3",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Add test first time
	test := &m.TestDocument{
		ID:     "test-upsert-1",
		Title:  "Original Title",
		Status: "RUNNING",
	}
	err = repo.UpsertTestBegin(ctx, test, "suite-root-upsert-3")
	if err != nil {
		t.Fatalf("First UpsertTestBegin failed: %v", err)
	}

	// Update test
	duration := int64(1000000)
	test.Title = "Updated Title"
	test.Status = "PASSED"
	test.Duration = &duration
	err = repo.UpsertTestBegin(ctx, test, "suite-root-upsert-3")
	if err != nil {
		t.Fatalf("Second UpsertTestBegin failed: %v", err)
	}

	// Verify test was updated, not duplicated
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-upsert-3"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 1 {
		t.Fatalf("Tests length = %v, want 1 (no duplicates)", len(doc.Tests))
	}

	updatedTest := doc.Tests[0]
	if updatedTest.Title != "Updated Title" {
		t.Errorf("Test Title = %v, want Updated Title", updatedTest.Title)
	}
	if updatedTest.Status != "PASSED" {
		t.Errorf("Test Status = %v, want PASSED", updatedTest.Status)
	}
	if updatedTest.Duration == nil || *updatedTest.Duration != duration {
		t.Errorf("Test Duration = %v, want %v", updatedTest.Duration, duration)
	}
}

// TestUpsertStepBegin_UpdateExisting verifies that calling UpsertStepBegin
// with the same ID updates the existing step instead of creating duplicates
func TestUpsertStepBegin_UpdateExisting(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite and test
	suite := &m.SuiteDocument{
		ID:     "suite-root-upsert-4",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	test := &m.TestDocument{
		ID:     "test-upsert-2",
		Title:  "Test with steps",
		Status: "RUNNING",
	}
	err = repo.UpsertTestBegin(ctx, test, "suite-root-upsert-4")
	if err != nil {
		t.Fatalf("Failed to create test: %v", err)
	}

	// Add step first time
	step := &m.StepDocument{
		ID:       "step-upsert-1",
		Status:   "RUNNING",
		Category: "action",
		Title:    "Original Step",
	}
	err = repo.UpsertStepBegin(ctx, step, "test-upsert-2", "")
	if err != nil {
		t.Fatalf("First UpsertStepBegin failed: %v", err)
	}

	// Give a small delay to ensure updated_at timestamp differs
	time.Sleep(10 * time.Millisecond)

	// Update step
	step.Status = "PASSED"
	step.Category = "assertion"
	step.Title = "Updated Step"
	err = repo.UpsertStepBegin(ctx, step, "test-upsert-2", "")
	if err != nil {
		t.Fatalf("Second UpsertStepBegin failed: %v", err)
	}

	// Verify step was updated, not duplicated
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-upsert-4"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 1 {
		t.Fatalf("Tests length = %v, want 1", len(doc.Tests))
	}

	if len(doc.Tests[0].Steps) != 1 {
		t.Fatalf("Steps length = %v, want 1 (no duplicates)", len(doc.Tests[0].Steps))
	}

	updatedStep := doc.Tests[0].Steps[0]
	if updatedStep.Title != "Updated Step" {
		t.Errorf("Step Title = %v, want Updated Step", updatedStep.Title)
	}
	if updatedStep.Status != "PASSED" {
		t.Errorf("Step Status = %v, want PASSED", updatedStep.Status)
	}
	if updatedStep.Category != "assertion" {
		t.Errorf("Step Category = %v, want assertion", updatedStep.Category)
	}
}

// TestUpsertFlow_CompleteScenario tests a realistic flow where events may be
// replayed or arrive out of order, ensuring upsert behavior prevents duplicates
func TestUpsertFlow_CompleteScenario(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	suite := &m.SuiteDocument{
		ID:     "suite-scenario",
		Name:   "Scenario Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create suite: %v", err)
	}

	// Add test
	test := &m.TestDocument{
		ID:     "test-scenario",
		Title:  "Scenario Test",
		Status: "RUNNING",
	}
	err = repo.UpsertTestBegin(ctx, test, "suite-scenario")
	if err != nil {
		t.Fatalf("Failed to create test: %v", err)
	}

	// Simulate event replay - send same test begin event again
	err = repo.UpsertTestBegin(ctx, test, "suite-scenario")
	if err != nil {
		t.Fatalf("Replayed test begin failed: %v", err)
	}

	// Add step
	step := &m.StepDocument{
		ID:       "step-scenario",
		Status:   "RUNNING",
		Category: "action",
		Title:    "Step 1",
	}
	err = repo.UpsertStepBegin(ctx, step, "test-scenario", "")
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	// Simulate event replay - send same step begin event again
	err = repo.UpsertStepBegin(ctx, step, "test-scenario", "")
	if err != nil {
		t.Fatalf("Replayed step begin failed: %v", err)
	}

	// End step (twice to simulate replay)
	err = repo.UpsertStepEnd(ctx, "step-scenario", "PASSED")
	if err != nil {
		t.Fatalf("First step end failed: %v", err)
	}
	err = repo.UpsertStepEnd(ctx, "step-scenario", "PASSED")
	if err != nil {
		t.Fatalf("Replayed step end failed: %v", err)
	}

	// End test (twice to simulate replay)
	duration := int64(1000000)
	err = repo.UpsertTestEnd(ctx, "test-scenario", "PASSED", &duration)
	if err != nil {
		t.Fatalf("First test end failed: %v", err)
	}
	err = repo.UpsertTestEnd(ctx, "test-scenario", "PASSED", &duration)
	if err != nil {
		t.Fatalf("Replayed test end failed: %v", err)
	}

	// End suite (twice to simulate replay)
	now := time.Now()
	suiteDuration := int64(5000000)
	err = repo.UpsertSuiteEnd(ctx, "suite-scenario", "PASSED", &now, &suiteDuration)
	if err != nil {
		t.Fatalf("First suite end failed: %v", err)
	}
	err = repo.UpsertSuiteEnd(ctx, "suite-scenario", "PASSED", &now, &suiteDuration)
	if err != nil {
		t.Fatalf("Replayed suite end failed: %v", err)
	}

	// Verify final state - should have exactly 1 suite, 1 test, 1 step
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-scenario"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	// Verify counts
	if len(doc.Tests) != 1 {
		t.Errorf("Tests count = %v, want 1", len(doc.Tests))
	}
	if len(doc.Tests[0].Steps) != 1 {
		t.Errorf("Steps count = %v, want 1", len(doc.Tests[0].Steps))
	}

	// Verify final statuses
	if doc.Status != "PASSED" {
		t.Errorf("Suite status = %v, want PASSED", doc.Status)
	}
	if doc.Tests[0].Status != "PASSED" {
		t.Errorf("Test status = %v, want PASSED", doc.Tests[0].Status)
	}
	if doc.Tests[0].Steps[0].Status != "PASSED" {
		t.Errorf("Step status = %v, want PASSED", doc.Tests[0].Steps[0].Status)
	}
}

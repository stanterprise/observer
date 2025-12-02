package models

import (
	"testing"
	"time"

	"gorm.io/datatypes"
)

func TestTestRunDocument_Fields(t *testing.T) {
	now := time.Now()
	doc := &TestRunDocument{
		ID:          "run-123",
		Name:        "Test Run",
		Description: "A test run",
		Status:      "PASSED",
		Metadata:    map[string]interface{}{"key": "value"},
		ProjectName: "test-project",
		StartTime:   &now,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tests:       []*TestDocument{},
		Suites:      []*SuiteDocument{},
	}

	if doc.ID != "run-123" {
		t.Errorf("ID = %v, want run-123", doc.ID)
	}
	if doc.Name != "Test Run" {
		t.Errorf("Name = %v, want Test Run", doc.Name)
	}
	if doc.Status != "PASSED" {
		t.Errorf("Status = %v, want PASSED", doc.Status)
	}
	if doc.ProjectName != "test-project" {
		t.Errorf("ProjectName = %v, want test-project", doc.ProjectName)
	}
}

func TestSuiteDocument_Fields(t *testing.T) {
	now := time.Now()
	suite := &SuiteDocument{
		ID:            "suite-123",
		ParentSuiteID: "parent-123",
		Name:          "Test Suite",
		Status:        "RUNNING",
		Metadata:      map[string]interface{}{"env": "test"},
		CreatedAt:     now,
		UpdatedAt:     now,
		Tests:         []*TestDocument{},
		Suites:        []*SuiteDocument{},
	}

	if suite.ID != "suite-123" {
		t.Errorf("ID = %v, want suite-123", suite.ID)
	}
	if suite.ParentSuiteID != "parent-123" {
		t.Errorf("ParentSuiteID = %v, want parent-123", suite.ParentSuiteID)
	}
	if suite.Name != "Test Suite" {
		t.Errorf("Name = %v, want Test Suite", suite.Name)
	}
}

func TestTestDocument_Fields(t *testing.T) {
	now := time.Now()
	duration := int64(1000000000)
	retryCount := int32(3)
	retryIndex := int32(0)
	timeout := int32(30000)

	test := &TestDocument{
		ID:         "test-123",
		RunID:      "run-456",
		SuiteID:    "suite-789",
		Title:      "Test Case",
		Status:     "PASSED",
		Metadata:   map[string]interface{}{"tag": "smoke"},
		Duration:   &duration,
		RetryCount: &retryCount,
		RetryIndex: &retryIndex,
		Timeout:    &timeout,
		CreatedAt:  now,
		UpdatedAt:  now,
		Steps:      []*StepDocument{},
	}

	if test.ID != "test-123" {
		t.Errorf("ID = %v, want test-123", test.ID)
	}
	if test.Title != "Test Case" {
		t.Errorf("Title = %v, want Test Case", test.Title)
	}
	if *test.Duration != 1000000000 {
		t.Errorf("Duration = %v, want 1000000000", *test.Duration)
	}
	if *test.RetryCount != 3 {
		t.Errorf("RetryCount = %v, want 3", *test.RetryCount)
	}
}

func TestStepDocument_Fields(t *testing.T) {
	now := time.Now()
	step := &StepDocument{
		ID:            "step-123",
		RunID:         "run-456",
		TestCaseRunID: "test-789",
		ParentStepID:  "",
		Status:        "PASSED",
		Category:      "assertion",
		Title:         "Check result",
		CreatedAt:     now,
		UpdatedAt:     now,
		Steps:         []*StepDocument{},
	}

	if step.ID != "step-123" {
		t.Errorf("ID = %v, want step-123", step.ID)
	}
	if step.Category != "assertion" {
		t.Errorf("Category = %v, want assertion", step.Category)
	}
	if step.Title != "Check result" {
		t.Errorf("Title = %v, want Check result", step.Title)
	}
}

func TestConvertTestSuiteRunToDocument(t *testing.T) {
	now := time.Now()
	duration := int64(5000000000)

	suite := &TestSuiteRun{
		ID:              "suite-123",
		Name:            "Test Suite",
		Description:     "Suite description",
		Status:          "PASSED",
		Metadata:        datatypes.JSONMap{"key": "value"},
		Duration:        &duration,
		TestSuiteSpecID: "spec-123",
		InitiatedBy:     "user@example.com",
		ProjectName:     "my-project",
		StartTime:       &now,
		EndTime:         &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	doc := ConvertTestSuiteRunToDocument(suite)

	if doc == nil {
		t.Fatal("ConvertTestSuiteRunToDocument() returned nil")
	}
	if doc.ID != "suite-123" {
		t.Errorf("ID = %v, want suite-123", doc.ID)
	}
	if doc.Name != "Test Suite" {
		t.Errorf("Name = %v, want Test Suite", doc.Name)
	}
	if doc.Status != "PASSED" {
		t.Errorf("Status = %v, want PASSED", doc.Status)
	}
	if doc.ProjectName != "my-project" {
		t.Errorf("ProjectName = %v, want my-project", doc.ProjectName)
	}
	if len(doc.Tests) != 0 {
		t.Errorf("Tests length = %v, want 0", len(doc.Tests))
	}
	if len(doc.Suites) != 0 {
		t.Errorf("Suites length = %v, want 0", len(doc.Suites))
	}
}

func TestConvertTestSuiteRunToDocument_Nil(t *testing.T) {
	doc := ConvertTestSuiteRunToDocument(nil)
	if doc != nil {
		t.Errorf("ConvertTestSuiteRunToDocument(nil) = %v, want nil", doc)
	}
}

func TestConvertTestCaseRunToDocument(t *testing.T) {
	now := time.Now()
	duration := int64(1000000000)
	retryCount := int32(3)
	retryIndex := int32(1)
	timeout := int32(60000)

	tc := &TestCaseRun{
		ID:         "test-123",
		RunID:      "run-456",
		Title:      "Test Case",
		Status:     "FAILED",
		Metadata:   datatypes.JSONMap{"browser": "chrome"},
		Duration:   &duration,
		RetryCount: &retryCount,
		RetryIndex: &retryIndex,
		Timeout:    &timeout,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	doc := ConvertTestCaseRunToDocument(tc)

	if doc == nil {
		t.Fatal("ConvertTestCaseRunToDocument() returned nil")
	}
	if doc.ID != "test-123" {
		t.Errorf("ID = %v, want test-123", doc.ID)
	}
	if doc.Title != "Test Case" {
		t.Errorf("Title = %v, want Test Case", doc.Title)
	}
	if doc.Status != "FAILED" {
		t.Errorf("Status = %v, want FAILED", doc.Status)
	}
	if *doc.RetryIndex != 1 {
		t.Errorf("RetryIndex = %v, want 1", *doc.RetryIndex)
	}
	if len(doc.Steps) != 0 {
		t.Errorf("Steps length = %v, want 0", len(doc.Steps))
	}
}

func TestConvertTestCaseRunToDocument_Nil(t *testing.T) {
	doc := ConvertTestCaseRunToDocument(nil)
	if doc != nil {
		t.Errorf("ConvertTestCaseRunToDocument(nil) = %v, want nil", doc)
	}
}

func TestConvertStepRunToDocument(t *testing.T) {
	now := time.Now()

	step := &StepRun{
		ID:            "step-123",
		RunID:         "run-456",
		TestCaseRunID: "test-789",
		ParentStepID:  "parent-step-123",
		Status:        "PASSED",
		Category:      "click",
		Title:         "Click button",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	doc := ConvertStepRunToDocument(step)

	if doc == nil {
		t.Fatal("ConvertStepRunToDocument() returned nil")
	}
	if doc.ID != "step-123" {
		t.Errorf("ID = %v, want step-123", doc.ID)
	}
	if doc.ParentStepID != "parent-step-123" {
		t.Errorf("ParentStepID = %v, want parent-step-123", doc.ParentStepID)
	}
	if doc.Category != "click" {
		t.Errorf("Category = %v, want click", doc.Category)
	}
	if len(doc.Steps) != 0 {
		t.Errorf("Steps length = %v, want 0", len(doc.Steps))
	}
}

func TestConvertStepRunToDocument_Nil(t *testing.T) {
	doc := ConvertStepRunToDocument(nil)
	if doc != nil {
		t.Errorf("ConvertStepRunToDocument(nil) = %v, want nil", doc)
	}
}

func TestTestRunDocument_EmbeddedStructures(t *testing.T) {
	now := time.Now()

	// Create a nested structure
	step := &StepDocument{
		ID:            "step-1",
		TestCaseRunID: "test-1",
		Status:        "PASSED",
		Title:         "Click login",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	test := &TestDocument{
		ID:        "test-1",
		RunID:     "run-1",
		Title:     "Login test",
		Status:    "PASSED",
		CreatedAt: now,
		UpdatedAt: now,
		Steps:     []*StepDocument{step},
	}

	suite := &SuiteDocument{
		ID:        "suite-1",
		Name:      "Auth Suite",
		CreatedAt: now,
		UpdatedAt: now,
		Tests:     []*TestDocument{test},
	}

	doc := &TestRunDocument{
		ID:        "run-1",
		Name:      "Full Test Run",
		CreatedAt: now,
		UpdatedAt: now,
		Suites:    []*SuiteDocument{suite},
		Tests:     []*TestDocument{}, // Empty root tests
	}

	// Verify nested access
	if len(doc.Suites) != 1 {
		t.Fatalf("Suites length = %v, want 1", len(doc.Suites))
	}
	if len(doc.Suites[0].Tests) != 1 {
		t.Fatalf("Suite tests length = %v, want 1", len(doc.Suites[0].Tests))
	}
	if len(doc.Suites[0].Tests[0].Steps) != 1 {
		t.Fatalf("Test steps length = %v, want 1", len(doc.Suites[0].Tests[0].Steps))
	}
	if doc.Suites[0].Tests[0].Steps[0].Title != "Click login" {
		t.Errorf("Step title = %v, want Click login", doc.Suites[0].Tests[0].Steps[0].Title)
	}
}

func TestTestSuiteRun_TableName(t *testing.T) {
	ts := TestSuiteRun{}
	expected := "test_suite_runs"
	if got := ts.TableName(); got != expected {
		t.Errorf("TableName() = %v, want %v", got, expected)
	}
}

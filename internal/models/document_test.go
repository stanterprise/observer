package models

import (
	"testing"
	"time"
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
	if suite.Name != "Test Suite" {
		t.Errorf("Name = %v, want Test Suite", suite.Name)
	}
	if suite.ParentSuiteID != "parent-123" {
		t.Errorf("ParentSuiteID = %v, want parent-123", suite.ParentSuiteID)
	}
}

func TestTestDocument_Fields(t *testing.T) {
	now := time.Now()
	test := &TestDocument{
		ID:        "test-123",
		RunID:     "run-456",
		SuiteID:   "suite-789",
		Title:     "Test Case",
		Status:    "PASSED",
		Metadata:  map[string]interface{}{"browser": "chrome"},
		CreatedAt: now,
		UpdatedAt: now,
		Steps:     []*StepDocument{},
	}

	if test.ID != "test-123" {
		t.Errorf("ID = %v, want test-123", test.ID)
	}
	if test.Title != "Test Case" {
		t.Errorf("Title = %v, want Test Case", test.Title)
	}
	if test.RunID != "run-456" {
		t.Errorf("RunID = %v, want run-456", test.RunID)
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
		Tests:     []*TestDocument{},
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

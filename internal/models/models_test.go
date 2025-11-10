package models

import (
	"testing"
	"time"

	"gorm.io/datatypes"
)

func TestTestCaseRun_TableName(t *testing.T) {
	tc := TestCaseRun{}
	expected := "test_case_runs"
	if got := tc.TableName(); got != expected {
		t.Errorf("TableName() = %v, want %v", got, expected)
	}
}

func TestStepRun_TableName(t *testing.T) {
	sr := StepRun{}
	expected := "step_runs"
	if got := sr.TableName(); got != expected {
		t.Errorf("TableName() = %v, want %v", got, expected)
	}
}

func TestTestCaseRun_Fields(t *testing.T) {
	now := time.Now()
	metadata := datatypes.JSONMap{
		"key1": "value1",
		"key2": "value2",
	}

	tc := TestCaseRun{
		ID:        "test-123",
		RunID:     "run-456",
		Title:     "Test Case Title",
		Status:    "PASSED",
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if tc.ID != "test-123" {
		t.Errorf("ID = %v, want test-123", tc.ID)
	}
	if tc.RunID != "run-456" {
		t.Errorf("RunID = %v, want run-456", tc.RunID)
	}
	if tc.Title != "Test Case Title" {
		t.Errorf("Title = %v, want Test Case Title", tc.Title)
	}
	if tc.Status != "PASSED" {
		t.Errorf("Status = %v, want PASSED", tc.Status)
	}
	if len(tc.Metadata) != 2 {
		t.Errorf("Metadata length = %v, want 2", len(tc.Metadata))
	}
}

func TestStepRun_Fields(t *testing.T) {
	now := time.Now()

	sr := StepRun{
		ID:            "step-123",
		RunID:         "run-456",
		TestCaseRunID: "test-789",
		Status:        "RUNNING",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if sr.ID != "step-123" {
		t.Errorf("ID = %v, want step-123", sr.ID)
	}
	if sr.RunID != "run-456" {
		t.Errorf("RunID = %v, want run-456", sr.RunID)
	}
	if sr.TestCaseRunID != "test-789" {
		t.Errorf("TestCaseRunID = %v, want test-789", sr.TestCaseRunID)
	}
	if sr.Status != "RUNNING" {
		t.Errorf("Status = %v, want RUNNING", sr.Status)
	}
}

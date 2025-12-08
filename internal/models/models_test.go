package models

import (
	"testing"
	"time"
)

// Note: TestTestDocument_Fields is in document_test.go to avoid duplication

func TestTestDocument_BasicFields(t *testing.T) {
	now := time.Now()
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	tc := TestDocument{
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
		ID:            "step-789",
		RunID:         "run-456",
		TestCaseRunID: "test-123",
		Status:        "RUNNING",
		Category:      "action",
		Title:         "Step Title",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if sr.ID != "step-789" {
		t.Errorf("ID = %v, want step-789", sr.ID)
	}
	if sr.RunID != "run-456" {
		t.Errorf("RunID = %v, want run-456", sr.RunID)
	}
	if sr.TestCaseRunID != "test-123" {
		t.Errorf("TestCaseRunID = %v, want test-123", sr.TestCaseRunID)
	}
	if sr.Status != "RUNNING" {
		t.Errorf("Status = %v, want RUNNING", sr.Status)
	}
	if sr.Category != "action" {
		t.Errorf("Category = %v, want action", sr.Category)
	}
}

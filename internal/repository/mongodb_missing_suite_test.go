package repository

import (
	"context"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// TestUpsertTestBegin_MissingSuite verifies that when a test references a suite
// that doesn't exist, it falls back to storing the test in the root suite
func TestUpsertTestBegin_MissingSuite(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	rootSuite := &m.SuiteDocument{
		ID:     "root-suite-123-suite-root",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Try to add test to a non-existent nested suite
	// The suite ID follows the pattern: {base-id}-suite-{path}
	test := &m.TestDocument{
		ID:     "test-orphan-1",
		Title:  "Test in Missing Suite",
		Status: "RUNNING",
	}

	missingSuiteID := "root-suite-123-suite-/some/nested/path"
	err = repo.UpsertTestBegin(ctx, test, missingSuiteID)
	if err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	// Verify test was stored in root suite as fallback
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "root-suite-123-suite-root"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 1 {
		t.Fatalf("Tests length = %v, want 1", len(doc.Tests))
	}

	if doc.Tests[0].ID != "test-orphan-1" {
		t.Errorf("Test ID = %v, want test-orphan-1", doc.Tests[0].ID)
	}

	if doc.Tests[0].Title != "Test in Missing Suite" {
		t.Errorf("Test Title = %v, want Test in Missing Suite", doc.Tests[0].Title)
	}
}

// TestExtractRootSuiteID verifies the root suite ID extraction logic
func TestExtractRootSuiteID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "root suite ID unchanged",
			input:    "abc123-suite-root",
			expected: "abc123-suite-root",
		},
		{
			name:     "nested suite extracts root",
			input:    "abc123-suite-/chromium/test.spec.ts",
			expected: "abc123-suite-root",
		},
		{
			name:     "deeply nested suite",
			input:    "abc123-suite-/chromium/api/v1/test.spec.ts/Suite Name",
			expected: "abc123-suite-root",
		},
		{
			name:     "ID without suite marker",
			input:    "abc123",
			expected: "abc123-suite-root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRootSuiteID(tt.input)
			if result != tt.expected {
				t.Errorf("extractRootSuiteID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestUpsertTestBegin_MultipleTestsMissingSuite verifies that multiple tests
// referencing different missing suites all get stored in the same root suite
func TestUpsertTestBegin_MultipleTestsMissingSuite(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	rootSuite := &m.SuiteDocument{
		ID:     "root-multi-123-suite-root",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Add tests referencing different missing nested suites
	testCases := []struct {
		testID  string
		suiteID string
		title   string
	}{
		{"test-1", "root-multi-123-suite-/path/suite1", "Test 1"},
		{"test-2", "root-multi-123-suite-/path/suite2", "Test 2"},
		{"test-3", "root-multi-123-suite-/path/suite1/nested", "Test 3"},
	}

	for _, tc := range testCases {
		test := &m.TestDocument{
			ID:     tc.testID,
			Title:  tc.title,
			Status: "RUNNING",
		}
		err = repo.UpsertTestBegin(ctx, test, tc.suiteID)
		if err != nil {
			t.Fatalf("UpsertTestBegin failed for %s: %v", tc.testID, err)
		}
	}

	// Verify all tests were stored in root suite
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "root-multi-123-suite-root"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 3 {
		t.Fatalf("Tests length = %v, want 3", len(doc.Tests))
	}

	// Verify all test IDs are present
	testIDs := make(map[string]bool)
	for _, test := range doc.Tests {
		testIDs[test.ID] = true
	}

	for _, tc := range testCases {
		if !testIDs[tc.testID] {
			t.Errorf("Test ID %s not found in root suite", tc.testID)
		}
	}
}

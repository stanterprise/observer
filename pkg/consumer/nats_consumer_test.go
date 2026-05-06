package consumer

import (
	"errors"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
)

func TestShouldDeferStepEvent(t *testing.T) {
	consumer := &NATSConsumer{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "parent test not found is deferred",
			err:  errors.New("parent test not found: runID=run-1 testID=test-1"),
			want: true,
		},
		{
			name: "active step buffer not found is deferred",
			err:  errors.New("active step buffer not found: runID=run-1, executionID=exec-1, testID=test-1, retryIndex=0"),
			want: true,
		},
		{
			name: "missing step in active buffer is deferred",
			err:  errors.New("step not found in active buffer: runID=run-1, executionID=exec-1, stepID=step-1, testID=test-1, retryIndex=0"),
			want: true,
		},
		{
			name: "unrelated errors are not deferred",
			err:  errors.New("unmarshal step begin event: invalid JSON"),
			want: false,
		},
		{
			name: "nil error is not deferred",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := consumer.shouldDeferStepEvent(tt.err)
			if got != tt.want {
				t.Fatalf("shouldDeferStepEvent(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestEnsureRelationalTestSuiteIDUsesRawSuiteID(t *testing.T) {
	relationalTest := &m.Test{ID: "test-123", RunID: "run-123"}

	ensureRelationalTestSuiteID(relationalTest, "run-123", "suite-123")

	if relationalTest.SuiteID == nil {
		t.Fatal("SuiteID was not assigned")
	}
	if *relationalTest.SuiteID != "suite-123" {
		t.Fatalf("SuiteID = %q, want suite-123", *relationalTest.SuiteID)
	}
}

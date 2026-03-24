package consumer

import (
	"encoding/json"
	"testing"
)

func TestExtractRunID(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		wantID string
	}{
		{
			name:   "run.start top-level run_id",
			data:   `{"run_id":"run-abc","name":"My Run"}`,
			wantID: "run-abc",
		},
		{
			name:   "run.end top-level run_id",
			data:   `{"run_id":"run-xyz","final_status":"PASSED"}`,
			wantID: "run-xyz",
		},
		{
			name:   "test.failure top-level run_id",
			data:   `{"run_id":"run-fail","test_id":"t1","failure_message":"boom"}`,
			wantID: "run-fail",
		},
		{
			name:   "stdout top-level run_id",
			data:   `{"run_id":"run-stdout","test_id":"t2","message":"hello"}`,
			wantID: "run-stdout",
		},
		{
			name:   "suite.begin nested suite.run_id",
			data:   `{"suite":{"id":"s1","run_id":"run-suite","name":"Suite"}}`,
			wantID: "run-suite",
		},
		{
			name:   "suite.end nested suite.run_id",
			data:   `{"suite":{"id":"s1","run_id":"run-suite-end","status":"PASSED"}}`,
			wantID: "run-suite-end",
		},
		{
			name:   "test.begin nested test_case.run_id",
			data:   `{"test_case":{"id":"t1","run_id":"run-test","name":"Test"}}`,
			wantID: "run-test",
		},
		{
			name:   "test.end nested test_case.run_id",
			data:   `{"test_case":{"id":"t1","run_id":"run-test-end","status":"PASSED"}}`,
			wantID: "run-test-end",
		},
		{
			name:   "step.begin nested step.run_id",
			data:   `{"step":{"id":"step1","run_id":"run-step","title":"Click"}}`,
			wantID: "run-step",
		},
		{
			name:   "step.end nested step.run_id",
			data:   `{"step":{"id":"step1","run_id":"run-step-end","status":"PASSED"}}`,
			wantID: "run-step-end",
		},
		{
			name:   "empty object returns empty string",
			data:   `{}`,
			wantID: "",
		},
		{
			name:   "invalid JSON returns empty string",
			data:   `not json`,
			wantID: "",
		},
		{
			name:   "run_id empty string treated as absent",
			data:   `{"run_id":""}`,
			wantID: "",
		},
		{
			name:   "heartbeat with no run_id returns empty string",
			data:   `{"source_id":"worker-1"}`,
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRunID(json.RawMessage(tt.data))
			if got != tt.wantID {
				t.Errorf("extractRunID(%q) = %q, want %q", tt.data, got, tt.wantID)
			}
		})
	}
}

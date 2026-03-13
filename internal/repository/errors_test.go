package repository

import (
	"errors"
	"testing"
)

func TestErrParentNotFound_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ErrParentNotFound
		expected string
	}{
		{
			name: "test parent missing for step",
			err: &ErrParentNotFound{
				ParentType: "test",
				ParentID:   "test-123",
				ChildType:  "step",
				ChildID:    "step-456",
			},
			expected: "test parent not found: parentID=test-123 (for step=step-456)",
		},
		{
			name: "suite parent missing for suite",
			err: &ErrParentNotFound{
				ParentType: "suite",
				ParentID:   "suite-abc",
				ChildType:  "suite",
				ChildID:    "suite-def",
			},
			expected: "suite parent not found: parentID=suite-abc (for suite=suite-def)",
		},
		{
			name: "run parent missing for test",
			err: &ErrParentNotFound{
				ParentType: "run",
				ParentID:   "run-xyz",
				ChildType:  "test",
				ChildID:    "test-789",
			},
			expected: "run parent not found: parentID=run-xyz (for test=test-789)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrParentNotFound_IsRetryable(t *testing.T) {
	err := &ErrParentNotFound{
		ParentType: "test",
		ParentID:   "test-123",
		ChildType:  "step",
		ChildID:    "step-456",
	}
	if !err.IsRetryable() {
		t.Error("ErrParentNotFound.IsRetryable() should return true")
	}
}

func TestErrParentNotFound_ErrorsAs(t *testing.T) {
	original := &ErrParentNotFound{
		ParentType: "test",
		ParentID:   "test-123",
		ChildType:  "step",
		ChildID:    "step-456",
	}

	var target *ErrParentNotFound
	if !errors.As(original, &target) {
		t.Fatal("errors.As should match *ErrParentNotFound")
	}
	if target.ParentType != "test" {
		t.Errorf("ParentType = %q, want %q", target.ParentType, "test")
	}
	if target.ParentID != "test-123" {
		t.Errorf("ParentID = %q, want %q", target.ParentID, "test-123")
	}
	if target.ChildType != "step" {
		t.Errorf("ChildType = %q, want %q", target.ChildType, "step")
	}
	if target.ChildID != "step-456" {
		t.Errorf("ChildID = %q, want %q", target.ChildID, "step-456")
	}
}

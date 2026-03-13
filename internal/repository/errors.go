package repository

import "fmt"

// ErrParentNotFound indicates that a parent entity (test, suite, attempt, or run)
// does not exist yet in the database for a child entity being persisted.
//
// This error is retryable: it typically indicates an out-of-order event delivery
// (e.g. StepBegin arrived before TestBegin), and the parent is expected to appear
// once its own event is processed.
type ErrParentNotFound struct {
	ParentType string // "test", "suite", "attempt", "run"
	ParentID   string
	ChildType  string // "step", "test", "suite"
	ChildID    string
}

func (e *ErrParentNotFound) Error() string {
	return fmt.Sprintf("%s parent not found: parentID=%s (for %s=%s)",
		e.ParentType, e.ParentID, e.ChildType, e.ChildID)
}

// IsRetryable returns true because a missing parent usually means the parent
// event has not been processed yet; NAK-ing with backoff will allow the parent
// event to be persisted first.
func (e *ErrParentNotFound) IsRetryable() bool {
	return true
}

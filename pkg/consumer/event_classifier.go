package consumer

import (
	"log/slog"
)

// EventClassification indicates how an event should be processed
type EventClassification string

const (
	// ClassifyImmediate means the event can be processed immediately
	ClassifyImmediate EventClassification = "immediate"
	// ClassifyBuffer means the event should be buffered for reconciliation
	ClassifyBuffer EventClassification = "buffer"
	// ClassifyReconcile means the event is a root suite end that triggers reconciliation
	ClassifyReconcile EventClassification = "reconcile"
)

// Classifier determines if an event can be immediately processed or needs buffering
type Classifier struct {
	logger *slog.Logger
}

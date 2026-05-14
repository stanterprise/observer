package consumer

import (
	"time"
)

// ReconciliationConfig holds configuration for event reconciliation
type ReconciliationConfig struct {
	// Enabled controls whether reconciliation is active
	Enabled bool

	// MaxBufferSize is the maximum number of events to buffer per run
	MaxBufferSize int

	// InactivityTTL is the duration after last event before triggering reconciliation
	InactivityTTL time.Duration

	// MaxPasses is the maximum number of reconciliation passes to attempt
	MaxPasses int

	// PassDelay is the delay between reconciliation passes
	PassDelay time.Duration

	// CleanupInterval is how often to check for inactive buffers
	CleanupInterval time.Duration
}

package consumer

import (
	"os"
	"strconv"
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

// DefaultReconciliationConfig returns configuration with sensible defaults
func DefaultReconciliationConfig() ReconciliationConfig {
	return ReconciliationConfig{
		Enabled:         false, // Disabled by default for gradual rollout
		MaxBufferSize:   10000,
		InactivityTTL:   5 * time.Minute,
		MaxPasses:       10,
		PassDelay:       100 * time.Millisecond,
		CleanupInterval: 1 * time.Minute,
	}
}

// LoadReconciliationConfigFromEnv loads reconciliation configuration from environment variables
// Falls back to defaults if environment variables are not set
func LoadReconciliationConfigFromEnv() ReconciliationConfig {
	cfg := DefaultReconciliationConfig()

	// RECONCILIATION_ENABLED
	if v := os.Getenv("RECONCILIATION_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Enabled = enabled
		}
	}

	// RECONCILIATION_MAX_BUFFER_SIZE
	if v := os.Getenv("RECONCILIATION_MAX_BUFFER_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil && size > 0 {
			cfg.MaxBufferSize = size
		}
	}

	// RECONCILIATION_INACTIVITY_TTL
	if v := os.Getenv("RECONCILIATION_INACTIVITY_TTL"); v != "" {
		if ttl, err := time.ParseDuration(v); err == nil && ttl > 0 {
			cfg.InactivityTTL = ttl
		}
	}

	// RECONCILIATION_MAX_PASSES
	if v := os.Getenv("RECONCILIATION_MAX_PASSES"); v != "" {
		if passes, err := strconv.Atoi(v); err == nil && passes > 0 {
			cfg.MaxPasses = passes
		}
	}

	// RECONCILIATION_PASS_DELAY
	if v := os.Getenv("RECONCILIATION_PASS_DELAY"); v != "" {
		if delay, err := time.ParseDuration(v); err == nil && delay > 0 {
			cfg.PassDelay = delay
		}
	}

	// RECONCILIATION_CLEANUP_INTERVAL
	if v := os.Getenv("RECONCILIATION_CLEANUP_INTERVAL"); v != "" {
		if interval, err := time.ParseDuration(v); err == nil && interval > 0 {
			cfg.CleanupInterval = interval
		}
	}

	return cfg
}

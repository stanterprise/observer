package consumer

import (
	"os"
	"testing"
	"time"
)

func TestLoadReconciliationConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(*testing.T, ReconciliationConfig)
	}{
		{
			name:    "defaults when no env vars set",
			envVars: map[string]string{},
		},
		{
			name: "enable reconciliation",
			envVars: map[string]string{
				"RECONCILIATION_ENABLED": "true",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if !cfg.Enabled {
					t.Error("Expected Enabled to be true")
				}
			},
		},
		{
			name: "custom buffer size",
			envVars: map[string]string{
				"RECONCILIATION_MAX_BUFFER_SIZE": "5000",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if cfg.MaxBufferSize != 5000 {
					t.Errorf("Expected MaxBufferSize 5000, got %d", cfg.MaxBufferSize)
				}
			},
		},
		{
			name: "custom inactivity TTL",
			envVars: map[string]string{
				"RECONCILIATION_INACTIVITY_TTL": "10m",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if cfg.InactivityTTL != 10*time.Minute {
					t.Errorf("Expected InactivityTTL 10m, got %v", cfg.InactivityTTL)
				}
			},
		},
		{
			name: "custom max passes",
			envVars: map[string]string{
				"RECONCILIATION_MAX_PASSES": "20",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if cfg.MaxPasses != 20 {
					t.Errorf("Expected MaxPasses 20, got %d", cfg.MaxPasses)
				}
			},
		},
		{
			name: "custom pass delay",
			envVars: map[string]string{
				"RECONCILIATION_PASS_DELAY": "200ms",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if cfg.PassDelay != 200*time.Millisecond {
					t.Errorf("Expected PassDelay 200ms, got %v", cfg.PassDelay)
				}
			},
		},
		{
			name: "custom cleanup interval",
			envVars: map[string]string{
				"RECONCILIATION_CLEANUP_INTERVAL": "30s",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if cfg.CleanupInterval != 30*time.Second {
					t.Errorf("Expected CleanupInterval 30s, got %v", cfg.CleanupInterval)
				}
			},
		},
		{
			name: "all custom settings",
			envVars: map[string]string{
				"RECONCILIATION_ENABLED":          "true",
				"RECONCILIATION_MAX_BUFFER_SIZE":  "8000",
				"RECONCILIATION_INACTIVITY_TTL":   "3m",
				"RECONCILIATION_MAX_PASSES":       "15",
				"RECONCILIATION_PASS_DELAY":       "50ms",
				"RECONCILIATION_CLEANUP_INTERVAL": "2m",
			},
			validate: func(t *testing.T, cfg ReconciliationConfig) {
				if !cfg.Enabled {
					t.Error("Expected Enabled to be true")
				}
				if cfg.MaxBufferSize != 8000 {
					t.Errorf("Expected MaxBufferSize 8000, got %d", cfg.MaxBufferSize)
				}
				if cfg.InactivityTTL != 3*time.Minute {
					t.Errorf("Expected InactivityTTL 3m, got %v", cfg.InactivityTTL)
				}
				if cfg.MaxPasses != 15 {
					t.Errorf("Expected MaxPasses 15, got %d", cfg.MaxPasses)
				}
				if cfg.PassDelay != 50*time.Millisecond {
					t.Errorf("Expected PassDelay 50ms, got %v", cfg.PassDelay)
				}
				if cfg.CleanupInterval != 2*time.Minute {
					t.Errorf("Expected CleanupInterval 2m, got %v", cfg.CleanupInterval)
				}
			},
		},
		{
			name: "invalid values fall back to defaults",
			envVars: map[string]string{
				"RECONCILIATION_ENABLED":          "not-a-bool",
				"RECONCILIATION_MAX_BUFFER_SIZE":  "invalid",
				"RECONCILIATION_INACTIVITY_TTL":   "invalid-duration",
				"RECONCILIATION_MAX_PASSES":       "not-a-number",
				"RECONCILIATION_PASS_DELAY":       "bad-duration",
				"RECONCILIATION_CLEANUP_INTERVAL": "wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			envKeys := []string{
				"RECONCILIATION_ENABLED",
				"RECONCILIATION_MAX_BUFFER_SIZE",
				"RECONCILIATION_INACTIVITY_TTL",
				"RECONCILIATION_MAX_PASSES",
				"RECONCILIATION_PASS_DELAY",
				"RECONCILIATION_CLEANUP_INTERVAL",
			}
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

package config

import (
	"os"
	"strconv"
	"time"
)

const (
	// DefaultStepBufferTTL is the default TTL for MongoDB live step buffer documents.
	DefaultStepBufferTTL = 15 * time.Minute

	// DefaultStepPayloadThreshold is the default size in bytes above which a step
	// payload is offloaded to object storage instead of stored inline in PostgreSQL JSONB.
	DefaultStepPayloadThreshold = 4 * 1024 * 1024 // ~4 MB
)

// StepBufferTTL reads the MONGO_STEP_BUFFER_TTL env var (Go duration string)
// and returns the configured TTL, or the default 15 minutes.
func StepBufferTTL() time.Duration {
	if v := os.Getenv("MONGO_STEP_BUFFER_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return DefaultStepBufferTTL
}

// StepPayloadThreshold reads the STEP_PAYLOAD_THRESHOLD env var (integer bytes)
// and returns the configured threshold, or the default ~4 MB.
func StepPayloadThreshold() int {
	if v := os.Getenv("STEP_PAYLOAD_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultStepPayloadThreshold
}

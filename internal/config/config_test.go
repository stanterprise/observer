package config

import (
	"os"
	"testing"
	"time"
)

func TestStepBufferTTL_Default(t *testing.T) {
	os.Unsetenv("MONGO_STEP_BUFFER_TTL")
	if got := StepBufferTTL(); got != DefaultStepBufferTTL {
		t.Errorf("StepBufferTTL() = %v, want %v", got, DefaultStepBufferTTL)
	}
}

func TestStepBufferTTL_Custom(t *testing.T) {
	os.Setenv("MONGO_STEP_BUFFER_TTL", "30m")
	defer os.Unsetenv("MONGO_STEP_BUFFER_TTL")

	if got := StepBufferTTL(); got != 30*time.Minute {
		t.Errorf("StepBufferTTL() = %v, want 30m", got)
	}
}

func TestStepBufferTTL_Invalid(t *testing.T) {
	os.Setenv("MONGO_STEP_BUFFER_TTL", "not-a-duration")
	defer os.Unsetenv("MONGO_STEP_BUFFER_TTL")

	if got := StepBufferTTL(); got != DefaultStepBufferTTL {
		t.Errorf("StepBufferTTL() with invalid value = %v, want default %v", got, DefaultStepBufferTTL)
	}
}

func TestStepPayloadThreshold_Default(t *testing.T) {
	os.Unsetenv("STEP_PAYLOAD_THRESHOLD")
	if got := StepPayloadThreshold(); got != DefaultStepPayloadThreshold {
		t.Errorf("StepPayloadThreshold() = %d, want %d", got, DefaultStepPayloadThreshold)
	}
}

func TestStepPayloadThreshold_Custom(t *testing.T) {
	os.Setenv("STEP_PAYLOAD_THRESHOLD", "8388608")
	defer os.Unsetenv("STEP_PAYLOAD_THRESHOLD")

	if got := StepPayloadThreshold(); got != 8388608 {
		t.Errorf("StepPayloadThreshold() = %d, want 8388608", got)
	}
}

func TestStepPayloadThreshold_Invalid(t *testing.T) {
	os.Setenv("STEP_PAYLOAD_THRESHOLD", "abc")
	defer os.Unsetenv("STEP_PAYLOAD_THRESHOLD")

	if got := StepPayloadThreshold(); got != DefaultStepPayloadThreshold {
		t.Errorf("StepPayloadThreshold() with invalid value = %d, want default %d", got, DefaultStepPayloadThreshold)
	}
}

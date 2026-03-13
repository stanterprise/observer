package consumer

import (
	"testing"
	"time"
)

func TestCalcNakDelay(t *testing.T) {
	tests := []struct {
		deliveryCount uint64
		expected      time.Duration
	}{
		// deliveryCount=0 is treated like the first attempt; returns 1s
		{deliveryCount: 0, expected: 1 * time.Second},
		// Exponential backoff: 1s, 2s, 4s, 8s, 16s then capped at 30s
		{deliveryCount: 1, expected: 1 * time.Second},
		{deliveryCount: 2, expected: 2 * time.Second},
		{deliveryCount: 3, expected: 4 * time.Second},
		{deliveryCount: 4, expected: 8 * time.Second},
		{deliveryCount: 5, expected: 16 * time.Second},
		// deliveryCount >= 6 should cap at 30s
		{deliveryCount: 6, expected: 30 * time.Second},
		{deliveryCount: 7, expected: 30 * time.Second},
		{deliveryCount: 10, expected: 30 * time.Second},
		{deliveryCount: 100, expected: 30 * time.Second},
	}

	for _, tt := range tests {
		got := calcNakDelay(tt.deliveryCount)
		if got != tt.expected {
			t.Errorf("calcNakDelay(%d) = %v, want %v", tt.deliveryCount, got, tt.expected)
		}
	}
}

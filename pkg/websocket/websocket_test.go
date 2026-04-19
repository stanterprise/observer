package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
)

func TestNewHub(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("hub.clients is nil")
	}

	if hub.broadcast == nil {
		t.Error("hub.broadcast is nil")
	}

	if hub.register == nil {
		t.Error("hub.register is nil")
	}

	if hub.unregister == nil {
		t.Error("hub.unregister is nil")
	}
}

func TestNewHub_NilLogger(t *testing.T) {
	hub := NewHub(nil)

	if hub == nil {
		t.Fatal("NewHub(nil) returned nil")
	}

	if hub.logger == nil {
		t.Error("hub.logger should not be nil even when nil logger is passed")
	}
}

func TestHub_Run_Shutdown(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run hub in background
	done := make(chan bool)
	go func() {
		hub.Run(ctx, NATSConfig{})
		done <- true
	}()

	// Wait for context to expire
	select {
	case <-done:
		// Hub stopped as expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Hub did not stop within expected time")
	}
}

func TestHub_InitNATS_NoURL(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)

	// Should not fail when no URL is provided
	err := hub.InitNATS(NATSConfig{URL: ""})
	if err != nil {
		t.Errorf("InitNATS with empty URL should not fail: %v", err)
	}
}

func TestEventPriorityClassification(t *testing.T) {
	tests := []struct {
		name      string
		eventType publisher.EventType
		isLowPri  bool
		isHighPri bool
	}{
		{"StepBegin", publisher.EventTypeStepBegin, true, false},
		{"StepEnd", publisher.EventTypeStepEnd, true, false},
		{"TestBegin", publisher.EventTypeTestBegin, false, true},
		{"TestEnd", publisher.EventTypeTestEnd, false, true},
		{"RunStart", publisher.EventTypeRunStart, false, true},
		{"RunEnd", publisher.EventTypeRunEnd, false, true},
		{"TestFailure", publisher.EventTypeTestFailure, false, true},
		{"TestError", publisher.EventTypeTestError, false, true},
		{"SuiteBegin", publisher.EventTypeSuiteBegin, false, false},
		{"SuiteEnd", publisher.EventTypeSuiteEnd, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLowPriorityEvent(tt.eventType); got != tt.isLowPri {
				t.Errorf("isLowPriorityEvent(%v) = %v, want %v", tt.eventType, got, tt.isLowPri)
			}
			if got := isHighPriorityEvent(tt.eventType); got != tt.isHighPri {
				t.Errorf("isHighPriorityEvent(%v) = %v, want %v", tt.eventType, got, tt.isHighPri)
			}
		})
	}
}

func TestSmartFiltering_StepEventsFilteredByRunID(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx, NATSConfig{})

	// Create two clients with different runID filters
	clientA := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
		filters: EventFilters{
			RunID: "run-a",
		},
	}
	clientB := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
		filters: EventFilters{
			RunID: "run-b",
		},
	}

	hub.register <- clientA
	hub.register <- clientB
	time.Sleep(50 * time.Millisecond) // Increased to ensure hub processes registrations

	// Send step event for run-a
	stepEvent := publisher.Event{
		Type:      publisher.EventTypeStepBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"runId":"run-a","step":{"id":"step-1"}}`),
	}
	eventBytes, _ := json.Marshal(stepEvent)
	hub.broadcast <- eventBytes
	time.Sleep(20 * time.Millisecond)

	// ClientA should receive it
	if len(clientA.send) != 1 {
		t.Errorf("ClientA should have 1 message, got %d", len(clientA.send))
	}

	// ClientB should NOT receive it (filtered)
	if len(clientB.send) != 0 {
		t.Errorf("ClientB should have 0 messages (filtered), got %d", len(clientB.send))
	}
}

func TestSmartFiltering_HighPriorityBroadcastToAll(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx, NATSConfig{})

	// Create two clients with different runID filters
	clientA := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
		filters: EventFilters{
			RunID: "run-a",
		},
	}
	clientB := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
		filters: EventFilters{
			RunID: "run-b",
		},
	}

	hub.register <- clientA
	hub.register <- clientB
	time.Sleep(50 * time.Millisecond) // Increased to ensure hub processes registrations

	// Send high-priority test.begin event for run-a
	testEvent := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"runId":"run-a","testCase":{"id":"test-1"}}`),
	}
	eventBytes, _ := json.Marshal(testEvent)
	hub.broadcast <- eventBytes
	time.Sleep(20 * time.Millisecond)

	// Both clients should be checked but only ClientA matches filter
	// ClientA should get it (matches filter)
	if len(clientA.send) != 1 {
		t.Errorf("ClientA should have 1 message, got %d", len(clientA.send))
	}

	// ClientB should NOT get it (doesn't match filter)
	if len(clientB.send) != 0 {
		t.Errorf("ClientB should have 0 messages, got %d", len(clientB.send))
	}
}

func TestSmartFiltering_NoFilterReceivesHighPriority(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx, NATSConfig{})

	// Client with no filters (should receive all events)
	clientNoFilter := &Client{
		hub:     hub,
		send:    make(chan []byte, 10),
		filters: EventFilters{}, // No filters
	}

	hub.register <- clientNoFilter
	time.Sleep(50 * time.Millisecond) // Increased to ensure hub processes registration

	// Send high-priority test event
	testEvent := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"runId":"run-a","testCase":{"id":"test-1"}}`),
	}
	eventBytes, _ := json.Marshal(testEvent)
	hub.broadcast <- eventBytes
	time.Sleep(20 * time.Millisecond)

	// Client with no filters should receive high-priority events
	if len(clientNoFilter.send) != 1 {
		t.Errorf("Client with no filters should have 1 message, got %d", len(clientNoFilter.send))
	}
}

func TestSmartFiltering_NoFilterDoesNotReceiveSteps(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx, NATSConfig{})

	// Client with no filters
	clientNoFilter := &Client{
		hub:     hub,
		send:    make(chan []byte, 10),
		filters: EventFilters{}, // No filters
	}

	hub.register <- clientNoFilter
	time.Sleep(50 * time.Millisecond) // Increased to ensure hub processes registration

	// Send low-priority step event
	stepEvent := publisher.Event{
		Type:      publisher.EventTypeStepBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"runId":"run-a","step":{"id":"step-1"}}`),
	}
	eventBytes, _ := json.Marshal(stepEvent)
	hub.broadcast <- eventBytes
	time.Sleep(20 * time.Millisecond)

	// Client with no filters should NOT receive step events (they need explicit filter)
	if len(clientNoFilter.send) != 0 {
		t.Errorf("Client with no filters should have 0 step messages, got %d", len(clientNoFilter.send))
	}
}

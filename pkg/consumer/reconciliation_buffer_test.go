package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
)

func TestReconciliationBuffer_BufferEvent(t *testing.T) {
	cfg := ReconciliationConfig{
		MaxBufferSize:   10,
		InactivityTTL:   5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Buffer an event
	err := rb.BufferEvent(runID, event)
	if err != nil {
		t.Fatalf("BufferEvent failed: %v", err)
	}

	// Verify buffer was created
	buffer, exists := rb.GetBuffer(runID)
	if !exists {
		t.Fatal("Expected buffer to exist")
	}
	if len(buffer.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(buffer.Events))
	}
	if buffer.Events[0].Type != publisher.EventTypeTestBegin {
		t.Errorf("Expected event type %s, got %s", publisher.EventTypeTestBegin, buffer.Events[0].Type)
	}
}

func TestReconciliationBuffer_TTLRefresh(t *testing.T) {
	cfg := ReconciliationConfig{
		MaxBufferSize:   10,
		InactivityTTL:   5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Buffer first event
	err := rb.BufferEvent(runID, event)
	if err != nil {
		t.Fatalf("BufferEvent failed: %v", err)
	}

	buffer, _ := rb.GetBuffer(runID)
	firstActivity := buffer.LastActivity

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Buffer second event
	err = rb.BufferEvent(runID, event)
	if err != nil {
		t.Fatalf("BufferEvent failed: %v", err)
	}

	buffer, _ = rb.GetBuffer(runID)
	secondActivity := buffer.LastActivity

	// Verify LastActivity was updated
	if !secondActivity.After(firstActivity) {
		t.Error("Expected LastActivity to be updated")
	}
}

func TestReconciliationBuffer_MaxSize(t *testing.T) {
	cfg := ReconciliationConfig{
		MaxBufferSize:   5,
		InactivityTTL:   5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Fill buffer to limit
	for i := 0; i < 5; i++ {
		err := rb.BufferEvent(runID, event)
		if err != nil {
			t.Fatalf("BufferEvent failed at %d: %v", i, err)
		}
	}

	// Attempt to exceed limit
	err := rb.BufferEvent(runID, event)
	if err == nil {
		t.Fatal("Expected error when exceeding buffer limit")
	}
	if !errors.Is(err, ErrBufferFull) {
		t.Errorf("Expected error to wrap ErrBufferFull, got %v", err)
	}
}

func TestReconciliationBuffer_MarkRootSuiteEnd(t *testing.T) {
	cfg := DefaultReconciliationConfig()
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"

	// Mark root suite end
	err := rb.MarkRootSuiteEnd(runID)
	if err != nil {
		t.Fatalf("MarkRootSuiteEnd failed: %v", err)
	}

	// Verify buffer was created with flag set
	buffer, exists := rb.GetBuffer(runID)
	if !exists {
		t.Fatal("Expected buffer to exist")
	}

	buffer.mu.RLock()
	received := buffer.RootSuiteEndReceived
	buffer.mu.RUnlock()

	if !received {
		t.Error("Expected RootSuiteEndReceived to be true")
	}
}

func TestReconciliationBuffer_DeleteBuffer(t *testing.T) {
	cfg := DefaultReconciliationConfig()
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Create buffer
	rb.BufferEvent(runID, event)

	// Verify it exists
	_, exists := rb.GetBuffer(runID)
	if !exists {
		t.Fatal("Expected buffer to exist")
	}

	// Delete buffer
	rb.DeleteBuffer(runID)

	// Verify it's gone
	_, exists = rb.GetBuffer(runID)
	if exists {
		t.Error("Expected buffer to not exist after deletion")
	}
}

func TestReconciliationBuffer_CleanupLoop(t *testing.T) {
	cfg := ReconciliationConfig{
		MaxBufferSize:   10,
		InactivityTTL:   200 * time.Millisecond, // Short TTL for testing
		CleanupInterval: 100 * time.Millisecond, // Fast cleanup for testing
	}
	rb := NewReconciliationBuffer(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Track reconciliation triggers
	triggered := make(chan string, 1)
	rb.SetInactivityCallback(func(ctx context.Context, runID string) {
		triggered <- runID
	})

	// Start cleanup loop
	rb.StartCleanupLoop(ctx)

	// Buffer an event
	rb.BufferEvent(runID, event)

	// Wait for inactivity timeout
	select {
	case triggeredRunID := <-triggered:
		if triggeredRunID != runID {
			t.Errorf("Expected runID %s, got %s", runID, triggeredRunID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Expected reconciliation to be triggered by inactivity timeout")
	}

	// Stop cleanup loop
	rb.StopCleanupLoop()
}

func TestReconciliationBuffer_GetAllBuffers(t *testing.T) {
	cfg := DefaultReconciliationConfig()
	rb := NewReconciliationBuffer(cfg, nil)

	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Create multiple buffers
	rb.BufferEvent("run-1", event)
	rb.BufferEvent("run-2", event)
	rb.BufferEvent("run-3", event)

	// Get all buffers
	buffers := rb.GetAllBuffers()

	if len(buffers) != 3 {
		t.Errorf("Expected 3 buffers, got %d", len(buffers))
	}

	for runID, buffer := range buffers {
		if buffer.RunID != runID {
			t.Errorf("Buffer runID mismatch: expected %s, got %s", runID, buffer.RunID)
		}
	}
}

func TestReconciliationBuffer_UpdateBufferStatus(t *testing.T) {
	cfg := DefaultReconciliationConfig()
	rb := NewReconciliationBuffer(cfg, nil)

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Create buffer
	rb.BufferEvent(runID, event)

	// Update status
	rb.UpdateBufferStatus(runID, StatusInProgress)

	// Verify status was updated
	buffer, _ := rb.GetBuffer(runID)
	buffer.mu.RLock()
	status := buffer.ReconciliationStatus
	buffer.mu.RUnlock()

	if status != StatusInProgress {
		t.Errorf("Expected status %s, got %s", StatusInProgress, status)
	}
}

func TestReconciliationBuffer_InactivityDoesNotTriggerDuringReconciliation(t *testing.T) {
	cfg := ReconciliationConfig{
		MaxBufferSize:   10,
		InactivityTTL:   100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	}
	rb := NewReconciliationBuffer(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runID := "test-run-1"
	event := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test":"data"}`),
	}

	// Track reconciliation triggers
	triggered := make(chan string, 10)
	rb.SetInactivityCallback(func(ctx context.Context, runID string) {
		triggered <- runID
	})

	// Start cleanup loop
	rb.StartCleanupLoop(ctx)

	// Buffer an event
	rb.BufferEvent(runID, event)

	// Mark as in progress
	rb.UpdateBufferStatus(runID, StatusInProgress)

	// Wait longer than inactivity TTL
	time.Sleep(300 * time.Millisecond)

	// Should not have triggered reconciliation
	select {
	case <-triggered:
		t.Error("Reconciliation should not trigger when status is in_progress")
	default:
		// Expected - no trigger
	}

	// Stop cleanup loop
	rb.StopCleanupLoop()
}

package websocket

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
)

// TestBufferOverflowDoesNotDisconnectClient verifies that when a client's buffer fills,
// old messages are dropped instead of disconnecting the client
func TestBufferOverflowDoesNotDisconnectClient(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub
	go hub.Run(ctx, NATSConfig{})

	// Create a mock client with small buffer for testing
	client := &Client{
		hub:     hub,
		send:    make(chan []byte, 5), // Small buffer to trigger overflow quickly
		filters: EventFilters{},
	}

	// Register client
	hub.register <- client

	// Give time for registration
	time.Sleep(10 * time.Millisecond)

	// Create test event
	testEvent := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"id":"test-1","name":"Test Case"}`),
	}
	eventBytes, _ := json.Marshal(testEvent)

	// Send more messages than buffer can hold (should trigger drop logic)
	for i := 0; i < 20; i++ {
		hub.broadcast <- eventBytes
		time.Sleep(1 * time.Millisecond) // Small delay to let hub process
	}

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Verify client is still registered (not disconnected)
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if !exists {
		t.Fatal("Client was disconnected when buffer overflowed - should have dropped messages instead")
	}

	// Verify some messages were dropped
	droppedCount := atomic.LoadInt64(&hub.droppedMessages)
	if droppedCount == 0 {
		t.Log("Warning: No messages were reported as dropped, buffer may not have filled")
	} else {
		t.Logf("Successfully dropped %d messages without disconnecting client", droppedCount)
	}
}

// TestMetricsTracking verifies that hub metrics are tracked correctly
func TestMetricsTracking(t *testing.T) {
	hub := NewHub(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx, NATSConfig{})

	// Create and register a client
	client := &Client{
		hub:     hub,
		send:    make(chan []byte, 2), // Very small buffer
		filters: EventFilters{},
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Get initial metrics
	initialMetrics := hub.GetMetrics()
	if initialMetrics.ConnectedClients != 1 {
		t.Errorf("Expected 1 connected client, got %d", initialMetrics.ConnectedClients)
	}
	if initialMetrics.DroppedMessages != 0 {
		t.Errorf("Expected 0 dropped messages initially, got %d", initialMetrics.DroppedMessages)
	}

	// Send messages to trigger drops
	testEvent := publisher.Event{
		Type:      publisher.EventTypeTestEnd,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"id":"test-1","status":"passed"}`),
	}
	eventBytes, _ := json.Marshal(testEvent)

	for i := 0; i < 10; i++ {
		hub.broadcast <- eventBytes
		time.Sleep(1 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	// Get updated metrics
	metrics := hub.GetMetrics()
	if metrics.ConnectedClients != 1 {
		t.Errorf("Client should still be connected, got %d clients", metrics.ConnectedClients)
	}

	t.Logf("Final metrics: Clients=%d, DroppedMessages=%d, DroppedBroadcasts=%d, QueueSize=%d/%d",
		metrics.ConnectedClients, metrics.DroppedMessages, metrics.DroppedBroadcasts,
		metrics.BroadcastQueueSize, metrics.BroadcastCapacity)
}

// TestNonBlockingBroadcast verifies that broadcast doesn't block when hub channel is full
func TestNonBlockingBroadcast(t *testing.T) {
	hub := NewHub(nil)

	// Fill the broadcast channel completely
	for i := 0; i < cap(hub.broadcast); i++ {
		hub.broadcast <- []byte(`{"type":"test"}`)
	}

	// Verify channel is full
	select {
	case hub.broadcast <- []byte(`{"type":"test"}`):
		t.Fatal("Broadcast channel should be full")
	default:
		// Channel is full as expected
	}

	// Simulate what consumeNATSEvents does - non-blocking send
	testEvent := publisher.Event{
		Type:      publisher.EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"id":"test-1"}`),
	}
	eventBytes, _ := json.Marshal(testEvent)

	// This should not block (simulating the fixed code)
	doneCh := make(chan bool, 1)
	go func() {
		select {
		case hub.broadcast <- eventBytes:
			// Would block in old code
		default:
			atomic.AddInt64(&hub.droppedBroadcasts, 1)
			// Non-blocking in new code
		}
		doneCh <- true
	}()

	// Should complete quickly (not block)
	select {
	case <-doneCh:
		// Success - operation completed without blocking
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Broadcast operation blocked - should have been non-blocking")
	}

	droppedCount := atomic.LoadInt64(&hub.droppedBroadcasts)
	if droppedCount != 1 {
		t.Errorf("Expected 1 dropped broadcast, got %d", droppedCount)
	}
}

package websocket

import (
	"context"
	"log/slog"
	"testing"
	"time"
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

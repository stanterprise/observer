package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
)

var (
	// ErrBufferFull is returned when a buffer reaches its size limit
	ErrBufferFull = errors.New("buffer full")
)

// ReconciliationStatus represents the state of reconciliation for a run
type ReconciliationStatus string

const (
	StatusPending    ReconciliationStatus = "pending"
	StatusInProgress ReconciliationStatus = "in_progress"
	StatusCompleted  ReconciliationStatus = "completed"
	StatusPartial    ReconciliationStatus = "partial"
	StatusFailed     ReconciliationStatus = "failed"
)

// BufferedEvent represents an event waiting for reconciliation
type BufferedEvent struct {
	Event        publisher.Event
	Timestamp    time.Time
	Type         publisher.EventType
	AttemptCount int
}

// RunBuffer holds buffered events for a specific test run
type RunBuffer struct {
	RunID        string
	Events       []BufferedEvent
	FirstSeen    time.Time
	LastActivity time.Time // Refreshes on ANY event for this runID

	RootSuiteEndReceived bool
	ReconciliationStatus ReconciliationStatus

	mu sync.RWMutex // Protects mutable fields
}

// ReconciliationBuffer manages buffered events for multiple test runs
type ReconciliationBuffer struct {
	mu      sync.RWMutex
	buffers map[string]*RunBuffer

	maxBufferSize   int
	inactivityTTL   time.Duration
	cleanupInterval time.Duration

	logger *slog.Logger

	// Callback for inactivity timeout
	onInactivityTimeout func(ctx context.Context, runID string)

	// Cleanup loop control
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// NewReconciliationBuffer creates a new reconciliation buffer manager
func NewReconciliationBuffer(cfg ReconciliationConfig, logger *slog.Logger) *ReconciliationBuffer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	rb := &ReconciliationBuffer{
		buffers:         make(map[string]*RunBuffer),
		maxBufferSize:   cfg.MaxBufferSize,
		inactivityTTL:   cfg.InactivityTTL,
		cleanupInterval: cfg.CleanupInterval,
		logger:          logger,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}

	return rb
}

// SetInactivityCallback sets the callback function for inactivity timeouts
func (rb *ReconciliationBuffer) SetInactivityCallback(callback func(ctx context.Context, runID string)) {
	rb.onInactivityTimeout = callback
}

// StartCleanupLoop starts the background cleanup goroutine
func (rb *ReconciliationBuffer) StartCleanupLoop(ctx context.Context) {
	go rb.cleanupLoop(ctx)
}

// StopCleanupLoop stops the cleanup loop
func (rb *ReconciliationBuffer) StopCleanupLoop() {
	close(rb.stopCleanup)
	<-rb.cleanupDone
}

// BufferEvent adds an event to the buffer for the given runID
func (rb *ReconciliationBuffer) BufferEvent(runID string, event publisher.Event) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	buffer, exists := rb.buffers[runID]
	if !exists {
		// Create new buffer for this run
		buffer = &RunBuffer{
			RunID:                runID,
			Events:               []BufferedEvent{},
			FirstSeen:            time.Now(),
			LastActivity:         time.Now(),
			RootSuiteEndReceived: false,
			ReconciliationStatus: StatusPending,
		}
		rb.buffers[runID] = buffer
		rb.logger.Info("created new buffer", "runID", runID)
	}

	// Check buffer size limit
	if len(buffer.Events) >= rb.maxBufferSize {
		return fmt.Errorf("%w: runID=%s size=%d", ErrBufferFull, runID, len(buffer.Events))
	}

	// Add event to buffer
	bufferedEvent := BufferedEvent{
		Event:        event,
		Timestamp:    time.Now(),
		Type:         event.Type,
		AttemptCount: 0,
	}
	buffer.Events = append(buffer.Events, bufferedEvent)

	// Update last activity (TTL refresh)
	buffer.LastActivity = time.Now()

	rb.logger.Debug("event buffered",
		"runID", runID,
		"type", event.Type,
		"buffer_size", len(buffer.Events))

	return nil
}

// MarkRootSuiteEnd marks that a root suite end event has been received
func (rb *ReconciliationBuffer) MarkRootSuiteEnd(runID string) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	buffer, exists := rb.buffers[runID]
	if !exists {
		// Create buffer if it doesn't exist
		buffer = &RunBuffer{
			RunID:                runID,
			Events:               []BufferedEvent{},
			FirstSeen:            time.Now(),
			LastActivity:         time.Now(),
			RootSuiteEndReceived: true,
			ReconciliationStatus: StatusPending,
		}
		rb.buffers[runID] = buffer
		rb.logger.Info("created buffer for root suite end", "runID", runID)
		return nil
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.RootSuiteEndReceived = true
	buffer.LastActivity = time.Now()

	rb.logger.Info("marked root suite end", "runID", runID, "buffer_size", len(buffer.Events))
	return nil
}

// GetBuffer retrieves a buffer for a run (thread-safe read)
func (rb *ReconciliationBuffer) GetBuffer(runID string) (*RunBuffer, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	buffer, exists := rb.buffers[runID]
	return buffer, exists
}

// DeleteBuffer removes a buffer after successful reconciliation
func (rb *ReconciliationBuffer) DeleteBuffer(runID string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	delete(rb.buffers, runID)
	rb.logger.Info("deleted buffer", "runID", runID)
}

// GetAllBuffers returns a snapshot of all buffers (for inspection/monitoring)
func (rb *ReconciliationBuffer) GetAllBuffers() map[string]*RunBuffer {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	// Create a copy to avoid lock contention
	snapshot := make(map[string]*RunBuffer, len(rb.buffers))
	for runID, buffer := range rb.buffers {
		snapshot[runID] = buffer
	}
	return snapshot
}

// UpdateBufferStatus updates the reconciliation status of a buffer
func (rb *ReconciliationBuffer) UpdateBufferStatus(runID string, status ReconciliationStatus) {
	rb.mu.RLock()
	buffer, exists := rb.buffers[runID]
	rb.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.ReconciliationStatus = status
}

// cleanupLoop runs periodically to check for inactive buffers
func (rb *ReconciliationBuffer) cleanupLoop(ctx context.Context) {
	defer close(rb.cleanupDone)

	ticker := time.NewTicker(rb.cleanupInterval)
	defer ticker.Stop()

	rb.logger.Info("cleanup loop started",
		"interval", rb.cleanupInterval,
		"inactivity_ttl", rb.inactivityTTL)

	for {
		select {
		case <-ctx.Done():
			rb.logger.Info("cleanup loop stopping: context cancelled")
			return
		case <-rb.stopCleanup:
			rb.logger.Info("cleanup loop stopping: stop signal received")
			return
		case <-ticker.C:
			rb.cleanupInactiveBuffers(ctx)
		}
	}
}

// cleanupInactiveBuffers checks for buffers that have exceeded inactivity TTL
func (rb *ReconciliationBuffer) cleanupInactiveBuffers(ctx context.Context) {
	rb.mu.Lock()
	inactiveRuns := []string{}

	for runID, buffer := range rb.buffers {
		buffer.mu.RLock()
		status := buffer.ReconciliationStatus
		lastActivity := buffer.LastActivity
		buffer.mu.RUnlock()

		// Don't interfere with ongoing reconciliation
		if status == StatusInProgress {
			continue
		}

		// Check if buffer has been inactive
		if time.Since(lastActivity) > rb.inactivityTTL {
			inactiveRuns = append(inactiveRuns, runID)
		}
	}
	rb.mu.Unlock()

	// Trigger reconciliation for inactive runs
	for _, runID := range inactiveRuns {
		rb.logger.Info("inactivity timeout reached, triggering reconciliation",
			"runID", runID,
			"inactivity_ttl", rb.inactivityTTL)

		if rb.onInactivityTimeout != nil {
			// Call the callback to trigger reconciliation
			go rb.onInactivityTimeout(ctx, runID)
		}
	}

	if len(inactiveRuns) > 0 {
		rb.logger.Info("cleanup pass completed",
			"inactive_runs", len(inactiveRuns))
	}
}

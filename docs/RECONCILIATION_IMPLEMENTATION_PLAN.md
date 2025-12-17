# Event Reconciliation Implementation Plan

## Overview

This document outlines the implementation plan for adding event reconciliation capabilities to the Observer Processor Service. The reconciliation system handles out-of-order test events by buffering unprocessable events and applying them once their dependencies are satisfied.

### Key Principles

- **Stateless Ingestion**: Ingestion service remains unchanged (gRPC → NATS only)
- **Stateful Processor**: Processor service manages event buffers and reconciliation
- **Inactivity-Based TTL**: Buffer timeout refreshes on EVERY event for a runID
- **Multiple Triggers**: Reconciliation triggered by root suite end OR inactivity timeout
- **Graceful Degradation**: Falls back to immediate processing on buffer overflow

### Architecture Decision

```
Test Reporter → Ingestion (gRPC) → NATS JetStream → Processor
                                                        ↓
                                    ┌───────────────────┴───────────────────┐
                                    ↓                                       ↓
                        Can Place Immediately?                      Cannot Place?
                                    ↓                                       ↓
                            MongoDB Upsert                          Buffer (per runID)
                                    ↓                                       ↓
                                  Ack                              Ack + Hold in Memory
                                                                            ↓
                                            ┌───────────────────────────────┘
                                            ↓
                                Root Suite End OR Inactivity Timeout
                                            ↓
                                Reconciliation Phase (recursive)
                                            ↓
                            Apply buffered events → Update buffer
```

---

## Stage 1: Foundation Components

**Duration**: 4-5 days

**Goal**: Build core reconciliation infrastructure without integrating into message flow

### Task 1.1: Reconciliation Buffer Manager

**File**: `pkg/consumer/reconciliation_buffer.go`

**Data Structures**:

```go
type ReconciliationBuffer struct {
    mu      sync.RWMutex
    buffers map[string]*RunBuffer

    maxBufferSize    int           // Default: 10,000 events per run
    inactivityTTL    time.Duration // Default: 5 minutes after last event
    cleanupInterval  time.Duration // Default: 1 minute

    metrics *ReconciliationMetrics
    logger  *slog.Logger
}

type RunBuffer struct {
    RunID         string
    Events        []BufferedEvent
    FirstSeen     time.Time
    LastActivity  time.Time  // CRITICAL: Refreshes on ANY event for this runID

    RootSuiteEndReceived bool
    ReconciliationStatus ReconciliationStatus
}

type BufferedEvent struct {
    Event     publisher.Event
    Timestamp time.Time
    Type      publisher.EventType
    AttemptCount int
}

type ReconciliationStatus string

const (
    StatusPending    ReconciliationStatus = "pending"
    StatusInProgress ReconciliationStatus = "in_progress"
    StatusCompleted  ReconciliationStatus = "completed"
    StatusPartial    ReconciliationStatus = "partial"
    StatusFailed     ReconciliationStatus = "failed"
)
```

**Methods to Implement**:

- `NewReconciliationBuffer(cfg ReconciliationConfig, logger *slog.Logger) *ReconciliationBuffer`
- `BufferEvent(runID string, event publisher.Event) error`
  - Creates buffer if doesn't exist
  - Appends event to buffer
  - **Updates LastActivity timestamp** (TTL refresh)
  - Checks buffer size limit
- `MarkRootSuiteEnd(runID string) error`
  - Sets `RootSuiteEndReceived = true`
  - Does NOT trigger reconciliation (caller does that)
- `GetBuffer(runID string) (*RunBuffer, bool)`
  - Thread-safe read access
- `DeleteBuffer(runID string)`
  - Removes buffer after successful reconciliation
- `startCleanupLoop(ctx context.Context)`
  - Background goroutine
  - Runs every `cleanupInterval`
  - Triggers reconciliation for buffers with `time.Since(LastActivity) > inactivityTTL`

**Unit Tests**:

- `TestReconciliationBuffer_BufferEvent` - Basic buffering
- `TestReconciliationBuffer_TTLRefresh` - Verify LastActivity updates
- `TestReconciliationBuffer_MaxSize` - Buffer size limit enforcement
- `TestReconciliationBuffer_MarkRootSuiteEnd` - Flag setting
- `TestReconciliationBuffer_CleanupLoop` - Inactivity timeout trigger

**Acceptance Criteria**:

- ✅ Buffer stores events correctly
- ✅ LastActivity updates on every BufferEvent call for same runID
- ✅ Buffer size limit enforced (returns error when full)
- ✅ Cleanup loop triggers reconciliation after inactivity TTL
- ✅ Thread-safe concurrent access

---

### Task 1.2: Event Classifier

**File**: `pkg/consumer/event_classifier.go`

**Purpose**: Determine if an event can be immediately processed or needs buffering

**Data Structures**:

```go
type EventClassification string

const (
    ClassifyImmediate   EventClassification = "immediate"   // Can process now
    ClassifyBuffer      EventClassification = "buffer"      // Buffer for reconciliation
    ClassifyReconcile   EventClassification = "reconcile"   // Root suite end - trigger reconciliation
)

type Classifier struct {
    repo   *repository.MongoRepository
    logger *slog.Logger
}
```

**Methods to Implement**:

- `NewClassifier(repo *repository.MongoRepository, logger *slog.Logger) *Classifier`
- `Classify(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Routes to specific classifier based on event type
- `classifySuiteBegin(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Root suite (no parent_suite_id in metadata) → ClassifyImmediate
  - Non-root suite → Check if parent exists via repo.SuiteExists()
    - Exists → ClassifyImmediate
    - Not exists → ClassifyBuffer
- `classifySuiteEnd(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Extract suite ID from event
  - Check if it's root suite (doc exists with \_id == suiteID)
    - Root suite → ClassifyReconcile
    - Non-root suite → ClassifyImmediate (or check existence)
- `classifyTestBegin(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Check if parent suite exists via repo.SuiteExists()
- `classifyStepBegin(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Check if parent test exists via repo.TestExists()
- `classifyTestEnd(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Check if test exists
- `classifyStepEnd(ctx context.Context, event publisher.Event) (EventClassification, error)`
  - Check if step exists

**Unit Tests**:

- `TestClassifier_RootSuiteBeginImmediate` - Root suite always immediate
- `TestClassifier_NestedSuiteWithParentImmediate` - Parent exists → immediate
- `TestClassifier_NestedSuiteWithoutParentBuffer` - Parent missing → buffer
- `TestClassifier_RootSuiteEndReconcile` - Root suite end → reconcile trigger
- `TestClassifier_TestBeginWithSuiteImmediate` - Suite exists → immediate
- `TestClassifier_TestBeginWithoutSuiteBuffer` - Suite missing → buffer

**Acceptance Criteria**:

- ✅ Root suite begin always returns ClassifyImmediate
- ✅ Root suite end returns ClassifyReconcile
- ✅ Nested entities return ClassifyBuffer when parent missing
- ✅ Nested entities return ClassifyImmediate when parent exists
- ✅ All event types handled

---

### Task 1.3: Repository Query Methods

**File**: `internal/repository/mongodb_query.go` (new file)

**Purpose**: Add existence check methods to support event classification

**Methods to Implement**:

- `SuiteExists(ctx context.Context, suiteID, parentSuiteID string) (bool, error)`
  - Extracts root document ID from parentSuiteID
  - Queries for suite in root document's suites array
  - Returns true if suite found
- `TestExists(ctx context.Context, testID, suiteID string) (bool, error)`
  - Extracts root document ID from suiteID
  - Checks root-level tests array
  - Checks nested suite tests arrays
  - Returns true if test found in either location
- `StepExists(ctx context.Context, stepID, testID, runID string) (bool, error)`
  - Extracts root document ID from runID
  - Queries for step in test's steps array
  - Handles both root-level and nested suite tests

**Implementation Notes**:

- Use `CountDocuments` for efficiency (no need to fetch full documents)
- Reuse `extractRootSuiteID` helper from existing repository code
- Handle both root-level and nested entities

**Unit Tests**:

- `TestMongoRepository_SuiteExists_RootLevel` - Root level suite
- `TestMongoRepository_SuiteExists_Nested` - Nested suite
- `TestMongoRepository_SuiteExists_NotFound` - Missing suite
- `TestMongoRepository_TestExists_RootLevel` - Test in root tests array
- `TestMongoRepository_TestExists_InSuite` - Test in nested suite
- `TestMongoRepository_StepExists` - Step in test

**Acceptance Criteria**:

- ✅ SuiteExists returns true when suite present in document hierarchy
- ✅ TestExists checks both root-level and nested suite tests
- ✅ StepExists handles nested test structures
- ✅ All methods handle non-existent entities gracefully
- ✅ Performance: queries use indexes effectively

---

### Task 1.4: Configuration & Feature Flag

**File**: `pkg/consumer/config.go` (new file)

**Data Structures**:

```go
type ReconciliationConfig struct {
    Enabled             bool          `env:"RECONCILIATION_ENABLED" envDefault:"false"`
    MaxBufferSize       int           `env:"RECONCILIATION_MAX_BUFFER_SIZE" envDefault:"10000"`
    InactivityTTL       time.Duration `env:"RECONCILIATION_INACTIVITY_TTL" envDefault:"5m"`
    MaxPasses           int           `env:"RECONCILIATION_MAX_PASSES" envDefault:"10"`
    PassDelay           time.Duration `env:"RECONCILIATION_PASS_DELAY" envDefault:"100ms"`
    CleanupInterval     time.Duration `env:"RECONCILIATION_CLEANUP_INTERVAL" envDefault:"1m"`
}
```

**Environment Variables**:

- `RECONCILIATION_ENABLED` - Master feature flag (default: false)
- `RECONCILIATION_MAX_BUFFER_SIZE` - Events per run (default: 10000)
- `RECONCILIATION_INACTIVITY_TTL` - Timeout after last event (default: 5m)
- `RECONCILIATION_MAX_PASSES` - Max reconciliation passes (default: 10)
- `RECONCILIATION_PASS_DELAY` - Delay between passes (default: 100ms)
- `RECONCILIATION_CLEANUP_INTERVAL` - Buffer cleanup frequency (default: 1m)

**Update**: `cmd/processor/main.go`

- Load reconciliation config from environment
- Pass to consumer constructor

**Acceptance Criteria**:

- ✅ Configuration loads from environment variables
- ✅ Defaults applied when env vars not set
- ✅ Feature flag can disable reconciliation entirely
- ✅ All timeouts parsed correctly (duration strings)

---

## Stage 2: Reconciliation Engine

**Duration**: 5-6 days

**Goal**: Implement reconciliation logic and integrate into consumer message flow

### Task 2.1: Reconciliation Engine Core

**File**: `pkg/consumer/reconciliation_engine.go`

**Data Structures**:

```go
type ReconciliationEngine struct {
    repo       *repository.MongoRepository
    classifier *Classifier
    logger     *slog.Logger

    maxPasses  int
    passDelay  time.Duration
}

type ReconciliationResult struct {
    RunID           string
    TotalEvents     int
    Status          ReconciliationStatus
    Passes          []PassResult
    RemainingEvents []BufferedEvent
    StartTime       time.Time
    EndTime         time.Time
    Error           error
}

type PassResult struct {
    PassNumber int
    Applied    int
    Remaining  int
    Duration   time.Duration
}
```

**Methods to Implement**:

- `NewReconciliationEngine(repo, classifier, logger, config) *ReconciliationEngine`
- `ReconcileRun(ctx context.Context, buffer *RunBuffer) (*ReconciliationResult, error)`
  - Main reconciliation entry point
  - Executes multiple passes until completion or max passes reached
  - Returns detailed result with pass-by-pass breakdown
- `attemptPass(ctx context.Context, events []BufferedEvent) (applied int, remaining []BufferedEvent, err error)`
  - Single reconciliation pass
  - Iterates through buffered events
  - Classifies each event (can it be applied now?)
  - Applies events that are ready
  - Returns remaining unprocessable events
- `applyEvent(ctx context.Context, event publisher.Event) error`
  - Applies a single event to MongoDB
  - Routes to appropriate handler (suite begin, test begin, etc.)
  - Same logic as immediate processing path

**Reconciliation Algorithm**:

```go
func (re *ReconciliationEngine) ReconcileRun(ctx context.Context, buffer *RunBuffer) (*ReconciliationResult, error) {
    result := &ReconciliationResult{
        RunID:       buffer.RunID,
        TotalEvents: len(buffer.Events),
        StartTime:   time.Now(),
    }

    remaining := buffer.Events

    for pass := 0; pass < re.maxPasses; pass++ {
        passStart := time.Now()
        appliedCount, newRemaining, err := re.attemptPass(ctx, remaining)
        passDuration := time.Since(passStart)

        result.Passes = append(result.Passes, PassResult{
            PassNumber: pass + 1,
            Applied:    appliedCount,
            Remaining:  len(newRemaining),
            Duration:   passDuration,
        })

        if err != nil {
            result.Error = err
            result.Status = StatusFailed
            result.EndTime = time.Now()
            return result, err
        }

        if appliedCount == 0 {
            // No progress made, stop trying
            re.logger.Warn("reconciliation stuck",
                "runID", buffer.RunID,
                "pass", pass+1,
                "remaining", len(newRemaining))
            break
        }

        remaining = newRemaining

        if len(remaining) == 0 {
            // All events successfully applied
            result.Status = StatusCompleted
            result.EndTime = time.Now()
            return result, nil
        }

        // Brief delay between passes to allow DB propagation
        if pass < re.maxPasses-1 && len(remaining) > 0 {
            time.Sleep(re.passDelay)
        }
    }

    // Some events remain unreconcilable
    result.Status = StatusPartial
    result.RemainingEvents = remaining
    result.EndTime = time.Now()

    return result, nil
}
```

**Unit Tests**:

- `TestReconciliationEngine_SimpleReconciliation` - 2 events, 1 pass
- `TestReconciliationEngine_DependencyResolution` - Events in wrong order, multiple passes
- `TestReconciliationEngine_MaxPassesExceeded` - Circular dependency, hits max passes
- `TestReconciliationEngine_NoProgress` - Some events unreconcilable, returns partial
- `TestReconciliationEngine_AllEventsApplied` - Returns completed status

**Acceptance Criteria**:

- ✅ Applies events recursively until all applied or no progress
- ✅ Respects max passes limit
- ✅ Returns detailed result with pass-by-pass breakdown
- ✅ Handles MongoDB errors gracefully
- ✅ Logs progress at appropriate levels

---

### Task 2.2: Consumer Integration

**File**: `pkg/consumer/nats_mongodb.go` (modify existing)

**Changes to MongoNATSConsumer struct**:

```go
type MongoNATSConsumer struct {
    nc         *nats.Conn
    js         jetstream.JetStream
    logger     *slog.Logger
    repo       *repository.MongoRepository

    // NEW: Reconciliation components (nil if disabled)
    reconciliationBuffer *ReconciliationBuffer
    classifier           *Classifier
    reconciliationEngine *ReconciliationEngine
    reconciliationConfig ReconciliationConfig

    stream     string
    consumer   jetstream.Consumer
}
```

**Modified/New Methods**:

- `NewMongoNATSConsumer(...)` - Initialize reconciliation components if enabled
- `processMessage(ctx, msg)` - Modified to include reconciliation logic
- `applyEventImmediately(ctx, event)` - Extract existing immediate processing logic
- `triggerReconciliation(ctx, runID)` - Async reconciliation trigger
- `extractRunID(event)` - Helper to extract runID from events

**New processMessage Logic**:

```go
func (c *MongoNATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
    var event publisher.Event
    if err := json.Unmarshal(msg.Data(), &event); err != nil {
        return fmt.Errorf("unmarshal event: %w", err)
    }

    // ALWAYS ACK to NATS immediately (decouple from reconciliation)
    defer func() {
        if ackErr := msg.Ack(); ackErr != nil {
            c.logger.Error("failed to ack message", "error", ackErr)
        }
    }()

    // If reconciliation disabled, process immediately (legacy behavior)
    if !c.reconciliationConfig.Enabled || c.reconciliationBuffer == nil {
        return c.applyEventImmediately(ctx, event)
    }

    // Extract runID from event
    runID, err := c.extractRunID(event)
    if err != nil {
        c.logger.Error("failed to extract runID", "error", err)
        return c.applyEventImmediately(ctx, event) // Fallback
    }

    // Update buffer LastActivity for this run (even if event processed immediately)
    // This ensures inactivity timeout only triggers after true inactivity
    if buf, exists := c.reconciliationBuffer.GetBuffer(runID); exists {
        buf.LastActivity = time.Now()
    }

    // Classify event
    classification, err := c.classifier.Classify(ctx, event)
    if err != nil {
        c.logger.Error("classification failed", "error", err)
        return c.applyEventImmediately(ctx, event) // Fallback
    }

    c.logger.Debug("event classified",
        "type", event.Type,
        "classification", classification,
        "runID", runID)

    switch classification {
    case ClassifyImmediate:
        // Process immediately
        return c.applyEventImmediately(ctx, event)

    case ClassifyBuffer:
        // Buffer for future reconciliation
        if err := c.reconciliationBuffer.BufferEvent(runID, event); err != nil {
            c.logger.Error("buffer event failed",
                "runID", runID,
                "error", err)
            // Fallback: try immediate application (best effort)
            return c.applyEventImmediately(ctx, event)
        }
        c.logger.Info("event buffered",
            "runID", runID,
            "type", event.Type)
        return nil

    case ClassifyReconcile:
        // This is root suite end - apply it then trigger reconciliation
        if err := c.applyEventImmediately(ctx, event); err != nil {
            return err
        }

        // Mark root suite end received
        if err := c.reconciliationBuffer.MarkRootSuiteEnd(runID); err != nil {
            c.logger.Error("mark root suite end failed",
                "runID", runID,
                "error", err)
        }

        // Trigger reconciliation asynchronously
        go c.triggerReconciliation(context.Background(), runID)

        return nil
    }

    return nil
}
```

**triggerReconciliation Implementation**:

```go
func (c *MongoNATSConsumer) triggerReconciliation(ctx context.Context, runID string) {
    c.logger.Info("starting reconciliation", "runID", runID)

    buffer, exists := c.reconciliationBuffer.GetBuffer(runID)
    if !exists {
        c.logger.Info("no buffer found for run", "runID", runID)
        return
    }

    if len(buffer.Events) == 0 {
        c.logger.Info("buffer empty, skipping reconciliation", "runID", runID)
        c.reconciliationBuffer.DeleteBuffer(runID)
        return
    }

    result, err := c.reconciliationEngine.ReconcileRun(ctx, buffer)
    if err != nil {
        c.logger.Error("reconciliation failed",
            "runID", runID,
            "error", err)
        return
    }

    c.logger.Info("reconciliation completed",
        "runID", runID,
        "status", result.Status,
        "total_events", result.TotalEvents,
        "passes", len(result.Passes),
        "duration", result.EndTime.Sub(result.StartTime).String(),
        "remaining", len(result.RemainingEvents))

    // Log pass-by-pass details
    for _, pass := range result.Passes {
        c.logger.Debug("reconciliation pass",
            "runID", runID,
            "pass", pass.PassNumber,
            "applied", pass.Applied,
            "remaining", pass.Remaining,
            "duration", pass.Duration.String())
    }

    if result.Status == StatusCompleted {
        // Clean up buffer
        c.reconciliationBuffer.DeleteBuffer(runID)
        c.logger.Info("reconciliation successful, buffer deleted", "runID", runID)
    } else if result.Status == StatusPartial {
        // Log remaining events for manual investigation
        c.logger.Warn("reconciliation incomplete",
            "runID", runID,
            "remaining_events", len(result.RemainingEvents))

        for i, evt := range result.RemainingEvents {
            c.logger.Warn("unreconciled event",
                "runID", runID,
                "index", i,
                "type", evt.Type,
                "timestamp", evt.Timestamp)
        }

        // Optionally: persist to dead letter queue or send alert
        // For now, keep buffer for manual inspection
    }
}
```

**Integration Tests**:

- `TestNATSMongoDB_ReconciliationDisabled` - Feature flag off, legacy behavior
- `TestNATSMongoDB_OutOfOrderEvents` - Test begin before suite begin
- `TestNATSMongoDB_RootSuiteEndTrigger` - Root suite end triggers reconciliation
- `TestNATSMongoDB_ComplexDependencies` - Nested suites, tests, steps
- `TestNATSMongoDB_PartialReconciliation` - Some events remain unreconcilable

**Acceptance Criteria**:

- ✅ Feature flag controls reconciliation behavior
- ✅ Events classified correctly
- ✅ Buffered events stored and reconciled
- ✅ Root suite end triggers reconciliation
- ✅ No event loss during reconciliation
- ✅ Immediate processing fallback works on errors

---

### Task 2.3: Inactivity Timeout Trigger

**File**: `pkg/consumer/reconciliation_buffer.go` (extend existing)

**Implementation**:

```go
func (rb *ReconciliationBuffer) startCleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(rb.cleanupInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            rb.logger.Info("cleanup loop stopping")
            return
        case <-ticker.C:
            rb.cleanupInactiveBuffers(ctx)
        }
    }
}

func (rb *ReconciliationBuffer) cleanupInactiveBuffers(ctx context.Context) {
    rb.mu.Lock()
    inactiveRuns := []string{}

    for runID, buffer := range rb.buffers {
        if buffer.ReconciliationStatus == StatusInProgress {
            // Don't interfere with ongoing reconciliation
            continue
        }

        if time.Since(buffer.LastActivity) > rb.inactivityTTL {
            inactiveRuns = append(inactiveRuns, runID)
        }
    }
    rb.mu.Unlock()

    // Trigger reconciliation for inactive runs
    for _, runID := range inactiveRuns {
        rb.logger.Info("inactivity timeout reached, triggering reconciliation",
            "runID", runID,
            "last_activity", rb.buffers[runID].LastActivity)

        // Trigger reconciliation via callback
        if rb.onInactivityTimeout != nil {
            go rb.onInactivityTimeout(ctx, runID)
        }
    }
}
```

**Update MongoNATSConsumer**:

- Start cleanup loop in `Start()` method
- Pass reconciliation callback to buffer
- Ensure cleanup loop stops on context cancellation

**Integration Tests**:

- `TestReconciliationBuffer_InactivityTimeout` - Wait for timeout, verify trigger
- `TestReconciliationBuffer_TTLRefreshPreventsTimeout` - Add events, timeout resets

**Acceptance Criteria**:

- ✅ Cleanup loop runs at configured interval
- ✅ Inactive buffers trigger reconciliation
- ✅ LastActivity update prevents premature timeout
- ✅ Cleanup loop stops gracefully on shutdown

---

## Stage 3: Observability & Monitoring

**Duration**: 3-4 days

**Goal**: Add metrics, logging, and inspection APIs

### Task 3.1: Reconciliation Metrics

**File**: `pkg/consumer/reconciliation_metrics.go`

**Data Structures**:

```go
type ReconciliationMetrics struct {
    // Atomic counters
    BufferedEvents         atomic.Int64
    ReconciliationsRun     atomic.Int64
    ReconciliationsSuccess atomic.Int64
    ReconciliationsPartial atomic.Int64
    ReconciliationsFailed  atomic.Int64
    EventsApplied          atomic.Int64
    EventsRemaining        atomic.Int64

    // Per-run metrics
    mu            sync.RWMutex
    activeBuffers map[string]*BufferMetrics
}

type BufferMetrics struct {
    RunID         string
    EventCount    int
    BufferSizeBytes int64
    FirstSeen     time.Time
    LastActivity  time.Time
    Status        ReconciliationStatus
}

type MetricsSnapshot struct {
    BufferedEvents         int64
    ReconciliationsRun     int64
    ReconciliationsSuccess int64
    ReconciliationsPartial int64
    ReconciliationsFailed  int64
    EventsApplied          int64
    EventsRemaining        int64
    ActiveBuffers          []BufferMetrics
    Timestamp              time.Time
}
```

**Methods to Implement**:

- `NewReconciliationMetrics() *ReconciliationMetrics`
- `RecordEventBuffered(runID string)`
- `RecordReconciliationStarted(runID string)`
- `RecordReconciliationCompleted(runID string, result *ReconciliationResult)`
- `GetMetrics() MetricsSnapshot`
- `GetBufferMetrics(runID string) (*BufferMetrics, bool)`

**Integration Points**:

- `ReconciliationBuffer.BufferEvent()` - Increment BufferedEvents
- `ReconciliationEngine.ReconcileRun()` - Record reconciliation results
- `MongoNATSConsumer.triggerReconciliation()` - Record success/failure

**Prometheus Metrics** (future):

- `observer_reconciliation_buffered_events_total`
- `observer_reconciliation_runs_total{status="success|partial|failed"}`
- `observer_reconciliation_events_applied_total`
- `observer_reconciliation_events_remaining_total`
- `observer_reconciliation_buffer_size_bytes`
- `observer_reconciliation_duration_seconds`

**Acceptance Criteria**:

- ✅ Metrics accurately track buffer state
- ✅ Reconciliation outcomes recorded
- ✅ Thread-safe concurrent updates
- ✅ Snapshot provides consistent view

---

### Task 3.2: API Endpoints for Buffer Inspection

**File**: `pkg/api/reconciliation_endpoints.go` (new file)

**Endpoints to Implement**:

1. **GET /api/reconciliation/status**

   - Returns global reconciliation status
   - Includes active buffer count, total events buffered
   - Response:
     ```json
     {
       "enabled": true,
       "active_buffers": 5,
       "buffered_events": 127,
       "reconciliations_run": 42,
       "reconciliations_success": 38,
       "reconciliations_partial": 3,
       "reconciliations_failed": 1,
       "events_applied": 3891,
       "events_remaining": 15
     }
     ```

2. **GET /api/reconciliation/buffers**

   - Returns list of all active buffers
   - Response:
     ```json
     {
       "buffers": [
         {
           "run_id": "test-run-123",
           "event_count": 15,
           "buffer_size_bytes": 45678,
           "first_seen": "2025-12-17T10:30:00Z",
           "last_activity": "2025-12-17T10:35:00Z",
           "status": "pending"
         }
       ]
     }
     ```

3. **GET /api/reconciliation/buffers/:runID**

   - Returns detailed buffer info for specific run
   - Includes list of buffered events (types, timestamps)
   - Response:
     ```json
     {
       "run_id": "test-run-123",
       "event_count": 15,
       "status": "pending",
       "first_seen": "2025-12-17T10:30:00Z",
       "last_activity": "2025-12-17T10:35:00Z",
       "root_suite_end_received": false,
       "events": [
         {
           "type": "test.begin",
           "timestamp": "2025-12-17T10:30:05Z",
           "attempt_count": 0
         }
       ]
     }
     ```

4. **POST /api/reconciliation/trigger/:runID**
   - Manually trigger reconciliation for a run
   - Useful for stuck buffers or manual intervention
   - Response:
     ```json
     {
       "run_id": "test-run-123",
       "triggered": true,
       "message": "Reconciliation triggered asynchronously"
     }
     ```

**Implementation**:

- Add routes to `pkg/api/rest_mongodb.go`
- Pass reconciliation buffer reference to API handlers
- Add proper error handling and validation

**Acceptance Criteria**:

- ✅ All endpoints return correct data
- ✅ Manual trigger successfully starts reconciliation
- ✅ Proper HTTP status codes (200, 404, 500)
- ✅ JSON responses well-formatted

---

### Task 3.3: Enhanced Logging

**Changes Across All Files**:

**Structured Logging Fields**:

- Add `runID` to all reconciliation-related logs
- Add `event_type`, `event_timestamp` to event logs
- Add pass details to reconciliation logs

**Log Levels**:

- `DEBUG`: Event classification, pass-by-pass progress
- `INFO`: Buffer creation, reconciliation start/complete, cleanup triggers
- `WARN`: Reconciliation incomplete, buffer overflow, stuck events
- `ERROR`: Classification failures, reconciliation errors, MongoDB errors

**Example Log Entries**:

```
INFO  event buffered runID=test-123 type=test.begin buffer_size=15
INFO  starting reconciliation runID=test-123 buffered_events=15
DEBUG reconciliation pass runID=test-123 pass=1 applied=8 remaining=7
DEBUG reconciliation pass runID=test-123 pass=2 applied=7 remaining=0
INFO  reconciliation completed runID=test-123 status=completed passes=2 duration=245ms
WARN  reconciliation incomplete runID=test-456 remaining_events=3 max_passes_reached=true
ERROR reconciliation failed runID=test-789 error="mongodb timeout"
```

**Acceptance Criteria**:

- ✅ All reconciliation operations logged appropriately
- ✅ Log levels used correctly
- ✅ Structured fields consistent across logs
- ✅ Logs parseable by log aggregation tools

---

### Task 3.4: Test Run API Extension

**File**: `pkg/api/rest_mongodb.go` (modify existing)

**Add Reconciliation Status to Test Run Response**:

```json
{
  "id": "test-run-123",
  "name": "Login Flow Tests",
  "status": "running",
  // ... existing fields ...
  "reconciliation": {
    "status": "pending",
    "buffered_events": 5,
    "last_activity": "2025-12-17T10:35:00Z"
  }
}
```

**Implementation**:

- Check reconciliation buffer for runID when serving test run
- Include reconciliation status if buffer exists
- Null/omit field if no buffer or reconciliation disabled

**Acceptance Criteria**:

- ✅ Test run response includes reconciliation status
- ✅ Field omitted when reconciliation disabled
- ✅ Field omitted when no buffer for run
- ✅ Backward compatible (existing clients ignore new field)

---

## Stage 4: Production Hardening

**Duration**: 4-5 days

**Goal**: Ensure production readiness with error handling, load testing, and resilience

### Task 4.1: Comprehensive Error Handling

**Areas to Cover**:

1. **Buffer Overflow Graceful Degradation**:

   ```go
   if err := c.reconciliationBuffer.BufferEvent(runID, event); err != nil {
       if errors.Is(err, ErrBufferFull) {
           c.logger.Warn("buffer full, falling back to immediate processing",
               "runID", runID,
               "buffer_size", c.reconciliationConfig.MaxBufferSize)
           // Fallback: attempt immediate processing
           return c.applyEventImmediately(ctx, event)
       }
       return err
   }
   ```

2. **MongoDB Transient Failures**:

   - Add retry logic with exponential backoff
   - Preserve buffer on transient failures
   - Don't delete buffer until reconciliation succeeds

3. **Classification Errors**:

   - Fall back to immediate processing if classification fails
   - Log classification errors for investigation

4. **Reconciliation Panics**:
   - Wrap reconciliation in recover()
   - Log panic details
   - Mark buffer as failed, preserve for inspection

**Acceptance Criteria**:

- ✅ Buffer overflow handled gracefully
- ✅ Transient errors don't cause data loss
- ✅ Panics don't crash processor
- ✅ All error paths logged appropriately

---

### Task 4.2: Circuit Breaker for Reconciliation

**File**: `pkg/consumer/reconciliation_circuit_breaker.go` (new file)

**Purpose**: Prevent cascading failures when reconciliation repeatedly fails

**Implementation**:

```go
type CircuitBreaker struct {
    mu                 sync.Mutex
    state              CircuitState
    failureCount       int
    successCount       int
    lastFailureTime    time.Time
    failureThreshold   int           // e.g., 5 failures
    successThreshold   int           // e.g., 2 successes
    openDuration       time.Duration // e.g., 1 minute
}

type CircuitState string

const (
    CircuitClosed    CircuitState = "closed"     // Normal operation
    CircuitOpen      CircuitState = "open"       // Stop reconciliation
    CircuitHalfOpen  CircuitState = "half_open"  // Test if recovered
)

func (cb *CircuitBreaker) Call(fn func() error) error {
    if !cb.AllowRequest() {
        return ErrCircuitOpen
    }

    err := fn()
    cb.RecordResult(err == nil)
    return err
}
```

**Integration**:

- Wrap `ReconcileRun()` calls in circuit breaker
- Log circuit state changes
- Expose circuit state in metrics

**Acceptance Criteria**:

- ✅ Circuit opens after repeated failures
- ✅ Circuit closes after successful recovery
- ✅ Open circuit prevents wasted reconciliation attempts
- ✅ Circuit state visible in metrics/logs

---

### Task 4.3: Load Testing

**File**: `tests/load/reconciliation_load_test.go` (new file)

**Test Scenarios**:

1. **High Event Rate**:

   - 10,000 events/second sustained
   - Mix of in-order and out-of-order events
   - Measure: throughput, latency, buffer sizes

2. **Large Buffer Reconciliation**:

   - Single run with 5,000 buffered events
   - Measure: reconciliation duration, memory usage, pass count

3. **Many Concurrent Runs**:

   - 1,000 concurrent test runs
   - Each with 10-50 buffered events
   - Measure: buffer memory usage, reconciliation success rate

4. **Sustained Load**:
   - Run for 1 hour
   - Verify: no memory leaks, no goroutine leaks, stable performance

**Acceptance Criteria**:

- ✅ Handles 10,000 events/sec without message loss
- ✅ Reconciles 5,000 events in < 10 seconds
- ✅ Supports 1,000 concurrent runs
- ✅ No memory leaks over 1 hour
- ✅ 99th percentile reconciliation latency < 5 seconds

---

### Task 4.4: Chaos Testing

**Test Scenarios**:

1. **MongoDB Disconnection During Reconciliation**:

   - Start reconciliation
   - Disconnect MongoDB mid-reconciliation
   - Verify: buffer preserved, reconciliation retries after reconnect

2. **Processor Restart with Active Buffers**:

   - Buffer events
   - Kill processor
   - Restart processor
   - Verify: events redelivered by NATS (since we ack immediately, new events recreate buffer)

3. **NATS Disconnection**:

   - Disconnect NATS during high load
   - Verify: processor reconnects, resumes processing

4. **Clock Skew Simulation**:
   - Simulate system clock changes
   - Verify: TTL calculations still work correctly

**Acceptance Criteria**:

- ✅ Recovers from MongoDB disconnection
- ✅ Handles processor restarts gracefully
- ✅ NATS reconnection works
- ✅ Clock skew handled correctly

---

### Task 4.5: Documentation

**Files to Create/Update**:

1. **docs/architecture/11-reconciliation.md**:

   - Architecture overview
   - Data flow diagrams
   - Component descriptions
   - Design decisions and trade-offs

2. **docs/RECONCILIATION_CONFIGURATION.md**:

   - Environment variables
   - Tuning guidelines
   - Performance considerations

3. **docs/RECONCILIATION_TROUBLESHOOTING.md**:

   - Common issues and solutions
   - How to inspect stuck buffers
   - Manual reconciliation steps
   - Metrics interpretation

4. **docs/RECONCILIATION_METRICS.md**:

   - Prometheus metrics reference
   - Grafana dashboard examples
   - Alerting recommendations

5. **README.md** (update):
   - Add reconciliation feature to feature list
   - Link to reconciliation documentation

**Acceptance Criteria**:

- ✅ Architecture documented with diagrams
- ✅ Configuration guide complete
- ✅ Troubleshooting runbook usable by ops team
- ✅ Metrics well-documented
- ✅ README updated

---

## Rollout Strategy

### Phase 1: Development Environment (Week 1)

- Deploy with `RECONCILIATION_ENABLED=true`
- Validate basic functionality
- Test out-of-order event scenarios
- Fix any critical bugs

### Phase 2: Staging Environment (Week 2)

- Deploy with `RECONCILIATION_ENABLED=true`
- Run production-like load tests
- Monitor metrics, logs, buffer behavior
- Tune configuration (buffer sizes, timeouts)
- Fix any performance issues

### Phase 3: Canary Deployment (Week 3)

- Deploy to 10% of production traffic
- Monitor for 48 hours
- Watch for:
  - Increased memory usage
  - Reconciliation success rate
  - Buffer overflow events
  - Stuck buffers
- Roll back if issues detected

### Phase 4: Full Production (Week 3-4)

- Gradually increase to 100%
- 10% → 25% → 50% → 100%
- Monitor each step for 24 hours
- Full rollback plan ready

### Phase 5: Feature Flag Removal (Future)

- After 1 month of stable operation
- Remove feature flag
- Make reconciliation default behavior
- Clean up legacy code paths

---

## Success Criteria Summary

### Performance

- ✅ 99th percentile reconciliation latency < 5 seconds
- ✅ Buffer memory usage < 100 MB per 10,000 events
- ✅ Supports 10,000 events/second sustained throughput
- ✅ Reconciliation success rate > 99%

### Reliability

- ✅ Zero event loss during reconciliation
- ✅ Graceful degradation under load (fallback to immediate)
- ✅ Recovery from failures within 30 seconds
- ✅ No memory leaks over extended operation

### Observability

- ✅ All reconciliation events logged with structured data
- ✅ Metrics exposed for monitoring
- ✅ API endpoints for buffer inspection
- ✅ Alerting on anomalies

### Production Readiness

- ✅ Comprehensive error handling
- ✅ Load tested at 2x expected production load
- ✅ Chaos testing passed
- ✅ Documentation complete
- ✅ Runbook for operations team

---

## Risk Mitigation

### Risk: Buffer Memory Exhaustion

**Mitigation**:

- Per-run buffer size limits
- Global memory monitoring
- Graceful degradation (fallback to immediate)

### Risk: Reconciliation Never Completes

**Mitigation**:

- Max passes limit
- Inactivity timeout backup trigger
- Manual trigger API endpoint

### Risk: Event Loss During Processor Restart

**Mitigation**:

- Ack to NATS immediately (NATS redelivery covers restarts)
- Optional: Persistent buffer in Redis/NATS KV (future enhancement)

### Risk: Performance Degradation

**Mitigation**:

- Feature flag for quick disable
- Load testing validates performance
- Circuit breaker prevents cascading failures

### Risk: MongoDB Contention

**Mitigation**:

- Classification queries use indexes
- Existence checks optimized (CountDocuments)
- Reconciliation passes include delay for DB propagation

---

## Future Enhancements (Post-Stage 4)

1. **Persistent Buffer (Redis/NATS KV)**:

   - Survive processor restarts
   - Support multi-instance deployment
   - Shared buffer across processor instances

2. **Dead Letter Queue**:

   - Persist unreconcilable events
   - Manual review and replay
   - Analytics on failed events

3. **Reconciliation Prediction**:

   - ML model to predict event dependencies
   - Optimize pass ordering
   - Reduce reconciliation duration

4. **Smart Buffering**:

   - Only buffer truly out-of-order events
   - Most events process immediately
   - Reduce buffer memory usage

5. **Distributed Reconciliation**:
   - Partition buffers across processor instances
   - Parallel reconciliation for large runs
   - Faster reconciliation for complex test suites

---

## Glossary

- **Reconciliation**: Process of applying buffered events once dependencies are satisfied
- **Buffer**: In-memory storage of events awaiting reconciliation
- **Inactivity TTL**: Time after last event before reconciliation trigger
- **Classification**: Determining if an event can be immediately processed
- **Pass**: Single iteration through buffered events during reconciliation
- **Root Suite End**: Event that signals test run completion and triggers reconciliation

---

## Contact & Support

For questions or issues during implementation:

- Architecture decisions: Architect Agent
- Implementation details: Developer Agent
- Testing strategy: QA Lead
- Production deployment: DevOps Team

---

**Document Version**: 1.0
**Last Updated**: 2025-12-17
**Status**: Approved for Implementation

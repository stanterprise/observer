# Race Condition Solutions for Step Persistence

## Date: January 19, 2026

## Context: Handling out-of-order event arrival in distributed test observability system

---

## Problem Statement

In a distributed event-driven system with NATS JetStream, events can arrive out of order due to:

- Network latency variations
- Concurrent workers/reporters
- NATS redelivery/retries
- Message processing speed differences

**Critical Race Conditions:**

1. **StepBegin arrives before TestBegin** → Parent test doesn't exist yet
2. **StepBegin arrives before attempt initialized** → Attempt[retry_index] doesn't exist
3. **Multiple steps for same attempt arrive concurrently** → Parallel writes conflict

---

## Current Implementation Analysis

### Existing Error Handling (Good ✅)

```go
// pkg/consumer/nats_mongodb.go:191-202
if err := c.processMessage(ctx, msg); err != nil {
    c.logger.Error("failed to process message", "error", err)
    // NAK message for redelivery
    if nakErr := msg.Nak(); nakErr != nil {
        c.logger.Error("failed to nak message", "error", nakErr)
    }
} else {
    if ackErr := msg.Ack(); ackErr != nil {
        c.logger.Error("failed to ack message", "error", ackErr)
    }
}
```

**Good:** Errors cause NAK → message redelivery → eventual consistency ✅
**Issue:** No distinction between "transient" vs "permanent" errors

---

## Recommended Solutions (3-Tier Strategy)

### **Tier 1: Defensive Persistence (Immediate - No Architecture Changes)**

Make repository methods create missing parent structures on-demand.

#### Solution 1.1: Auto-Create Missing Attempts in UpsertStepBegin

**Implementation:**

```go
// internal/repository/mongodb_step_begin.go
// After line 164, when MatchedCount == 0

if result.MatchedCount == 0 {
    r.logger.Warn("attempt not found for step, creating attempt on-demand",
        "runID", runID,
        "testID", testID,
        "retryIndex", retry_index,
        "stepID", step.ID)

    // Check if parent test exists first
    testExists, err := r.testExistsWithID(ctx, runID, testID)
    if err != nil {
        return fmt.Errorf("check test existence: %w", err)
    }

    if !testExists {
        // Parent test doesn't exist - this is a true ordering issue
        // Return specific error for consumer to NAK with delay
        return &ParentNotFoundError{
            ParentType: "test",
            ParentID:   testID,
            ChildType:  "step",
            ChildID:    step.ID,
        }
    }

    // Test exists but attempt doesn't - create it
    if err := r.createAttemptIfMissing(ctx, runID, testID, retry_index, now); err != nil {
        return fmt.Errorf("create missing attempt: %w", err)
    }

    // Retry step insertion
    result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
    if err != nil {
        return fmt.Errorf("retry append step after creating attempt: %w", err)
    }

    if result.MatchedCount == 0 {
        return fmt.Errorf("step insertion failed after creating attempt: runID=%s, testID=%s", runID, testID)
    }

    r.logger.Info("step begin (inserted after creating missing attempt)",
        "runID", runID,
        "stepID", step.ID,
        "testID", testID,
        "retryIndex", retry_index)
    return nil
}
```

**Helper Methods to Add:**

```go
// internal/repository/mongodb_helpers.go

// testExistsWithID checks if a test exists in the document
func (r *MongoRepository) testExistsWithID(ctx context.Context, runID, testID string) (bool, error) {
    filter := bson.M{
        "_id":      runID,
        "tests.id": testID,
    }
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// createAttemptIfMissing creates an attempt if it doesn't exist
func (r *MongoRepository) createAttemptIfMissing(ctx context.Context, runID, testID string, retryIndex int32, now time.Time) error {
    // Check if attempt already exists
    filter := bson.M{
        "_id":                        runID,
        "tests.id":                   testID,
        "tests.attempts.retry_index": retryIndex,
    }
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return fmt.Errorf("check attempt existence: %w", err)
    }

    if count > 0 {
        // Attempt already exists (race condition: created between checks)
        r.logger.Debug("attempt already exists (concurrent creation)",
            "runID", runID,
            "testID", testID,
            "retryIndex", retryIndex)
        return nil
    }

    // Create new attempt
    newAttempt := &m.AttemptDocument{
        RetryIndex: retryIndex,
        Steps:      []*m.StepDocument{},
        Status:     "RUNNING",
        CreatedAt:  now,
        UpdatedAt:  now,
    }

    filter = bson.M{
        "_id":      runID,
        "tests.id": testID,
    }
    update := bson.M{
        "$push": bson.M{"tests.$[test].attempts": newAttempt},
        "$set":  bson.M{"updated_at": now},
    }
    arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
        Filters: []interface{}{
            bson.M{"test.id": testID},
        },
    })

    result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
    if err != nil {
        return fmt.Errorf("insert attempt: %w", err)
    }

    if result.MatchedCount == 0 {
        return fmt.Errorf("parent test not found: testID=%s", testID)
    }

    r.logger.Info("created missing attempt on-demand",
        "runID", runID,
        "testID", testID,
        "retryIndex", retryIndex)
    return nil
}
```

**Custom Error Type:**

```go
// internal/repository/errors.go (new file)
package repository

import "fmt"

// ParentNotFoundError indicates a parent entity doesn't exist yet
// This is a retry-able error - NAK with delay
type ParentNotFoundError struct {
    ParentType string // "test", "suite", "attempt"
    ParentID   string
    ChildType  string // "step", "test"
    ChildID    string
}

func (e *ParentNotFoundError) Error() string {
    return fmt.Sprintf("%s parent not found: parentID=%s (for %s=%s)",
        e.ParentType, e.ParentID, e.ChildType, e.ChildID)
}

func (e *ParentNotFoundError) IsRetryable() bool {
    return true
}
```

---

### **Tier 2: Smart Consumer Retry Logic (Medium Effort)**

Enhance consumer to distinguish between transient and permanent errors.

#### Solution 2.1: Typed Error Handling with Exponential Backoff

```go
// pkg/consumer/nats_mongodb.go

import (
    "errors"
    "time"
    "github.com/stanterprise/observer/internal/repository"
)

func (c *MongoNATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
    // ... existing unmarshal logic ...

    var err error
    switch event.Type {
    case publisher.EventTypeStepBegin:
        err = c.handleStepBegin(ctx, event.Data)
    // ... other cases ...
    }

    if err != nil {
        return c.handleProcessingError(ctx, msg, err, event.Type)
    }

    return nil
}

func (c *MongoNATSConsumer) handleProcessingError(ctx context.Context, msg jetstream.Msg, err error, eventType publisher.EventType) error {
    var parentNotFound *repository.ParentNotFoundError

    if errors.As(err, &parentNotFound) {
        // Parent not found - likely ordering issue
        // NAK with delay for retry
        c.logger.Warn("parent not found, will retry",
            "error", err,
            "eventType", eventType,
            "metadata", msg.Metadata())

        // Get current delivery count
        meta, metaErr := msg.Metadata()
        if metaErr == nil {
            deliveryCount := meta.NumDelivered

            // Exponential backoff: 1s, 2s, 4s, 8s, 16s (max 5 retries)
            if deliveryCount > 5 {
                c.logger.Error("max retries exceeded for parent-not-found, sending to DLQ",
                    "deliveryCount", deliveryCount,
                    "error", err)
                // Terminate message (goes to DLQ if configured)
                return msg.Term()
            }

            // NAK with exponential backoff
            delay := time.Duration(1<<(deliveryCount-1)) * time.Second
            if delay > 30*time.Second {
                delay = 30 * time.Second // Cap at 30s
            }

            c.logger.Debug("nak with delay",
                "delay", delay,
                "deliveryCount", deliveryCount)

            return msg.NakWithDelay(delay)
        }

        // Fallback: NAK without delay
        return msg.Nak()
    }

    // Other errors - check if MongoDB connection issue
    if isTransientDBError(err) {
        c.logger.Warn("transient DB error, will retry", "error", err)
        return msg.NakWithDelay(2 * time.Second)
    }

    // Permanent error (bad data, validation failure, etc.)
    c.logger.Error("permanent processing error, terminating message",
        "error", err,
        "eventType", eventType)

    // Terminate (goes to DLQ if configured, otherwise discarded)
    return msg.Term()
}

func isTransientDBError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    // MongoDB transient errors
    return strings.Contains(errStr, "connection") ||
        strings.Contains(errStr, "timeout") ||
        strings.Contains(errStr, "network") ||
        strings.Contains(errStr, "temporary")
}
```

**Update NATS Consumer Configuration:**

```go
// cmd/processor/main.go or pkg/consumer/nats_mongodb.go config

consumerConfig := jetstream.ConsumerConfig{
    Durable:       "processor",
    FilterSubject: "tests.events.v1.>",
    AckPolicy:     jetstream.AckExplicitPolicy,
    MaxDeliver:    10,  // Allow up to 10 redeliveries
    AckWait:       30 * time.Second,
    MaxAckPending: 100,

    // Backoff configuration for automatic retry
    BackOff: []time.Duration{
        1 * time.Second,
        2 * time.Second,
        5 * time.Second,
        10 * time.Second,
        30 * time.Second,
    },
}
```

---

### **Tier 3: Event Buffering & Reordering (Advanced - Future)**

For high-throughput scenarios with frequent out-of-order events.

#### Solution 3.1: In-Memory Event Buffer with Dependency Tracking

**Architecture:**

```
NATS → Consumer → Event Buffer → Dependency Checker → Repository
                     ↓
                [Pending Events]
                 {testID: [stepEvent1, stepEvent2]}
```

**Implementation Sketch:**

```go
// pkg/consumer/event_buffer.go (new file)

type EventBuffer struct {
    mu              sync.RWMutex
    pendingSteps    map[string][]pendingStep  // testID -> []steps
    maxBufferSize   int
    maxBufferAge    time.Duration
    repo            *repository.MongoRepository
    logger          *slog.Logger
}

type pendingStep struct {
    step       *m.StepDocument
    testID     string
    retryIndex int32
    receivedAt time.Time
    attempts   int
}

func (eb *EventBuffer) HandleStep(ctx context.Context, step *m.StepDocument, testID string, retryIndex int32) error {
    // Try immediate persistence
    err := eb.repo.UpsertStepBegin(ctx, step.RunID, step, testID, retryIndex)

    var parentNotFound *repository.ParentNotFoundError
    if errors.As(err, &parentNotFound) {
        // Buffer for later retry
        return eb.bufferStep(step, testID, retryIndex)
    }

    if err != nil {
        return err
    }

    // Success - check if any buffered steps can now be persisted
    return eb.flushBufferedSteps(ctx, step.RunID, testID)
}

func (eb *EventBuffer) bufferStep(step *m.StepDocument, testID string, retryIndex int32) error {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    key := fmt.Sprintf("%s:%s:%d", step.RunID, testID, retryIndex)

    pending := pendingStep{
        step:       step,
        testID:     testID,
        retryIndex: retryIndex,
        receivedAt: time.Now(),
        attempts:   0,
    }

    eb.pendingSteps[key] = append(eb.pendingSteps[key], pending)

    if len(eb.pendingSteps[key]) > eb.maxBufferSize {
        eb.logger.Warn("buffer size exceeded, dropping oldest step",
            "key", key,
            "bufferSize", len(eb.pendingSteps[key]))
        // Drop oldest
        eb.pendingSteps[key] = eb.pendingSteps[key][1:]
    }

    eb.logger.Debug("buffered step for later persistence",
        "stepID", step.ID,
        "testID", testID,
        "retryIndex", retryIndex,
        "bufferSize", len(eb.pendingSteps[key]))

    return nil
}

func (eb *EventBuffer) flushBufferedSteps(ctx context.Context, runID, testID string) error {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    key := fmt.Sprintf("%s:%s:*", runID, testID)
    // Find all keys matching this test

    for k, steps := range eb.pendingSteps {
        if !strings.HasPrefix(k, runID+":"+testID+":") {
            continue
        }

        var remaining []pendingStep
        for _, ps := range steps {
            err := eb.repo.UpsertStepBegin(ctx, ps.step.RunID, ps.step, ps.testID, ps.retryIndex)
            if err != nil {
                ps.attempts++
                if ps.attempts < 10 {
                    remaining = append(remaining, ps)
                } else {
                    eb.logger.Error("max buffer retries exceeded, dropping step",
                        "stepID", ps.step.ID,
                        "attempts", ps.attempts)
                }
            } else {
                eb.logger.Info("flushed buffered step",
                    "stepID", ps.step.ID,
                    "testID", ps.testID)
            }
        }

        if len(remaining) == 0 {
            delete(eb.pendingSteps, k)
        } else {
            eb.pendingSteps[k] = remaining
        }
    }

    return nil
}

// Background goroutine to age out old buffered events
func (eb *EventBuffer) StartAgeoutWorker(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            eb.ageOutOldEvents()
        }
    }
}

func (eb *EventBuffer) ageOutOldEvents() {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    now := time.Now()
    for key, steps := range eb.pendingSteps {
        var remaining []pendingStep
        for _, ps := range steps {
            age := now.Sub(ps.receivedAt)
            if age > eb.maxBufferAge {
                eb.logger.Warn("dropping aged-out buffered step",
                    "stepID", ps.step.ID,
                    "age", age,
                    "maxAge", eb.maxBufferAge)
            } else {
                remaining = append(remaining, ps)
            }
        }

        if len(remaining) == 0 {
            delete(eb.pendingSteps, key)
        } else {
            eb.pendingSteps[key] = remaining
        }
    }
}
```

---

## Recommendation: Phased Rollout

### **Phase 1 (Immediate - This Week):**

✅ Implement **Tier 1: Defensive Persistence**

- Add `createAttemptIfMissing()` helper
- Add `ParentNotFoundError` type
- Modify `UpsertStepBegin` to auto-create missing attempts
- Add validation for step.ID

**Effort:** 2-4 hours  
**Risk:** Low (only makes code more defensive)  
**Benefit:** Handles 80% of race condition cases

### **Phase 2 (Next Sprint):**

✅ Implement **Tier 2: Smart Consumer Retry**

- Add typed error handling in consumer
- Implement exponential backoff for NAK
- Configure NATS consumer with backoff
- Add metrics for retry counts

**Effort:** 1-2 days  
**Risk:** Medium (changes consumer behavior)  
**Benefit:** Graceful handling of ordering issues, better observability

### **Phase 3 (Future - If Needed):**

⏸️ **Tier 3: Event Buffering** (only if Tier 1+2 insufficient)

- Implement event buffer
- Add dependency tracking
- Background flush workers
- Monitoring and alerts

**Effort:** 1 week  
**Risk:** High (complex state management, memory concerns)  
**Benefit:** Handles extreme out-of-order scenarios

---

## Additional Safeguards

### 1. **Idempotency Keys**

Add to step events for duplicate detection:

```go
type StepDocument struct {
    // ... existing fields ...
    IdempotencyKey string `bson:"idempotency_key,omitempty"` // Hash of (runID, testID, stepID, retryIndex)
}
```

### 2. **Event Ordering Validation**

Add sequence numbers to events:

```go
type Event struct {
    // ... existing fields ...
    SequenceNumber int64 `json:"sequence_number"`
}
```

### 3. **Metrics & Alerting**

```go
// Add prometheus metrics
var (
    stepPersistenceErrors = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "observer_step_persistence_errors_total",
        },
        []string{"error_type"},
    )

    stepOrderingRetries = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "observer_step_ordering_retries_total",
        },
    )
)

// In error handling:
if errors.As(err, &parentNotFound) {
    stepPersistenceErrors.WithLabelValues("parent_not_found").Inc()
    stepOrderingRetries.Inc()
}
```

---

## Testing Strategy

Add integration tests for race conditions:

```go
// tests/race_conditions_test.go

func TestStepBeforeTest_RaceCondition(t *testing.T) {
    // 1. Send StepBegin event
    // 2. Wait 100ms
    // 3. Send TestBegin event
    // 4. Verify step eventually persisted (retry logic worked)
}

func TestConcurrentStepsToSameAttempt(t *testing.T) {
    // Send 10 steps concurrently to same attempt
    // Verify all 10 persisted without data loss
}

func TestMissingAttempt_AutoCreation(t *testing.T) {
    // Create test with retry_index=0
    // Send step with retry_index=1 (attempt doesn't exist)
    // Verify attempt[1] auto-created and step persisted
}
```

---

## Conclusion

**Recommended Approach: Tier 1 + Tier 2**

1. **Make repository defensive** (auto-create missing attempts)
2. **Make consumer smart** (typed errors, exponential backoff, NAK with delay)
3. **Add observability** (metrics, logs for ordering issues)

This provides robust handling of race conditions without over-engineering. Tier 3 (buffering) only needed if seeing >5% ordering issues in production.

**Key Principle:** _Eventual consistency through intelligent retries_ rather than trying to prevent all race conditions.

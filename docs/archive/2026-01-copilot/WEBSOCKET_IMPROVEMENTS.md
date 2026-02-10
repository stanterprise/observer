# WebSocket Event Handling Improvements

## Current Issues Identified (January 5, 2026)

### 1. Event Type Coverage Gaps

**Backend → Frontend Mismatch:**

| Event Type     | Backend Publisher | Backend Consumer | WebSocket Normalization | Frontend Handler                 |
| -------------- | ----------------- | ---------------- | ----------------------- | -------------------------------- |
| `test.begin`   | ✅                | ✅               | ✅ Normalized           | ✅                               |
| `test.end`     | ✅                | ✅               | ✅ Normalized           | ✅                               |
| `run.start`    | ✅                | ✅               | ✅ Normalized           | ✅                               |
| `run.end`      | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `suite.begin`  | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `suite.end`    | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `step.begin`   | ✅                | ✅               | ⚠️ Pass-through         | ⚠️ Partial (TestDetailPage only) |
| `step.end`     | ✅                | ✅               | ⚠️ Pass-through         | ⚠️ Partial (TestDetailPage only) |
| `test.failure` | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `test.error`   | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `stdout`       | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `stderr`       | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |
| `heartbeat`    | ✅                | ✅               | ⚠️ Pass-through         | ❌ **Missing**                   |

### 2. Heavy Load Issues

#### Issue 2.1: Client Disconnection on Buffer Overflow

**Location:** `pkg/websocket/websocket.go:236`

```go
select {
case client.send <- message:
default:
    // Client's send channel is full, close and remove
    close(client.send)
    delete(h.clients, client)  // 🚨 Client dropped!
}
```

**Problem:** When a client's 256-message buffer fills (e.g., slow network, high event rate), the client is **disconnected** instead of applying backpressure or dropping old messages.

**Impact Under Heavy Load:**

- Multiple concurrent test runs (100+ tests/second)
- Slow client networks (mobile, remote connections)
- Result: Clients repeatedly disconnect/reconnect, losing real-time updates

#### Issue 2.2: Broadcast Channel Blocking

**Location:** `pkg/websocket/websocket.go:304`

```go
h.broadcast <- normalizedData  // Can block if hub is slow
```

**Problem:** NATS consumer goroutine blocks if hub's broadcast channel (256 buffer) is full.

**Impact:** NATS message processing stalls → messages accumulate in JetStream → increased latency → potential redelivery timeouts.

#### Issue 2.3: Mutex Contention

**Location:** `pkg/websocket/websocket.go:227-240`

```go
h.mu.RLock()
for client := range h.clients {
    if !client.matchesFilters(&event) {
        continue
    }
    select {
    case client.send <- message:
    default:
        close(client.send)
        delete(h.clients, client)
    }
}
h.mu.RUnlock()
```

**Problem:** Read lock held while:

1. Iterating all clients (O(n))
2. Filtering each event (JSON parsing + field extraction)
3. Attempting channel sends (can take microseconds if buffer nearly full)

With 50+ concurrent WebSocket clients, this creates lock contention with registration/unregistration operations.

#### Issue 2.4: No Backpressure Mechanism

**Current flow:**

```
NATS → [Fetch Batch: 10] → Normalize → [Broadcast: 256] → Hub → [Send: 256] → Client
```

**Problem:** No feedback loop from slow clients to NATS consumer. Fast events from NATS overwhelm slow WebSocket clients.

### 3. Missing Event Normalization

Only 3 of 13 event types are normalized to model-based JSON:

- `run.start`, `test.begin`, `test.end`

**Remaining events pass through raw protobuf JSON**, causing inconsistency with REST API responses (which use MongoDB document models).

**Example inconsistency:**

- REST API: `{ "runId": "abc", "testCaseId": "xyz", ... }` (camelCase, model fields)
- WebSocket (raw): `{ "run_id": "abc", "test_case_id": "xyz", ... }` (snake_case, protobuf fields)

## Recommended Fixes

### Priority 1: Fix Client Disconnection Under Load

#### Option A: Drop Old Messages (Preferred for Real-Time)

```go
select {
case client.send <- message:
default:
    // Channel full - drop oldest message and add new one
    select {
    case <-client.send: // Remove oldest
    default:
    }
    select {
    case client.send <- message: // Add new
    default:
        // Still full - log and continue (don't disconnect)
        h.logger.Warn("client buffer overflow", "client", client.conn.RemoteAddr())
    }
}
```

**Tradeoff:** Clients may miss some intermediate events but stay connected.

#### Option B: Configurable Buffer Sizes

Add configuration:

```go
type HubConfig struct {
    BroadcastBufferSize int  // Default: 256
    ClientBufferSize    int  // Default: 256
    DisconnectOnFull    bool // Default: false
}
```

Allow operators to tune buffer sizes based on load profile.

#### Option C: Per-Client Event Sampling

When buffer is at 80% capacity, start dropping non-critical events:

```go
if len(client.send) > int(float64(cap(client.send)) * 0.8) {
    // Under pressure - only send critical events
    if !isCriticalEvent(event.Type) {
        h.logger.Debug("dropping non-critical event", "type", event.Type)
        continue
    }
}

func isCriticalEvent(eventType publisher.EventType) bool {
    return eventType == publisher.EventTypeTestEnd ||
           eventType == publisher.EventTypeRunEnd ||
           eventType == publisher.EventTypeTestFailure
}
```

### Priority 2: Fix Broadcast Channel Blocking

#### Solution: Non-Blocking Broadcast

```go
// In NATS consumer goroutine
select {
case h.broadcast <- normalizedData:
    // Successfully queued
default:
    // Hub overwhelmed - log and drop
    h.logger.Warn("broadcast channel full, dropping event", "type", event.Type)
    // Still ACK the NATS message to avoid redelivery
}
```

Alternative: Increase broadcast buffer size based on expected load:

- Single-digit concurrent runs: 256 (current)
- High-throughput CI (10+ runs): 2048
- Enterprise scale (50+ runs): 8192

### Priority 3: Reduce Mutex Contention

#### Solution A: Per-Client Goroutines for Broadcasting

Instead of hub iterating clients, each client has a goroutine listening to a shared channel:

```go
type Hub struct {
    broadcast chan *BroadcastMessage
    // ... other fields
}

type BroadcastMessage struct {
    event   *publisher.Event
    payload []byte
}

// Client goroutine (in addition to read/write pumps)
func (c *Client) broadcastPump() {
    for msg := range c.hub.broadcast {
        // Each client filters independently (no central lock)
        if !c.matchesFilters(msg.event) {
            continue
        }

        select {
        case c.send <- msg.payload:
        default:
            // Apply buffer overflow strategy here
        }
    }
}
```

**Benefit:** No global iteration under lock. Filtering is parallelized across clients.

#### Solution B: Shard Clients by Filter

Group clients by their filters (e.g., by runID) to avoid broadcasting to all clients:

```go
type Hub struct {
    clientsByRunID map[string]map[*Client]bool
    allClients     map[*Client]bool  // Clients with no runID filter
    mu sync.RWMutex
}
```

**Benefit:** When broadcasting a `run.start` event for runID="abc", only send to clients filtered for "abc" or with no filter.

### Priority 4: Complete Event Normalization

Add model converters for remaining event types:

**Required functions in `pkg/websocket/proto_to_model.go`:**

```go
func protoToSuiteDocument(req *events.SuiteBeginEventRequest) *models.SuiteDocument
func protoToSuiteEndDocument(req *events.SuiteEndEventRequest) *models.SuiteDocument
func protoToStepDocument(req *events.StepBeginEventRequest) *models.StepDocument
func protoToStepEndDocument(req *events.StepEndEventRequest) *models.StepDocument
func protoToRunEndDocument(req *events.ReportRunEndEventRequest) *models.TestRunDocument
```

Update `normalizeEventData()` switch statement to handle all event types.

### Priority 5: Complete Frontend Event Handlers

Add handlers for missing event types in frontend components:

#### In `TestSuiteRunsPage.tsx`:

```typescript
if (type === "run.end") {
  handleRunEnd(data as WebSocketRunData, setRuns);
}

if (type === "suite.begin" || type === "suite.end") {
  handleSuiteEvent(data, type, setRuns);
}
```

#### In `TestRunDetailPage.tsx`:

```typescript
if (type === "run.end") {
  // Mark run as completed, update final statistics
}
```

#### In `TestDetailPage.tsx`:

```typescript
if (type === "test.failure" || type === "test.error") {
  // Display error details inline
}

if (type === "stdout" || type === "stderr") {
  // Append to test output log
}
```

### Priority 6: Add Monitoring Metrics

Instrument WebSocket hub with metrics:

```go
type HubMetrics struct {
    TotalEvents       int64
    DroppedEvents     int64
    ClientDisconnects int64
    BroadcastQueueSize int
    SlowClients       int64
}

// Periodically log or expose via /metrics endpoint
```

Use these metrics to:

- Alert on high drop rates
- Identify slow clients
- Tune buffer sizes
- Detect backpressure issues

## Implementation Plan

### Phase 1: Critical Fixes (Immediate)

1. ✅ Document current issues (this file)
2. Fix client disconnection strategy (Option A: drop old messages)
3. Fix broadcast blocking (non-blocking send with drop + log)
4. Add metrics logging

**Estimated effort:** 4-6 hours  
**Impact:** Prevents client disconnections under load

### Phase 2: Performance Optimization (Week 1)

1. Implement per-client broadcast goroutines (Solution A)
2. Increase buffer sizes with configuration
3. Add event sampling under pressure
4. Load test with 100+ concurrent clients

**Estimated effort:** 2-3 days  
**Impact:** 10x throughput improvement

### Phase 3: Complete Event Support (Week 2)

1. Add model converters for all event types
2. Update WebSocket normalization
3. Add frontend handlers for missing events
4. Update TypeScript types
5. End-to-end testing

**Estimated effort:** 3-4 days  
**Impact:** Full real-time observability

### Phase 4: Advanced Features (Week 3)

1. Client-side event acknowledgment
2. Automatic reconnection with replay from last received event
3. WebSocket compression (permessage-deflate)
4. Rate limiting per client

**Estimated effort:** 3-5 days  
**Impact:** Production-grade reliability

## Testing Strategy

### Load Testing

Create load test that simulates:

- 50 concurrent test runs
- 10,000 events/minute
- 20 WebSocket clients (mixed slow/fast)
- Measure: drop rate, latency, disconnections

```go
// tests/websocket_load_test.go
func TestWebSocketUnderHeavyLoad(t *testing.T) {
    // Start hub with default config
    // Connect 20 clients with varying simulated network delays
    // Publish 10k events over 1 minute
    // Assert:
    //   - Zero client disconnections
    //   - <5% event drops for slow clients
    //   - <100ms p99 latency
}
```

### Event Coverage Testing

```typescript
// web/src/__tests__/websocket-events.test.ts
describe('WebSocket Event Handlers', () => {
    it('handles all 13 event types', () => {
        const eventTypes = [
            'test.begin', 'test.end',
            'suite.begin', 'suite.end',
            'step.begin', 'step.end',
            'run.start', 'run.end',
            'test.failure', 'test.error',
            'stdout', 'stderr', 'heartbeat'
        ];

        eventTypes.forEach(type => {
            const result = handleWebSocketEvent({ type, ... });
            expect(result).toBeDefined();
        });
    });
});
```

## References

- [WEBSOCKET_IMPLEMENTATION.md](./WEBSOCKET_IMPLEMENTATION.md) - Original implementation doc
- [WEBSOCKET_FILTERS.md](./WEBSOCKET_FILTERS.md) - Filter specification
- [pkg/websocket/websocket.go](../pkg/websocket/websocket.go) - Current implementation
- [web/src/hooks/useWebSocket.ts](../web/src/hooks/useWebSocket.ts) - Frontend hook

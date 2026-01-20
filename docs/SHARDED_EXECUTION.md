# Sharded Test Execution in Observer

## Overview

Observer supports **sharded test execution** where multiple test runners execute tests in parallel and report results for the same test run. This enables:

- ✅ Faster CI/CD pipelines through parallel test execution
- ✅ Distributed testing across multiple workers
- ✅ Playwright sharded runs (`--shard=X/Y`)
- ✅ Any test framework supporting test splitting

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Test Orchestrator                        │
│               (CI/CD, Kubernetes, etc.)                     │
└──────┬─────────────────┬──────────────────┬─────────────────┘
       │                 │                  │
       ▼                 ▼                  ▼
  ┌────────┐        ┌────────┐        ┌────────┐
  │ Shard 1│        │ Shard 2│        │ Shard 3│
  │ 50 tests│       │ 30 tests│       │ 20 tests│
  └────┬───┘        └────┬───┘        └────┬───┘
       │                 │                  │
       │ RunStart(       │ RunStart(        │ RunStart(
       │  run_id=123,    │  run_id=123,     │  run_id=123,
       │  total=50,      │  total=30,       │  total=20,
       │  suites=[A,B])  │  suites=[C,D])   │  suites=[E])
       │                 │                  │
       └─────────────────┴──────────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │  Observer gRPC   │
              │  Ingestion       │
              └────────┬─────────┘
                       │
                       ▼
              ┌──────────────────┐
              │  NATS JetStream  │
              └────────┬─────────┘
                       │
                       ▼
              ┌──────────────────┐
              │  Processor       │
              │  (MapSuites)     │
              └────────┬─────────┘
                       │
                       ▼
              ┌──────────────────┐
              │  MongoDB         │
              │  ┌────────────┐  │
              │  │ run_id: 123│  │
              │  │ total: 100 │  │ ← Accumulated!
              │  │ shards: 3  │  │ ← Counted!
              │  │ suites:    │  │
              │  │  [A,B,C,D,E]│  │ ← Merged!
              │  └────────────┘  │
              └──────────────────┘
```

## How It Works

### 1. Accumulative Total Tests

Observer uses MongoDB's `$inc` operator to accumulate test counts:

```go
// Each shard's total_tests is added to the run
if totalTests > 0 {
    runUpdate["$inc"].(bson.M)["total_tests"] = totalTests
    runUpdate["$inc"].(bson.M)["shard_count"] = 1
}
```

**Result:**
- Shard 1: 50 tests → `total_tests = 50`
- Shard 2: 30 tests → `total_tests = 80` 
- Shard 3: 20 tests → `total_tests = 100` ✅

### 2. Suite Accumulation

Suites from all shards are appended to the run document:

```go
update := bson.M{
    "$push": bson.M{"suites": bson.M{"$each": suites}},
}
```

**Result:**
- Shard 1: [A, B] → `suites = [A, B]`
- Shard 2: [C, D] → `suites = [A, B, C, D]`
- Shard 3: [E] → `suites = [A, B, C, D, E]` ✅

### 3. Metadata Merge Strategy

Metadata keys are merged individually (last write wins per key):

```go
// Key-level merge preserves unique keys across shards
for k, v := range metadata {
    runUpdate["$set"].(bson.M)[fmt.Sprintf("metadata.%s", k)] = v
}
```

**Example:**
```javascript
// Shard 1 metadata
{ "environment": "ci", "browser": "chromium", "shard_id": "1" }

// Shard 2 metadata
{ "environment": "ci", "browser": "firefox", "shard_id": "2" }

// Final merged metadata (last write wins)
{ "environment": "ci", "browser": "firefox", "shard_id": "2" }
```

**Tip:** Use unique keys per shard to avoid conflicts:
```javascript
{ 
  "environment": "ci",
  "shard_1_browser": "chromium",
  "shard_2_browser": "firefox"
}
```

## Usage Examples

### Playwright Sharded Tests

```bash
# GitHub Actions workflow
jobs:
  test:
    strategy:
      matrix:
        shard: [1, 2, 3]
    steps:
      - name: Run tests
        env:
          RUN_ID: ${{ github.run_id }}-${{ github.run_attempt }}
        run: |
          playwright test --shard=${{ matrix.shard }}/3 \
            --reporter=@stanterprise/playwright-reporter
```

### Manual Sharded Execution

```bash
# Terminal 1 - Shard 1
RUN_ID="test-run-123" playwright test --shard=1/3

# Terminal 2 - Shard 2
RUN_ID="test-run-123" playwright test --shard=2/3

# Terminal 3 - Shard 3
RUN_ID="test-run-123" playwright test --shard=3/3
```

All shards report to the same `run_id`, and Observer accumulates the results automatically.

### Custom Test Framework

Configure your test reporter to:

1. **Use the same `run_id`** across all shards
2. **Send `ReportRunStart`** with shard-specific metadata
3. **Report total test count** for the shard

```javascript
// Shard 1 configuration
const reporter = new ObserverReporter({
  runId: process.env.RUN_ID,
  totalTests: 50,
  metadata: {
    shard_id: "shard-1",
    worker_id: "worker-1"
  }
});
```

## Verification

Query the MongoDB database to see accumulated results:

```bash
# Connect to MongoDB
make mongodb-shell

# Query a test run
db.test_runs.findOne({"_id": "test-run-123"})
```

**Expected output:**

```json
{
  "_id": "test-run-123",
  "name": "CI Test Run",
  "total_tests": 100,
  "shard_count": 3,
  "suites": [
    { "id": "suite-A", "name": "Suite A", ... },
    { "id": "suite-B", "name": "Suite B", ... },
    { "id": "suite-C", "name": "Suite C", ... },
    { "id": "suite-D", "name": "Suite D", ... },
    { "id": "suite-E", "name": "Suite E", ... }
  ],
  "metadata": {
    "environment": "ci",
    "shard_id": "shard-3"
  }
}
```

## Idempotency

The implementation is **idempotent** - replaying the same event is safe:

- ✅ Duplicate suite IDs will be appended (not deduplicated)
- ✅ Total tests will accumulate again
- ✅ Shard count will increment again

**Important:** Ensure your test framework doesn't retry/replay run start events unless intentional. Use NATS JetStream acknowledgment to prevent message redelivery.

## Limitations

1. **Suite deduplication not implemented** - Replaying the same run start will append duplicate suites
2. **Metadata last-write-wins** - Conflicting keys will use the last shard's value
3. **No shard completion tracking** - Observer doesn't know if all expected shards have reported

## Testing

Run the sharded execution tests:

```bash
# Unit tests (no containers needed)
go test -v ./tests/ -run TestShardedExecution_Idempotent
go test -v ./tests/ -run TestShardedExecution_ZeroTotalTests

# Integration test (requires Docker)
go test -v ./tests/ -run TestShardedExecution_MultipleRunStarts -timeout 5m
```

Expected output:
```
✓ TestShardedExecution_MultipleRunStarts (11.63s)
✓ TestShardedExecution_IdempotentRunStart (0.67s)
✓ TestShardedExecution_ZeroTotalTests (0.71s)
```

## Implementation Details

### Modified Files

1. **`internal/models/document.go`**
   - Added `ShardCount int32` field to track shards

2. **`internal/repository/mongodb_map.go`**
   - Changed `total_tests` from `$set` to `$inc`
   - Added `shard_count` increment
   - Implemented key-level metadata merge

3. **`pkg/consumer/event_classifier.go`**
   - Updated comments to reflect idempotent run start behavior

4. **`tests/sharded_execution_test.go`**
   - Added comprehensive integration tests

### Key MongoDB Operations

**Accumulation:**
```go
runUpdate["$inc"].(bson.M)["total_tests"] = totalTests
runUpdate["$inc"].(bson.M)["shard_count"] = 1
```

**Metadata merge:**
```go
for k, v := range metadata {
    runUpdate["$set"].(bson.M)[fmt.Sprintf("metadata.%s", k)] = v
}
```

**Suite append:**
```go
update := bson.M{
    "$push": bson.M{"suites": bson.M{"$each": suites}},
}
```

## Best Practices

1. **Use unique shard IDs** in metadata to track which shards reported
2. **Set consistent run IDs** across all shards (use CI build ID)
3. **Coordinate run end event** - only one shard should send `ReportRunEnd`
4. **Monitor shard_count** to verify all expected shards reported
5. **Use metadata to track shard configuration** (browsers, workers, etc.)

## Troubleshooting

**Problem:** Total tests don't match expected value

- Check that all shards are using the same `run_id`
- Verify each shard is reporting its correct `total_tests`
- Look for duplicate run start events (idempotency will accumulate again)

**Problem:** Missing suites

- Check NATS consumer logs for processing errors
- Verify MongoDB connection is stable
- Ensure all shards successfully sent their run start events

**Problem:** Unexpected shard_count

- Zero total_tests don't increment shard_count
- Check if shards are sending `total_tests > 0`
- Verify no duplicate/retry events

## Future Enhancements

Potential improvements:

- [ ] Suite deduplication based on suite ID
- [ ] Expected shard count validation
- [ ] Shard completion tracking
- [ ] Shard-specific metadata namespacing
- [ ] Configurable metadata merge strategies
- [ ] Web UI visualization of shard distribution

## References

- [MongoDB Repository Architecture](./MONGODB_REPOSITORY_ARCHITECTURE.md)
- [README - Sharded Test Execution](../README.md#sharded-test-execution)
- [Test Suite](../tests/sharded_execution_test.go)

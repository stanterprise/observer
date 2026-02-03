# Attempt-Based Retries Implementation - Summary

## Overview

Successfully implemented attempt-based retry architecture for the Observer test system. Tests now maintain an `attempts` array where each retry attempt stores its own steps, failures, errors, and execution metadata.

## Implementation Complete ✅

### 1. Models Updated ([internal/models/document.go](internal/models/document.go))

**New Structure: AttemptDocument**

```go
type AttemptDocument struct {
    RetryIndex    int32              `bson:"retry_index"`
    Steps         []*StepDocument    `bson:"steps"`
    Status        string             `bson:"status"`
    StartTime     *time.Time         `bson:"start_time"`
    EndTime       *time.Time         `bson:"end_time"`
    Duration      *int64             `bson:"duration"`
    Attachments   []string           `bson:"attachments"`
    ErrorMessage  string             `bson:"error_message"`
    StackTrace    string             `bson:"stack_trace"`
    ErrorList     []string           `bson:"error_list"`
    Failures      []interface{}      `bson:"failures"`
    Errors        []interface{}      `bson:"errors"`
    StdOut        string             `bson:"stdout"`
    StdErr        string             `bson:"stderr"`
    CreatedAt     time.Time          `bson:"created_at"`
    UpdatedAt     time.Time          `bson:"updated_at"`
}
```

**Updated TestDocument**

- Added `Attempts []*AttemptDocument` field
- Marked legacy fields as DEPRECATED (Steps, Attachments, ErrorMessage, etc.)
- Test-level Status/StartTime/EndTime/Duration now represent aggregated values

### 2. Repository Functions Updated

#### Test Functions ([internal/repository/mongodb_testcase.go](internal/repository/mongodb_testcase.go))

**UpsertTestBegin:**

- Initializes `attempts` array sized to `retry_count + 1`
- Each attempt pre-initialized with `retry_index` and empty `steps` array
- Sets `start_time` and `status` for `attempts[retry_index]`

**UpsertTestEnd (BREAKING CHANGE):**

- New signature: `UpsertTestEnd(ctx, runID, testID, status, retryIndex, endTime, duration)`
- Updates BOTH test-level status AND `attempts[retry_index]` status/end_time/duration
- Uses literal index notation: `tests.$[test].attempts.%d.status`

#### Step Functions ([internal/repository/mongodb_step.go](internal/repository/mongodb_step.go))

**Complete rewrite - Clean 176-line implementation:**

- `UpsertStepBegin`: Creates/updates steps in `attempts[retry_index].steps`
- `upsertStepInTestAttempt`: Helper using literal index paths like `tests.$[test].attempts.%d.steps`
- `UpsertStepEnd` (BREAKING CHANGE): New signature includes `retry_index` parameter

**MongoDB Update Pattern:**

```go
// Uses literal indices instead of positional operators
stepPath := fmt.Sprintf("tests.$[test].attempts.%d.steps.$[step]", retry_index)
```

#### Helper Functions ([internal/repository/mongodb_helpers.go](internal/repository/mongodb_helpers.go))

**AppendTestFailure (BREAKING CHANGE):**

- New signature: `AppendTestFailure(ctx, runID, testID, retryIndex, failure)`
- Targets: `tests.$[test].attempts.%d.failures`

**AppendTestError (BREAKING CHANGE):**

- New signature: `AppendTestError(ctx, runID, testID, retryIndex, error)`
- Targets: `tests.$[test].attempts.%d.errors`

### 3. Consumer Handlers Updated

#### Test Handlers ([pkg/consumer/nats_test_handlers.go](pkg/consumer/nats_test_handlers.go))

**handleTestEnd:**

- Extracts `endTime` from protobuf: `req.TestCase.EndTime.AsTime()`
- Passes `endTime` to `UpsertTestEnd` along with `retry_index`

**handleTestFailure:**

- Fetches test document using `GetTestFromRun(ctx, req.TestId)`
- Extracts `retry_index` from fetched document (defaults to 0 if nil)
- Passes `retry_index` to `AppendTestFailure`

**handleTestError:**

- Fetches test document using `GetTestFromRun(ctx, req.TestId)`
- Extracts `retry_index` from fetched document (defaults to 0 if nil)
- Passes `retry_index` to `AppendTestError`

**Rationale:** TestFailure/TestError protobuf events don't include `retry_index`, so must query database

#### Step Handlers ([pkg/consumer/nats_step_handlers.go](pkg/consumer/nats_step_handlers.go))

**handleStepEnd:**

- Extracts `retry_index` from protobuf: `req.Step.RetryIndex`
- Passes `retry_index` to `UpsertStepEnd`
- Added logging: `"retryIndex", retryIndex`

### 4. Tests Fixed ([tests/nats_mongodb_integration_test.go](tests/nats_mongodb_integration_test.go))

Updated function calls to match new signatures:

- `UpsertStepEnd(ctx, runID, stepID, testID, 0, "PASSED")` - added retry_index=0
- `UpsertTestEnd(ctx, runID, testID, "PASSED", 0, &testEndTime, &testDuration)` - added endTime parameter

## Design Decisions

1. **Single Document Structure**: One test document with `attempts` array (not separate documents per retry)
2. **Test Status = Current Attempt Status**: Test-level status mirrors `attempts[retry_index].status`
3. **Failures/Errors per Attempt**: Moved from test-level to attempt-level arrays
4. **Attachments/Output per Attempt**: Each attempt has own attachments, stdout, stderr
5. **Test-Level Timing**:
   - `start_time`: Earliest start from first attempt
   - `end_time`: Latest end from current attempt
   - `duration`: Duration of current attempt
6. **Literal Index Notation**: MongoDB paths use `attempts.0.steps` instead of positional operators (simpler, faster)
7. **Backward Compatibility**: Legacy fields kept on TestDocument marked as DEPRECATED

## Breaking Changes

### Function Signatures Changed

**Repository:**

```go
// BEFORE
UpsertTestEnd(ctx, runID, testID, status, retryIndex, duration)
UpsertStepEnd(ctx, runID, stepID, testID, status)
AppendTestFailure(ctx, runID, testID, failure)
AppendTestError(ctx, runID, testID, error)

// AFTER
UpsertTestEnd(ctx, runID, testID, status, retryIndex, endTime, duration)
UpsertStepEnd(ctx, runID, stepID, testID, retry_index, status)
AppendTestFailure(ctx, runID, testID, retryIndex, failure)
AppendTestError(ctx, runID, testID, retryIndex, error)
```

## Files Modified

### Core Implementation

- ✅ `/internal/models/document.go` - Added AttemptDocument, updated TestDocument
- ✅ `/internal/repository/mongodb_testcase.go` - Updated test begin/end logic
- ✅ `/internal/repository/mongodb_step.go` - Complete rewrite for attempt-based architecture
- ✅ `/internal/repository/mongodb_helpers.go` - Updated failure/error helpers

### Consumer Layer

- ✅ `/pkg/consumer/nats_test_handlers.go` - Updated test event handlers
- ✅ `/pkg/consumer/nats_step_handlers.go` - Updated step event handlers

### Tests

- ✅ `/tests/nats_mongodb_integration_test.go` - Fixed function signatures

## Cleanup Performed

**Deleted corrupted files:**

- `mongodb_step_new.go` - Duplicate file causing compilation errors
- Corrupted version of `mongodb_step.go` - Had duplicate functions and syntax errors

**Final clean state:**

- `mongodb_step.go` - 176 lines with 3 functions only

## MongoDB Schema Example

```json
{
  "_id": "run-123",
  "tests": [
    {
      "id": "test-456",
      "title": "My Test",
      "retry_count": 2,
      "retry_index": 1,
      "status": "FAILED",           // Current attempt status
      "start_time": "2026-01-16T10:00:00Z",  // First attempt start
      "end_time": "2026-01-16T10:05:00Z",    // Current attempt end
      "duration": 300000,                     // Current attempt duration
      "attempts": [
        {
          "retry_index": 0,
          "steps": [...],
          "status": "FAILED",
          "start_time": "2026-01-16T10:00:00Z",
          "end_time": "2026-01-16T10:02:00Z",
          "duration": 120000,
          "failures": [...],
          "errors": [...]
        },
        {
          "retry_index": 1,
          "steps": [...],
          "status": "FAILED",
          "start_time": "2026-01-16T10:03:00Z",
          "end_time": "2026-01-16T10:05:00Z",
          "duration": 120000,
          "failures": [...],
          "errors": [...]
        }
      ],
      // DEPRECATED FIELDS (for backward compatibility)
      "steps": [],          // Use attempts[retry_index].steps instead
      "attachments": [],    // Use attempts[retry_index].attachments instead
      "error_message": "",  // Use attempts[retry_index].error_message instead
      "failures": [],       // Use attempts[retry_index].failures instead
      "errors": []          // Use attempts[retry_index].errors instead
    }
  ]
}
```

## Next Steps

### Immediate Verification

```bash
# 1. Build all components
make build-all

# 2. Run unit tests
make test

# 3. Start infrastructure
make mongo-up nats-up

# 4. Run integration tests
make test-nats-integration

# 5. Start services
NATS_URL=nats://localhost:4222 ./bin/ingestion
MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
  NATS_URL=nats://localhost:4222 ./bin/processor
MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
  NATS_URL=nats://localhost:4222 ./bin/api
```

### Future Enhancements

1. Update API endpoints to expose `attempts` array structure
2. Update Web UI to display retry attempts separately
3. Add query functions to fetch specific attempts
4. Migrate legacy data from deprecated fields to attempts array
5. Add MongoDB indexes on `attempts.retry_index` for performance
6. Update GraphQL schema (when implemented) to include attempts

## Migration Notes

**For existing data:**

- Legacy fields remain populated on TestDocument for backward compatibility
- New data will populate `attempts` array
- Consider running migration script to move legacy data into attempts[0]
- API consumers should check for both formats during transition period

**For new features:**

- Always read from `attempts[retry_index]` arrays
- Always write to `attempts[retry_index]` arrays
- Maintain test-level aggregated values for summary views

## Testing Strategy

1. **Unit Tests**: Verify attempt array initialization and updates
2. **Integration Tests**: Validate full retry flow with multiple attempts
3. **E2E Tests**: Run Playwright tests with retries enabled
4. **Performance Tests**: Verify MongoDB query performance with nested arrays

## Success Criteria ✅

- [x] All MongoDB repository functions updated
- [x] All consumer handlers updated
- [x] Test signatures fixed
- [x] File corruption cleaned up
- [x] Compilation successful
- [ ] All tests passing (pending verification)
- [ ] Integration tests with retries (pending verification)

---

**Implementation Date:** January 16, 2026  
**Status:** Implementation Complete - Verification Pending

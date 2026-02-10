# Attempt-Based Test Retries Implementation Plan

## Overview

This document describes the implementation of the new attempt-based retry structure for tests, where each test contains an `attempts` array and steps are nested within the appropriate attempt.

## Architectural Changes

### Current Structure

```
TestDocument {
  id, name, status, ...
  retry_count, retry_index
  steps: []StepDocument
}
```

### New Structure

```
TestDocument {
  id, name, status, ...
  retry_count, retry_index
  attempts: []AttemptDocument  // NEW: Array sized by retry_count
  steps: []StepDocument        // DEPRECATED: Kept for backwards compatibility
}

AttemptDocument {
  steps: []StepDocument
  status
  start_time, end_time, duration
  attachments
  error_message, stack_trace, error_list
  updated_at
}
```

## Implementation Steps

### 1. Model Changes (`internal/models/document.go`)

**Add new AttemptDocument struct** before TestDocument:

```go
// AttemptDocument represents a single test attempt/retry.
// Each test can have multiple attempts based on retry_count.
type AttemptDocument struct {
	Steps        []*StepDocument          `bson:"steps,omitempty" json:"steps,omitempty"`
	Status       string                   `bson:"status,omitempty" json:"status,omitempty"`
	StartTime    *time.Time               `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime      *time.Time               `bson:"end_time,omitempty" json:"endTime,omitempty"`
	Duration     *int64                   `bson:"duration,omitempty" json:"duration,omitempty"`
	Attachments  []map[string]interface{} `bson:"attachments,omitempty" json:"attachments,omitempty"`
	ErrorMessage string                   `bson:"error_message,omitempty" json:"errorMessage,omitempty"`
	StackTrace   string                   `bson:"stack_trace,omitempty" json:"stackTrace,omitempty"`
	ErrorList    []string                 `bson:"error_list,omitempty" json:"errorList,omitempty"`
	UpdatedAt    time.Time                `bson:"updated_at" json:"updatedAt"`
}
```

**Update TestDocument** to add:

```go
// NEW: Attempts array containing all test attempts (including retries)
// Each attempt contains its own steps, status, timing, and error information
Attempts []*AttemptDocument `bson:"attempts,omitempty" json:"attempts,omitempty"`

// DEPRECATED: Steps field kept for backward compatibility but no longer used
// Steps are now stored in Attempts[retry_index].Steps
Steps []*StepDocument `bson:"steps,omitempty" json:"steps,omitempty"`
```

### 2. Repository Changes

#### `UpsertTestBegin` (`internal/repository/mongodb_testcase.go`)

**Current behavior**: Creates/updates test document with flat steps array

**New behavior**:

- Initialize `attempts` array with size = `retry_count + 1` (default 1)
- Each attempt starts with empty steps array and no status
- Initialize start_time for attempt[retry_index]

**Key changes**:

1. After ensuring retry_count and retry_index are set
2. Create attempts array:

```go
attempts := make([]*m.AttemptDocument, *test.RetryCount+1)
for i := range attempts {
	now := time.Now()
	attempts[i] = &m.AttemptDocument{
		Steps:     []*m.StepDocument{},
		UpdatedAt: now,
	}
	if int32(i) == *test.RetryIndex {
		attempts[i].StartTime = test.StartTime
	}
}
test.Attempts = attempts
```

#### `UpsertTestEnd` (`internal/repository/mongodb_testcase.go`)

**Current behavior**: Updates test-level status and duration

**New behavior**:

- Update both test-level status AND attempt-level status
- Set attempt[retry_index] status, end_time, duration
- Set test-level status (aggregate of all attempts)

**Key changes**:

1. Update attempt status:

```go
filter := bson.M{
	"_id":               runID,
	"tests.id":          testID,
	"tests.retry_index": retryIndex,
}

// Build update for both test and attempt
setFields := bson.M{
	"updated_at":                                   now,
	"tests.$[test].status":                         status,
	"tests.$[test].duration":                       duration,
	"tests.$[test].updated_at":                     now,
	fmt.Sprintf("tests.$[test].attempts.%d.status", retryIndex):   status,
	fmt.Sprintf("tests.$[test].attempts.%d.duration", retryIndex): duration,
	fmt.Sprintf("tests.$[test].attempts.%d.updated_at", retryIndex): now,
}
```

#### `UpsertStepBegin` (`internal/repository/mongodb_step.go`)

**Current behavior**: Upserts step in flat `tests.steps` array

**New behavior**:

- Upsert step in `tests.attempts[retry_index].steps` array
- Use `retry_index` parameter to target correct attempt

**Key changes**:

```go
// Try to update existing step in attempt
filter := bson.M{
	"_id":                             runID,
	"tests.id":                        testID,
	"tests.retry_index":               retry_index,
	fmt.Sprintf("tests.attempts.%d.steps.id", retry_index): step.ID,
}

update := bson.M{
	"$set": bson.M{
		fmt.Sprintf("tests.$[test].attempts.%d.steps.$[step].title", retry_index):       step.Title,
		fmt.Sprintf("tests.$[test].attempts.%d.steps.$[step].description", retry_index): step.Description,
		// ... other fields
		"updated_at": now,
	},
}

// If not found, append to attempt's steps
filter = bson.M{
	"_id":               runID,
	"tests.id":          testID,
	"tests.retry_index": retry_index,
}
update = bson.M{
	"$push": bson.M{
		fmt.Sprintf("tests.$[test].attempts.%d.steps", retry_index): step,
	},
	"$set": bson.M{"updated_at": now},
}
```

#### `UpsertStepEnd` (`internal/repository/mongodb_step.go`)

**Current behavior**: Updates step in flat `tests.steps` array

**New behavior**:

- Update step in `tests.attempts[retry_index].steps` array
- Need to add retry_index parameter (currently missing)

**Key changes**:

```go
// Function signature change
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, retryIndex int32, status string) error

// Update step in attempt
filter := bson.M{
	"_id":                             runID,
	"tests.id":                        testID,
	"tests.retry_index":               retryIndex,
	fmt.Sprintf("tests.attempts.%d.steps.id", retryIndex): stepID,
}

setFields := bson.M{
	fmt.Sprintf("tests.$[test].attempts.%d.steps.$[step].status", retryIndex):     status,
	fmt.Sprintf("tests.$[test].attempts.%d.steps.$[step].updated_at", retryIndex): now,
	"updated_at": now,
}
```

### 3. Consumer Changes (`pkg/consumer/nats_step_handlers.go`)

**Update handleStepEnd** to extract and pass retry_index:

```go
func (c *MongoNATSConsumer) handleStepEnd(ctx context.Context, data json.RawMessage) error {
	// ... existing unmarshaling ...

	retryIndex := int32(0) // Extract from req.Step if available
	if req.Step.RetryIndex != nil {
		retryIndex = *req.Step.RetryIndex
	}

	return c.repo.UpsertStepEnd(ctx, runID, req.Step.Id, testID, retryIndex, status)
}
```

### 4. Testing Strategy

#### Unit Tests

1. **Model Tests** (`internal/models/document_test.go`):
   - Add `TestAttemptDocument_Fields`
   - Update `TestTestDocument_Fields` to verify Attempts array
2. **Repository Tests** (`internal/repository/mongodb_testcase_test.go`):
   - Test `UpsertTestBegin` creates correct attempts array size
   - Test `UpsertTestEnd` updates both test and attempt status
   - Test retry scenarios with retry_index > 0
3. **Step Repository Tests** (`internal/repository/mongodb_step_test.go`):
   - Test steps are inserted into correct attempt
   - Test step updates target correct attempt
   - Test multiple attempts with different steps

#### Integration Tests

Update `tests/nats_mongodb_integration_test.go`:

- Test full flow with retry_count > 0
- Verify attempts array structure in final document
- Test steps nested in correct attempts

## Migration Path

### Backward Compatibility

1. **Keep `Steps` field** on TestDocument for backward compatibility
2. **API layer** should check both `Attempts` and `Steps` when reading
3. **Query updates** may need to search both locations temporarily

### Data Migration Script

For existing data (if needed):

```javascript
db.test_runs.updateMany({ "tests.steps": { $exists: true, $ne: [] } }, [
  {
    $set: {
      tests: {
        $map: {
          input: "$tests",
          as: "test",
          in: {
            $mergeObjects: [
              "$$test",
              {
                attempts: [
                  {
                    steps: "$$test.steps",
                    status: "$$test.status",
                    start_time: "$$test.start_time",
                    end_time: "$$test.end_time",
                    duration: "$$test.duration",
                    updated_at: "$$test.updated_at",
                  },
                ],
              },
            ],
          },
        },
      },
    },
  },
]);
```

## Rollout Plan

1. ✅ Update models with Attempts structure
2. ✅ Update repository UpsertTestBegin
3. ✅ Update repository UpsertTestEnd
4. ✅ Update repository UpsertStepBegin
5. ✅ Update repository UpsertStepEnd signature + implementation
6. ✅ Update consumer step handlers
7. ✅ Add/update tests
8. ✅ Test with protobuf reporter
9. ✅ Monitor and validate in production
10. (Future) Remove deprecated Steps field

## Breaking Changes

- `UpsertStepEnd` signature changed to include `retryIndex` parameter
- Consumers of TestDocument need to read from `Attempts` array
- Web UI needs updates to display attempts structure

## Benefits

1. **Clear retry semantics**: Each attempt is isolated with its own steps and status
2. **Better debugging**: Can see all retry attempts and their individual failures
3. **Accurate reporting**: Can report on first-attempt vs retry success rates
4. **Scalability**: Supports N retries without structural changes

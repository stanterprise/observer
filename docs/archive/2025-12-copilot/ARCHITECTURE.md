# MongoDB Repository Architecture (Simplified)

## Design Principles

### 1. Document Identification

- **Every document is identified by its `runID`** (stored as `_id` in MongoDB)
- No complex ID extraction or manipulation logic
- No cross-document operations ever

### 2. Entity Operations

- **Every entity (suite, test, step) has its own unique ID**
- Operations: Find entity by ID within the document, then update or insert
- Recursive search through nested structures using MongoDB aggregation pipeline
- Single atomic operation per request

### 3. Method Signatures

All upsert methods follow this pattern:

```go
func (r *MongoRepository) UpsertEntityBegin(ctx context.Context, runID string, entity *EntityDocument, parentID string) error
```

- `runID`: Identifies the document (required, error if empty)
- `entity`: The entity to upsert (with its own ID)
- `parentID`: Parent entity ID (empty string for root-level entities)

### 4. Operation Flow

**For Begin Events:**

1. Validate `runID` is not empty → error if missing
2. Find document by `_id == runID`
3. Search for entity by `entity.ID` recursively
4. If found: Update entity fields (preserving children)
5. If not found: Insert entity into parent's array

**For End Events:**

1. Validate `runID` is not empty → error if missing
2. Find document by `_id == runID`
3. Search for entity by `entityID` recursively
4. Update only status/duration/endTime fields

### 5. No More Complex Helpers

- Remove `extractRootSuiteID()` - no ID manipulation
- Remove complex filters with multiple conditions
- Remove `UpdateMany()` calls - always `UpdateOne()` with `_id`

### 6. Error Handling

- Empty `runID` → return error immediately
- Document not found → return specific error
- Entity not found for End event → return specific error
- All errors include context (runID, entityID, operation)

## Implementation Strategy

### Phase 1: Core Helper Functions

Create recursive search/update functions using MongoDB aggregation pipeline:

- `findAndUpdateEntity(ctx, runID, entityID, updateFields, entityType)`
- `findAndInsertEntity(ctx, runID, parentID, entity, entityType)`

### Phase 2: Refactor Operations

Simplify each operation file:

- `mongodb_suite.go`: Suite begin/end
- `mongodb_testcase.go`: Test begin/end
- `mongodb_step.go`: Step begin/end

### Phase 3: Update Consumer

Ensure all consumer handlers extract and pass `runID`:

- Suite events: `suite.RunId` or `suite.Id` for root
- Test events: `test.RunId`
- Step events: `step.RunId`

### Phase 4: Testing

Update all tests to pass explicit `runID` in all calls.

## Benefits

1. **Correctness**: No cross-document mutations possible
2. **Simplicity**: Single responsibility per method
3. **Debuggability**: Clear error messages with context
4. **Performance**: Single atomic operation per request
5. **Maintainability**: No complex ID extraction logic

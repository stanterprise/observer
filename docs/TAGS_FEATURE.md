# Tags Feature Documentation

## Overview

The Observer system now supports displaying tags on all test entities (Test Runs, Test Suites, Tests, and Steps). Tags provide a flexible way to categorize and organize test executions, making it easier to filter and identify specific test scenarios.

## Implementation Status

### ✅ Completed

#### Backend (Go)
- **Data Models**: Tags field added to all document models:
  - `TestDocument` (already existed)
  - `SuiteDocument` (newly added)
  - `StepDocument` (newly added)

- **Database**: MongoDB schema supports tags on all entities via BSON tags:
  ```go
  Tags []string `bson:"tags,omitempty" json:"tags,omitempty"`
  ```

- **Consumer Handlers**: 
  - Test handler processes tags from protobuf `TestCaseRun.Tags` ✅
  - Suite handler ready for tags (commented, awaiting protobuf update) 🔜
  - Step handler ready for tags (commented, awaiting protobuf update) 🔜

#### Frontend (React/TypeScript)
- **Type Definitions**: Tags added to all TypeScript interfaces:
  - `Test` interface in `types/testCase.ts`
  - `TestSuite` interface in `types/testSuite.ts`
  - `Step` interface in `types/testCase.ts`

- **UI Components**: Tags displayed across all entity views:
  - `TagList` component: Reusable component for rendering tag badges
  - `TestCaseRecord`: Shows tags on test cards
  - `TestSuiteRecord`: Shows tags on suite headers
  - `Step` component: Shows tags on step cards
  - `TestDetailPage`: Shows tags in test detail view

### 🔜 Future Work

#### Protobuf Schema Updates
Currently, only `TestCaseRun` in the protobuf schema includes a `Tags` field. To fully enable tags for suites and steps:

1. Update `proto-go` repository to add tags to:
   - `TestSuiteRun` message
   - `StepRun` message

2. Update Observer consumer handlers:
   - Uncomment `Tags: req.Suite.Tags` in `nats_suite_handlers.go`
   - Uncomment `Tags: req.Step.Tags` in `nats_step_handlers.go`

3. Update Playwright reporter to send tags for suites and steps

## Usage

### Sending Tags from Test Reporters

#### For Tests (Currently Supported)
```typescript
// In Playwright reporter
const testCase = {
  id: "test-123",
  name: "Login test",
  tags: ["@smoke", "@authentication", "@critical"],
  // ... other fields
};
```

#### For Suites and Steps (Ready for Future Use)
Once protobuf is updated, tags can be sent similarly:
```typescript
const suite = {
  id: "suite-456",
  name: "Authentication Suite",
  tags: ["@auth", "@api"],
  // ... other fields
};

const step = {
  id: "step-789",
  title: "Click login button",
  tags: ["@ui", "@interaction"],
  // ... other fields
};
```

### Viewing Tags in UI

Tags are automatically displayed in:

1. **Test Run Detail Page**: Tags appear below test metadata in test cards
2. **Test Suite Headers**: Tags appear below suite names
3. **Step Cards**: Tags appear below step titles
4. **Test Detail Page**: Tags appear in a dedicated "Tags" section

### Tag Display Format

Tags are rendered as pill-shaped badges with:
- Blue background (`bg-blue-100`)
- Blue text (`text-blue-800`)
- Blue border (`border-blue-200`)
- Icon indicator (tag icon from lucide-react)

Example:
```
🏷️ @smoke  @critical  @api
```

## API Response Format

Tags are included in JSON responses from the API:

```json
{
  "id": "test-123",
  "title": "Login test",
  "tags": ["@smoke", "@authentication", "@critical"],
  "status": "PASSED",
  ...
}
```

## Database Schema

MongoDB documents store tags as arrays:

```javascript
{
  "_id": "test-123",
  "name": "Login test",
  "tags": ["@smoke", "@authentication", "@critical"],
  "status": "PASSED",
  ...
}
```

## Code Locations

### Backend
- Models: `internal/models/document.go`
- Test handler: `pkg/consumer/nats_test_handlers.go` (line 76)
- Suite handler: `pkg/consumer/nats_suite_handlers.go` (line 76, commented)
- Step handler: `pkg/consumer/nats_step_handlers.go` (line 60, commented)

### Frontend
- Component: `web/src/components/TagList.tsx`
- Types: `web/src/types/testCase.ts`, `web/src/types/testSuite.ts`
- Usage:
  - `web/src/pages/TestRunDetailPage/TestCaseRecord.tsx`
  - `web/src/pages/TestRunDetailPage/TestSuiteRecord.tsx`
  - `web/src/pages/TestDetailPage/Step.tsx`
  - `web/src/pages/TestDetailPage/TestDetailPage.tsx`

## Testing

### Manual Testing
1. Start Observer services: `docker compose --profile web-dev up -d`
2. Run Playwright tests with custom reporter that includes tags
3. View tags in Observer UI at `http://localhost:3000`

### Automated Testing
- Backend model tests: `go test ./internal/models/...`
- Frontend build: `cd web && npm run build`

## Backward Compatibility

- Tags are optional fields (omitempty in Go, optional in TypeScript)
- Missing tags are handled gracefully (TagList returns null if no tags)
- Existing data without tags will continue to work without modification
- Database queries don't require tags field to be present

## Future Enhancements

1. **Tag Filtering**: Add ability to filter tests/suites/steps by tags in UI
2. **Tag Statistics**: Show tag distribution and usage statistics
3. **Tag Management**: UI for managing common tags and tag suggestions
4. **Tag-based Search**: Search functionality across all entities by tags
5. **Tag Categories**: Support for tag categories (e.g., priority, type, module)

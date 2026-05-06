# Step Component Quick Reference

## TL;DR - What Changed

✅ **Status badges now show correct status** - No more "pending" for skipped/broken/timed out steps
✅ **Error messages now visible** - Failed steps display error information

## Usage

### Basic Step
```tsx
<Step step={{
  id: "step-1",
  title: "Test step title",
  status: "PASSED"  // Any valid TestStatus
}} />
```

### Step with Error
```tsx
<Step step={{
  id: "step-2",
  title: "Failed test step",
  status: "FAILED",
  error: "AssertionError: Expected true but got false"
}} />
```

## Status Types & Display

| Status | Badge Color | Icon | When to Use |
|--------|-------------|------|-------------|
| `PASSED` | Green | ✓ CheckCircle | Step completed successfully |
| `FAILED` | Red | ✗ XCircle | Step failed with error |
| `SKIPPED` | Gray | ⊖ MinusCircle | Step was skipped |
| `BROKEN` | Orange | ⚠ AlertTriangle | Test infrastructure issue |
| `TIMEDOUT` | Purple | ⏱ Clock | Step exceeded time limit |
| `INTERRUPTED` | Pink | ⊘ Ban | Execution was interrupted |
| `RUNNING` | Blue | ▶ Play | Currently executing |
| `PENDING` | Yellow | ⏲ Clock | Waiting to execute |
| `NOT_RUN` | Gray | ⊖ MinusCircle | Not executed |
| `UNKNOWN` | Gray | ○ Circle | Status unavailable |

## Error Display Rules

Errors are shown when:
- Step has `error` or `errors` field populated, AND
- Status is `FAILED`, `BROKEN`, or `TIMEDOUT`

```tsx
// Error shown
{ status: "FAILED", error: "Something went wrong" }

// Error NOT shown (wrong status)
{ status: "PASSED", error: "Ignored" }

// Error NOT shown (no error data)
{ status: "FAILED", error: undefined }
```

## Props

```typescript
interface StepProps {
  step: Step;                    // Required: Step data
  globalExpandAll?: boolean;     // Optional: Control expand state
}

interface Step {
  id: string;                    // Required: Unique identifier
  title: string;                 // Required: Step display name
  status?: TestStatus;           // Optional: Step status (defaults to UNKNOWN)
  error?: string;                // Optional: Single error message
  errors?: string[];             // Optional: Array of errors
  tags?: string[];               // Optional: Tags to display
  steps?: Step[];                // Optional: Nested child steps
  // ... other fields
}
```

## Component Hierarchy

```
Step (parent)
├─ Card
│  └─ CardContent
│     ├─ Expand/Collapse Button (if has children)
│     ├─ Badge (status indicator)
│     ├─ Title (h3)
│     ├─ TagList (if has tags)
│     └─ Error Box (if shouldShowError)
└─ Nested Steps (if expanded)
   └─ Step (recursive)
```

## Styling Classes

### Card Container
```tsx
<Card className="mb-4">          // 1rem margin bottom
<CardContent className="py-4">   // 1rem padding vertical
```

### Badge
```tsx
<Badge status={step.status || "UNKNOWN"} />
// Badge handles its own styling
```

### Error Box
```tsx
<div className="mt-4 p-3 bg-red-50 border border-red-200 rounded">
  <p className="text-sm font-medium text-red-800">Error</p>
  <p className="text-sm text-red-700 mt-1 whitespace-pre-wrap break-words">
    {errorMessage}
  </p>
</div>
```

## Common Patterns

### Display Steps from Test
```tsx
{test.steps?.map((step) => (
  <Step key={step.id} step={step} />
))}
```

### Nested Steps with Expand All
```tsx
const [expandAll, setExpandAll] = useState(false);

<Step step={parentStep} globalExpandAll={expandAll} />
```

### Filter Steps by Status
```tsx
const failedSteps = test.steps?.filter(s => s.status === "FAILED");
{failedSteps?.map((step) => (
  <Step key={step.id} step={step} />
))}
```

## Accessibility

### Screen Readers
- Badge announces: "Test status: [status]"
- Expand button announces: "Expand substeps" / "Collapse substeps"
- Error section read as part of step content

### Keyboard Navigation
- `Tab`: Navigate to expand/collapse button
- `Enter` or `Space`: Toggle expand/collapse
- Title is not interactive (h3 element)

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Performance

- Lightweight: ~85 lines of code
- No expensive calculations
- Efficient re-renders with React hooks
- Expand/collapse state managed per-step

## Testing Tips

### Unit Tests
```tsx
// Test status display
<Step step={{ status: "PASSED" }} />
// Assert Badge receives "PASSED"

// Test error display
<Step step={{ status: "FAILED", error: "Test error" }} />
// Assert error box is visible

// Test no error for passed steps
<Step step={{ status: "PASSED", error: "Ignored" }} />
// Assert error box is NOT visible
```

### Visual Tests
- Check badge colors match status
- Verify error box only shows for failed states
- Test expand/collapse functionality
- Verify responsive layout

## Migration Guide

### Before (Hardcoded Icons)
```tsx
// ❌ Old way - hardcoded status checks
{step.status === "PASSED" ? (
  <>
    <CheckCircle2 className="..." />
    <Badge status="success" />
  </>
) : step.status === "FAILED" ? (
  // ...
) : (
  // ...
)}
```

### After (Badge Component)
```tsx
// ✅ New way - let Badge handle it
<Badge status={step.status || "UNKNOWN"} />
```

## Troubleshooting

### Badge shows "unknown"
**Cause**: `step.status` is undefined or not a valid `TestStatus`
**Fix**: Ensure backend returns valid status, or check TypeScript types

### Error not displaying
**Cause**: Status is not FAILED/BROKEN/TIMEDOUT, or error field is empty
**Fix**: Check `step.status` and `step.error` / `step.errors` values

### Expand/collapse not working
**Cause**: Step has no children (`step.steps` is empty or undefined)
**Fix**: Expand button only shows if `step.steps?.length > 0`

## Related Components

- **Badge** (`/web/src/components/Badge.tsx`) - Status indicator
- **TagList** (`/web/src/components/TagList.tsx`) - Tag display
- **Card** (`/web/src/components/Card.tsx`) - Container component
- **StepContainer** (`/web/src/pages/TestDetailPage/StepContainer.tsx`) - Step list wrapper

## API Reference

### Step Type (Backend)
```go
// pkg/api/rest_mongodb.go
type Step struct {
    ID       string      `json:"id"`
    Title    string      `json:"title"`
    Status   TestStatus  `json:"status"`
    Error    string      `json:"error,omitempty"`
    Errors   []string    `json:"errors,omitempty"`
    Steps    []Step      `json:"steps,omitempty"`
    // ...
}
```

### TestStatus Enum
```typescript
type TestStatus = 
  | "PASSED"
  | "FAILED"
  | "SKIPPED"
  | "BROKEN"
  | "TIMEDOUT"
  | "INTERRUPTED"
  | "RUNNING"
  | "PENDING"
  | "NOT_RUN"
  | "UNKNOWN";
```

## Examples

### Playwright Test Steps
```tsx
<Step step={{
  id: "pw-1",
  title: "Login to application",
  status: "PASSED",
  tags: ["authentication", "smoke"]
}} />
```

### Pytest Test Steps
```tsx
<Step step={{
  id: "py-1",
  title: "test_user_registration",
  status: "FAILED",
  error: "AssertionError: assert 200 == 201",
  tags: ["unit", "users"]
}} />
```

### JUnit Test Steps
```tsx
<Step step={{
  id: "junit-1",
  title: "testDatabaseConnection",
  status: "BROKEN",
  error: "ConnectionException: Unable to connect to database",
  tags: ["integration", "database"]
}} />
```

## FAQ

**Q: Can I customize badge colors?**
A: Badge colors are defined in the Badge component. Pass `className` prop for additional styling.

**Q: How do I hide the error box?**
A: Error box only shows for FAILED/BROKEN/TIMEDOUT statuses. It's automatically hidden for other statuses.

**Q: Can I nest steps more than one level?**
A: Yes! Steps can be nested infinitely. Each child step can have its own `steps` array.

**Q: How do I control expand/collapse programmatically?**
A: Use the `globalExpandAll` prop to control all steps' expand state.

**Q: Does the component support dark mode?**
A: Not currently, but can be added with `dark:` Tailwind variants.

## Version History

### v2.0 (Current)
- ✅ Fixed status badge display for all status types
- ✅ Added error message display
- ✅ Simplified component logic
- ✅ Removed unused imports

### v1.0 (Legacy)
- ❌ Only handled PASSED/FAILED statuses
- ❌ No error display
- ❌ Hardcoded status checks

---

**Component**: Step.tsx
**Location**: `/web/src/pages/TestDetailPage/Step.tsx`
**Dependencies**: Badge, Card, TagList, lucide-react
**Last Updated**: [Current Date]

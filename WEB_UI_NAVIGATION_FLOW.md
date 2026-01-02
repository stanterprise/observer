# Observer Web UI - Navigation Flow

## Application Structure (After Improvements)

```
┌─────────────────────────────────────────────────────────────┐
│                         Layout                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Header: Observer | Dashboard | Test Runs | Status  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Route Content                      │  │
│  │                                                      │  │
│  │   [Dashboard, Test Runs, Run Detail, Test Detail]  │  │
│  │                                                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Footer: Observer | Docs | Support | v1.0.0        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Page Hierarchy

```
Root (/)
│
├─ Dashboard (/)
│  └─ Displays: Statistics, Recent Runs, Quick Actions
│
├─ Test Suite Runs (/suite_runs)
│  │  └─ Displays: Table of all test suite runs with stats
│  │
│  └─ Test Suite Run Detail (/suite_runs/:runId)
│     │  └─ Displays: Run statistics, progress bar, test cases list
│     │
     └─ Test Detail (/suite_runs/:runId/tests/:testId)
│        └─ Displays: Test info, metadata, execution steps
```

## User Flow Examples

### Flow 1: New User Landing

```
1. User visits root (/) → Dashboard
   ├─ Sees empty state with welcome message
   ├─ Clicks "View Test Runs" button
   └─ Navigated to Test Suite Runs page

2. User on Test Suite Runs (/suite_runs)
   ├─ Sees empty state "No test runs found"
   └─ Waits for first test execution
```

### Flow 2: Monitoring Active Tests

```
1. User visits Dashboard (/)
   ├─ Sees 4 statistics cards
   │  ├─ Total Runs: 15
   │  ├─ Total Tests: 243
   │  ├─ Pass Rate: 94%
   │  └─ Test Health: Excellent
   │
   ├─ Sees recent runs table (5 most recent)
   └─ Clicks run ID link → Navigated to Run Detail

2. User on Run Detail (/suite_runs/abc-123)
   ├─ Sees progress bar (visual overview)
   ├─ Sees run statistics (total, passed, failed, skipped)
   ├─ Sees list of test cases
   └─ Clicks test case → Navigated to Test Case Detail

3. User on Test Detail (/suite_runs/abc-123/tests/test-xyz)
   ├─ Sees test information (ID, Run ID, Duration, Retries)
   ├─ Sees execution timeline
   ├─ Sees metadata (if available)
   ├─ Sees execution steps with hierarchy
   └─ Clicks back arrow → Returns to Run Detail
```

### Flow 3: Investigating Failures

```
1. User on Dashboard (/)
   ├─ Sees Pass Rate: 75% (yellow/warning)
   ├─ Sees Test Health: Fair
   ├─ Sees recent run with failed tests
   └─ Clicks run ID → Navigated to Run Detail

2. User on Run Detail (/suite_runs/failed-run)
   ├─ Progress bar shows red section (failures)
   ├─ Failed count prominently displayed
   ├─ Scrolls to test cases list
   ├─ Identifies failed test (red badge)
   └─ Clicks failed test → Navigated to Test Case Detail

3. User on Test Detail (/suite_runs/failed-run/tests/failed-test)
   ├─ Badge shows "failed" status with red background
   ├─ Scrolls to execution steps
   ├─ Failed step highlighted with:
   │  ├─ Red background (bg-red-50)
   │  ├─ Red border (border-red-300)
   │  ├─ AlertCircle icon (red)
   │  └─ Failed badge with icon
   │
   ├─ Reviews step details
   ├─ Notes timestamp and duration
   └─ Clicks back to investigate other failures
```

## Component Interaction Flow

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │
       ├─ HTTP Request → API Server (/api/runs/stats, /api/runs/:id, etc.)
       │                      │
       │                      └─ MongoDB (reads test data)
       │
       ├─ WebSocket Connection → API Server (/ws)
       │                              │
       │                              └─ NATS JetStream (real-time events)
       │
       └─ Renders UI
          │
          ├─ Dashboard
          │  ├─ Fetches stats on mount
          │  ├─ Displays cards and table
          │  └─ Updates on WebSocket events
          │
          ├─ Test Suite Runs
          │  ├─ Fetches all runs on mount
          │  ├─ Displays table
          │  ├─ Updates on WebSocket events
          │  └─ Supports sorting
          │
          ├─ Test Run Detail
          │  ├─ Fetches specific run data
          │  ├─ Displays progress and tests
          │  ├─ Updates on WebSocket events
          │  └─ Links to test details
          │
          └─ Test Detail
             ├─ Fetches test and steps
             ├─ Displays hierarchical steps
             ├─ Updates on WebSocket events
             └─ Links back to run
```

## Real-Time Update Flow

```
Test Execution → Reporter → Observer gRPC → NATS → API Consumer → WebSocket → Web UI

Example: Test "login-test" fails

1. Playwright executes test
2. Reporter sends test.end event to Observer gRPC
3. Observer publishes to NATS stream
4. API Consumer receives event
5. API Consumer broadcasts via WebSocket
6. Web UI receives WebSocket message
7. Components update local state:
   ├─ Dashboard: Decrements pass rate
   ├─ Test Suite Runs: Updates run status to "failed"
   ├─ Test Run Detail: Increments failed count
   └─ Test Detail: Updates test status badge
```

## Navigation Patterns

### Primary Navigation

- **Dashboard** → Overview of all test activity
- **Test Runs** → List of all test suite runs

### Contextual Navigation

- **Run Detail** → Click run ID from Dashboard or Test Runs
- **Test Case Detail** → Click test case from Run Detail
- **Back Navigation** → Arrow icons to return to previous page

### Quick Actions

- **Refresh** buttons on Dashboard and Test Runs
- **"View All →"** link from Dashboard to Test Runs
- **Documentation** link in footer and Quick Actions

## Responsive Behavior

### Desktop (≥1024px)

```
┌──────────────────────────────────────────────┐
│  Header [Logo | Dashboard | Test Runs | ●]  │
├──────────────────────────────────────────────┤
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐│
│  │ Card 1 │ │ Card 2 │ │ Card 3 │ │ Card 4 ││  (4 columns)
│  └────────┘ └────────┘ └────────┘ └────────┘│
│  ┌────────────────────────────────────────┐  │
│  │         Recent Runs Table             │  │
│  └────────────────────────────────────────┘  │
├──────────────────────────────────────────────┤
│  Footer [Logo | Links | Version]            │
└──────────────────────────────────────────────┘
```

### Tablet (768px - 1023px)

```
┌────────────────────────────────────┐
│  Header [Logo | Dash | Runs | ●]  │
├────────────────────────────────────┤
│  ┌──────────┐ ┌──────────┐        │
│  │  Card 1  │ │  Card 2  │        │  (2 columns)
│  └──────────┘ └──────────┘        │
│  ┌──────────┐ ┌──────────┐        │
│  │  Card 3  │ │  Card 4  │        │
│  └──────────┘ └──────────┘        │
│  ┌─────────────────────────────┐  │
│  │   Recent Runs Table         │  │  (horizontal scroll)
│  └─────────────────────────────┘  │
├────────────────────────────────────┤
│  Footer [stacked]                 │
└────────────────────────────────────┘
```

### Mobile (<768px)

```
┌────────────────────┐
│ Header [☰ | Logo] │
│ [Home][Runs][●]   │
├────────────────────┤
│  ┌──────────────┐ │
│  │   Card 1     │ │  (1 column, stacked)
│  └──────────────┘ │
│  ┌──────────────┐ │
│  │   Card 2     │ │
│  └──────────────┘ │
│  ┌──────────────┐ │
│  │   Card 3     │ │
│  └──────────────┘ │
│  ┌──────────────┐ │
│  │   Card 4     │ │
│  └──────────────┘ │
│  ┌──────────────┐ │
│  │ Recent Runs  │ │  (scroll table)
│  │ (scroll →)   │ │
│  └──────────────┘ │
├────────────────────┤
│ Footer [stacked]  │
└────────────────────┘
```

## Accessibility Navigation

### Keyboard Navigation

```
Tab Order:
1. Skip to main content (optional, could add)
2. Logo / Home link
3. Dashboard link
4. Test Runs link
5. Connection status (non-interactive, but focusable for SR)
6. Main content area
   ├─ Refresh button
   ├─ Statistics cards
   ├─ Run links in table
   └─ Action buttons
7. Footer links

Shortcuts:
- Tab: Move to next interactive element
- Shift+Tab: Move to previous interactive element
- Enter/Space: Activate buttons and links
- Arrow keys: Navigate within table (native browser)
```

### Screen Reader Navigation

```
Landmarks:
- <nav> → "Navigation"
- <main> → "Main content"
- <footer> → "Footer"

Headings:
- h1: Page title (e.g., "Dashboard", "Test Suite Runs")
- h2: Section headings (e.g., "Recent Test Runs", "Test Cases")
- h3: Card titles, subsection headings

ARIA:
- role="status" → Connection indicator, status badges
- aria-label → Icon buttons, navigation elements
- aria-live="polite" → Connection status changes
- scope="col" → Table headers
```

## Error State Flows

### Network Error

```
1. User on Dashboard
2. API request fails (network error)
3. Error state displays:
   ├─ AlertTriangle icon (red)
   ├─ Error message
   └─ Retry button
4. User clicks Retry
5. Request retries
6. Success → Dashboard loads
   OR Failure → Error persists
```

### Not Found Error

```
1. User visits /suite_runs/invalid-id
2. API returns 404
3. Error state displays:
   ├─ Back link to Test Runs
   ├─ AlertCircle icon
   ├─ "Run not found" message
   └─ Descriptive text
4. User clicks Back link
5. Navigated to Test Runs page
```

### Empty State

```
1. User on Test Runs page
2. API returns empty array
3. Empty state displays:
   ├─ Play icon (gray)
   ├─ "No test runs found" heading
   └─ Descriptive text
4. User waits for tests to execute
5. WebSocket event arrives
6. New run appears in list
```

## Performance Considerations

### Data Fetching

- Initial fetch on page load
- Real-time updates via WebSocket (no polling)
- Local state updates for immediate feedback
- Optional manual refresh button

### Rendering

- React functional components (fast re-renders)
- Minimal prop drilling (local state where possible)
- Conditional rendering for loading/error/success states
- CSS transitions (GPU-accelerated)

### Bundle Size

- Tailwind CSS (JIT, minimal bundle)
- Lucide React icons (tree-shakeable)
- React Router (code splitting ready)
- No heavy dependencies

## Summary

The Observer web UI now provides:

- Clear, intuitive navigation structure
- Comprehensive dashboard for quick insights
- Real-time updates without page refresh
- Responsive design for all devices
- Accessible interface for all users
- Performant, lightweight application

All changes maintain backward compatibility with the existing API and data structures, requiring only a frontend rebuild for deployment.

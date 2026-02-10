# Test Map Feature Documentation

## Overview
The Test Map is a visual representation page that displays test suites and their tests in a canvas-like grid layout with interactive tag-based filtering.

## Access
Navigate to Test Map from any Test Run Detail page via the "View Test Map" button in the header.

**Route**: `/suite_runs/:runId/map`

## Features

### Visual Test Grid
- Test suites displayed as cards in a responsive grid (1-3 columns)
- Tests shown as 32x32px colored squares within each suite
- Hover over any test to see detailed tooltip with:
  - Test title
  - Status (including "Flaky" indicator)
  - Duration in milliseconds
  - Tags
  - Retry count for flaky tests

### Status Colors
- 🟢 **Green**: PASSED
- 🟡 **Amber**: FLAKY (passed with retries)
- 🔴 **Red**: FAILED
- 🔵 **Blue**: RUNNING
- ⚫ **Gray**: SKIPPED/NOT_RUN
- 🟠 **Orange**: BROKEN
- 🟣 **Purple**: TIMEDOUT
- 🔴 **Pink**: INTERRUPTED

### Tag Filtering
- Right sidebar shows all tags sorted by occurrence
- Click tags to highlight matching tests (blue ring + scale effect)
- Multi-select enabled - select multiple tags to filter
- Counter shows number of highlighted tests
- "Clear Selection" button to reset filters

### Real-time Updates
- Automatically polls for updates every 5 seconds
- Updates are silent (no loading spinner on refresh)
- Test status colors update automatically

### Navigation
- **Click any test**: Navigate to Test Detail page
- **Back button**: Return to Test Run Detail page
- **Keyboard navigation**: Full keyboard support

## Technical Implementation

### Component Structure
```
TestMapPage (web/src/pages/TestMapPage/TestMapPage.tsx)
├── SuiteBox: Displays a test suite with tests
└── TestBox: Individual test square with hover tooltip
```

### Data Flow
1. Fetches test run data: `GET /api/runs/:runId`
2. Extracts and counts tags from all tests
3. Sorts tags by occurrence (descending)
4. Filters tests based on selected tags
5. Polls for updates every 5 seconds

### State Management
- `runDetail`: Full test run data
- `selectedTags`: Set of currently selected tags
- `highlightedTestIds`: Computed set of tests to highlight
- `loading`, `error`: Standard UI states

## Use Cases

### Finding Smoke Tests
1. Open Test Map
2. Click "smoke" tag in sidebar
3. All smoke tests are highlighted
4. Visual distribution across suites is clear

### Identifying Flaky Tests
1. Open Test Map
2. Look for amber-colored test squares
3. Hover to see retry count
4. Click to investigate details

### Multi-tag Analysis
1. Select "integration" tag
2. Also select "api" tag
3. Only tests with both tags are highlighted
4. Shows overlap between test categories

## Responsive Design
- **Desktop (lg+)**: 3-column grid with sticky sidebar
- **Tablet (md)**: 2-column grid
- **Mobile**: Single column with sidebar below

## Accessibility
- Semantic HTML structure
- ARIA labels on all interactive elements
- Keyboard navigation support
- Status indicated by both color and text
- Focus indicators on all interactive elements

## Browser Support
Same as main Observer web UI (modern browsers supporting ES2022).

## Development

### Build
```bash
cd web
npm install
npm run build
```

### Development Server
```bash
cd web
npm run dev
```

### Route Configuration
Route is defined in `web/src/App.tsx`:
```typescript
<Route path="suite_runs/:runId/map" element={<TestMapPage />} />
```

## Future Enhancements
- Search/filter by test name
- Export map as image
- Adjustable test box sizes
- Suite type filtering
- WebSocket for real-time updates
- Virtual scrolling for large test suites

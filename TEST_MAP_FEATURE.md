# Test Map Feature Documentation

## Overview
The Test Map is a visual representation page that displays all tests as a single rectangular grid with dynamic sizing based on test count. Features an 8:6 aspect ratio for balanced layout and visual fading for tag filtering. No scrolling required - all tests are visible at once.

## Access
Navigate to Test Map from any Test Run Detail page via the "View Test Map" button in the header.

**Route**: `/suite_runs/:runId/map`

## Features

### Visual Test Grid
- **Single rectangular grid** of all tests (no suite wrappers)
- **8:6 aspect ratio**: Grid maintains consistent 8:6 (1.333...) ratio for balanced rectangular layout
- **Dynamic sizing**: Test boxes automatically scale from 4px to 32px based on total test count
- **No scrolling**: All tests fit in viewport
- Tests flow left-to-right, top-to-bottom
- Hover over any test to see detailed tooltip with:
  - Test title
  - Status (including "Flaky" indicator)
  - Duration in milliseconds
  - Tags
  - Retry count for flaky tests

### Dynamic Sizing
Test box size is automatically calculated to fill the available viewport space while maintaining the 8:6 aspect ratio:
- **Minimum size**: 32px (ensures readability)
- **Maximum size**: No limit - boxes scale up to fill available space
- **Few tests (10-50)**: Larger boxes (50-100px+) to fill viewport
- **Many tests (100-200)**: Medium boxes (32-60px) filling space efficiently
- **Very many tests (500+)**: Boxes default to 32px minimum

The algorithm ensures test objects always fill the map container, with larger boxes for fewer tests and maintaining 32px minimum for readability.

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
- **Visual fading**: Non-highlighted tests fade to 25% opacity when tags are selected for clear distinction

### Real-time Updates
- Automatically polls for updates every 5 seconds
- Updates are silent (no loading spinner on refresh)
- Test status colors update automatically
- Recalculates box sizes if test count changes

### Navigation
- **Click any test**: Navigate to Test Detail page
- **Back button**: Return to Test Run Detail page
- **Keyboard navigation**: Full keyboard support

## Technical Implementation

### Component Structure
```
TestMapPage (web/src/pages/TestMapPage/TestMapPage.tsx)
└── TestBox: Individual test square with hover tooltip and dynamic sizing
```

### Dynamic Sizing Algorithm
```typescript
// 1. Measure available space
const availableWidth = containerWidth - 48;   // Card padding (24px each side)
const availableHeight = viewportHeight - 80;  // Header and padding

// 2. Calculate optimal grid dimensions with 8:6 aspect ratio
const TARGET_ASPECT_RATIO = 8 / 6;
const cols = Math.ceil(Math.sqrt(totalTests * TARGET_ASPECT_RATIO));
const rows = Math.ceil(totalTests / cols);

// 3. Calculate size that fits
const gap = 2;
const sizeByWidth = (availableWidth - (cols - 1) * gap) / cols;
const sizeByHeight = (availableHeight - (rows - 1) * gap) / rows;

// 4. Use the LARGER dimension to maximize box size and fill container
const calculatedSize = Math.floor(Math.max(sizeByWidth, sizeByHeight));
const size = Math.max(32, calculatedSize);
```

This maximizes box size by using the larger of the two calculated dimensions, ensuring the container is filled as much as possible. One dimension may slightly overflow (with flex-wrap handling), but this provides better space utilization.

### Visual Fading Logic
When tags are selected:
```typescript
const isHighlighted = highlightedTestIds.has(test.id);
const isFaded = selectedTags.size > 0 && !isHighlighted;

// In TestBox component:
style={{ opacity: isFaded ? 0.25 : 1 }}
```

This creates a clear visual distinction:
- **Highlighted tests**: 100% opacity + blue ring + scale
- **Non-highlighted tests**: 25% opacity (faded)
- **No tags selected**: All tests at 100% opacity

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
- `containerDimensions`: Current container width/height for sizing
- `testBoxSize`: Computed optimal box size (4-32px) using 8:6 aspect ratio
- `loading`, `error`: Standard UI states

## Use Cases

### Finding Smoke Tests with Visual Focus
1. Open Test Map
2. Click "smoke" tag in sidebar
3. All smoke tests are highlighted with blue ring at full opacity
4. All other tests fade to 25% opacity
5. Instantly see distribution and focus on smoke tests

### Identifying Flaky Tests
1. Open Test Map
2. Look for amber-colored test squares
3. Hover to see retry count
4. Click to investigate details

### Multi-tag Analysis with Clear Distinction
1. Select "integration" tag - non-matching tests fade
2. Also select "api" tag
3. Only tests with both tags remain bright
4. All other tests at 25% opacity
5. Shows overlap between test categories with clear visual hierarchy

### Assessing Test Run Health
1. Open Test Map
2. Get instant visual overview of entire run
3. Red squares (failures) stand out immediately
4. See distribution and patterns at a glance

## Responsive Design
- **Desktop**: Full viewport height minus headers (~calc(100vh - 320px))
- **Tablet**: Same behavior with slightly smaller container
- **Mobile**: Tag sidebar moves below map, full width available
- **Window Resize**: Automatically recalculates box sizes

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
- Zoom control to override calculated size
- Density selector (compact/normal/comfortable)
- Optional suite boundary dividers
- Search/filter by test name
- Export map as image
- WebSocket for real-time updates instead of polling

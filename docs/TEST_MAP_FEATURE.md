# Test Map Feature Documentation

## Overview

The Test Map is a visual representation page that displays all tests as rectangular bars with dynamic sizing based on test count. Each test is shown as a wide, short rectangle optimized for displaying text. Features visual fading for tag filtering and maximum space utilization.

## Access

Navigate to Test Map from any Test Run Detail page via the "View Test Map" button in the header.

**Route**: `/runs/:runId/map`

## Features

### Visual Test Grid

- **Rectangular test bars**: Each test is displayed as a wide rectangle (not square)
- **Text-optimized height**: Height fixed at ~28px (comfortable for a line of text)
- **Dynamic width**: Width scales to fill available horizontal space
- **No scrolling**: All tests fit in viewport
- Tests flow left-to-right, top-to-bottom, wrapping to new rows
- Hover over any test to see detailed tooltip with:
  - Test title
  - Status (including "Flaky" indicator)
  - Duration in milliseconds
  - Tags
  - Retry count for flaky tests

### Dynamic Sizing

Test boxes are rectangular with dimensions calculated to maximize space utilization:

- **Height**: Fixed at 24-32px (optimized for text display)
- **Width**: Dynamically calculated based on available space and test count
- **Minimum width**: 60px (ensures usability)
- **No maximum width**: Boxes expand to fill horizontal space
- **Layout**: Rows calculated based on available vertical space, columns adjust accordingly

**Examples:**

- **50 tests**: Wide bars (~200-300px width × 28px height) in few rows
- **125 tests**: Medium bars (~100-150px width × 28px height) in more rows
- **500+ tests**: Narrower bars (~60-80px width × 28px height) filling all rows

The algorithm maximizes horizontal space usage while keeping a comfortable text-readable height.

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
└── TestBox: Individual rectangular test bar with hover tooltip and dynamic dimensions
```

### Dynamic Sizing Algorithm

```typescript
// 1. Measure available space
const availableWidth = containerWidth - 48; // Card padding (24px each side)
const availableHeight = viewportHeight - 80; // Header and padding

// 2. Fixed height optimized for text (one line)
const BOX_HEIGHT = 32; // Fixed height for comfortable text display
const MIN_HEIGHT = 24;
const MAX_HEIGHT = 32;
const boxHeight = Math.max(MIN_HEIGHT, Math.min(MAX_HEIGHT, BOX_HEIGHT));

// 3. Calculate how many rows we can fit
const gap = 2;
const maxRows = Math.floor((availableHeight + gap) / (boxHeight + gap));

// 4. Calculate columns needed
const cols = Math.ceil(totalTests / maxRows);

// 5. Calculate width to fill available space
const boxWidth = Math.floor((availableWidth - (cols - 1) * gap) / cols);

// 6. Apply minimum width constraint
const MIN_WIDTH = 60;
const finalWidth = Math.max(MIN_WIDTH, boxWidth);
```

This creates rectangular test bars with fixed, text-optimized height and variable width that maximizes horizontal space utilization.

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
- `testBoxDimensions`: Computed box dimensions with independent width/height (height ~24–32px, width ≥ ~60px, width scales to fill horizontal space)
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
<Route path="/runs/:runId/map" element={<TestMapPage />} />
```

## Future Enhancements

- Zoom control to override calculated size
- Density selector (compact/normal/comfortable)
- Optional suite boundary dividers
- Search/filter by test name
- Export map as image
- WebSocket for real-time updates instead of polling

# Tag Territory Map

An interactive visualization that provides a bird's-eye view of test runs by displaying tests and tags in a self-organizing semantic map.

## Overview

The Tag Territory Map uses a force-directed physics simulation to layout tests and tags in 2D space, where:
- Tests "pull" their associated tags together
- Tags with similar test coverage cluster naturally
- Tag territories emerge as overlapping regions
- The layout stabilizes into a coherent, deterministic structure

## Features

### Visual Elements
- **Test Nodes**: Colored dots representing individual tests
  - Green: Passed tests
  - Red: Failed tests
  - Gray: Skipped tests
  - Orange: Timed out tests
  - Size reflects test importance (duration, retries, tag count)

- **Tag Territories**: Soft gradient halos showing tag influence regions
  - Color-coded per tag (deterministic hashing)
  - Overlapping regions indicate tag co-occurrence
  - Intensity shows test density

- **Tag Labels**: Positioned at the centroid of their test clusters
  - Bordered boxes with tag names
  - Click to focus/filter
  - Legend shows top tags by impact

### Interactions

#### Hover
- **Test nodes**: Display tooltip with test name, status, duration, retries, and tags
- **Tag labels**: Highlight related tests and dim others

#### Click
- **Tag labels**: Toggle focus mode (dim non-matching tests)
- **Legend items**: Same as clicking tag labels

#### Pan & Zoom
- **Drag**: Pan the canvas
- **Mouse wheel**: Zoom in/out
- **Reset View button**: Return to default view

### Controls

- **Max Visible Tags**: Select 10/30/60/100 top tags to display
- **Render tag regions**: Toggle gradient halos on/off for performance
- **Mock Data Generator**: (Demo page only) Generate test datasets of various sizes

## Implementation

### Architecture

```
┌─────────────────────┐
│  TagTerritoryMap    │ ← React Component (Canvas UI)
│  Component          │
└──────────┬──────────┘
           │
           ├─→ Web Worker ─→ Physics Simulation
           │                 (Force calculation)
           │
           ├─→ Utils
           │   ├─ tagSelection.ts   (Impact scoring)
           │   ├─ tagSimilarity.ts  (Jaccard index)
           │   └─ seededRandom.ts   (Deterministic RNG)
           │
           └─→ Canvas Renderer
               (Test nodes + Tag halos + Labels)
```

### Key Files

- **`components/TagTerritoryMap.tsx`**: Main React component with Canvas rendering
- **`workers/tagTerritoryWorker.ts`**: Web Worker for physics simulation
- **`utils/tagSelection.ts`**: Tag impact scoring and selection logic
- **`utils/tagSimilarity.ts`**: Jaccard similarity for tag relationships
- **`utils/seededRandom.ts`**: Deterministic random number generator
- **`types/tagTerritory.ts`**: TypeScript type definitions
- **`pages/TagTerritoryPage/`**: Integration with test run data
- **`pages/TagTerritoryDemoPage/`**: Standalone demo with mock data

### Physics Forces

The simulation uses four forces:

1. **Test → Tag Attraction**: Each test attracts its tags (proportional to distance)
2. **Tag ↔ Tag Repulsion**: Tags repel each other (inversely proportional to similarity)
3. **Separation**: Prevents node overlap (collision detection)
4. **Centering**: Gentle pull toward canvas center (keeps layout on-screen)

### Tag Selection

Tags are ranked by **impact score**:
```
impact = test_count 
       + (2 × failed_test_count) 
       + slow_test_count
```

Only the top N tags are displayed (configurable via UI).

### Performance

- **Web Worker**: Simulation runs in a separate thread (non-blocking)
- **Canvas Rendering**: Efficient 2D graphics (no DOM overhead)
- **Batched Updates**: State updates throttled to ~60fps
- **Deterministic**: Same test data produces same layout (seeded RNG)

**Tested with:**
- ✅ 500 tests, 30 tags: ~500ms to stabilize
- ✅ 130 tests, 23 tags: ~300ms to stabilize
- 🎯 Target: 2,000 tests, 150 tags

## Usage

### Accessing the Visualization

1. **From Test Run Detail Page**
   - Navigate to any test run: `/suite_runs/:runId`
   - Click the "Tag Territory Map" button
   - View the visualization at `/suite_runs/:runId/territory`

2. **Demo Page**
   - Navigate to `/demo/territory`
   - Experiment with mock data of various sizes
   - Test interactions and controls

### Integration with Test Data

The `TagTerritoryPage` component automatically:
1. Fetches test run data from the API
2. Transforms tests to `TagTerritoryTest` format
3. Passes data to `TagTerritoryMap` component
4. Renders with user-configurable controls

Example:
```typescript
const territoryTests: TagTerritoryTest[] = tests.map(test => ({
  id: test.id,
  name: test.title,
  tags: test.tags || [],
  status: test.status,
  durationMs: test.duration / 1_000_000, // ns → ms
  retries: test.retryCount || 0,
}));
```

## Future Enhancements

### Potential Features
- [ ] Marching squares for precise isocontour boundaries
- [ ] Multi-tag selection for complex filtering
- [ ] Animation presets (attract/repel, explode/collapse)
- [ ] Export as image/SVG
- [ ] Time-series view (animate layout over multiple runs)
- [ ] Heatmap overlay (test frequency, failure rate)
- [ ] Custom force parameters (user-adjustable physics)
- [ ] Search/highlight by test name or tag pattern

### Optimizations
- [ ] Spatial indexing for collision detection (quadtree)
- [ ] WebGL renderer for 5000+ tests
- [ ] Progressive rendering for large datasets
- [ ] Cached layouts (resume from previous state)

## Technical Notes

### Why Web Workers?
Running the physics simulation in the main thread would block rendering and cause UI jank. The Web Worker allows:
- Continuous simulation at 60fps
- Responsive UI during layout calculation
- Easy termination on component unmount

### Why Canvas over SVG?
- **Performance**: Canvas is much faster for large numbers of elements
- **Smooth animations**: 60fps rendering with thousands of nodes
- **Gradient halos**: Efficient radial gradient rendering
- **Interaction**: Custom hit detection (no DOM overhead)

### Why Deterministic RNG?
- **Reproducibility**: Same test data = same layout
- **Debugging**: Easier to reason about layout behavior
- **Testing**: Predictable snapshots for visual regression tests

## License

Part of the Observer Test Observability Platform. See main repository for license information.

# MARKER-Based Historical Statistics Page

## Overview

This document describes the implementation of a new feature that allows users to view historical statistics for test runs filtered by a specific `MARKER` metadata value.

## Implementation

### Backend API

**Endpoint**: `GET /api/marker/{markerValue}/stats`

**Query Parameters**:
- `limit` (optional): Maximum number of runs to return (default: 100)

**Response Format**:
```json
{
  "marker": "string",
  "runs": [
    {
      "runId": "string",
      "runName": "string",
      "status": "string",
      "metadata": {},
      "startTime": "timestamp",
      "endTime": "timestamp",
      "duration": number,
      "createdAt": "timestamp",
      "updatedAt": "timestamp",
      "total": number,
      "passed": number,
      "failed": number,
      "skipped": number,
      "running": number,
      "broken": number,
      "timedout": number,
      "interrupted": number,
      "unknown": number
    }
  ],
  "total": number,
  "count": number
}
```

**Implementation Details**:
- Located in `pkg/api/rest_mongodb.go`
- Method: `handleMarkerStats`
- Filters test runs by `metadata.MARKER` field value
- Calculates aggregate statistics for each run
- Returns runs sorted by creation date (descending)

### Frontend Page

**Route**: `/marker/:markerValue/stats`

**Component**: `MarkerStatsPage`

**Features**:
1. **Summary Cards**: Display aggregate statistics (total runs, total tests, pass rate, passed, failed)
2. **Pass Rate Timeline**: Line chart showing pass rate trends over time
3. **Status Distribution**: Bar chart showing distribution of test statuses across all runs
4. **Run History Table**: Detailed table of all runs with links to run details

**Implementation Details**:
- Located in `web/src/pages/MarkerStatsPage/`
- Uses Recharts for visualization
- Responsive design with Tailwind CSS
- Real-time data fetching on mount

### Database Query

The implementation uses MongoDB's flexible document structure to filter runs by metadata:

```go
filter := bson.M{
    "metadata.MARKER": markerValue,
}
```

This takes advantage of MongoDB's dot notation to query nested fields efficiently.

## Usage

### Accessing the Page

Navigate to `/marker/{your-marker-value}/stats` where `{your-marker-value}` is the value of the MARKER metadata you want to filter by.

Example:
- `/marker/release-1.0/stats` - Shows all runs with `MARKER=release-1.0`
- `/marker/nightly/stats` - Shows all runs with `MARKER=nightly`

### Setting MARKER Metadata

When creating test runs, include the MARKER field in the metadata:

```javascript
// Example: Playwright Reporter configuration
{
  metadata: {
    MARKER: "release-1.0",
    environment: "production"
  }
}
```

## Future Enhancements

Potential improvements for this feature:

1. **Multiple Marker Support**: Filter by multiple marker values
2. **Time Range Filter**: Add date range picker to filter runs by time period
3. **Export Functionality**: Export statistics to CSV/Excel
4. **Comparison View**: Compare statistics between different marker values
5. **Trend Analysis**: Show long-term trends and anomaly detection
6. **Custom Metrics**: Allow users to define custom success criteria

## Technical Notes

- The API endpoint performs aggregation in the API layer rather than using MongoDB aggregation pipelines for simplicity
- Frontend caching could be added to improve performance for frequently accessed markers
- Consider adding pagination for marker values with large numbers of runs
- Indexes on `metadata.MARKER` field would improve query performance for large datasets

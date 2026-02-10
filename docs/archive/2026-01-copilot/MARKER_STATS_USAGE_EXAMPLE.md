# MARKER Statistics Feature - Usage Example

## Quick Example

This example shows how to use the MARKER-based historical statistics feature to track test runs across different releases or environments.

### Step 1: Configure Your Test Reporter

When running your tests, add a `MARKER` field to your test run metadata:

```javascript
// playwright.config.ts
export default defineConfig({
  reporter: [
    ['@stanterprise/observer-reporter', {
      serverUrl: 'http://localhost:50051',
      metadata: {
        MARKER: 'release-2.0-beta',  // <-- Add this
        environment: 'staging',
        branch: 'feature/new-ui'
      }
    }]
  ]
});
```

### Step 2: Run Your Tests

Execute your test suite as normal. The reporter will send events to Observer with the MARKER metadata:

```bash
npx playwright test
```

### Step 3: View Statistics

Navigate to the MARKER statistics page in the Observer Web UI:

```
http://localhost:3000/marker/release-2.0-beta/stats
```

You'll see:
- **Summary Cards**: Total runs, tests, pass rate
- **Pass Rate Timeline**: Historical trend of test success
- **Status Distribution**: Breakdown of test results (passed/failed/skipped)
- **Run History Table**: Detailed list of all runs with this MARKER

## Real-World Use Cases

### Use Case 1: Release Tracking

Track test quality across different software releases:

```javascript
// For release 1.0
metadata: { MARKER: 'v1.0.0' }

// For release 2.0
metadata: { MARKER: 'v2.0.0' }

// For beta releases
metadata: { MARKER: 'v2.1.0-beta.3' }
```

View trends:
- `/marker/v1.0.0/stats` - All tests for version 1.0
- `/marker/v2.0.0/stats` - All tests for version 2.0

### Use Case 2: Environment-Specific Monitoring

Track test stability in different environments:

```javascript
// Production smoke tests
metadata: { MARKER: 'prod-smoke' }

// Staging full suite
metadata: { MARKER: 'staging-full' }

// Nightly regression tests
metadata: { MARKER: 'nightly-regression' }
```

### Use Case 3: Feature Branch Testing

Monitor test quality during feature development:

```javascript
// Feature branch tests
metadata: { 
  MARKER: 'feature-new-payment-flow',
  jiraTicket: 'PAY-123'
}
```

View at: `/marker/feature-new-payment-flow/stats`

### Use Case 4: CI/CD Pipeline Stages

Track tests at different pipeline stages:

```javascript
// Unit tests stage
metadata: { MARKER: 'ci-unit', pipeline: 'main' }

// Integration tests stage
metadata: { MARKER: 'ci-integration', pipeline: 'main' }

// E2E tests stage
metadata: { MARKER: 'ci-e2e', pipeline: 'main' }
```

## API Integration

You can also access the statistics programmatically via the REST API:

```bash
# Get statistics for a specific marker
curl http://localhost:8080/api/marker/release-2.0-beta/stats

# With limit parameter
curl http://localhost:8080/api/marker/release-2.0-beta/stats?limit=50
```

Response format:
```json
{
  "marker": "release-2.0-beta",
  "runs": [
    {
      "runId": "abc123...",
      "runName": "E2E Test Suite",
      "status": "PASSED",
      "createdAt": "2026-01-14T10:30:00Z",
      "total": 150,
      "passed": 148,
      "failed": 2,
      "skipped": 0
    }
  ],
  "total": 25,
  "count": 25
}
```

## Tips & Best Practices

1. **Use Consistent Naming**: Establish a naming convention for MARKER values (e.g., `{environment}-{type}` or `v{version}`)

2. **Keep MARKER Values Descriptive**: Use clear, meaningful names that explain what the runs represent

3. **Avoid Too Many Markers**: Too many unique MARKER values can make it hard to find specific data. Consider using other metadata fields for fine-grained filtering.

4. **Combine with Other Metadata**: Use MARKER for high-level categorization and other metadata fields for additional details:
   ```javascript
   metadata: {
     MARKER: 'v2.0.0',          // Primary filter
     environment: 'staging',      // Additional context
     browser: 'chrome',           // Test configuration
     os: 'linux'                  // Test environment
   }
   ```

5. **Document Your MARKER Values**: Keep a list of active MARKER values and their purposes for your team

## Troubleshooting

**No data showing for my MARKER?**
- Verify the MARKER value matches exactly (case-sensitive)
- Check that test runs were sent to Observer with the metadata
- Confirm the API endpoint is accessible

**Too many runs returned?**
- Use the `limit` query parameter to reduce the result set
- Consider using more specific MARKER values

**Want to see all available MARKER values?**
- Currently, you need to know the MARKER value to query
- Future enhancement: Add an endpoint to list all unique MARKER values

## Next Steps

- See [MARKER_STATS_FEATURE.md](./MARKER_STATS_FEATURE.md) for complete API documentation
- Explore other Observer features in the [README.md](../README.md)
- Join the discussions to request new features or report issues

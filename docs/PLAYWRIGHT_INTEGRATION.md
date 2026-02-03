# Playwright Reporter Integration Guide

This guide explains how to use the Observer service with the [stanterprise-playwright-reporter](https://github.com/stanterprise/stanterprise-playwright-reporter) to collect test execution data from Playwright tests.

## Overview

The Observer system collects test execution events via gRPC and stores them for analysis. The Playwright reporter acts as a client that sends test events during Playwright test execution.

## Architecture

```
Playwright Tests → Playwright Reporter → Observer Ingestion (gRPC) → NATS → Processor → Database
                                                ↓
                                           API Service → Web UI
```

## Prerequisites

- Node.js 16+ (for Playwright)
- Observer services running (ingestion, processor, API)
- NATS and MongoDB (for distributed mode) or AIO container

## Installation

### Option 1: Install Reporter from npm

```bash
npm install --save-dev github:stanterprise/stanterprise-playwright-reporter
```

### Option 2: Clone and Link Locally

```bash
git clone https://github.com/stanterprise/stanterprise-playwright-reporter
cd stanterprise-playwright-reporter
npm install
npm link

cd your-project
npm link stanterprise-playwright-reporter
```

## Observer Setup

### Distributed Mode (Recommended for CI/Production)

Start all services:

```bash
# Start infrastructure
make mongo-up nats-up

# Build and start services
make build-all

# Start ingestion service (gRPC endpoint)
NATS_URL='nats://localhost:4222' ./bin/ingestion &

# Start processor service (consumes NATS, writes to DB)
MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
NATS_URL='nats://localhost:4222' \
./bin/processor &

# Start API service (optional, for future web UI)
MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
./bin/api &
```

### All-in-One Mode (Development/Local)

```bash
docker compose --profile aio up -d
```

This starts a single container with:

- Ingestion service on port 50051
- API service on port 8080
- Embedded NATS on port 4222
- Embedded MongoDB

## Playwright Configuration

Configure Playwright to use the Observer reporter in your `playwright.config.ts`:

```typescript
import { defineConfig } from "@playwright/test";

export default defineConfig({
  reporter: [
    ["list"], // Keep console output
    [
      "stanterprise-playwright-reporter",
      {
        // Observer ingestion endpoint
        endpoint: "localhost:50051",

        // Optional: TLS configuration (for production)
        // useTLS: true,
        // tlsCert: '/path/to/cert.pem',

        // Optional: Additional metadata
        metadata: {
          environment: "ci",
          branch: process.env.GITHUB_REF_NAME || "main",
          buildId: process.env.GITHUB_RUN_ID || "local",
        },

        // Optional: Batch configuration
        // batchSize: 10,
        // flushInterval: 1000,
      },
    ],
  ],

  // ... rest of your Playwright config
});
```

## Running Tests

```bash
# Run Playwright tests normally
npx playwright test

# Tests will automatically report to Observer
```

## Example Test

```typescript
import { test, expect } from "@playwright/test";

test.describe("Login Flow", () => {
  test("should login successfully", async ({ page }) => {
    // Navigate
    await page.goto("https://example.com/login");

    // Fill form
    await page.fill('[name="username"]', "testuser");
    await page.fill('[name="password"]', "password123");

    // Submit
    await page.click('button[type="submit"]');

    // Verify
    await expect(page.locator(".welcome")).toBeVisible();
  });
});
```

The reporter automatically captures:

- Test start/end events
- Step execution (each Playwright action)
- Test status (passed, failed, skipped)
- Timing information
- Error messages and stack traces
- Browser and environment metadata

## Verifying Data Collection

### Check Database (MongoDB)

```bash
make mongo-shell

# List recent test runs (collection names may vary by implementation)
db.test_runs.find({}, { _id: 1, run_id: 1, title: 1, status: 1, created_at: 1 }).sort({ created_at: -1 }).limit(10)
```

### Check NATS Stream

```bash
# View NATS monitoring
curl http://localhost:8222/streaming/channelsz

# Or use NATS CLI
nats stream info tests_events
```

## Protocol Compatibility

The Observer implements the gRPC protocol defined in `github.com/stanterprise/proto-go/testsystem/v1/observer`.

Current version: `v0.0.9`

### Supported Events

1. **TestBeginEvent** - Sent when a test starts
   - `id` - Unique test identifier
   - `runId` - Test run identifier (shared across related tests)
   - `title` - Test name
   - `metadata` - Additional key-value data
   - `retryCount` - Total number of retry attempts allowed (optional)
   - `retryIndex` - Current retry attempt index (optional)
   - `timeout` - Timeout in milliseconds (optional)

2. **TestEndEvent** - Sent when a test completes
   - `id` - Test identifier
   - `status` - PASSED, FAILED, SKIPPED, BROKEN, TIMEDOUT, INTERRUPTED
   - `duration` - Execution time (protobuf Duration type)

3. **StepBeginEvent** - Sent when a test step starts
   - `id` - Step identifier
   - `testCaseRunId` - Parent test identifier
   - `title` - Step description
   - `type` - Step type (e.g., "action", "assertion")
   - `category` - Step category (e.g., "hook", "fixture", "test.step") (optional)

4. **StepEndEvent** - Sent when a test step completes
   - `id` - Step identifier
   - `status` - Step result
   - `error` - Error message if failed
   - `category` - Step category (optional)

### Suite Events

5. **SuiteBeginEvent** - Sent when a test suite starts
   - `suite.id` - Unique suite run identifier
   - `suite.name` - Suite name
   - `suite.description` - Suite description
   - `suite.projectName` - Project name (e.g., browser/device configuration) (new in v0.0.9)
   - `suite.testSuiteSpecId` - Test suite specification identifier
   - `suite.initiatedBy` - User or system that initiated the suite
   - `suite.metadata` - Additional metadata

6. **SuiteEndEvent** - Sent when a test suite completes
   - `suite.id` - Suite identifier
   - `suite.status` - Suite execution status
   - `suite.duration` - Execution time (protobuf Duration type)

## Troubleshooting

### Connection Refused

**Problem**: Reporter can't connect to Observer

**Solutions**:

1. Verify Observer ingestion service is running: `ps aux | grep ingestion`
2. Check port is open: `netstat -an | grep 50051`
3. Ensure no firewall blocking: `telnet localhost 50051`

### No Data in Database

**Problem**: Tests run but no data appears

**Solutions**:

1. Check NATS connection: `curl http://localhost:8222/varz`
2. Verify processor is running: `ps aux | grep processor`
3. Check processor logs for errors
4. Verify database connection in processor

### Slow Test Execution

**Problem**: Tests run slower with reporter enabled

**Solutions**:

1. Increase batch size in reporter config
2. Use async mode for event sending
3. Run Observer ingestion service locally to reduce network latency

## Advanced Configuration

### CI/CD Integration

GitHub Actions example:

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      observer-db:
        image: mongo:7
        env:
          MONGO_INITDB_ROOT_USERNAME: root
          MONGO_INITDB_ROOT_PASSWORD: change-me

      observer-nats:
        image: nats:2.10-alpine
        options: --health-cmd "wget --spider http://localhost:8222/healthz"

    steps:
      - uses: actions/checkout@v4

      - name: Start Observer Services
        run: |
          docker compose up -d ingestion processor

      - name: Install dependencies
        run: npm ci

      - name: Run Playwright tests
        run: npx playwright test
        env:
          OBSERVER_ENDPOINT: localhost:50051

      - name: View test results
        if: always()
        run: |
          docker compose exec -T observer-db mongosh --username root --password change-me --authenticationDatabase admin --eval \
            "db.getSiblingDB('observer').test_runs.find({}, { _id: 1, title: 1, status: 1 }).sort({ created_at: -1 }).limit(10).toArray()"
```

### Custom Metadata

Add custom metadata to all tests:

```typescript
// In playwright.config.ts
metadata: {
  // Git information
  gitCommit: process.env.GITHUB_SHA?.slice(0, 8),
  gitBranch: process.env.GITHUB_REF_NAME,

  // CI information
  ciProvider: 'github-actions',
  ciJobUrl: process.env.GITHUB_SERVER_URL + '/' +
            process.env.GITHUB_REPOSITORY + '/actions/runs/' +
            process.env.GITHUB_RUN_ID,

  // Environment
  nodeVersion: process.version,
  platform: process.platform,

  // Custom tags
  team: 'qa',
  feature: 'authentication',
}
```

## Performance Considerations

1. **Batch Events**: Use batching to reduce network overhead
2. **Async Reporting**: Reporter sends events asynchronously to not block test execution
3. **Local Services**: Run Observer services locally during development
4. **Database Indexing**: Ensure proper indexes on `test_case_runs` and `step_runs` tables

## Future Enhancements

- Web UI for viewing test results and trends
- GraphQL API for flexible queries
- Retention policies and data archiving
- Real-time notifications for test failures
- Artifact storage (screenshots, videos, traces)
- Test analytics and flakiness detection

## Support

- Observer Issues: https://github.com/stanterprise/observer/issues
- Reporter Issues: https://github.com/stanterprise/stanterprise-playwright-reporter/issues

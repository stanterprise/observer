# Observer Test Suite

This directory contains comprehensive tests for the Observer test observability system.

## Test Files

### `api_test.go`

Comprehensive API test suite validating gRPC service functionality:

- Full test lifecycle flows
- Error handling and validation
- Concurrent request handling
- Idempotency
- Various test scenarios (passed, failed, skipped)
- Metadata persistence

### `e2e_integration_test.go`

End-to-end integration tests for the distributed architecture:

- Complete gRPC → NATS → Consumer → Database flow
- NATS event format validation
- Async event processing verification

**Note**: Requires `NATS_TEST_URL` environment variable

### `nats_integration_test.go`

NATS JetStream publisher integration tests:

- Event publishing to NATS
- Message consumption and acknowledgment
- Event type routing

**Note**: Requires `NATS_TEST_URL` environment variable

### `main_test.go` & `helper_test.go`

Legacy unit tests using in-process bufconn:

- Basic lifecycle validation
- Input validation
- Error handling

## Running Tests

### All Tests (Unit Tests Only)

```bash
make test
# or
go test ./tests -v
```

### With NATS Integration Tests

```bash
# Start NATS
make nats-up

# Run tests
NATS_TEST_URL=nats://localhost:4222 make test-nats-integration
# or
NATS_TEST_URL=nats://localhost:4222 go test ./tests -v
```

### Specific Test Suite

```bash
# API tests only
go test ./tests -v -run "^TestFull|^TestError|^TestConcurrent"

# E2E tests only
NATS_TEST_URL=nats://localhost:4222 go test ./tests -v -run "^TestEndToEnd"

# NATS integration only
NATS_TEST_URL=nats://localhost:4222 go test ./tests -v -run "^TestNATSIntegration"
```

### With Coverage

```bash
go test ./tests -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Infrastructure

### Database

- **Integration Tests**: Use MongoDB (via `testcontainers-go`) for persistence validation
- **Isolation**: Each test uses an isolated database/collection

### NATS

- **Streams**: Unique stream per test run (auto-created/deleted)
- **Consumers**: Durable consumers for event processing
- **Cleanup**: Automatic stream deletion after tests

### gRPC

- **Transport**: In-process bufconn (no TCP ports needed)
- **Server**: Started in TestMain for shared use
- **Client**: Created per test with custom dialer
- **Interceptors**: Logging and panic recovery enabled

## Test Patterns

### Setup Pattern

```go
conn, db, cleanup := setupTestServerWithDB(t)
defer cleanup()

client := observer.NewTestEventCollectorClient(conn)
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

### Database Verification

```go
var doc bson.M
err := collection.FindOne(ctx, bson.M{"_id": testID}).Decode(&doc)
if err != nil {
    t.Fatalf("Failed to find test doc: %v", err)
}
```

### Event Publishing (E2E)

```go
pub, err := publisher.NewNATSPublisher(cfg, logger)
if err != nil {
    t.Fatalf("NewNATSPublisher() error = %v", err)
}
defer pub.Close()
```

## Environment Variables

| Variable           | Required              | Description                                          | Example                     |
| ------------------ | --------------------- | ---------------------------------------------------- | --------------------------- |
| `NATS_TEST_URL`    | For integration tests | NATS server URL                                      | `nats://localhost:4222`     |
| `MONGODB_TEST_URI` | Optional              | MongoDB connection string (used by Makefile targets) | `mongodb://localhost:27017` |

## Writing New Tests

### API Test Template

```go
func TestMyFeature(t *testing.T) {
    conn, db, cleanup := setupTestServerWithDB(t)
    defer cleanup()

    client := observer.NewTestEventCollectorClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Test implementation
    req := &events.TestBeginEventRequest{
        TestCase: &entities.TestCaseRun{
            Id:    "my-test-id",
            RunId: "my-run-id",
            Title: "My Test",
        },
    }

    resp, err := client.ReportTestBegin(ctx, req)
    if err != nil {
        t.Fatalf("ReportTestBegin failed: %v", err)
    }

    // Verify database
    var testCase models.TestCaseRun
    result := db.Where("id = ?", "my-test-id").First(&testCase)
    if result.Error != nil {
        t.Fatalf("Test not found: %v", result.Error)
    }

    // Assertions
    if testCase.Title != "My Test" {
        t.Errorf("Expected title 'My Test', got '%s'", testCase.Title)
    }
}
```

### Integration Test Template

```go
func TestMyIntegration(t *testing.T) {
    natsURL := os.Getenv("NATS_TEST_URL")
    if natsURL == "" {
        t.Skip("Skipping - NATS_TEST_URL not set")
    }

    // Setup NATS publisher
    cfg := publisher.NATSConfig{
        URL:           natsURL,
        StreamName:    "test_stream_" + time.Now().Format("20060102150405"),
        SubjectPrefix: "test.events",
    }

    pub, err := publisher.NewNATSPublisher(cfg, logger)
    if err != nil {
        t.Fatalf("NewNATSPublisher() error = %v", err)
    }
    defer pub.Close()

    // Test implementation
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Unit Tests
  run: make test

- name: Start NATS
  run: make nats-up

- name: Run Integration Tests
  run: NATS_TEST_URL=nats://localhost:4222 make test-nats-integration
  env:
    NATS_TEST_URL: nats://localhost:4222
```

## Troubleshooting

### Tests Hang

- Check timeout contexts are set properly
- Verify NATS is running for integration tests
- Check MongoDB container startup and connectivity

### Database Errors

- Ensure migrations are run in setup
- Check unique database names are used
- Verify cleanup functions are called

### NATS Connection Errors

- Verify NATS is running: `docker compose ps nats`
- Check NATS health: `curl http://localhost:8222/healthz`
- Ensure correct NATS_TEST_URL format

### Flaky Tests

- Increase timeouts for slow CI environments
- Check for race conditions with `-race` flag
- Verify cleanup in deferred functions

## Best Practices

1. **Always use timeouts** on contexts
2. **Always defer cleanup** functions
3. **Use unique IDs** for test data
4. **Clean up resources** (databases, streams)
5. **Test error cases** as well as success cases
6. **Use table-driven tests** for similar scenarios
7. **Add descriptive test names** and comments
8. **Verify database state** after operations
9. **Use sub-tests** for logical groupings
10. **Skip integration tests** when dependencies unavailable

## References

- [Observer Architecture](../README.md)
- [Playwright Integration](../docs/PLAYWRIGHT_INTEGRATION.md)
- [Test Report](../docs/TEST_REPORT.md)

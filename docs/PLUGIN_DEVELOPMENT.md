# Plugin Development Guide

Observer is designed around four well-defined extension points that let you add new
behaviour without touching the core event pipeline. This document explains each one,
why it exists, and how to build a plugin against it.

---

## Extension points at a glance

| Extension point | Where to add code | What it does |
|---|---|---|
| **Test reporter** | New binary / library | Sends test events to Observer over gRPC |
| **Storage driver** | `pkg/storage/` | Stores and retrieves file attachments |
| **HTTP API handler** | `pkg/api/` | Adds REST endpoints to the API service |
| **gRPC interceptor** | `pkg/server/server.go` | Wraps every gRPC call (auth, metrics, rate-limit) |
| **NATS event handler** | `pkg/consumer/` | Reacts to a new or existing event type inside the processor |

The sections below explain each one from the simplest to the most involved.

---

## 1. Test reporter (external plugin)

A reporter is code in _your_ test framework that calls Observer's gRPC service.
It is the most natural "plugin" because it lives entirely outside the Observer
repository and needs no changes there.

### How it works

```
Your test framework
      │
      │  gRPC (proto-go/testsystem/v1/events)
      ▼
Ingestion service  →  NATS  →  Processor  →  MongoDB
```

The protobuf service contract lives in the separate
[`stanterprise/proto-go`](https://github.com/stanterprise/proto-go) repository.
The ingestion service at `:50051` implements `TestEventCollector`.

### What you need to implement

Pick up the generated client for your language from `proto-go` and call the
relevant RPCs in order:

```
ReportRunStart       (once per run)
ReportSuiteBegin     (once per top-level suite)
  ReportTestBegin    (once per test)
    ReportStepBegin  (once per step)
    ReportStepEnd
  ReportTestEnd
  ReportSuiteEnd
ReportRunEnd
```

The existing Playwright reporter
(`github.com/stanterprise/stanterprise-playwright-reporter`) is the reference
implementation. Read it before writing your own.

### Key constraints

- Every event needs a `run_id` that is consistent across the entire run.
- `suite_id` and `test_id` must be stable (the processor uses them as MongoDB
  document keys via upsert; duplicates are safe, but ID changes are not).
- You do not need to send events in real time — the processor is idempotent and
  handles out-of-order delivery gracefully.

---

## 2. Storage driver plugin

Observer stores file attachments (screenshots, videos, traces) through a `Driver`
interface in `pkg/storage/`. Two drivers ship out of the box: `local` (filesystem)
and `s3` (AWS S3 / MinIO). A third-party driver (GCS, Azure Blob, etc.) is a
pure Go implementation with no changes to any service binary.

### The interface

```go
// pkg/storage/driver.go
type Driver interface {
    Upload(ctx context.Context, name, mimeType string, content io.Reader) (*AttachmentMetadata, error)
    Download(ctx context.Context, storageKey string) (io.ReadCloser, error)
    GetMetadata(ctx context.Context, storageKey string) (*AttachmentMetadata, error)
    Delete(ctx context.Context, storageKey string) error
    GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error)
    Name() string
    Close() error
}
```

### Adding a new driver

**Step 1 — Implement the interface**

Create `pkg/storage/gcs.go` (or a separate module if you prefer):

```go
package storage

type GCSDriver struct { /* ... */ }

func NewGCSDriver(cfg Config, logger *slog.Logger) (*GCSDriver, error) { /* ... */ }

func (d *GCSDriver) Name() string { return "gcs" }
func (d *GCSDriver) Upload(...) (*AttachmentMetadata, error) { /* ... */ }
// ... remaining methods
```

**Step 2 — Register the driver name in the factory**

Add a case to `NewDriver` in `pkg/storage/driver.go`:

```go
case "gcs":
    return NewGCSDriver(cfg, logger)
```

**Step 3 — Add any new configuration fields to `Config`**

`Config` is a plain struct in `driver.go`; add your fields there and read them
from environment variables inside `NewDriverFromEnv`.

**Step 4 — Configure via environment variable**

```bash
STORAGE_DRIVER=gcs
STORAGE_GCS_BUCKET=my-bucket
# ... any other env vars your driver needs
```

No service code changes are required — `NewDriverFromEnv` is already called by
both the processor (`cmd/processor/main.go`) and the API service
(`cmd/api/main.go`).

---

## 3. HTTP API handler plugin

The API service wires together one or more *handler structs* that each own a
group of related REST routes. The pattern is:

```go
type MyHandler struct {
    repo   *repository.MongoRepository
    logger *slog.Logger
}

func (h *MyHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/api/my-feature", h.handleMyFeature)
}
```

`RegisterRoutes` is called once from `cmd/api/main.go` after the mux is created.

### Adding a new handler

**Step 1 — Create the handler file**

```
pkg/api/my_feature.go
```

Follow the same struct-and-constructor pattern used by `MongoHandler`
(`pkg/api/rest_mongodb.go`) and `AttachmentHandler`
(`pkg/api/attachments.go`).

**Step 2 — Register it in the API entrypoint**

In `cmd/api/main.go`, add two lines after the existing handler registrations:

```go
// existing
mongoHandler.RegisterRoutes(mux)
attachmentHandler.RegisterRoutes(mux)

// new
myHandler := api.NewMyHandler(repo, logger)
myHandler.RegisterRoutes(mux)
```

**Step 3 — Update the root index handler (optional)**

The `"/"` handler in `cmd/api/main.go` prints a plaintext index of available
endpoints. Add your new routes there so operators know they exist.

---

## 4. gRPC interceptor plugin

Interceptors wrap every gRPC unary call that passes through the ingestion service.
The existing chain handles panic recovery and structured logging. New interceptors
add cross-cutting behaviour such as authentication, rate limiting, or tracing.

### The interceptor chain

```go
// pkg/server/server.go  –  NewGRPCServer
grpc.ChainUnaryInterceptor(
    recoveryInterceptor(logger),   // 1. catch panics
    loggingInterceptor(logger),    // 2. log call + duration
    // your interceptor goes here  // 3. ...
)
```

Interceptors execute in declaration order on the way **in** and in reverse order
on the way **out**.

### Writing an interceptor

```go
func authInterceptor(secret string) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok || !validateToken(md["authorization"], secret) {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        return handler(ctx, req)  // call the next interceptor / handler
    }
}
```

### Registering the interceptor

Pass it to `grpc.ChainUnaryInterceptor` inside `NewGRPCServer` in
`pkg/server/server.go`:

```go
grpc.ChainUnaryInterceptor(
    recoveryInterceptor(logger),
    loggingInterceptor(logger),
    authInterceptor(os.Getenv("API_SECRET")),
)
```

The interceptor applies to **all** methods. If you only want it on some methods,
check `info.FullMethod` and call through for the rest.

---

## 5. NATS event handler plugin

The processor (`cmd/processor`) consumes events from NATS and routes them by
type to handler methods. Adding a handler for a new event type requires three
small changes.

### How the dispatcher works

```go
// pkg/consumer/nats_mongodb.go  –  processMessage
switch event.Type {
case publisher.EventTypeTestBegin:
    return c.handleTestBegin(ctx, event.Data)
// ...
default:
    c.logger.Warn("unknown event type", "type", event.Type)
    return nil  // unknown types are acked — no redelivery loop
}
```

Each `event.Data` field is a JSON-encoded protobuf message (`protojson`
format). The handler unmarshals it, maps it to internal models, and upserts
to MongoDB via `c.repo`.

### Adding a handler for a new event type

**Step 1 — Declare the event type constant**

In `pkg/publisher/nats.go`, add a new constant alongside the existing ones:

```go
const (
    // ... existing constants ...
    EventTypeMyEvent EventType = "my.event"
)
```

**Step 2 — Add the protobuf message and gRPC method (if needed)**

If the event originates from a reporter calling a new gRPC method, add the RPC
to the proto definition in `stanterprise/proto-go`, bump the Go module version,
and implement the handler in `pkg/server/server.go` following the existing
validate → publish pattern.

**Step 3 — Implement the handler method**

Create (or add to an existing) handler file, e.g.
`pkg/consumer/nats_my_handlers.go`:

```go
func (c *MongoNATSConsumer) handleMyEvent(ctx context.Context, data json.RawMessage) error {
    var req myproto.MyEventRequest
    if err := protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(data, &req); err != nil {
        return fmt.Errorf("unmarshal my event: %w", err)
    }
    // persist to MongoDB via c.repo ...
    return nil
}
```

**Step 4 — Wire it into the dispatcher**

Add the case to `processMessage` in `pkg/consumer/nats_mongodb.go`:

```go
case publisher.EventTypeMyEvent:
    return c.handleMyEvent(ctx, event.Data)
```

### Idempotency requirement

All handler methods **must** be idempotent. The JetStream consumer retries a
message (up to 5 times) whenever a handler returns an error or the process
restarts before sending an ack. Use MongoDB upsert operations (`$setOnInsert`,
`$set`) rather than inserts to meet this requirement. See the existing handlers
in `pkg/consumer/nats_*_handlers.go` for examples.

---

## Choosing the right extension point

| Goal | Recommended approach |
|---|---|
| Add support for a new test framework (Pytest, JUnit, …) | New **test reporter** using the gRPC client |
| Store attachments in a different cloud provider | New **storage driver** |
| Expose new data from MongoDB as a REST endpoint | New **HTTP API handler** |
| Add authentication, rate-limiting, or distributed tracing to gRPC | New **gRPC interceptor** |
| React to a new kind of test event (e.g., flakiness score) | New **NATS event handler** |
| Subscribe to all events for alerting or analytics | Independent NATS JetStream consumer (same stream, new durable consumer name) |

---

## Cross-cutting concerns

### Nil-safe loggers

Any struct that accepts a `*slog.Logger` must guard against `nil`:

```go
if logger == nil {
    logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
}
```

See `pkg/server/server.go`, `pkg/publisher/nats.go`, and `pkg/consumer/nats_mongodb.go`
for the canonical pattern.

### Configuration via environment variables

All runtime configuration is read from environment variables, never from config
files. Use `os.Getenv` with a sensible default via a local helper:

```go
func envOr(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### Graceful shutdown

Every long-running goroutine should respect a `context.Context` that is cancelled
on `SIGINT`/`SIGTERM`. Pass the context from the service entrypoint into your
plugin and return when it is done. See `cmd/processor/main.go` for the canonical
pattern.

### Both deployment modes

Test your plugin under both the **AIO** (single container) and **distributed**
(multi-container) Docker Compose profiles to ensure it works with embedded NATS
and MongoDB as well as external ones.

---

## Further reading

- [Architecture overview](architecture/00-overview.md)
- [Component descriptions](architecture/01-components.md)
- [Data flow](architecture/02-dataflow.md)
- [Playwright reporter integration](architecture/08-reporter-integration.md)
- [WebSocket endpoint](WEBSOCKET_IMPLEMENTATION.md)
- [Database schema](architecture/07-database-schema.md)

# Reporter Integration

## Playwright Reporter

- Uses gRPC client from generated protobuf code.
- Streams events:
  - RunStarted / RunFinished
  - TestStarted / TestFinished
  - StepStarted / StepFinished
  - AttachmentAdded

## Event Delivery
- Chunked binary streams for attachments (64–256 KB).
- Retries with exponential backoff.
- Local spillover queue when broker is offline.

## Configuration Example

```bash
export OBSERVER_ENDPOINT=localhost:50051
export OBSERVER_TOKEN=dev-token
```

## Other Frameworks (Future)
- pytest-observer  
- junit-observer  
- jest-observer

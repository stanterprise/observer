# Attachment Storage Feature

## Overview

Observer now supports deployment-aware attachment storage that automatically selects the appropriate storage backend based on configuration. This feature enables efficient storage of test artifacts (screenshots, videos, logs) in both local and cloud deployments.

## Storage Architecture

### Storage Abstraction Layer

The system uses a pluggable `storage.Driver` interface located in `pkg/storage/` that supports multiple backends:

- **Local Filesystem** (`local`): For Docker/local deployments
- **S3-Compatible Storage** (`s3`): For AWS S3, MinIO, Google Cloud Storage
- **Inline Storage** (fallback): Small attachments stored directly in MongoDB

### Size-Based Storage Strategy

The processor service automatically selects storage method based on attachment size:

- **< 100KB**: Stored inline in MongoDB as base64-encoded content
- **≥ 100KB**: Stored in external storage (if configured)
- **Fallback**: If no storage driver is configured, all attachments are stored inline

## Configuration

### Local Filesystem Storage

For Docker and local deployments:

```bash
STORAGE_DRIVER=local
STORAGE_LOCAL_BASE_PATH=/data/artifacts
STORAGE_LOCAL_BASE_URL=http://localhost:8080/api/attachments
```

**Docker Compose Example (Distributed Mode):**

```yaml
processor:
  environment:
    STORAGE_DRIVER: local
    STORAGE_LOCAL_BASE_PATH: /data/artifacts
    STORAGE_LOCAL_BASE_URL: http://localhost:8080/api/attachments
  volumes:
    - observer-artifacts:/data/artifacts

api:
  environment:
    STORAGE_DRIVER: local
    STORAGE_LOCAL_BASE_PATH: /data/artifacts
    STORAGE_LOCAL_BASE_URL: http://localhost:8080/api/attachments
  volumes:
    - observer-artifacts:/data/artifacts
```

### S3-Compatible Storage

For Kubernetes deployments with AWS S3:

```bash
STORAGE_DRIVER=s3
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BUCKET=observer-attachments
STORAGE_S3_ACCESS_KEY_ID=<your-access-key>
STORAGE_S3_SECRET_ACCESS_KEY=<your-secret-key>
```

**For MinIO:**

```bash
STORAGE_DRIVER=s3
STORAGE_S3_ENDPOINT=http://minio:9000
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BUCKET=observer-attachments
STORAGE_S3_ACCESS_KEY_ID=minioadmin
STORAGE_S3_SECRET_ACCESS_KEY=minioadmin
STORAGE_S3_USE_PATH_STYLE=true
```

**For Google Cloud Storage (S3-compatible mode):**

```bash
STORAGE_DRIVER=s3
STORAGE_S3_ENDPOINT=https://storage.googleapis.com
STORAGE_S3_REGION=us-central1
STORAGE_S3_BUCKET=observer-attachments
STORAGE_S3_ACCESS_KEY_ID=<gcs-access-key>
STORAGE_S3_SECRET_ACCESS_KEY=<gcs-secret-key>
```

### No External Storage (Inline Only)

If `STORAGE_DRIVER` is not set, all attachments are stored inline in MongoDB regardless of size:

```bash
# No STORAGE_DRIVER environment variable
```

## API Endpoints

### Retrieve Attachment

```
GET /api/attachments/{storageKey}
```

The API automatically handles different storage types:

- **inline**: Returns content directly from MongoDB
- **local**: Streams from local filesystem
- **s3**: Redirects to presigned URL (15-minute expiration) or proxies the content
- **external**: Redirects to external URI

**Example Request:**

```bash
curl -O http://localhost:8080/api/attachments/abc123-screenshot.png
```

## Attachment Metadata in MongoDB

Attachments are stored in test documents with the following structure:

### Inline Attachment

```json
{
  "name": "small-log.txt",
  "mime_type": "text/plain",
  "content": "log content here...",
  "storage": "inline",
  "size": 512
}
```

### External Storage Attachment (Local)

```json
{
  "name": "screenshot.png",
  "mime_type": "image/png",
  "storage_key": "attachment-id.png",
  "storage_uri": "http://localhost:8080/api/attachments/attachment-id.png",
  "size": 153600,
  "storage": "local",
  "uploaded_at": "2026-01-28T00:00:00Z"
}
```

### External Storage Attachment (S3)

```json
{
  "name": "screenshot.png",
  "mime_type": "image/png",
  "storage_key": "attachments/attachment-id.png",
  "storage_uri": "http://localhost:8080/api/attachments/attachments/attachment-id.png",
  "size": 153600,
  "storage": "s3",
  "uploaded_at": "2026-01-28T00:00:00Z"
}
```

> **Note:** Storage key format differs between drivers. Local driver uses flat keys (`{id}.{ext}`), while S3 driver uses prefixed keys (`attachments/{id}.{ext}`) for better organization in bucket storage.

### External URI Reference

```json
{
  "name": "external-video.mp4",
  "mime_type": "video/mp4",
  "uri": "https://cdn.example.com/videos/test-run-123.mp4",
  "storage": "external"
}
```

## Implementation Details

### Processor Service Integration

The processor service (`pkg/consumer/nats_mongodb.go`) initializes the storage driver on startup:

```go
// Initialize storage driver (optional)
storageDriver, err := storage.NewDriverFromEnv(logger)
if err != nil {
    return nil, fmt.Errorf("initialize storage driver: %w", err)
}
```

When processing attachments, it uses the `processAttachment` helper:

```go
attMap, err := c.processAttachment(ctx, attachment)
```

This helper automatically:

1. Checks attachment size
2. Uploads to external storage if > 100KB and driver is available
3. Falls back to inline storage if upload fails or no driver is configured
4. Logs errors without failing the entire event

### API Service Integration

The API service (`cmd/api/main.go`) initializes the storage driver and attachment handler:

```go
storageDriver, err := storage.NewDriverFromEnv(logger)
// ...
attachmentHandler := api.NewAttachmentHandler(repo, storageDriver, logger)
attachmentHandler.RegisterRoutes(mux)
```

### Backward Compatibility

Existing inline attachments continue to work seamlessly. The system automatically detects the storage type from the `storage` field in the attachment document.

## Testing

### Unit Tests

```bash
# Test local driver
go test -v ./pkg/storage/... -run TestLocal

# Test integration
go test -v ./pkg/storage/... -run TestStorageIntegration
```

### Integration Tests with MinIO

```bash
# Run S3 integration tests (requires Docker)
go test -v ./pkg/storage/... -run TestS3Driver_Integration
```

## Performance Considerations

### Local Storage

- **Pros**: Simple, fast, no external dependencies
- **Cons**: Not suitable for multi-pod Kubernetes deployments
- **Use Case**: Docker Compose, single-node deployments

### S3 Storage

- **Pros**: Scalable, durable, works with multi-pod deployments
- **Cons**: Network latency, external dependency, cost
- **Use Case**: Production Kubernetes clusters, multi-region deployments

### Inline Storage

- **Pros**: No external storage needed, simple
- **Cons**: Increases MongoDB document size, not suitable for large files
- **Use Case**: Small attachments (< 100KB), text logs

## Security Considerations

### S3 Presigned URLs

The API generates presigned URLs with 15-minute expiration for S3 attachments. This provides temporary, secure access without exposing AWS credentials.

### Access Control

Attachment retrieval requires knowledge of the storage key. For production use, consider adding:

- Authentication on the `/api/attachments/` endpoint
- Run-based authorization (ensure user has access to the test run)
- Rate limiting to prevent abuse

## Troubleshooting

### Attachments Not Uploading

1. Check storage driver initialization logs:

   ```
   storage driver initialized driver=local
   ```

2. Verify environment variables are set correctly

3. Check processor logs for upload errors:
   ```
   storage upload failed, falling back to inline
   ```

### Cannot Retrieve Attachments

1. Verify storage driver is initialized in API service
2. Check volume mounts match between processor and API (for local storage)
3. Verify S3 credentials and bucket permissions (for S3 storage)

### Disk Space Issues (Local Storage)

Monitor the artifacts directory:

```bash
du -sh /data/artifacts
```

Consider implementing cleanup policies for old attachments.

## Future Enhancements

Potential improvements for future versions:

1. **Attachment Cleanup**: Automatic deletion of old attachments
2. **Compression**: Compress large attachments before storage
3. **CDN Integration**: Serve attachments via CDN for better performance
4. **Azure Blob Storage**: Support for Azure as an additional backend
5. **Attachment Index**: Dedicated collection for faster attachment lookup
6. **Retention Policies**: Configurable retention periods per attachment type

## Example Usage

### Playwright Reporter with Attachments

The Playwright reporter automatically attaches screenshots and videos to test results:

```typescript
// Screenshot is automatically uploaded
await page.screenshot({ path: "screenshot.png" });

// Large video is stored externally
// Small logs are stored inline
```

### Retrieving in Web UI

The web UI can display attachments using the API:

```typescript
const attachmentUrl = `/api/attachments/${attachment.storage_key}`;
<img src={attachmentUrl} alt={attachment.name} />
```

## Migration Guide

### Existing Deployments

No migration is required. The system gracefully handles:

1. **Old inline attachments**: Continue to work as before
2. **New external attachments**: Use configured storage driver
3. **Mixed mode**: Both inline and external attachments coexist

### Enabling External Storage

To enable external storage on an existing deployment:

1. Add storage environment variables to docker-compose.yml or Kubernetes manifests
2. Restart the processor and API services
3. New attachments will use external storage
4. Old inline attachments remain unchanged

## References

- Storage Driver Interface: `pkg/storage/driver.go`
- Local Driver Implementation: `pkg/storage/local.go`
- S3 Driver Implementation: `pkg/storage/s3.go`
- Consumer Integration: `pkg/consumer/nats_test_handlers.go`
- API Integration: `pkg/api/attachments.go`
- Docker Configuration: `docker-compose.yml`

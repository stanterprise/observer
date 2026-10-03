# Dockerfile Design

## Multi-Stage Build

The AIO build uses multiple stages:

1. The Go builder compiles the ingestion, processor, API, and migration binaries.
2. The web builder compiles the React application.
3. The Debian runtime stage includes Nginx, s6-overlay, NATS, MongoDB, PostgreSQL, and the service binaries.

Distributed services use their own Dockerfiles: `Dockerfile.ingestion`, `Dockerfile.processor`, `Dockerfile.api`, and `Dockerfile.web`.

### Current Runtime Environment Variables

```bash
MONGODB_URI=mongodb://user:pass@host:27017/observer?authSource=admin
STORAGE_DRIVER=local|s3|minio
NATS_URL=nats://nats:4222
ARTIFACTS_DIR=/data/artifacts
```

### Legacy / Planned Variables

```bash
MODE=aio|service
AUTH_MODE=dev|oidc
```

`MODE` is retained from the earlier single-binary design; current image roles are selected through image-specific Dockerfiles and Compose/Helm configuration. Authentication mode (`AUTH_MODE`) is planned, not implemented.

For end-user image selection and installation, see the [public installation guide](https://observer.stanterprise.com/docs/install/). For Helm-specific image and runtime configuration, see [the chart guide](../../DEPLOYMENT.md).

# Dockerfile Design

## Multi-Stage Build

1. **Build stage** — Go builder image compiles the `observer` binary.
2. **Runtime stage** — Debian slim base with `s6-overlay` and optional `nats-server`.

### Key Environment Variables

```bash
MODE=aio|service
MONGODB_URI=mongodb://user:pass@host:27017/observer?authSource=admin
STORAGE_DRIVER=local|s3
AUTH_MODE=dev|oidc
NATS_URL=nats://nats:4222
ARTIFACTS_DIR=/data/artifacts
```

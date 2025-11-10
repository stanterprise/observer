# Dockerfile Design

## Multi-Stage Build

1. **Build stage** — Go builder image compiles the `observer` binary.  
2. **Runtime stage** — Debian slim base with `s6-overlay` and optional `nats-server`.

### Key Environment Variables

```bash
MODE=aio|service
DB_DRIVER=sqlite|postgres
STORAGE_DRIVER=local|s3
AUTH_MODE=dev|oidc
NATS_URL=nats://nats:4222
ARTIFACTS_DIR=/data/artifacts
```

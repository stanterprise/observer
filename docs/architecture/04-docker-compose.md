# Docker Compose Architecture

## Profiles

- `aio` → Single all-in-one container.  
- `dist` → Multi-container distributed stack.

### Usage

```bash
docker compose --profile aio up -d
docker compose --profile dist up -d
```

### Services

| Service | Description |
|----------|-------------|
| `aio` | Single compact container |
| `nats` | Message broker |
| `postgres` | Main database |
| `minio` | Object storage |
| `ingestion` | gRPC endpoint |
| `processor` | Event consumer |
| `api` | Web UI + API service |

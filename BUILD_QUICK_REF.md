# Observer Build & Run Quick Reference

## Build

```bash
make build-all
```

## Run (local binaries + local infra)

```bash
make mongo-up
make nats-up

NATS_URL=nats://localhost:4222 ./bin/ingestion

MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
NATS_URL=nats://localhost:4222 ./bin/processor

MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
NATS_URL=nats://localhost:4222 ./bin/api
```

## Run (Docker Compose)

```bash
# Distributed
docker compose --profile dist up -d

# All-in-one
docker compose --profile aio up -d
```

## Test

```bash
make test

# NATS integration tests (requires NATS running)
make nats-up
make test-nats-integration
```

## Format / lint

```bash
make fmt
make lint
make vet
```

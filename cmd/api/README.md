# API Service

The API service provides HTTP endpoints for the web UI and external integrations. It serves as the query interface for test data.

## Architecture

The API service will provide:

1. GraphQL API for flexible querying
2. RESTful endpoints for simple operations
3. WebSocket connections for real-time updates
4. Static file serving for the web UI
5. Authentication middleware (OIDC in distributed mode)

## Current State

Currently, the API service is a minimal HTTP server with:
- Health check endpoint (`/health`)
- Basic information endpoint (`/`)
- Database connection (read-only mode)

## Running

### Without database

```bash
./bin/api
# or
make build-api && ./bin/api
```

### With database (read-only)

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/api
```

Default port: `8080`

### Custom port

```bash
PORT=3000 ./bin/api
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service information |
| `/health` | GET | Health check |
| `/api/graphql` | POST | GraphQL endpoint (future) |
| `/metrics` | GET | Prometheus metrics (future) |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listening port |
| `DATABASE_URL` | - | PostgreSQL connection string (optional) |
| `AUTH_MODE` | `dev` | Authentication mode: `dev` or `oidc` (future) |
| `OIDC_ISSUER` | - | OIDC issuer URL (future) |

## Testing

Test the API service:

```bash
# Health check
curl http://localhost:8080/health

# Service info
curl http://localhost:8080/
```

## Future Enhancements

- [ ] GraphQL API implementation (using gqlgen)
- [ ] WebSocket support for real-time updates
- [ ] Web UI static file serving
- [ ] Authentication middleware (dev token, OIDC)
- [ ] Rate limiting
- [ ] CORS configuration
- [ ] Metrics endpoint
- [ ] OpenTelemetry tracing

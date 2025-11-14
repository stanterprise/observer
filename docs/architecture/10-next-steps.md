# Next Steps

## Completed ✅

### 1. Service Decomposition ✅

- ✅ Separate services: `ingestion`, `processor`, `api`
- ✅ Independent binaries with proper entrypoints
- ✅ Graceful shutdown and signal handling

### 2. Event Bus Integration ✅

- ✅ **Phase 1**: NATS JetStream publisher in ingestion service
- ✅ **Phase 2**: NATS JetStream consumer in processor service
- ✅ Pull-based consumer with batch fetching
- ✅ Durable consumer for horizontal scaling
- ✅ Event envelope with type routing

### 3. Storage Layer ✅

- ✅ SQLite support for development/AIO mode
- ✅ PostgreSQL support for production
- ✅ Multi-dialect GORM integration
- ✅ Idempotent upsert patterns

### 4. AIO Runtime ✅

- ✅ s6-overlay integration for multi-process container
- ✅ Embedded NATS server
- ✅ All services in single container

### 5. Compose Setup ✅

- ✅ `aio` profile for all-in-one deployment
- ✅ `dist` profile for distributed deployment
- ✅ Health checks for all services
- ✅ Proper service dependencies

### 6. Testing Infrastructure ✅

- ✅ Comprehensive test suite (17 tests)
- ✅ E2E integration tests with NATS
- ✅ In-process bufconn testing
- ✅ Playwright reporter validation
- ✅ Test documentation and guides

### 7. WebSocket Real-Time Events ✅

- ✅ WebSocket hub with connection management
- ✅ NATS JetStream consumer for event relay
- ✅ Integration into API service
- ✅ Support for distributed and AIO modes
- ✅ HTML test client for validation
- ✅ Documentation and examples

## Remaining Work 🚧

### Phase 3: Full Event-Driven Architecture

- [ ] Remove database dependency from ingestion service
- [ ] Make ingestion fully stateless (NATS-only)
- [ ] Update dual-write pattern to NATS-exclusive

### Phase 4: API Service Implementation

- [ ] GraphQL schema design
- [ ] Query resolvers for test data
- [ ] ~~Real-time subscriptions (WebSocket)~~ ✅ Completed
- [ ] Pagination and filtering

### Phase 5: UI Development

- [ ] React dashboard setup (Vite + TypeScript)
- [ ] Test run listing and detail views
- [ ] ~~Real-time updates via WebSocket~~ ✅ Completed (infrastructure ready)
- [ ] Artifact viewer (screenshots, videos, traces)
- [ ] Tailwind CSS + shadcn/ui components

### Phase 6: Object Storage

- [ ] MinIO/S3 integration for artifacts
- [ ] Pre-signed URL generation
- [ ] Artifact upload endpoint
- [ ] Retention policies

### Phase 7: Authentication & Authorization

- [ ] Dev token authentication
- [ ] OIDC integration (optional)
- [ ] Role-based access control
- [ ] API key management

### Phase 8: Observability

- [ ] Prometheus metrics export
- [ ] OpenTelemetry tracing
- [ ] Structured logging enhancements
- [ ] Grafana dashboards

### Phase 9: CI/CD & Deployment

- [ ] Multi-arch Docker builds (amd64, arm64)
- [ ] SBOM generation
- [ ] Trivy security scanning
- [ ] Kubernetes Helm charts
- [ ] Automated release pipeline

### Phase 10: Production Hardening

- [ ] Rate limiting
- [ ] Circuit breakers
- [ ] Dead letter queue handling
- [ ] Data retention policies
- [ ] Backup and restore procedures

## Priority Order

**High Priority (Next Sprint):**

1. Phase 3: Remove DB from ingestion
2. Phase 4: Basic GraphQL API
3. Phase 5: Minimal UI for viewing tests (WebSocket infrastructure ready ✅)

**Medium Priority:** 4. Phase 6: Object storage for artifacts 5. Phase 8: Basic observability (metrics)

**Low Priority:** 6. Phase 7: Authentication 7. Phase 9: Advanced CI/CD 8. Phase 10: Production hardening

---

**Last Updated**: November 14, 2025  
**Current Phase**: Phase 2 Complete + WebSocket Component ✅

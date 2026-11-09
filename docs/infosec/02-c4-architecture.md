# 02 - C4 Architecture

**Status:** Intended distributed deployment target. Platform controls are
deployment requirements, not claims that Helm or Observer enforces them. The
current implementation and chart state are summarized in
[the security-review overview](../security-review/deployment-security-overview.md).
Managed dependencies may live outside the cluster.

## C4 Level 1 - System Context

```mermaid
flowchart LR
  reporter["External system: CI runners and test reporters"]
  user["Person: engineer using Observer"]
  operator["Person: platform and operations administrator"]
  grpc["External system: public gRPC gateway<br/>Operator-managed TLS / HTTP2 on 443"]
  vpn["External system: organization VPN / private web gateway<br/>HTTPS / WSS on 443; no application login"]
  observer["Software system: Observer on Kubernetes<br/>Ingest, persist, and display test execution data"]
  platform["External system: platform security services<br/>Secrets, audit, monitoring, backup"]

  reporter -->|"Lifecycle events and attachments<br/>gRPC over TLS; shared bearer key"| grpc
  grpc -->|"Private gRPC route to ingestion<br/>TLS re-encryption required by target"| observer
  user -->|"Results, artifacts, and live updates"| vpn
  vpn -->|"Restricted HTTPS/WSS route"| observer
  operator -->|"Controlled deployment and administration<br/>Private management access"| observer
  observer -->|"Obtain secrets; export audit/telemetry; protect backups<br/>Authenticated private interfaces"| platform
```

Observer's system boundary includes its dedicated data services and storage,
whether operated in-cluster or supplied as managed dependencies. Shared platform
security services are outside that ownership boundary.

## C4 Level 2 - Containers

Here, "container" means a C4 runtime/data-store boundary, not necessarily a
single Kubernetes container. Gateways are platform components outside Observer's
application boundary.

```mermaid
flowchart TB
  reporter["CI reporter<br/>External client"]
  user["Engineer<br/>Browser"]
  edge["Public gRPC gateway<br/>Operator-owned TLS / HTTP2 :443<br/>Limits and bypass protection required"]
  access["Organization VPN / private web gateway<br/>HTTPS/WSS :443; network restriction only"]

  subgraph observer["Observer software system"]
    ingestion["Ingestion<br/>Go / gRPC<br/>Checks shared bearer key;<br/>no project/run authorization"]
    nats[("NATS JetStream<br/>Private durable event stream")]
    processor["Processor<br/>Go event consumer<br/>Persist execution state and artifacts"]
    api["API<br/>Go REST + WebSocket<br/>No application user AuthN/AuthZ"]
    web["Web UI<br/>React static assets / web server<br/>Reverse proxy for API and WebSocket"]
    pg[("PostgreSQL<br/>Authoritative durable records")]
    mongo[("MongoDB<br/>Transient live_step_buffers")]
    objects[("Private artifact storage<br/>S3-compatible")]
  end

  reporter -->|"gRPC / TLS 443; shared bearer key"| edge
  edge -->|"Private gRPC :50051;<br/>re-encrypt per deployment policy"| ingestion
  user -->|"HTTPS / WSS via approved private access"| access
  access -->|"Restricted route to web :80"| web
  web -->|"HTTP :8080 in current chart;<br/>TLS is a deployment gap"| api
  ingestion -->|"Publish validated events<br/>(project/run authorization is a gap)"| nats
  nats -->|"Consume events"| processor
  nats -->|"Consume live events"| api
  processor -->|"PostgreSQL :5432;<br/>TLS/auth required by target"| pg
  processor -->|"MongoDB :27017;<br/>TLS/auth required by target"| mongo
  processor -->|"Write artifacts"| objects
  api -->|"PostgreSQL :5432;<br/>TLS/auth required by target"| pg
  api -->|"Read authorized artifacts"| objects
```

The shared bearer key is checked by Observer; it is not project/run authorization.
The gateway must only route to the intended ingestion endpoint, and the cluster
must prevent direct gateway bypass. The VPN/private route is a network boundary,
not application user authentication or authorization. Browser-to-API requests
pass through the web route in this topology; API authorization is still required
for REST, mutations, attachments, and WebSocket subscriptions before that control
can be considered complete.

Database, broker, and object-storage connections must be authenticated and
encrypted according to the network matrix. The current chart defaults do not
provide all of these protections; see the current-state comparison below.

## Current implementation vs target

| Area | Current implementation / chart | Intended target |
| --- | --- | --- |
| Public gRPC | Distributed ingestion enforces one shared bearer key; Service is `ClusterIP`. Helm creates no public gateway, TLS route, or rate limit. | Operator-managed public TLS/HTTP2 gateway on 443, private-only backend path with bypass prevention, abuse limits, and project/run authorization. |
| Web/API access | No application user AuthN/AuthZ. The chart creates ClusterIP Services, not a VPN gateway or Ingress. | Web/API reachable only through the organization's VPN/private access route. Treat VPN restriction as a compensating network control, not user authorization. |
| Internal transport and isolation | Bundled NATS defaults to unauthenticated NATS; PostgreSQL defaults to `sslmode=disable`; MongoDB and Web-to-API use no TLS by default. No Observer/NATS default-deny policy is rendered; bundled database policies are permissive. | Authenticated and encrypted service-to-service connections, scoped credentials, and default-deny network policies with explicit flows. |
| Scope and diagrams | [Current-state security-review diagrams](../security-review/architecture/) document application/chart behavior and operator-selected routes. | This document and the [deployment/network target](03-deployment-and-network.md) describe required deployment controls; they are not deployment evidence. |

## Interpretation and boundaries

- Reporters and event payloads are untrusted even after authentication.
- NATS contains potentially sensitive events; live WebSocket delivery crosses the
  same project authorization boundary as persisted reads.
- MongoDB is not a second durable query store. PostgreSQL remains canonical.
- Artifact bytes are untrusted content, not executable application assets.
- Secrets, migration jobs, backup paths, and infrastructure boundaries appear in
  [the deployment view](03-deployment-and-network.md).
- Level 3 component diagrams should be added when the selected authentication,
  authorization, or artifact-handling design needs internal implementation review.

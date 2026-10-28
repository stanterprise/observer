# 02 - C4 Architecture

**Status:** Proposed target state; security controls shown are requirements, not
implemented-capability claims. Managed dependencies may live outside the cluster.

## C4 Level 1 - System Context

```mermaid
flowchart LR
  reporter["External system: CI runners and test reporters"]
  user["Person: authorized engineer or test analyst"]
  operator["Person: platform and operations administrator"]
  idp["External system: organizational identity provider / machine issuer"]
  observer["Software system: Observer on Kubernetes<br/>Ingest, persist, and display test execution data"]
  platform["External system: platform security services<br/>Secrets, audit, monitoring, backup"]

  reporter -->|"Lifecycle events and attachments<br/>Authenticated gRPC over TLS; public edge 443"| observer
  user -->|"Results, artifacts, and live updates<br/>SSO-protected HTTPS/WSS 443"| observer
  operator -->|"Controlled deployment and administration<br/>Private management access"| observer
  observer -->|"Validate identity / obtain trusted keys<br/>TLS; selected identity mechanism"| idp
  observer -->|"Obtain secrets; export audit/telemetry; protect backups<br/>Authenticated private interfaces"| platform
```

Observer's system boundary includes its dedicated data services and storage,
whether operated in-cluster or supplied as managed dependencies. Shared identity
and platform systems are outside that ownership boundary.

## C4 Level 2 - Containers

Here, "container" means a C4 runtime/data-store boundary, not necessarily a
single Kubernetes container. Gateways are platform components outside Observer's
application boundary.

```mermaid
flowchart TB
  reporter["CI reporter<br/>External client"]
  user["Authorized user<br/>Browser"]
  idp["Identity provider / machine credential issuer"]
  edge["Public gRPC gateway<br/>HTTP/2, TLS, machine authentication, limits"]
  access["Private / identity-aware web gateway<br/>SSO, HTTPS/WSS"]

  subgraph observer["Observer software system"]
    ingestion["Ingestion<br/>Go / gRPC<br/>Authorize and validate events"]
    nats[("NATS JetStream<br/>Private durable event stream")]
    processor["Processor<br/>Go event consumer<br/>Persist execution state and artifacts"]
    api["API<br/>Go REST + WebSocket<br/>Authorized queries, downloads, live relay"]
    web["Web UI<br/>React static assets / web server<br/>Reverse proxy for API and WebSocket"]
    pg[("PostgreSQL<br/>Authoritative durable records")]
    mongo[("MongoDB<br/>Transient live_step_buffers")]
    objects[("Private artifact storage<br/>S3-compatible")]
  end

  reporter -->|"gRPC / TLS 443"| edge
  edge -->|"Authenticated identity + gRPC / TLS"| ingestion
  user -->|"HTTPS / WSS 443"| access
  access -->|"TLS; trusted identity context"| web
  web -->|"REST / WebSocket over TLS"| api
  edge -.->|"Validate machine credentials"| idp
  access -.->|"SSO"| idp
  ingestion -->|"Publish validated, authorized events"| nats
  nats -->|"Consume events"| processor
  nats -->|"Consume live events"| api
  processor -->|"Write durable run data"| pg
  processor -->|"Buffer in-flight steps"| mongo
  processor -->|"Write artifacts"| objects
  api -->|"Query authorized records"| pg
  api -->|"Read authorized artifacts"| objects
```

The edge validates machine identity; ingestion must still enforce authorization
for the target project/run. API authorization must apply equally to REST, artifact
retrieval, and WebSocket subscriptions. Browser-to-API logical requests pass
through the protected web route in this topology.

Database/broker/object-storage connections must be authenticated and encrypted
according to the network matrix. Dashed identity links represent conceptual
validation dependencies; offline JWT validation need not call the issuer per request.

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

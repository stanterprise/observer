# Helm Platform Exposure Task Packet

> **Archived 2026-09-22.** This proposal was scoped to a one-off homelab k3s
> deployment and the separate `observer-mcp` chart; it is unrelated to the
> ongoing `charts/observer` development effort and is not maintained. See
> [HELM_CHART_HARDENING_CHECKLIST.md](../../HELM_CHART_HARDENING_CHECKLIST.md)
> for the current, maintained chart status.

**Status:** Proposed
**Source:** [Helm chart deployment retrospective](HELM_CHART_DEPLOYMENT_RETROSPECTIVE.md)
**Scope:** Deployment and platform configuration for Observer and observer-mcp
**Related:** [Helm chart issue-resolution task packet](HELM_CHART_ISSUE_RESOLUTION_TASK_PACKET.md)

## 1. Objective

Create a declarative, repeatable platform configuration that exposes the application Services according to cluster standards. This packet owns DNS, ingress or Gateway routing, TLS certificates, and private-PKI trust. It does not add those responsibilities to the portable application charts.

The application charts are inputs to this work. They must provide stable Services and documented ports; this packet consumes those Services and configures the platform around them.

## 2. Current Observed State

The working deployment currently relies on resources outside the two Helm releases:

- manually managed Traefik `IngressRoute` resources;
- separate TLS `Certificate` resources and namespace-local TLS Secrets;
- a CoreDNS custom map resolving `.k8s.lan` names to a hard-coded Traefik ClusterIP;
- private homelab CA trust on clients and selected workloads;
- h2c routing for gRPC ingestion;
- MCP routing for both `/sse` and `/message` on one origin;
- separate S3 ingress and certificate resources in the `seaweedfs` namespace.

The target is to manage these through one platform configuration that can be applied, upgraded, inspected, and rolled back independently of application chart releases.

## 3. Platform Contract

The first supported profile is k3s with Traefik and a private homelab CA. The design should keep the application-facing contract portable where practical.

| Application endpoint | Backend Service                | Required platform behavior                                                |
| -------------------- | ------------------------------ | ------------------------------------------------------------------------- |
| Observer web/API     | `observer-web`, `observer-api` | HTTP routing with API path precedence and web fallback                    |
| Observer ingestion   | `observer-ingestion`           | gRPC routing to port `50051`, with h2c where required                     |
| Observer MCP         | observer-mcp Service           | HTTPS host routing for `/sse` and `/message`, no path rewrite, one origin |
| SeaweedFS S3         | SeaweedFS Service              | HTTPS routing with path-style S3 support                                  |

The exact Service names must be derived from the installed release name and documented in the platform values or templates. The platform layer must not assume that a chart creates an Ingress resource.

## 4. Work Items

### P0. Make the current exposure reproducible

#### P0-1. Choose and document the platform profile

Document the supported ingress controller, entry points, TLS policy, certificate issuer, DNS authority, and namespace ownership. Explicitly identify which resources are platform-owned and which are operator prerequisites.

**Acceptance criteria:**

- A clean cluster can be prepared from documented prerequisites.
- The platform profile does not depend on undocumented kubectl commands.
- Traefik-specific resources are isolated from portable application chart logic.

#### P0-2. Manage web, API, and gRPC routes declaratively

Create versioned route resources for the Observer web/API and ingestion Services. Preserve API route priority, configure gRPC/h2c correctly, and verify service ports against the chart contract.

**Acceptance criteria:**

- Web and API requests reach the correct backend Services.
- gRPC ingestion succeeds through the external endpoint.
- Route resources are applied and upgraded without manual patches.
- A chart release or ingress Service recreation does not require editing route manifests.

#### P0-3. Manage MCP SSE and message routing declaratively

Create one host-based route for observer-mcp that supports both `GET /sse` and POST requests to the session-specific `/message` endpoint. Do not rewrite either path or redirect the client to another scheme or host.

**Acceptance criteria:**

- `https://mcp.<domain>/sse` establishes an SSE session.
- The returned relative `/message?sessionId=...` endpoint remains reachable on the same origin.
- Keepalives remain observable through the route.
- The profile documents the one-replica/session-affinity limitation.

#### P0-4. Replace hard-coded internal DNS mapping

Replace the CoreDNS map that points directly to a Traefik ClusterIP with one of:

- managed split-horizon DNS pointing to a stable ingress address; or
- a generated platform address that is stable across Service recreation.

The design must support Pods resolving the same S3 and application hostnames that clients use for signed HTTPS URLs.

**Acceptance criteria:**

- In-cluster and LAN clients resolve the expected names.
- Recreating the ingress Service does not leave stale DNS records.
- S3 hostname resolution works from Observer API and processor Pods.
- DNS changes are versioned and recoverable.

#### P0-5. Manage certificates and private-CA trust

Define the certificate strategy for Observer, gRPC, MCP, and S3. Keep TLS Secrets beside their consuming route resources. Automate issuance and renewal through cert-manager where appropriate, or document pre-created Secret rotation. Distribute the private root CA to clients and workloads that must verify the internal S3 endpoint.

**Acceptance criteria:**

- Certificate SANs cover every supported hostname.
- Certificates and routes are namespace-correct.
- Trusted clients and workloads connect without `-k` or disabled verification.
- Renewal and root-CA recovery procedures are documented and tested.
- Certificate rotation does not require application chart template edits.

### P1. Add operational safeguards

- Add route, certificate, and DNS smoke tests to the deployment pipeline.
- Add an in-cluster test Pod that verifies name resolution and HTTPS trust.
- Validate Traefik timeouts and buffering for long-lived MCP SSE connections.
- Add NetworkPolicies or documented policy requirements for ingress and backend namespaces.
- Record the ingress Service address through a stable load-balancer or DNS contract rather than a mutable ClusterIP.
- Verify S3 path-style requests and private-object URL behavior through the platform endpoint.

### P2. Improve portability and recovery

- Provide a generic Kubernetes Ingress or Gateway API variant if a second controller is required.
- Separate shared wildcard certificates from per-service certificates according to blast-radius and renewal needs.
- Automate private-root CA distribution for supported client platforms.
- Test full recovery of DNS, certificates, routes, and platform state independently of application rollback.

## 5. Deliverables

1. Versioned Traefik or generic Ingress/Gateway resources for web/API, gRPC, MCP, and S3.
2. Versioned certificate and issuer resources or a documented pre-created Secret contract.
3. Versioned DNS/CoreDNS configuration without a fragile hard-coded ingress ClusterIP.
4. Private-root CA distribution and recovery documentation.
5. Platform smoke tests for DNS resolution, TLS trust, HTTP, gRPC, S3, and MCP SSE/message flows.
6. Ownership and rollback documentation for all platform resources.

## 6. Out Of Scope

- Adding Ingress, Gateway, Certificate, or DNS templates to the portable Observer application chart by default.
- Changing application Service names or ports without coordinating with the chart packet.
- Application datastore Secrets, image tags, migrations, dependency wiring, or runtime S3 environment variables.
- Treating a Helm application rollback as a rollback of DNS, certificates, or ingress state.

## 7. Exit Criteria

This packet is complete when a clean supported cluster can apply the platform configuration after installing the application charts, and all four endpoint groups are reachable with trusted DNS and TLS. No manual route creation, DNS edit, certificate patch, or insecure TLS bypass may be required.

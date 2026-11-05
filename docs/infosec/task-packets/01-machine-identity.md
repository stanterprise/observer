# TP-01 - Authenticate Machine Ingestion

**Status:** Proposed; no owner assigned.
**Priority:** P0; part of the public-ingestion gate.
**Accountable roles:** Application and identity owners, with platform support.
**Findings:** 1.
**Controls:** IAM-03, ING-01, ING-08/09, NET-01/02/16, SCP-04; tests V-02/03.
**Prerequisites:** D-03 and D-06 from [TP-00](00-decisions-and-scope.md).
Coordinate the principal contract with [TP-02](02-project-ownership.md) and the
gateway/trust path with [TP-14](14-network-and-platform.md).

## Objective and current state

Authenticate every event-bearing gRPC call and supply a verified principal to
machine project/run authorization, without trusting caller-supplied identity
headers. This is a public-ingestion requirement even while product AuthN/AuthZ
and human access controls are deferred. An approved gateway may provide machine
authentication; no particular product-side identity mechanism is assumed. The
[server constructor](../../../pkg/server/server.go#L338) configures recovery
and logging interceptors but no authentication interceptor, and the
[ingestion entrypoint](../../../cmd/ingestion/main.go) serves the gRPC server.
The [identity requirements](../04-identity-and-ingestion.md) describe two
possible mechanisms, neither of which should be presumed selected.

## Work and acceptance criteria

1. Implement or integrate the approved short-lived, per-reporter machine-token
   or mTLS design at the selected boundary, including issuer/audience/signature
   or client-CA validation, expiry, rotation, and the approved revocation bound.
   Fail closed if required trust configuration is absent or invalid; never log
   credentials. Do not treat a shared long-lived key or source allowlist alone
   as the package's approved default.
2. Ensure every event-bearing RPC goes through the same authentication boundary,
   including newly added methods. Produce a typed, validated principal and
   approved grant information for TP-02; do not infer identity from event data.
3. Use an authenticated, certificate-verified TLS path between reporter,
   gateway, and ingestion as required by the package and selected under D-06.
   If a gateway propagates identity, strip spoofed client headers and accept
   assertions only over the explicitly trusted, bypass-protected path.
4. Missing, invalid, expired, wrong-audience/issuer, untrusted, or revoked
   credentials fail with appropriate gRPC statuses and do not publish an event.
   Bound key refresh and authority outages without silently accepting unverifiable
   credentials. Preserve valid gRPC deadlines and retry/status behavior.
5. Document reporter setup and compatible migration/rotation. Preserve supported
   unauthenticated local Docker/AIO and Compose demonstrations without requiring
   an enterprise gateway or identity provider; keep local-only settings from
   silently weakening the distributed profile.
6. Before public exposure, verify TP-02, TP-06, TP-07's required stream bounds,
   TP-08's trusted broker path, and TP-14 as well as this packet against SCP-04,
   or obtain the specifically authorized, time-bounded exception. Authentication
   alone does not establish project authorization.

## Verification

- Test every RPC category and invalid case at the actual authentication boundary;
  rejected calls must publish nothing. Add `bufconn` tests for any application
  authentication or trusted-identity handling, and gateway configuration tests
  when it enforces authentication. Run `go test ./pkg/server ./tests` for
  application changes and the applicable gateway tests.
- In an authorized non-production deployment, execute V-02/03 through the real
  HTTP/2 gateway: valid and invalid credentials, wrong CA/host, forged identity,
  and direct-backend bypass. Record expiry/revocation timing against D-03.
- Verify the existing local reporter workflow still works without the new
  production credential setup.
- A unit test or TLS proxy alone is not evidence that project authorization or
  the deployed gateway path is secure.

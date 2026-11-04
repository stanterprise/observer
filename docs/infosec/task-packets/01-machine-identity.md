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
authorization, without trusting caller-supplied identity headers. The
[server constructor](../../../pkg/server/server.go#L338) configures recovery
and logging interceptors but no authentication interceptor, and the
[ingestion entrypoint](../../../cmd/ingestion/main.go) serves the gRPC server.
The [identity requirements](../04-identity-and-ingestion.md) describe two
possible mechanisms, neither of which should be presumed selected.

## Work and acceptance criteria

1. Implement the approved short-lived machine-token or mTLS design, including
   issuer/audience/signature or client-CA validation, expiry, rotation, and the
   approved revocation bound. Fail startup or requests closed if required trust
   configuration is absent or invalid; never log credentials.
2. Ensure every event-bearing RPC goes through the same authentication boundary,
   including newly added methods. Produce a typed, validated principal and
   approved grant information for TP-02; do not infer identity from event data.
3. Use an authenticated, certificate-verified TLS path between reporter,
   gateway, and ingestion as chosen under D-06. If a gateway propagates identity,
   strip spoofed client headers and accept assertions only over the explicitly
   trusted, bypass-protected path.
4. Missing, invalid, expired, wrong-audience/issuer, untrusted, or revoked
   credentials fail with appropriate gRPC statuses and do not publish an event.
   Bound key refresh and authority outages without silently accepting unverifiable
   credentials. Preserve valid gRPC deadlines and retry/status behavior.
5. Document reporter setup, compatible migration/rotation, and explicit
   development-only behavior without presenting it as production conformance.
   Do not expose ingestion until TP-02, TP-06 and TP-14 also satisfy SCP-04.

## Verification

- Add `bufconn` authentication tests for every RPC category and for all invalid
  cases above; check that a rejected call publishes nothing. Run
  `go test ./pkg/server ./tests`.
- In an authorized non-production deployment, execute V-02/03 through the real
  HTTP/2 gateway: valid and invalid credentials, wrong CA/host, forged identity,
  and direct-backend bypass. Record expiry/revocation timing against D-03.
- A unit test or TLS proxy alone is not evidence that project authorization or
  the deployed gateway path is secure.

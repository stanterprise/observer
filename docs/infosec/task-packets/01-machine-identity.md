# TP-01 - Authenticate Machine Ingestion

**Status:** Partially implemented: distributed Observer enforces shared-key
authentication, and the current Playwright and pytest reporter sources can send
the key. Project authorization, reporter release/deployment verification,
TLS evidence, and risk disposition remain open. No owner assigned.
**Priority:** P0; part of the public-ingestion gate.
**Accountable roles:** Application and identity owners, with platform support.
**Findings:** 1.
**Controls:** IAM-03, ING-01, ING-08/09, NET-01/02/16, SCP-04; tests V-02/03.
**Prerequisites:** D-03 and D-06 from [TP-00](00-decisions-and-scope.md).
Coordinate the principal contract with [TP-02](02-project-ownership.md) and the
gateway/trust path with [TP-14](14-network-and-platform.md).

## Objective and current state

Require authenticated submission on every event-bearing gRPC call, without
trusting caller-supplied identity headers. This is a public-ingestion requirement
even while product AuthN/AuthZ and human access controls are deferred. The
current implementation uses one shared bearer key from the
`INGESTION_AUTH_KEY` environment variable in distributed mode. The
[server constructor](../../../pkg/server/server.go#L338) configures recovery
and logging interceptors; the
[ingestion entrypoint](../../../cmd/ingestion/main.go) selects authenticated
distributed mode or keyless local mode. The
[identity requirements](../04-identity-and-ingestion.md) prefer per-reporter
identities; the shared key is a lower-assurance compromise and does not provide
a verified reporter principal, per-reporter revocation, or project
authorization. Do not treat it as satisfying SCP-04 by itself.

The current `stanterprise-playwright-reporter` and `pytest-observer` sources
read `STANTERPRISE_INGESTION_AUTH_KEY` and send it as bearer gRPC metadata on
ingestion calls, including retries. If unset, they send no authorization
metadata, preserving keyless local/AIO use. Their authentication diagnostics
do not print the key, and the key is not part of event/debug payloads. This
source-level support does not establish that a published package or deployed
CI job uses a compatible version or securely receives the key.

## Work and acceptance criteria

1. Require one random, high-entropy shared bearer key (at least 32 bytes) from
   `INGESTION_AUTH_KEY` in distributed mode. Load it from an operator-managed
   Kubernetes Secret; reject missing, weak, or malformed configuration at
   startup. Never put the key in Helm values, logs, source control, or
   diagnostics.
2. Require exactly one `authorization: Bearer <key>` gRPC metadata value on
   every event-bearing unary or streaming RPC. Compare credentials in constant
   time, return `Unauthenticated` for missing/invalid values, and never publish
   rejected requests. Keep only the health methods needed by probes
   unauthenticated.
3. Keep keyless local mode explicit and separate. AIO and local Docker/Compose
   workflows must not require, generate, or use an ingestion key. Do not allow
   local settings or gRPC reflection to be enabled in distributed mode.
4. Configure CI secret stores to provide `STANTERPRISE_INGESTION_AUTH_KEY` to
   both reporter clients. Do not put the key in reporter configuration files,
   event metadata, debug output, Helm values, logs, source control, or
   diagnostics. Document key provisioning, protected TLS transport, coordinated
   rotation/restart, and compromise response. This shared key authenticates
   possession only: it is not per-reporter identity, grants no project scope,
   and cannot provide per-reporter revocation.
5. Add a separate P0 TP-02 control that binds each submitted run to an authorized
   project before data is accepted. Until that is implemented and the shared-key
   risk is assessed, public ingestion remains blocked unless the authorized
   risk owner grants an explicit, time-bounded exception. Do not claim SCP-04
   satisfied by this packet alone.

## Verification

- Test missing, duplicate, malformed, weak, and incorrect bearer metadata on
  every RPC category; verify health probes remain usable and rejected calls
  publish nothing. Run `go test ./pkg/server ./cmd/ingestion ./tests`.
- Render the distributed chart and verify the referenced Secret supplies the
  key without placing it in rendered values; verify missing Secret references
  fail chart validation. Verify AIO/local templates set local mode and contain
  no key reference.
- In an authorized non-production deployment, test authenticated and
  unauthenticated reporters through TLS, invalid/missing keys, rotation,
  health-check behavior, reflection disabled, and direct-backend restrictions.
  Use the actual reporter package versions and CI secret-injection path intended
  for deployment; verify both reporter integrations end to end.
- Verify each reporter attaches one bearer credential on every ingestion call
  and retry, sends no authorization metadata when the key is unset, does not
  retry `UNAUTHENTICATED`, and emits no key in diagnostics or debug payloads.
- Verify local AIO, Compose, and reporter workflows remain keyless.
- A unit test or TLS proxy alone is not evidence that project authorization or
  the deployed gateway path is secure.

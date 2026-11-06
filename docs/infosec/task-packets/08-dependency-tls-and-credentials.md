# TP-08 - Secure Dependency Connections and Credentials

**Status:** Proposed; no owner assigned.
**Priority:** P2 for internal dependency TLS if deferred with approval; P1 for
external dependency TLS and scoped credentials; P0 for broker controls required
for trusted ingestion scope.
**Accountable roles:** Platform, application, identity, and data owners.
**Findings:** 11, 17, and 21.
**Controls:** IAM-05, NET-05 through NET-10, DATA-02, K8S-07/08/13/15,
OPS-05; tests V-09/10/12.
**Prerequisites:** D-01 and the workload-identity portion of D-04 for early
grants; D-06 for dependency TLS. Human SSO is not a prerequisite for broker
producer controls. Coordinate service accounts and migration job wiring with
[TP-12](12-kubernetes-workloads.md) and dependency deployment with
[TP-14](14-network-and-platform.md). Public reporter-to-gateway and
gateway-to-ingestion TLS remain P0 in TP-01/14, not part of this deferral.

## Objective and current state

Use verified TLS and distinct least-privilege identities for each dependency
path. [PostgreSQL connection configuration](../../../internal/database/postgres.go#L118)
defaults split-variable `sslmode` to `disable`; the
[migration job](../../../charts/observer/templates/distributed/migration/migration-job.yaml)
can use the runtime PostgreSQL credential. The chart's
[custom CA wiring](../../../charts/observer/templates/distributed/api/api-deployment.yaml#L90)
sets `AWS_CA_BUNDLE` for the relevant S3 path; that does not itself demonstrate
trust configuration for every database, broker, and migration connection.

The package requires verified TLS on the documented dependency paths, including
in-cluster connections. Scheduling internal TLS later is a proposed plan, not
an approved exception: D-06 identifies the selected transport but cannot waive
NET-05 through NET-10 or DATA-02. Any unmet requirement needs a scoped,
time-bounded exception from the appropriate authority before the applicable
approval gate. Prioritize connections crossing infrastructure boundaries according
to the actual threat and policy assessment; do not assume all dependencies are
inside one trusted cluster. Verify and implement TLS for external dependency
paths before use unless an applicable exception is approved. Do not disable a
dependency's existing TLS protection in order to defer work.

## Work and acceptance criteria

1. Inventory each selected NATS, PostgreSQL, MongoDB, and S3 endpoint, TLS
   termination, hostname, trust roots, credentials, and role. The full target
   requires authenticated TLS with hostname/certificate verification; reject
   wrong CA, wrong hostname, expired certificates, and insecure production
   configuration unless an explicitly approved exception covers the exact path.
2. Support and document CA distribution and reload/rotation for every actual
   client path, including ingestion, processor, API, and migration. Verify
   NATS URL/client-library behavior rather than assuming TLS is unavailable or
   automatically secure. Avoid disabling verification to make a connection work.
3. **Early workload milestone:** Issue distinct scoped credentials: ingestion
   publishes only required broker subjects; processor and API use their own
   consumer and database/storage privileges; MongoDB is limited to the
   live-buffer scope; migration has a time-bounded schema identity not mounted
   in runtime pods. Ensure privileges cover actual required JetStream consumer
   operations without granting broad administration. Complete the producer
   restrictions or equivalent trust boundary needed by TP-02 before public
   ingestion, even if other credential work remains scheduled for later.
   Assess any alternative separately against the package's broker-credential
   and subject requirements; a network boundary does not by itself satisfy them.
4. Separate stream/schema provisioning from runtime privileges where required:
   the current publisher's stream-creation behavior must not force a
   publish-only workload to hold broker-administrator rights.
5. Wire secret references per workload and document compatibility and rotation.
   Prefer approved workload federation where supported; never place reusable
   credentials in values, release metadata, source, logs, or evidence. Verify
   Kubernetes secret encryption and storage encryption through platform evidence.
   Do not require distributed workload credentials or verified dependency TLS
   for currently supported local Docker/AIO and Compose demos.

## Verification

- Add configuration and integration tests for all four dependency types with a
  valid certificate, wrong hostname/CA, expired certificate, missing or revoked
  credentials, forbidden privileges, and successful rotation. Verify a
  publish-only identity cannot administer streams and the runtime DB users cannot
  perform migrations. Track the early credential milestone separately; mark
  deferred TLS tests and controls blocked or not met, not passed.
- Run `go test ./internal/database ./pkg/publisher ./pkg/consumer ./pkg/storage
  ./pkg/websocket` and `HELM_REQUIRE_DEPENDENCIES=true make helm-test` for chart
  changes. Use an authorized environment for V-09/10/12 and record actual
  grants, trust roots, and rotation timing in sanitized form. Verify existing
  local dependency configurations still work after shared connection changes.

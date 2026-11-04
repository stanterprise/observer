# TP-08 - Secure Dependency Connections and Credentials

**Status:** Proposed; no owner assigned.
**Priority:** P0 for the selected production dependency paths.
**Accountable roles:** Platform, application, identity, and data owners.
**Findings:** 11, 17, and 21.
**Controls:** IAM-05, NET-05 through NET-10, DATA-02, K8S-07/08/13/15,
OPS-05; tests V-09/10/12.
**Prerequisites:** D-01/D-04/D-06. Coordinate service accounts and migration
job wiring with [TP-12](12-kubernetes-workloads.md) and dependency deployment
with [TP-14](14-network-and-platform.md).

## Objective and current state

Use verified TLS and distinct least-privilege identities for each dependency
path. [PostgreSQL connection configuration](../../../internal/database/postgres.go#L118)
defaults split-variable `sslmode` to `disable`; the
[migration job](../../../charts/observer/templates/distributed/migration/migration-job.yaml)
can use the runtime PostgreSQL credential. The chart's
[custom CA wiring](../../../charts/observer/templates/distributed/api/api-deployment.yaml#L90)
sets `AWS_CA_BUNDLE` for the relevant S3 path; that does not itself demonstrate
trust configuration for every database, broker, and migration connection.

## Work and acceptance criteria

1. Inventory each selected NATS, PostgreSQL, MongoDB, and S3 endpoint, TLS
   termination, hostname, trust roots, credentials, and role. Require authenticated
   TLS with hostname/certificate verification; reject wrong CA, wrong hostname,
   expired certificates, and insecure production configuration.
2. Support and document CA distribution and reload/rotation for every actual
   client path, including ingestion, processor, API, and migration. Verify
   NATS URL/client-library behavior rather than assuming TLS is unavailable or
   automatically secure. Avoid disabling verification to make a connection work.
3. Issue distinct scoped credentials: ingestion publishes only required broker
   subjects; processor and API use their own consumer and database/storage
   privileges; MongoDB is limited to the live-buffer scope; migration has a
   time-bounded schema identity not mounted in runtime pods. Ensure privileges
   cover actual required JetStream consumer operations without granting broad
   administration.
4. Separate stream/schema provisioning from runtime privileges where required:
   the current publisher's stream-creation behavior must not force a
   publish-only workload to hold broker-administrator rights.
5. Wire secret references per workload and document compatibility and rotation.
   Prefer approved workload federation where supported; never place reusable
   credentials in values, release metadata, source, logs, or evidence. Verify
   Kubernetes secret encryption and storage encryption through platform evidence.

## Verification

- Add configuration and integration tests for all four dependency types with a
  valid certificate, wrong hostname/CA, expired certificate, missing or revoked
  credentials, forbidden privileges, and successful rotation. Verify a
  publish-only identity cannot administer streams and the runtime DB users cannot
  perform migrations.
- Run `go test ./internal/database ./pkg/publisher ./pkg/consumer ./pkg/storage
  ./pkg/websocket` and `HELM_REQUIRE_DEPENDENCIES=true make helm-test` for chart
  changes. Use an authorized environment for V-09/10/12 and record actual
  grants, trust roots, and rotation timing in sanitized form.

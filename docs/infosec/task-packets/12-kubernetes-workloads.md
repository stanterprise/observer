# TP-12 - Harden Helm Workloads and Migration Jobs

**Status:** Proposed; no owner assigned.
**Priority:** P1.
**Accountable roles:** Platform, application, and release owners.
**Findings:** 15, 16, 17, 19, 20, and 23.
**Controls:** K8S-01 through K8S-04, K8S-10/11/13/15/16; tests V-09/12/14.
**Prerequisites:** D-01/D-07. Coordinate per-workload credential contracts with
[TP-08](08-dependency-tls-and-credentials.md), image digest wiring with
[TP-13](13-supply-chain.md), and admission/network evidence with
[TP-14](14-network-and-platform.md).

## Objective and current state

Make the selected distributed chart workloads compatible with the approved
restricted pod baseline and explicit resource/availability policy. The
[chart defaults](../../../charts/observer/values.yaml#L169) run the web container
as root; the shared distributed security context drops capabilities but does not
enable read-only root or specify seccomp. The chart provides one
[service account](../../../charts/observer/templates/serviceaccount.yaml), and
the [migration hook](../../../charts/observer/templates/distributed/migration/migration-job.yaml)
has retry and cleanup settings but no explicit active deadline.

## Work and acceptance criteria

1. Run each distributed workload and migration container non-root, prohibit
   privilege escalation and unnecessary capabilities/host access, and use
   `RuntimeDefault` seccomp. Use read-only root filesystems where feasible, with
   narrowly scoped writable volumes and documented exceptions. Set non-root
   runtime users in the distributed application images as well as pod specs.
   Preserve the existing web Service contract even if the internal listener
   changes; do not assume the AIO supervisor can run unmodified without root.
2. Give ingestion, processor, API, web, and migration distinct service accounts;
   disable token automount unless a documented workload-identity integration
   requires a scoped token. Prove effective RBAC does not include runtime
   secret listing or cluster administration. Coordinate separate data-service
   credentials with TP-08.
3. Bound the migration job's execution, retries, resources, and identity; do
   not reuse its schema-change credential in runtime pods. Document hook
   ordering, upgrade, rollback, and migration compatibility. Support the
   selected embedded/external dependency modes.
4. Set or expose approved CPU, memory, ephemeral storage, replica and capacity
   limits. Configure safe probes, bounded graceful shutdown, and disruption/HA
   behavior appropriate to D-07 and availability targets. Add a PDB or startup
   probe where justified rather than asserting either is universally required.
5. Test the selected embedded dependency workloads and init containers against
   the same baseline; explicitly record any narrow exceptions. Do not silently
   break AIO/Compose development or chart Service ports while hardening the
   distributed production profile.
6. Extend the existing [Helm contract tests](../../../scripts/test-helm-chart.sh)
   and [chart CI](../../../.github/workflows/helm-publish.yml); do not introduce
   a second test framework merely because the preliminary review overlooked
   existing coverage. Keep the chart schema, values, and README aligned.

## Verification

- Run `HELM_REQUIRE_DEPENDENCIES=true make helm-test` and `make helm-lint`;
  verify the default, production, AIO, embedded, external, and migration variants
  relevant to the change. The test script can otherwise skip rendering when
  dependencies are unavailable, so record skipped or blocked runs explicitly.
- Assert rendered service-account, token, security-context, volume, resource,
  image, probe, and migration settings, including negative misconfiguration
  cases. In a non-production cluster, test restricted admission, workload
  startup, graceful termination, effective RBAC, and migration rollback.
- Record V-09/12/14 evidence, selected workload exceptions, and the actual
  admission policy; successful Helm rendering alone is not enforcement.

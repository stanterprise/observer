# Helm Chart Publish-Readiness Rubric

**Purpose:** Give downstream infrastructure repositories a repeatable way to decide whether the Observer chart can be consumed directly, wrapped internally, or blocked from adoption.
**Last Reviewed:** 2026-09-22

## Scoring Scale

Use the same scale for every domain:

- `0` Broken: advertised behavior does not work, is misleading, or cannot be safely consumed.
- `1` Internal-only: works only with team-specific knowledge, manual patching, or undocumented assumptions.
- `2` Controlled: usable by a private infrastructure repository with some wrappers, guardrails, or pinned conventions.
- `3` Public-ready: works as documented for external consumers and is backed by release evidence.

## Downstream Expectation Model

A separate infrastructure repository should expect the upstream chart to provide:

- a stable values contract for supported install modes
- working external dependency paths for any feature claimed as supported
- secret-safe configuration patterns
- deterministic install and upgrade behavior
- validated networking examples for exposed public endpoints
- release notes that explain behavioral changes and deprecations

If the chart cannot provide those guarantees, the downstream repository should treat it as an internal dependency and own the missing guardrails itself.

## Domain Rubric

### 1. Installation Contract

**What the infra repo should expect:** default install, recommended presets, and documented optional modes all render and install successfully.

| Score | Criteria                                                                                                |
| ----- | ------------------------------------------------------------------------------------------------------- |
| `0`   | Default or documented install paths fail to render, collide, or require manual chart edits.             |
| `1`   | One install path works, but other advertised presets or modes are broken or ambiguous.                  |
| `2`   | Core install paths work, but some optional surfaces still rely on repo-specific knowledge or wrappers.  |
| `3`   | Every supported install path is render-tested, install-tested, and documented with clear prerequisites. |

**Evidence:** `helm lint`, render matrix, install smoke tests, supported mode documentation.

### 2. Configuration Contract

**What the infra repo should expect:** values files, schema, and docs describe the same public API.

| Score | Criteria                                                                                                         |
| ----- | ---------------------------------------------------------------------------------------------------------------- |
| `0`   | Values contract is inconsistent, nil-prone, or contradicted by examples and docs.                                |
| `1`   | Core values work, but optional trees or example files drift from the real contract.                              |
| `2`   | Core values are stable, but some optional features lack schema validation or failure guards.                     |
| `3`   | Values are schema-backed, examples stay in sync, and unsupported combinations fail early with actionable errors. |

**Evidence:** `values.yaml`, `values.schema.json`, example values renders, doc review.

### 3. External Dependency Contract

**What the infra repo should expect:** if the chart claims support for external PostgreSQL, MongoDB, NATS, or other services, those paths work without hidden coupling to embedded dependencies.

| Score | Criteria                                                                                                       |
| ----- | -------------------------------------------------------------------------------------------------------------- |
| `0`   | Claimed external modes are broken or still reference embedded services.                                        |
| `1`   | External modes can work, but only after local template patches or undocumented overrides.                      |
| `2`   | External modes work for core dependencies, but some combinations or examples remain weakly tested.             |
| `3`   | Every advertised external dependency path is CI-covered, documented, and free of embedded-service assumptions. |

**Evidence:** render matrix for external modes, install tests, explicit docs for each dependency path.

### 4. Secrets And Security Contract

**What the infra repo should expect:** secrets are handled safely enough that the downstream repo does not need to patch templates just to avoid leaking credentials.

| Score | Criteria                                                                                                                              |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `0`   | Passwords are hardcoded in public defaults or rendered directly into manifests without an acceptable secret path.                     |
| `1`   | Some secret references exist, but major paths still rely on raw values or incomplete secret wiring.                                   |
| `2`   | Secret-backed paths exist for core services, but optional features or examples still expose unsafe patterns.                          |
| `3`   | Public defaults avoid reusable passwords, secret references are first-class, and restricted-cluster security defaults are documented. |

**Evidence:** workload templates, values contract, example manifests, security review.

### 5. Networking And Exposure Contract

**What the infra repo should expect:** ingress, Gateway API, TLS, and gRPC exposure options are explicit, coherent, and controller-aware.

| Score | Criteria                                                                                                                       |
| ----- | ------------------------------------------------------------------------------------------------------------------------------ |
| `0`   | Networking surfaces conflict, duplicate resources, or rely on hidden cloud-controller assumptions.                             |
| `1`   | One environment works, but support boundaries are not clearly defined.                                                         |
| `2`   | Recommended networking paths are usable, while non-primary paths are clearly marked as limited or experimental.                |
| `3`   | Each supported exposure model is validated, controller-specific behavior is gated, and public endpoint guidance is consistent. |

**Evidence:** ingress or gateway render tests, TLS examples, controller-specific docs.

### 6. Upgrade And Migration Contract

**What the infra repo should expect:** upgrades, schema changes, and rollbacks have a documented and deterministic path.

| Score | Criteria                                                                                                       |
| ----- | -------------------------------------------------------------------------------------------------------------- |
| `0`   | Migration behavior is duplicated, unsafe, or undocumented.                                                     |
| `1`   | Upgrade path exists, but operators still need internal knowledge to avoid outages.                             |
| `2`   | Upgrade and migration behavior is usable for maintained environments, with some remaining operational caveats. |
| `3`   | Upgrade, rollback, and migration strategy are documented, tested, and release-noted.                           |

**Evidence:** migration hooks or jobs, release notes, upgrade docs, install-to-upgrade test results.

### 7. Operability Contract

**What the infra repo should expect:** the chart is supportable after install without forking basic health or scaling behavior.

| Score | Criteria                                                                                                |
| ----- | ------------------------------------------------------------------------------------------------------- |
| `0`   | Probes, init behavior, or notes are misleading enough to break normal operations.                       |
| `1`   | Core runtime works, but day-2 operations still depend on team knowledge.                                |
| `2`   | Health, scaling, and access patterns are mostly usable, with limited rough edges.                       |
| `3`   | Probes, autoscaling, service exposure, and post-install instructions reflect actual operator workflows. |

**Evidence:** template review, smoke tests, operational runbooks, `NOTES.txt` validation.

### 8. Documentation And Release Contract

**What the infra repo should expect:** the upstream chart behaves like a product, not an internal folder.

| Score | Criteria                                                                                                                |
| ----- | ----------------------------------------------------------------------------------------------------------------------- |
| `0`   | README and release artifacts materially misrepresent the chart.                                                         |
| `1`   | Docs exist, but defaults, examples, and support boundaries drift from reality.                                          |
| `2`   | Docs are serviceable for internal consumers, but still require maintainers to explain caveats out of band.              |
| `3`   | README, deployment docs, architecture docs, release notes, and published artifacts are current and mutually consistent. |

**Evidence:** README review, docs index, CHANGELOG, OCI package metadata, release notes.

## Decision Policy

Use the rubric to make one of three decisions:

### 1. Block Direct Consumption

Choose this when:

- any of domains 1 through 4 scores `0`
- install, secret, or external dependency behavior is misleading
- the infra repo would need to patch the upstream chart just to make the advertised basics work

### 2. Consume Behind An Internal Wrapper

Choose this when:

- domains 1 through 4 score at least `2`
- one or more domains 5 through 8 still score `1` or `2`
- the infra repo can safely add policy, defaults, and secret wiring without carrying template forks

Typical wrapper responsibilities:

- pinning safe defaults
- enforcing secret references
- selecting one supported ingress or gateway pattern
- disabling experimental surfaces
- guarding external dependency combinations

### 3. Consume Upstream Directly

Choose this only when:

- domains 1 through 4 all score `3`
- domains 5 through 8 score at least `2`, with a target of `3`
- no release blocker from the hardening checklist remains open
- published OCI artifacts and release notes are available for the pinned version

## Release Evidence Bundle

Before a downstream infra repo accepts a new chart version, expect these artifacts from upstream:

- packaged chart version and OCI reference
- matching `Chart.yaml` and `Chart.lock`
- CI results for lint, render matrix, and install smoke tests
- release notes or changelog entry for behavior changes
- supported values files and any deprecation notice for renamed keys or removed modes

## Practical Guidance For Observer

Based on the 2026-09-22 review against the current `charts/observer` state:

- `3` for installation, configuration, and external dependency contracts (domains 1-3): every advertised mode renders, install-tests via `scripts/test-helm-chart.sh`, and external PostgreSQL/MongoDB/NATS paths fail render early with actionable errors when required endpoints are missing.
- `3` for secrets handling (domain 4): public defaults ship no reusable passwords, and `NATS_URL`/`POSTGRES_DSN`/`MONGODB_URI` are always injected via `secretKeyRef`, never as literal `value:` entries.
- `2` for networking and exposure (domain 5): the chart-managed boundary (no Ingress/Gateway/cloud-specific resources) is clear and documented, but downstream ingress/TLS guidance for k3s/homelab-style clusters lives only in archived, unmaintained notes (see [archive/2026-09-copilot/](archive/2026-09-copilot/)).
- `1` to `2` for upgrade/migration (domain 6): the migration hook is single-path and documented for `helm upgrade`, but there is no documented rollback story or compatibility matrix for stateful dependency schema/data changes.
- `2` for operability (domain 7): probes, HPA, and `NOTES.txt` reflect real behavior; no PodDisruptionBudget or NetworkPolicy hooks exist yet.
- `2` to `3` for documentation (domain 8): chart README and values contract are in sync as of this review; the remaining gap is an automated, real-cluster install/upgrade/rollback smoke test (CI currently only renders, schema-validates, and inspects packaged artifacts).

That places the chart in **Consume Behind An Internal Wrapper**, moving toward **Consume Upstream Directly** once a real-cluster smoke test (checklist HC083) and stateful-dependency upgrade/rollback documentation are added.

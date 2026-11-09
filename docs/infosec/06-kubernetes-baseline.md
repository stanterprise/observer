# 06 - Kubernetes and Helm Security Baseline

**Status:** Expected deployment baseline; chart defaults are not evidence of compliance.

## Required controls

| ID | Requirement | Responsible role | Expected evidence |
| --- | --- | --- | --- |
| K8S-01 | Supported cluster/node/controller/dependency versions with documented patch ownership | Platform | Version inventory and support policy |
| K8S-02 | Dedicated namespace; enforced restricted pod-security posture or documented narrow exceptions | Platform | Namespace/admission policy and admission tests |
| K8S-03 | Run non-root, drop capabilities, prohibit privilege escalation/host namespaces/hostPath, use runtime-default seccomp and read-only root filesystem where feasible | Application + platform | Rendered pod specs; justified writeable volumes/exceptions |
| K8S-04 | Separate service accounts; disable token automount unless required; least-privilege RBAC; no runtime secret-listing or cluster-admin | Platform | Service accounts, bindings, effective permission checks |
| K8S-05 | Default-deny ingress/egress with explicit flows from document 03, enforced by the chosen CNI | Platform | Policies and positive/negative connectivity tests |
| K8S-06 | No public dependency, API bypass, monitoring, or administration endpoint | Platform | Services, gateway routes, external exposure verification |
| K8S-07 | External secret management or approved equivalent; encrypted Kubernetes secrets at rest; scoped mounts/env; rotation without leakage | Platform + identity | Sanitized references, KMS/access policy, rotation drill |
| K8S-08 | Authenticated TLS for dependency access; verified hostnames/trust roots; scoped DB/broker/storage roles | Platform + application | Sanitized connection policy, ACLs, negative access tests |
| K8S-09 | Pin reviewed images by digest or enforce verified digest resolution; control registries and signature/provenance admission | Release + platform | Release manifest, admission policy, provenance |
| K8S-10 | Bound CPU/memory, ephemeral storage, replicas, namespace quotas, PVC/object/broker capacity; protect critical dependencies from ingestion overload | Operations | Rendered limits, quota policy, capacity exercise |
| K8S-11 | Safe probes and graceful termination; restrict disruption; select replica/HA policy against approved availability targets | Application + operations | Probe specs, PDB where applicable, failover exercise |
| K8S-12 | Restrict persistent storage and snapshots; approved encryption, reclaim and backup policy | Platform + data | StorageClass/PVC and provider configuration |
| K8S-13 | Dedicated bounded migration job and schema identity; reviewed SQL; backup/rollback compatibility plan | Release + data | Hook manifest, role grants, migration rehearsal |
| K8S-14 | Secure web configuration: explicit origins, safe proxy trust, headers, TLS routes, no public debug endpoints | Application + platform | Proxy/API configuration and response checks |
| K8S-15 | Review embedded dependency subcharts separately; their security contexts, credentials, TLS, storage and exposure must meet this baseline | Platform + release | Dependency-specific evidence |
| K8S-16 | Keep production data and credentials out of chart source, public values, Helm command history, build logs, and review exports | All owners | Secret workflow and sanitized artifacts |

## Helm deployment contract

The chart owns workloads, Services, resources/probes/security-context configuration,
and the dedicated PostgreSQL migration hook when enabled. The operator owns
ingress/gateway routing, DNS, certificates, private CA distribution, external secret
integration, NetworkPolicies, and cluster-level controls.

Use the [chart README](../../charts/observer/README.md) for supported values:

- Supply distributed connection strings through `runtime.existingSecret` for
  production; never paste credentials into this package.
- Supply S3 settings through `storage.s3.existingSecret`; prefer workload identity
  where supported and assessed.
- Use `customCA.existingSecret` where private endpoints require custom trust.
  Verify trust for every connection, not just S3.
- Select embedded or external dependencies explicitly. Do not assume disabling a
  subchart configures a secured managed service.
- Keep operator-owned exposure/policy manifests under change control alongside
  sanitized values and dependency versions.

Do not assume the current chart can express every requirement. Record missing
capabilities as implementation work rather than inventing unsupported Helm keys
or treating unset values as secure defaults.

## Baseline acceptance

Render the exact selected chart, dependency versions, values, and operator bundle.
Assess every workload, including web, migration hooks, dependency workloads,
init containers, and supporting proxies. Any exception must identify its smallest
scope, compensating control, owner, expiry, and verification evidence.

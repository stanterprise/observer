# 08 - Supply Chain and Security Verification

**Status:** Required evidence and planned verification; no scans or security tests
are asserted to have been performed by this package.

## Supply-chain requirements

- **SUP-01:** Inventory exact application images, base images, reporter versions,
  Helm chart and locked dependencies, gateways/proxies, and data services. Associate
  immutable digests with source revision and build identity.
- **SUP-02:** Produce machine-readable SPDX or CycloneDX SBOMs for submitted images
  and relevant frontend/backend dependencies. Identify generator/version/date and
  correlate to the deployed digest.
- **SUP-03:** Attach dated dependency, container, secret, static-analysis, and IaC/
  manifest scan results. Record tool versions, rules/database freshness, coverage,
  exclusions, unresolved findings, and remediation/exception decisions.
- **SUP-04:** Protect source/build/release workflows with review and least-privilege
  credentials; verify provenance/signatures against trusted builders before deployment.
  Document registry access and artifact integrity policy.
- **SUP-05:** Define vulnerability intake, triage, patch SLAs, emergency rebuilds,
  dependency support/EOL review, and exception expiry. A passing scan is not proof
  that a build is free of vulnerabilities.

## Verification plan

Use an authorized non-production deployment and synthetic data. Agree scope,
rate ceilings, allowed targets, stop conditions, and cleanup before abuse/load
testing. Do not scan unrelated cluster tenants or third-party services without
authorization.

| Test ID | Requirements / scenario | Expected result and required evidence |
| --- | --- | --- |
| V-01 | SCP-01; SUP-01/02 | Evidence matches exact image digests, chart, dependencies, rendered configuration and SBOMs |
| V-02 | NET-01/02/16; ING-01/08 | Valid gRPC works over public TLS/HTTP2; missing, invalid, expired credentials and untrusted certificates fail; status/deadline behavior preserved |
| V-03 | IAM-02/03; ING-02; T-01/05 | Cross-project submission, ownership changes, forged identity headers, and direct gateway-bypass attempts are denied |
| V-04 | IAM-01/02/06; DATA-05 | Unauthorized REST reads/mutations/downloads and CSRF attempts fail; role/project matrix tested |
| V-05 | IAM-04; DATA-06 | WS subscriptions cannot leak other projects; token/session expiry and revocation close or reauthorize within approved bounds |
| V-06 | ING-03/04 | Malformed, oversized, duplicate, replayed and reordered events follow defined behavior without corruption or silent legitimate-event loss |
| V-07 | ING-05/06; K8S-10 | At and just above each approved limit, bounded resource usage and explicit throttling/rejection; backlog/storage/replicas stay within approved caps |
| V-08 | DATA-03/04/05 | Traversal, cross-project keys, unsafe active content, disallowed types and remote-fetch destinations are blocked; signed URLs obey scope/expiry if used |
| V-09 | NET-05 through NET-18; K8S-04/05/06/08 | Required flows work; forbidden pod/dependency/public/admin paths fail; service identities cannot exceed grants |
| V-10 | DATA-02; K8S-07/08; OPS-05 | TLS rejects wrong host/CA; rotation succeeds; revoked credentials cease access within approved bound; secrets absent from ordinary logs |
| V-11 | DATA-07/08 | Expired/deleted runs leave no unauthorized accessible copies across stores/versions; orphan buffers expire; restored backups reapply deletions |
| V-12 | K8S-02/03/09/13/15 | Rendered workloads meet admission policy; unsigned/untrusted artifacts rejected under selected policy; migrations use isolated privileges |
| V-13 | OPS-01 through OPS-04 | Security events produce sanitized audit records and actionable alerts; access/tampering controls and response routing verified |
| V-14 | ING-06; OPS-06/07/08 | Dependency outage/recovery, restore, upgrade and rollback meet agreed integrity, RPO/RTO and compatibility criteria |
| V-15 | SUP-03/04/05 | Dated scans and provenance validate the submitted build; exclusions and unresolved findings have explicit disposition |

For quantitative tests, record configured limit, observed value, workload shape,
duration, monitoring source, expected boundary behavior, and pass/fail criterion.
Testing a small payload is not evidence that a maximum-size requirement is enforced.

## Independent testing

InfoSec should decide whether a scoped penetration test is required, covering the
public gRPC endpoint, web/API/WS access, artifact handling, project isolation, and
gateway bypass. Record authorization, tested revision/configuration, limitations,
findings, and retest evidence. Planned testing must never be reported as completed.

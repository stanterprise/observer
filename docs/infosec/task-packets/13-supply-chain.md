# TP-13 - Secure Release Artifacts and Supply Chain

**Status:** Proposed; no owner assigned.
**Priority:** P2; required for final approval where applicable.
**Accountable roles:** Release, platform, and InfoSec owners.
**Findings:** 18 and 24 through 26.
**Controls:** SUP-01 through SUP-05, K8S-09/16, OPS-06; tests V-01/12/15.
**Prerequisites:** D-01/D-09. Coordinate deployment admission with
[TP-14](14-network-and-platform.md) and final evidence with
[TP-16](16-assessment-and-approval.md).

## Objective and current state

Tie the deployed images and chart to reviewed source, reproducible inventory,
scans, and verified provenance. The [chart values](../../../charts/observer/values.yaml#L9)
select image tags; [Docker publishing](../../../.github/workflows/docker-publish.yml)
builds and retags images. [Helm publishing](../../../.github/workflows/helm-publish.yml)
already performs chart tests and kubeconform schema validation; those checks are
not equivalent to the full required security scanning and provenance evidence.

## Work and acceptance criteria

1. Support immutable image digests for every selected distributed component,
   with explicit precedence and validation versus tags. Keep documented
   tag-based development compatibility without treating it as production
   evidence. Inventory chart, locked dependencies, base images, reporter,
   proxies, data-service versions, source revision, and build identity.
2. Produce machine-readable SPDX or CycloneDX SBOMs for submitted images and
   relevant backend/frontend dependencies, correlated to the deployed digest,
   generator version, and build date.
3. Add dated dependency, container, secret, static-analysis, and IaC/manifest
   security scans. Record tool versions, rules/database freshness, coverage,
   exclusions, findings, and disposition. Existing kubeconform is useful schema
   validation, not a substitute for these scans.
4. Implement approved signing and provenance for release images and charts, and
   verify trusted builder identity and digest before deployment, including
   image-retag and packaged-chart paths. Verify registry and cluster admission
   enforcement; generating an attestation without verifying it is insufficient.
5. Harden source/build/release access: least-privilege workflow permissions,
   reviewed action/base-image pins, dependency updates, and protected review
   policy. Add or update CODEOWNERS and repository policy as appropriate, while
   treating actual branch protection as separately verified platform evidence.
6. Update the [vulnerability reporting policy](../../../SECURITY.md) with the
   approved intake/triage process, patch SLAs, emergency rebuild and
   time-bounded exception workflow. Do not invent organizational deadlines.

## Verification

- Extend chart and workflow tests for tag/digest precedence, malformed digests,
  all selected component images, locked dependencies, and release/retag paths;
  run `HELM_REQUIRE_DEPENDENCIES=true make helm-test` for chart changes.
- On an approved test release, verify each SBOM, scan record, signature, and
  provenance matches the actual deployed digest. Demonstrate rejection of an
  unsigned, wrong-builder, or tampered artifact under the selected admission
  policy. Record V-01/12/15 results and unresolved finding dispositions.

# TP-14 - Secure Gateways, Networks, and Platform Dependencies

**Status:** Proposed; no owner assigned.
**Priority:** P0; gateway-bypass prevention is part of the ingestion gate.
**Accountable roles:** Platform, identity, data, and application owners.
**Findings:** 21 and 22.
**Controls:** NET-01 through NET-19, K8S-01/02/05/06/07/08/12/15,
SCP-04; tests V-02/03/09/10/12.
**Prerequisites:** D-01/D-02/D-06. Coordinate dependency clients and grants with
[TP-08](08-dependency-tls-and-credentials.md) and workload labels/accounts with
[TP-12](12-kubernetes-workloads.md).

## Objective and current state

Deliver and verify the operator-owned network, TLS, secret, and storage controls
around the selected chart. Distributed Services default to `ClusterIP` in
[values.yaml](../../../charts/observer/values.yaml), but can be configured
otherwise. The [chart contract](../../../charts/observer/README.md) assigns
gateway, DNS, certificates, and NetworkPolicies to the operator; a chart render
does not prove those controls are deployed or enforced.

## Work and acceptance criteria

1. Select the gateway/controller, exposure model, DNS, TLS termination and
   re-encryption, trust roots, certificate lifecycle, and SSO integration.
   Publish only approved gateway routes; deny direct access to ingestion, API,
   dependency, monitoring, and administration endpoints. Preserve HTTP/2 gRPC,
   status/deadline behavior, HTTPS/WSS, and approved WebSocket upgrades.
2. Maintain a versioned operator bundle or clearly scoped examples for default
   deny and explicit ingress/egress flows in the
   [network matrix](../03-deployment-and-network.md#network-flow-matrix). Cover
   gateway namespaces, DNS, identity, telemetry, secrets, administration and
   backup exceptions, and selected managed services. Validate that the chosen
   CNI actually enforces the policies, including any needed external-endpoint
   egress controls.
3. Configure and separately assess each selected embedded or managed dependency:
   private exposure, supported versions, TLS, scoped ACLs, security contexts,
   resource/storage bounds, and administrative paths. Coordinate broker/DB/S3
   application credentials with TP-08 rather than granting shared admin access.
4. Configure approved secret management, encrypted Kubernetes secrets, storage
   and snapshots, key ownership/rotation, residency, and access/reclaim policy.
   If signed artifact URLs are used, include the browser-to-storage path and
   its controls in the actual flow matrix.
5. Keep application-chart and platform ownership explicit. Do not silently add
   or publish provider-specific ingress resources from the portable chart, or
   treat an illustrative manifest as production evidence.

## Verification

- For the exact deployment, render and inspect the chart and operator bundle;
  test each required connection and attempt forbidden cross-pod, cross-namespace,
  public, dependency-admin, and direct-backend paths. Confirm allowed traffic
  still works under default deny.
- Execute V-02/03/09/10/12 with valid and invalid certificates, forged identity,
  denied network paths, subject/role ACLs, and certificate rotation. Record CNI,
  gateway, endpoint, and policy versions and sanitized results.

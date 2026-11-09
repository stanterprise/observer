# TP-14 - Secure Gateways, Networks, and Platform Dependencies

**Status:** Proposed; no owner assigned.
**Priority:** P0 for the public gRPC/private web split and ingestion gate.
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

The intended exposure is a public TLS gRPC gateway for external CI and private
organizational access to web, API, WebSocket, and artifacts. The private network
boundary is an important interim mitigation, not proof of human identity,
per-project authorization, or compliance with the package's SSO requirements.
Verify both intended access and denied alternate paths in the actual deployment.

## Work and acceptance criteria

1. Select the gateway/controller, exposure model, DNS, TLS termination and
   re-encryption, trust roots, and certificate lifecycle. Expose only the
   approved gRPC route publicly; restrict web, API, WebSocket, and artifact
   routes to approved private/corporate access. Deny direct or alternate
   public and unauthorized in-cluster access to ingestion, API, dependency,
   monitoring, and administration endpoints. Preserve HTTP/2 gRPC,
   status/deadline behavior, HTTPS/WSS, and approved WebSocket upgrades.
2. Coordinate reporter authentication and machine project/run authorization
   with TP-01/02. Prevent gateway bypass and forged propagated identities;
   verify the gateway and downstream enforcement together. A private endpoint,
   source allowlist, or authenticated gateway without project/run checks does
   not satisfy SCP-04.
3. Decide whether an identity-aware private web gateway is available as an
   interim human-authentication control. If not, record the applicable gaps and
   proposed exceptions under TP-16; gateway sign-in alone is not proof of
   server-side project authorization. Do not silently downgrade NET-03/04.
4. Maintain a versioned operator bundle or clearly scoped examples for default
   deny and explicit ingress/egress flows in the
   [network matrix](../03-deployment-and-network.md#network-flow-matrix). Cover
   gateway namespaces, DNS, identity, telemetry, secrets, administration and
   backup exceptions, and selected managed services. Validate that the chosen
   CNI actually enforces the policies, including any needed external-endpoint
   egress controls.
5. Configure and separately assess each selected embedded or managed dependency:
   private exposure, supported versions, TLS, scoped ACLs, security contexts,
   resource/storage bounds, and administrative paths. Internal dependency TLS
   scheduled later requires the applicable approved exception; it is not waived
   merely by selecting D-06. Coordinate broker/DB/S3 application credentials
   with TP-08 rather than granting shared admin access.
6. Configure approved secret management, encrypted Kubernetes secrets, storage
   and snapshots, key ownership/rotation, residency, and access/reclaim policy.
   If signed artifact URLs are used, include the browser-to-storage path and
   its controls in the actual flow matrix.
7. Keep application-chart and platform ownership explicit. Do not silently add
   or publish provider-specific ingress resources from the portable chart, or
   treat an illustrative manifest as production evidence. Do not impose these
   platform integrations on supported local Docker/AIO or Compose workflows.

## Verification

- For the exact deployment, render and inspect the chart and operator bundle;
  test each required connection and attempt forbidden cross-pod, cross-namespace,
  public web/API/WebSocket/artifact, dependency-admin, and direct-backend paths,
  including alternate Services and gateway routes. Confirm external approved CI
  can use gRPC and approved private users can reach web/API/WS under default deny.
- Execute V-02/03/09/10/12 with valid and invalid certificates, forged identity,
  denied network paths, reporter project/run denial, subject/role ACLs, and
  certificate rotation. Record CNI, gateway, endpoint, and policy versions and
  sanitized results. Mark deferred human access checks as not met or otherwise
  assessed, not passed because the network is private.

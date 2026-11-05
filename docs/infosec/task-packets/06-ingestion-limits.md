# TP-06 - Validate and Bound Ingestion

**Status:** Proposed; no owner assigned.
**Priority:** P0; part of the public-ingestion gate.
**Accountable roles:** Application and operations owners.
**Findings:** 8 and 10.
**Controls:** ING-03/05/06/07/08, K8S-10, SCP-04; tests V-06/07.
**Prerequisites:** Approved D-07 limits; [TP-01](01-machine-identity.md) and
[TP-02](02-project-ownership.md) for identity/project enforcement. Global
validation and capacity foundations can proceed independently. Coordinate
stream capacity with [TP-07](07-replay-and-event-delivery.md), workload quotas with
[TP-12](12-kubernetes-workloads.md), and gateway limits with
[TP-14](14-network-and-platform.md).

## Objective and current state

Reject invalid input and constrain work before an external reporter can exhaust
the application or dependencies. [Ingestion handlers](../../../pkg/server/server.go#L57)
already validate some nil fields and required IDs, but the
[gRPC constructor](../../../pkg/server/server.go#L338) does not configure the
complete package limits, and production reflection is registered by
[service registration](../../../pkg/server/server.go#L401).

## Work and acceptance criteria

1. Record numeric, owner-approved limits and units for message and attachment
   bytes, decoded/decompressed size, identifier and metadata lengths/counts,
   nesting, event age, rate, concurrent calls, connections, storage, stream
   capacity, and backlog. Distinguish per-identity/project, per-instance, and
   deployment-wide limits; do not describe a per-pod limit as a global quota.
2. Validate every event-bearing RPC, supported protobuf structure, allowed event
   type, required identifier, timestamp, lifecycle transition, attachment type,
   and size before publication. Preserve legitimate reporter payloads and
   return explicit client errors for invalid input.
3. Apply bounded gRPC receive/decompression, time, connection, concurrency, and
   rate controls at the correct application and gateway layers. Enforce
   identity/project limits using validated scope, not caller-supplied names.
   Configure overload and backpressure responses and bounded retries.
4. Document precisely what a successful RPC acknowledges. A required publisher
   being unavailable or a failed JetStream publish must not appear successful.
   Stream/backlog limits and existing-stream behavior are owned by TP-07.
5. Disable or tightly restrict production reflection/debug services and protect
   health endpoints. Preserve HTTP/2, deadlines, status codes, and supported
   reporter behavior through the gateway. Configure and test the distributed
   protection without silently breaking supported local Docker/AIO and Compose
   reporters or requiring production identity infrastructure locally.

## Verification

- Add `bufconn` tests for all supported RPC families and negative validation
  paths; run `go test ./pkg/server ./tests`. Add publish-failure and overload
  tests that assert no success-shaped response.
- In an authorized isolated environment, test **at and immediately above each
  approved limit**, including concurrent clients and multiple replicas. Record
  configured value, observed result, resource use, workload shape, and rejection
  status; a single small request is not evidence of a maximum.
- Execute V-06/07 through the selected gateway and broker. Record unavailable
  integration tests as blocked or skipped, not passed.
- Exercise an existing local demo reporter and normal payloads after shared
  validation or server-configuration changes.

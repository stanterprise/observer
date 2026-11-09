# 04 - Identity, Access, and Public Ingestion

**Status:** Expected capabilities; authentication and authorization implementation
must be assessed separately.

## Access matrix

Roles below are proposed logical roles, not existing Observer role names.

| Principal | Allowed actions | Scope / prohibited actions |
| --- | --- | --- |
| CI reporter | Submit lifecycle events and attachments | Assigned project and permitted runs only; no result reads or deletion |
| Viewer | Read results, download allowed artifacts, subscribe to live events | Assigned projects only |
| Project maintainer | Viewer actions and approved run/data-management actions | Assigned projects; deletion requires explicit privilege and audit |
| Platform administrator | Deploy/configure workloads and infrastructure | No routine application-data access; break-glass access audited |
| Ingestion workload | Publish accepted events | Required NATS subjects only; no database/storage credentials |
| Processor workload | Consume events, persist records, buffer steps, write artifacts | Dedicated database/schema/bucket scope; no schema administration |
| API workload | Read records/artifacts, consume live events, execute authorized mutations | Least-privilege queries/mutations; no migration rights |
| Migration job | Apply reviewed schema migrations | Target database only, execution window only |
| Backup/recovery identity | Backup and authorized restore | Separate from runtime; restricted backup destinations |

## Human and service access requirements

- **IAM-01:** Require organizational SSO and policy-mandated MFA for human access.
  Protect static UI, REST, WebSocket, artifact routes, and alternate hostnames.
- **IAM-02:** Enforce deny-by-default project/resource authorization server-side.
  Never trust client-supplied project names, run IDs, or subscription filters as grants.
- **IAM-03:** Propagate validated identity through a documented trust mechanism.
  Strip caller-supplied identity headers; reject untrusted upstreams and prevent
  direct access to backends. Define issuer/audience/signature/expiry validation.
- **IAM-04:** Re-evaluate long-lived WebSocket sessions on expiry/revocation within
  a defined maximum interval; filter every delivered event to authorized scope.
- **IAM-05:** Use separate workload identities and credentials. Prefer federation
  over static keys; prohibit broad shared administrator credentials.
- **IAM-06:** Protect browser sessions with secure cookie/token handling, defined
  expiry/logout, CSRF defenses for cookie-authenticated mutations, explicit trusted
  origins, and WebSocket Origin checks. CORS is not authentication.
- **IAM-07:** Audit privilege changes, privileged mutations, denied access, and
  break-glass access without logging bearer tokens or sensitive payloads.

## Public gRPC credential decision

Select one primary mechanism under D-03; do not imply either is already available.

| Option | Required design |
| --- | --- |
| Short-lived machine tokens | Trusted CI identity exchange; restricted issuer/audience and project grants; signature/expiry checks; safe key refresh; bounded revocation latency |
| mTLS | Trusted client CA; per-reporter identity mapped to project grants; certificate issuance, expiry, rotation, and revocation enforcement |

Long-lived shared API keys should not be the production default. Any exception
must define scope, secure CI storage, rotation, revocation, and expiry.
Network allowlists supplement identity; they do not replace it.

## Submission requirements

- **ING-01:** Authenticate every event-bearing RPC before accepting work. Missing,
  expired, invalid, or revoked credentials must fail closed with suitable gRPC
  status codes, without exposing internals.
- **ING-02:** Bind project/run ownership to the validated principal. Prevent callers
  from writing another project's runs or replacing existing ownership through metadata.
  Preserve trusted scope in the event envelope so asynchronous processing cannot
  widen access.
- **ING-03:** Validate protobuf structure, allowed event types, identifiers, lengths,
  nesting depth, metadata, timestamps, attachment types/sizes, and lifecycle
  transitions. Treat identifiers and filenames as untrusted strings.
- **ING-04:** Define idempotency/event identity and replay rules. Duplicate delivery
  must not duplicate records, bypass quotas, or corrupt state; retries must not
  silently discard legitimate events. Set allowed event age and ordering behavior.
- **ING-05:** Set measurable per-identity/project and global limits for message size,
  attachment size, request rate, connections, concurrent work, storage, stream
  retention, and backlog. Bound memory, processing time, and decompression.
- **ING-06:** Apply overload/backpressure behavior with bounded retries and explicit
  failures. Document what an RPC acknowledgement guarantees: accepted into the
  stream is not the same as committed to PostgreSQL.
- **ING-07:** Restrict public RPC methods to the intended service. Disable or tightly
  restrict production reflection/debug interfaces; protect health interfaces.
- **ING-08:** Maintain gRPC compatibility through the gateway, including TLS trust,
  authentication metadata, status codes, deadlines, and retry semantics.
- **ING-09:** Define anti-replay, compromised-reporter containment, and revocation
  propagation targets. A valid identity proves the sender, not the truth of test results.

Record actual numeric limits and revocation bounds in D-07/D-03. No unspecified
limit should be treated as an effective control.

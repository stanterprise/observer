# Observer on k3s: deployment retrospective and upgrade notes

Captured: 2026-08-31 (America/Chicago)  
Scope: Observer `0.8.3` and observer-mcp `0.0.3` in the `observer` namespace

This document records the complications encountered while installing the two Helm charts on the local ARM64 k3s cluster, the workarounds now in production, and the changes that would make a future install or upgrade repeatable. It intentionally names Secrets and keys but contains no credential values.

## Current working baseline

| Item                 | Current state                                                                         |
| -------------------- | ------------------------------------------------------------------------------------- |
| Kubernetes           | k3s `v1.30.4+k3s1`, ARM64 Debian 12 nodes                                             |
| Helm client          | Helm `v4.2.0`                                                                         |
| Namespace            | `observer`                                                                            |
| Observer release     | `observer`, OCI chart/app `0.8.3`, images explicitly pinned to `v0.8.3`               |
| observer-mcp release | `observer-mcp`, OCI chart/app `0.0.3`, image explicitly pinned to `v0.0.3`            |
| PostgreSQL           | Separate `postgresql` release, service `postgresql.observer.svc.cluster.local:5432`   |
| MongoDB              | Separate `observer-mongodb` workload using `mongo:8.0.4` and a 10 Gi `local-path` PVC |
| NATS                 | Observer subchart, one replica with JetStream and a 10 Gi `local-path` PVC            |
| Object storage       | SeaweedFS S3, private bucket `observer`, path-style access                            |
| Ingress              | k3s Traefik with manually managed `IngressRoute` resources                            |
| TLS                  | cert-manager `v1.18.6`, private homelab CA trusted on local clients                   |

The working endpoints are:

| Purpose            | Endpoint                           |
| ------------------ | ---------------------------------- |
| Observer web/API   | `https://observer.k8s.lan`         |
| Observer ingestion | `https://grpc.observer.k8s.lan`    |
| SeaweedFS S3       | `https://s3.observer.k8s.lan`      |
| Observer MCP       | `https://mcp.observer.k8s.lan/sse` |

## Executive summary

The difficult part was not running the application containers. The difficult part was reconciling assumptions across chart versions, ARM64 dependencies, Secret lifecycles, local TLS, internal DNS, and resources that the charts do not render.

The five largest upgrade risks are:

1. Observer's S3 environment and private-CA mount are manual Deployment mutations. A Helm upgrade can remove them.
2. Traefik routes, certificates, DNS entries, the external MongoDB workload, and several Secrets are not owned by either application release.
3. Both charts have image-tag traps. The published chart version does not safely select the published `v`-prefixed image tag.
4. Runtime Secret requirements are insufficiently validated and are needed before hook Jobs or Pods can start.
5. The cluster is pinned to an old cert-manager because k3s/Kubernetes is old. That should be corrected before treating this as a low-maintenance production installation.

## 1. Source-of-truth confusion

### What happened

The original deployment reference was a DigitalOcean-oriented `helm-infra` bundle. It was useful for understanding the broad topology, but it was not the authoritative current installation path for the local cluster. It mixed provider infrastructure concerns with application release concerns and did not map cleanly to k3s, Traefik, local DNS, a private CA, or SeaweedFS.

The current authoritative artifacts are the OCI charts published from the application repositories:

- Observer: `oci://ghcr.io/stanterprise/observer/charts/observer`
- observer-mcp: `oci://ghcr.io/stanterprise/observer-mcp/charts/observer-mcp`

### Why this was costly

- It was easy to infer values and resource ownership from the DigitalOcean workflow that the current packaged chart did not actually support.
- Provider-managed DNS, certificates, load balancers, and externally provisioned databases concealed work that must be explicit on a homelab cluster.
- There was no single checked-in local values file or release bundle to act as the installation contract.

### Improvement

Publish one deployment matrix in each repository that clearly separates:

- self-contained demonstration install;
- external-database production install;
- cloud-provider examples; and
- local/k3s requirements.

The chart, its README, and the release workflow should be tested together at every tag.

## 2. Observer chart complications

### 2.1 Chart and image tags do not line up safely

The working chart is `0.8.3`, but the application image tags are `v0.8.3`. The chart's normal image selection produced or implied the non-existent `0.8.3` tag, so each Observer component had to be explicitly pinned to `v0.8.3`.

**Current workaround:** explicitly set the image tag to `v0.8.3`.

**Upgrade risk:** an upgrade that changes only `--version` can leave the old image in place or attempt to pull a tag that does not exist.

**Quality-of-life fix:** default the image tag from a canonical, tested `.Chart.AppVersion`, document whether tags include `v`, and add optional digest pinning. CI should install the packaged chart against the images produced by the same release.

### 2.2 The runtime Secret must exist before migration hooks

The migration Job failed with:

```text
Error: secret "observer-runtime-env" not found
```

The exact required Secret is:

```text
Secret: observer-runtime-env
Keys:
  POSTGRES_DSN
  MONGODB_URI
  NATS_URL
```

The chart setting `runtime.existingSecret: observer-runtime-env` tells workloads which Secret to consume; it does not create the Secret. The migration hook starts early enough that this prerequisite must be present before the Helm release is installed or upgraded.

An attempted bootstrap command also assumed two things that did not exist: a Secret named `observer-mongodb-auth` and a local file named `~/observer-k3s-values.yaml`. That led to an empty template pipeline and no Secret being applied.

**Current workaround:** create `observer-runtime-env` independently, with a PostgreSQL DSN for `postgresql.observer.svc.cluster.local:5432`, the actual external MongoDB URI, and the NATS URL, before running Helm.

**Upgrade risk:** deleting or renaming the Secret blocks hooks and all components. Rotating the Secret does not update already-running processes until their Pods are restarted.

**Quality-of-life fix:** add render-time validation with `required`/`fail` plus a preflight Job or script that verifies the Secret and its keys without printing values. Support configurable Secret names and key names for each datastore. Document that Secret rotation requires a controlled rollout.

A safe key-only check is:

```bash
kubectl -n observer get secret observer-runtime-env \
  -o go-template='{{range $key, $value := .data}}{{printf "%s\n" $key}}{{end}}'
```

### 2.3 Credentials can leak into Helm release metadata

The bootstrap path used chart values such as PostgreSQL and external-database passwords while also using a runtime Secret. Helm stores supplied values in its release Secret, so credentials can be duplicated in Helm metadata even when the application consumes `observer-runtime-env` at runtime.

**Quality-of-life fix:** make external Secret references a complete alternative to literal credential values. Passwords should never need to appear in a values file, a `--set` argument, shell history, or Helm release metadata.

### 2.4 The PostgreSQL dependency referenced a retired image path

The Bitnami PostgreSQL chart requested:

```text
docker.io/bitnami/postgresql:17.6.0-debian-12-r4
```

That versioned image had moved to the legacy repository. Pulls first showed a Docker Hub TLS timeout and then the definitive `not found` error.

The working override was:

```yaml
global:
  security:
    allowInsecureImages: true

image:
  repository: bitnamilegacy/postgresql
```

The resulting service is healthy at `postgresql.observer.svc.cluster.local:5432`, backed by a 10 Gi `local-path` PVC.

**Upgrade risk:** `bitnamilegacy` is a compatibility stopgap, not a durable supply-chain strategy. A dependency upgrade can change image names, credentials, PVC behavior, or PostgreSQL major versions.

**Quality-of-life fix:** choose and document a supported PostgreSQL distribution, pin chart and image digests, add an ARM64 manifest check, and make database backup/restore a mandatory upgrade gate.

### 2.5 The bundled MongoDB path was not viable on ARM64

The MongoDB image selected by the Observer `0.8.3` dependency path was AMD64-only. It could not run on the ARM64 k3s nodes.

**Current workaround:** disable the chart's MongoDB dependency and run an external `observer-mongodb` Deployment using the official multi-architecture `mongo:8.0.4` image, with service `observer-mongodb.observer.svc.cluster.local:27017` and a 10 Gi `local-path` PVC.

MongoDB 8 also requires ARMv8.2-A. That is suitable for Raspberry Pi 5-class hardware but must be validated before scheduling it on older ARM64 nodes such as some Raspberry Pi 4 installations.

**Upgrade risk:** the external MongoDB workload and credentials are outside the Observer Helm release; `helm uninstall observer` will not remove them, and `helm upgrade observer` will not validate or upgrade them.

**Quality-of-life fix:** make external MongoDB a first-class, documented chart mode, add architecture checks to release CI, and publish a supported dependency matrix. If a bundled MongoDB remains available, verify its image has both `linux/amd64` and `linux/arm64` manifests.

### 2.6 NATS worked, but its persistence is an explicit operational dependency

The embedded NATS path works on ARM64. The current installation uses one NATS replica with JetStream and a 10 Gi `local-path` PVC.

**Upgrade considerations:** preserve the PVC, inspect NATS and JetStream version changes, and do not assume a one-replica local-path deployment is highly available. It is adequate for this local cluster but ties recovery to the node and volume backup strategy.

### 2.7 S3 support is not expressible cleanly through the chart

Observer `0.8.3` can use S3 at runtime, but the chart has no complete first-class values for:

- an existing S3 Secret;
- arbitrary `envFrom` entries;
- extra volumes and volume mounts; or
- a custom CA bundle.

The working Secret is:

```text
Secret: observer-storage-env
Keys:
  STORAGE_DRIVER
  STORAGE_S3_ENDPOINT
  STORAGE_S3_REGION
  STORAGE_S3_BUCKET
  STORAGE_S3_USE_PATH_STYLE
  STORAGE_S3_ACCESS_KEY_ID
  STORAGE_S3_SECRET_ACCESS_KEY
```

Its non-secret settings currently resolve to:

```text
STORAGE_DRIVER=s3
STORAGE_S3_ENDPOINT=https://s3.observer.k8s.lan
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BUCKET=observer
STORAGE_S3_USE_PATH_STYLE=true
```

The Secret was injected into `observer-api` and `observer-processor` out of band. Both Deployments were also patched with:

```text
AWS_CA_BUNDLE=/etc/ssl/homelab/ca.crt
```

and a volume that projects `ca.crt` from `observer-edge-tls` into `/etc/ssl/homelab`.

**Upgrade risk:** these Deployment mutations are Helm drift. A Helm upgrade can replace the Pod template and silently remove the S3 variables, CA bundle, volume, or mount. Attachments would then fall back to inline storage or fail TLS verification.

`STORAGE_S3_PUBLIC_URL` was not a solution for the private bucket: in this application version it produces a direct public URL instead of a presigned private-object URL. The endpoint hostname used to sign requests therefore needs to be reachable consistently from both Pods and clients.

**Quality-of-life fix:** add `storage.s3.existingSecret`, `extraEnv`, `extraEnvFrom`, `extraVolumes`, and `extraVolumeMounts` to the chart. A dedicated `customCA.existingSecret` option would make the common private-PKI case safer. Until then, keep these patches in a checked-in Kustomize post-renderer rather than issuing ad hoc `kubectl set env` and patch commands.

### 2.8 The chart did not render usable ingress resources

The chart's ingress-related values did not produce the resources needed for this deployment, and the packaged SSL guidance did not match the current chart behavior. Standalone Traefik resources were required for:

- web and API routing on `observer.k8s.lan`;
- gRPC ingestion on `grpc.observer.k8s.lan` using h2c to the backend; and
- both HTTP and HTTPS entry points where desired.

The correct CRD API group is `traefik.io/v1alpha1`. The cluster also contains the legacy `traefik.containo.us` CRD. An unqualified command such as `kubectl get ingressroutes` can select the empty legacy group and misleadingly report that a live route does not exist.

Use the fully qualified resource during diagnostics:

```bash
kubectl get ingressroutes.traefik.io -A
```

**Quality-of-life fix:** add maintained generic Kubernetes Ingress support or maintained Traefik examples to the chart. The examples should include route priority for `/api`, a web fallback, TLS Secret references, and h2c for the ingestion backend.

## 3. Shared k3s, DNS, and TLS complications

### 3.1 Internal Pods could not resolve the public-facing local hostname

Observer needed to access SeaweedFS using the same HTTPS hostname used in signed S3 URLs. Pods initially could not resolve `s3.observer.k8s.lan`.

The current `coredns-custom` map sends these names to Traefik's Service ClusterIP `10.43.181.50`:

```text
observer.k8s.lan
grpc.observer.k8s.lan
s3.observer.k8s.lan
mcp.observer.k8s.lan
```

**Upgrade risk:** the map hard-codes a Service ClusterIP. If the Traefik Service is deleted and recreated, DNS can continue returning a stale address even though all application Pods look healthy.

**Quality-of-life fix:** manage split-horizon LAN DNS centrally, or generate the CoreDNS override from a stable load-balancer address. At minimum, keep the DNS configuration beside the releases and add a smoke test from inside the cluster.

### 3.2 `.lan` requires a private certificate authority

A public ACME CA cannot issue a normal certificate for the private `.lan` names. The solution was a private homelab CA managed through cert-manager, with its root certificate manually trusted on clients.

Current leaf certificates include:

- `observer-edge` / Secret `observer-edge-tls` for `observer.k8s.lan`, `grpc.observer.k8s.lan`, and `mcp.observer.k8s.lan`;
- `observer-s3-edge` / Secret `observer-s3-tls` in the `seaweedfs` namespace for `s3.observer.k8s.lan`.

Traefik TLS Secrets must be in the same namespace as their `IngressRoute`.

Adding the MCP hostname to the shared Observer certificate rotated the key and certificate. Traefik hot-reloaded it successfully, but sharing a certificate increases the blast radius of every SAN change.

**Quality-of-life fixes:**

- Use a wildcard certificate containing `observer.k8s.lan` and `*.observer.k8s.lan`, or issue one leaf certificate per service.
- Automate private-root distribution and document recovery/rotation.
- Keep certificate and route manifests under the same declarative release bundle.

### 3.3 Kubernetes and cert-manager compatibility forced an old release

k3s currently provides Kubernetes `1.30.4`. Newer supported cert-manager releases require newer Kubernetes minors, so cert-manager was pinned to `v1.18.6` with local-CA-only settings:

```text
crds.enabled=true
crds.keep=true
global.rbac.aggregateClusterRoles=false
prometheus.enabled=false
```

That cert-manager line is now end-of-life and has a high-severity DNS01 advisory. This installation does not configure DNS01 credentials, which reduces exposure to that specific path, but the platform pin is still technical debt.

**Quality-of-life fix:** upgrade k3s to a currently supported Kubernetes minor, then move cert-manager to a supported release before adding more certificate automation.

## 4. observer-mcp chart complications

### 4.1 The chart, source tree, release, and image report inconsistent versions

The installed OCI chart and GitHub release are `0.0.3`, while the source tree's `Chart.yaml` reports `0.1.0`. The packaging workflow rewrites chart metadata to the Git tag. The application also reports server version `0.1.0` during MCP initialization.

The chart defaults to the mutable image tag `latest`; the released image is tagged `v0.0.3`, and a bare `0.0.3` image tag does not exist.

**Current workaround:** pin all three independently:

```text
Chart: 0.0.3
Image: ghcr.io/stanterprise/observer-mcp:v0.0.3
Replicas: 1
```

**Upgrade risk:** a chart-only upgrade can silently continue to run `latest`, and runtime version reporting cannot currently prove which release is installed.

**Quality-of-life fix:** make `image.tag` default to the packaged `.Chart.AppVersion`, publish immutable digests, preserve one version value through build and runtime, and add a release test that compares chart metadata, image tag, and MCP `serverInfo.version`.

### 4.2 Secret references are rigid

The chart correctly defaults to `observer-runtime-env`, but it unconditionally references both:

```text
POSTGRES_DSN
MONGODB_URI
```

The application documents PostgreSQL as required and MongoDB as optional, but the Pod cannot be created when the `MONGODB_URI` key is absent. Supplying a non-empty but unreachable Mongo URI is worse: startup exits when the database connection fails.

The chart does not expose independent Secret names, configurable key names, `optional`, `envFrom`, or the application's alternate `DATABASE_URL` and `MONGO_URI` variables.

**Current workaround:** reuse `observer-runtime-env` in the same namespace and keep both keys present.

**Upgrade risk:** observer-mcp is coupled to the Secret contract of another Helm release. Secret rotation also requires a Deployment restart because environment variables are read only at process start.

**Quality-of-life fix:** provide separate `postgres.existingSecret/name/key` and `mongodb.enabled/existingSecret/name/key/optional` values, plus generic `extraEnv` and `extraEnvFrom`. Add a values schema and render-time validation.

### 4.3 The network protocol is easy to configure incorrectly

The deployed version uses the legacy MCP HTTP/SSE flow, protocol version `2024-11-05`:

1. A client opens `GET /sse`.
2. The server returns a session-specific relative endpoint such as `/message?sessionId=...`.
3. The client sends messages to that endpoint while the SSE connection remains open.

It is not the newer Streamable HTTP `/mcp` endpoint. The repository documentation also refers to stdio in places, so documentation and deployed behavior are not aligned.

Sessions are kept in an in-process map. A restart loses sessions, and multiple replicas require sticky routing or a shared session store. The current one-replica deployment is therefore intentional. The server emits a keepalive roughly every 25 seconds.

**Ingress consequences:** both `/sse` and `/message` must reach the same service and origin, with no path rewrite. Clients should be configured directly with `https://mcp.observer.k8s.lan/sse`; an HTTP redirect was avoided because redirect handling, particularly for POST requests, is not reliable across MCP clients.

**Quality-of-life fix:** document the exact transport and client URL in chart notes, keep `replicaCount: 1` as an explicitly supported default, and warn or fail when scaling without session affinity. Longer term, support Streamable HTTP and either stateless or shared sessions.

### 4.4 The chart has no ingress, TLS, or safe authentication integration

The chart renders a Deployment, ClusterIP Service, and tokenless ServiceAccount, but no Ingress, Certificate, NetworkPolicy, or authentication Secret integration.

The application supports `MCP_AUTH_TOKEN`, but the chart can only receive it as a literal environment value. That would expose the token in Helm values and release metadata. The current LAN-only endpoint is intentionally unauthenticated, per deployment decision, and plain HTTP has no route.

**Security consequence:** TLS protects traffic in transit; it does not authorize callers. Any host that can reach this LAN endpoint can invoke the read-only analytical tools. The application also permits CORS from `*`.

**Quality-of-life fix:** add `auth.enabled`, `auth.existingSecret`, and a configurable key rendered with `secretKeyRef`. Add optional Ingress/TLS and NetworkPolicy values, or provide maintained examples that cover SSE timeouts and both protocol paths. Keep the unauthenticated LAN choice explicit rather than accidental.

### 4.5 It queries Observer storage directly

observer-mcp connects directly to PostgreSQL and MongoDB. It does not use NATS, S3, a PVC, or the Kubernetes API.

This is efficient, but an Observer schema migration can break MCP queries even when both Helm releases roll out cleanly. Helm cannot detect that compatibility problem.

**Quality-of-life fix:** publish an observer-mcp-to-Observer compatibility matrix and run integration tests against the exact Observer migrations for each release. The test should exercise `initialize`, `tools/list`, a PostgreSQL-backed tool, and the Mongo-backed live-buffer tool.

### 4.6 What worked well

The observer-mcp image is available for ARM64, uses modest resources, starts as a non-root user, has a read-only root filesystem, drops Linux capabilities, prevents privilege escalation, and does not mount its ServiceAccount token. Its named service ports also made the external Traefik route straightforward.

## 5. Inventory of resources outside the two Helm releases

These resources are part of the functioning system even though `helm upgrade observer` or `helm upgrade observer-mcp` does not own all of them:

| Resource                                      | Namespace             | Current purpose                                      | Upgrade concern                                    |
| --------------------------------------------- | --------------------- | ---------------------------------------------------- | -------------------------------------------------- |
| `observer-runtime-env` Secret                 | `observer`            | PostgreSQL, MongoDB, and NATS connection strings     | Must pre-exist; rotation needs Pod rollouts        |
| `observer-storage-env` Secret                 | `observer`            | SeaweedFS S3 configuration and credentials           | Manually injected into two Deployments             |
| `postgresql` release/PVC                      | `observer`            | Durable relational database                          | Independent lifecycle and backups                  |
| `observer-mongodb` workload/PVC               | `observer`            | ARM64-compatible MongoDB                             | Not upgraded or validated by Observer Helm release |
| Observer API/processor S3 patches             | `observer`            | `envFrom`, `AWS_CA_BUNDLE`, CA volume/mount          | Highest drift risk; Helm can remove them           |
| `observer`, `observer-https` routes           | `observer`            | Web/API HTTP and HTTPS                               | Manual Traefik resources                           |
| `observer-grpc`, `observer-grpc-https` routes | `observer`            | Ingestion, h2c backend                               | Manual Traefik resources                           |
| `observer-mcp-https` route                    | `observer`            | Host-only HTTPS route covering `/sse` and `/message` | Not removed or rolled back with MCP release        |
| `observer-s3`, `observer-s3-https` routes     | `seaweedfs`           | S3 HTTP/HTTPS                                        | Separate namespace and lifecycle                   |
| `observer-edge` Certificate                   | `observer`            | Web, gRPC, and MCP TLS                               | Shared SAN changes rotate all three                |
| `observer-s3-edge` Certificate                | `seaweedfs`           | S3 TLS                                               | Must remain beside its route                       |
| `coredns-custom` map                          | `kube-system`         | Internal resolution of `.lan` names                  | Hard-coded Traefik ClusterIP                       |
| Homelab root CA trust                         | clients and workloads | Trust for `.lan` certificates                        | Manual distribution and recovery                   |

This inventory is the strongest argument for moving to one declarative release bundle.

## 6. Safe upgrade checklist

### 6.1 Before either application upgrade

1. Read both release notes and inspect datastore migration changes.
2. Back up PostgreSQL before running any Observer migration. Back up MongoDB and persistent volumes in proportion to the data's value.
3. Capture current rollback evidence without displaying Secret contents:

   ```bash
   helm history observer -n observer
   helm history observer-mcp -n observer
   helm get values observer -n observer --all > observer-values.before.yaml
   helm get values observer-mcp -n observer --all > observer-mcp-values.before.yaml
   helm get manifest observer -n observer > observer-manifest.before.yaml
   helm get manifest observer-mcp -n observer > observer-mcp-manifest.before.yaml
   ```

   Treat the captured values as sensitive because the present release metadata may contain credentials.

4. Confirm required Secret keys without printing values:

   ```bash
   kubectl -n observer get secret observer-runtime-env \
     -o go-template='{{range $key, $value := .data}}{{printf "%s\n" $key}}{{end}}'
   kubectl -n observer get secret observer-storage-env \
     -o go-template='{{range $key, $value := .data}}{{printf "%s\n" $key}}{{end}}'
   ```

5. Confirm database endpoints, PVCs, certificates, DNS, and Traefik routes are healthy.
6. Verify that every proposed image exists for `linux/arm64`; do this for application and dependency images.
7. Inspect the exact packaged chart rather than assuming the source tree matches it:

   ```bash
   helm show chart oci://ghcr.io/stanterprise/observer/charts/observer --version X.Y.Z
   helm show chart oci://ghcr.io/stanterprise/observer-mcp/charts/observer-mcp --version X.Y.Z
   ```

8. Render with a pinned, checked-in values file. Compare the rendered images, Secret references, Jobs, ports, probes, Services, PVCs, and security contexts. Use `helm diff upgrade` if the plugin is available.
9. Do not use `--reuse-values` blindly; it hides changed defaults and can preserve obsolete fields.

### 6.2 Observer upgrade

The desired command shape for Helm 4 is:

```bash
helm upgrade --install observer \
  oci://ghcr.io/stanterprise/observer/charts/observer \
  --version X.Y.Z \
  --namespace observer \
  -f observer-k3s-values.yaml \
  --rollback-on-failure \
  --wait \
  --timeout 10m
```

Helm 4 warns that `--atomic` is deprecated; use `--rollback-on-failure` in new automation.

Immediately after the upgrade:

1. Confirm the migrate Job completed.
2. Reconcile the checked-in post-render/Kustomize overlay that supplies the storage Secret and CA mount. Until that overlay exists, explicitly verify the manual patches survived.
3. Wait for every Deployment and the NATS StatefulSet.
4. Confirm the exact running image tags/digests.
5. Inspect API and processor logs for PostgreSQL, MongoDB, NATS, and S3 initialization.
6. Perform an authenticated S3 operation over trusted HTTPS; do not use `-k` as the normal test.
7. Verify the UI, API, and gRPC endpoints from both a LAN client and an in-cluster test Pod.

Useful checks:

```bash
kubectl -n observer get pods,svc,pvc
kubectl -n observer get jobs
kubectl -n observer rollout status deployment/observer-api --timeout=2m
kubectl -n observer rollout status deployment/observer-processor --timeout=2m
kubectl -n observer logs deployment/observer-api --tail=100
kubectl -n observer logs deployment/observer-processor --tail=100
kubectl get ingressroutes.traefik.io -A
kubectl -n observer get certificate
```

### 6.3 observer-mcp upgrade

Keep the chart, image, and running version independently pinned until the upstream versioning issue is fixed:

```yaml
image:
  tag: vX.Y.Z

runtime:
  existingSecret: observer-runtime-env

replicaCount: 1
```

Then use:

```bash
helm upgrade --install observer-mcp \
  oci://ghcr.io/stanterprise/observer-mcp/charts/observer-mcp \
  --version X.Y.Z \
  --namespace observer \
  -f observer-mcp-values.yaml \
  --rollback-on-failure \
  --wait \
  --timeout 5m
```

After the upgrade:

1. Confirm one ready Pod, one ready Service endpoint, no unexpected restarts, and the exact running image.
2. Check `/health` locally, then open `/sse` and retain the connection long enough to observe a keepalive.
3. Run a real MCP `initialize`, `tools/list`, one PostgreSQL-backed query, and the Mongo-backed live-buffer query when data exists.
4. Test `https://mcp.observer.k8s.lan/sse` with the trusted homelab CA and confirm `/message` remains routed without rewriting.
5. Confirm plain HTTP still has no route if HTTPS-only remains the policy.
6. Recheck `observer-edge`, the CoreDNS answer, and `observer-mcp-https`; Helm does not reconcile them.
7. Watch for query failures that indicate Observer schema drift.

Rollback uses the prior Helm revision:

```bash
helm rollback observer-mcp REVISION -n observer --wait --timeout 5m
```

That rollback does not revert the Certificate, DNS, Traefik route, or runtime Secret. Those must be assessed separately.

## 7. Prioritized quality-of-life backlog

### P0 — eliminate silent upgrade regressions

1. **Observer chart:** add `storage.s3.existingSecret`, `extraEnv`, `extraEnvFrom`, `extraVolumes`, `extraVolumeMounts`, and custom-CA support.
2. **Both charts:** derive immutable image tags from the packaged app version, test `v` prefix behavior, and support digest pins.
3. **Observer chart:** fully support pre-existing datastore Secrets so credentials never enter Helm values.
4. **observer-mcp chart:** add configurable PostgreSQL/Mongo Secret refs, truly optional MongoDB, and Secret-backed bearer-token support.
5. **Local deployment:** put all current manual mutations into a checked-in Kustomize post-renderer or equivalent declarative layer.
6. **Both charts:** add `values.schema.json` and render-time validation for mutually exclusive or required settings.

### P1 — make installation repeatable

1. Create one Helmfile or umbrella deployment repository that pins PostgreSQL, external MongoDB, Observer, observer-mcp, NATS settings, routes, certificates, DNS, and Secrets/ExternalSecrets.
2. Add an ARM64 preflight that checks every image manifest before install.
3. Add a preflight command that validates Kubernetes version, storage classes, CRDs, Secret keys, DNS resolution, certificate trust, and database reachability.
4. Add chart-owned or maintained example ingress resources for Traefik, including h2c ingestion and MCP SSE behavior.
5. Add release smoke tests for migrations, S3, MCP handshake/tools, TLS SANs, and internal/external DNS.
6. Publish Observer/observer-mcp/database compatibility matrices.

### P2 — reduce platform maintenance risk

1. Upgrade k3s, then cert-manager, to supported releases.
2. Replace the hard-coded CoreDNS-to-Traefik ClusterIP map with managed split-horizon DNS or a stable address.
3. Automate homelab root-CA distribution and recovery.
4. Add PostgreSQL, MongoDB, NATS, and SeaweedFS backup/restore procedures with periodic restore tests.
5. Add NetworkPolicies and tighten wildcard CORS where practical.
6. Add a Secret reloader or explicit checksum annotations so controlled Secret changes trigger rollouts.

## 8. Recommended target layout

The practical end state is one small infrastructure repository in which Helm owns the applications and a declarative overlay owns everything the charts cannot yet express:

```text
observer-homelab/
├── helmfile.yaml
├── values/
│   ├── postgresql.yaml
│   ├── observer.yaml
│   └── observer-mcp.yaml
├── manifests/
│   ├── mongodb/
│   ├── ingressroutes/
│   ├── certificates/
│   ├── coredns/
│   └── networkpolicies/
├── overlays/
│   └── observer-storage-and-ca/
├── secrets/
│   └── README.md              # ExternalSecret/SOPS contract; no plaintext
└── scripts/
    ├── preflight.sh
    ├── smoke-test.sh
    ├── backup.sh
    └── restore-test.sh
```

Success means a clean cluster can be reconstructed from that repository plus encrypted credentials, and an upgrade does not require remembering any manual `kubectl set env`, patch, DNS, route, or certificate step.

## Upstream references

- [Observer repository](https://github.com/stanterprise/observer)
- [Observer Helm chart source](https://github.com/stanterprise/observer/tree/master/charts/observer)
- [Observer v0.8.3 release](https://github.com/stanterprise/observer/releases/tag/v0.8.3)
- [observer-mcp repository](https://github.com/stanterprise/observer-mcp)
- [observer-mcp Helm chart source](https://github.com/stanterprise/observer-mcp/tree/master/charts/observer-mcp)
- [observer-mcp v0.0.3 release](https://github.com/stanterprise/observer-mcp/releases/tag/v0.0.3)
- [cert-manager release support policy](https://cert-manager.io/docs/releases/)

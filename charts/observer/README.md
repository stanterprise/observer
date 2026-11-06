# Observer Helm Chart

This Helm chart deploys the Observer test observability system on Kubernetes. The chart defaults to distributed mode.

Shipped presets:

- `values-production.yaml`: distributed example with larger resource and HPA settings, embedded dependencies, and no chart-managed exposure resources
- `values-aio.yaml`: all-in-one evaluation preset that disables the PostgreSQL, MongoDB, and NATS subcharts
- `values-aio-gateway.yaml`: legacy AIO compatibility preset retained for existing automation names; the chart itself no longer renders Gateway API resources

## Current Contract

### Chart-Owned Responsibilities

- Deployment and runtime configuration for Observer services (ingestion, processor, API, web).
- Kubernetes Services exposing stable ports and protocols.
- PostgreSQL migrations through a dedicated Helm hook job (when `postgres.migration.enabled=true`).
- Health probes, resource requests/limits, and security contexts.
- Image selection and tag management.

### Operator-Owned Responsibilities

- Ingress routing, TLS certificates, and DNS (see platform exposure task packet).
- Private CA distribution and trust.
- External Secret providers (ExternalSecret, SOPS, or vault).
- Kubernetes network policies and cluster-level security.

### Key Features

**Deployment Modes:**

- Distributed mode is the primary install path.
- AIO is intended for development and evaluation only.
- Both modes support embedded or external PostgreSQL, MongoDB, and NATS.

**Connection Management:**

- Distributed workloads consume connection strings from Kubernetes Secrets.
- Set `runtime.existingSecret` to reuse a pre-created Secret containing `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`.
- If `runtime.existingSecret` is empty, the chart renders a generated Secret for non-production installs.
- Distributed ingestion requires `distributed.ingestion.auth.existingSecret` to reference an operator-managed Secret containing `INGESTION_AUTH_KEY` (at least 32 bytes). Do not put the key in Helm values, `extraEnv`, or release metadata.
- Ingestion uses one shared bearer key per deployment. This authenticates possession only; it does not identify individual reporters or authorize project/run ownership. Do not treat it as satisfying the public-ingestion gate until machine project authorization is implemented or an authorized exception is recorded.
- AIO/local mode does not use or generate an ingestion key.
- `distributed.ingestion.env`, `distributed.api.env`, and `distributed.processor.env` must not set `NATS_URL`, `POSTGRES_DSN`, or `MONGODB_URI`; the chart rejects those keys at validation time.

**External Dependencies:**

- When disabling an embedded dependency, the external endpoint is required: set `postgres.host` when `postgresql.enabled=false`, etc.
- The chart validates missing external endpoints and fails with a clear error message.
- The bundled Bitnami legacy PostgreSQL image is pinned to `17.6.0-debian-12-r4` and supports linux/amd64 and linux/arm64.
- The bundled Bitnami legacy MongoDB image is pinned to `8.0.4-debian-12-r0` and currently supports linux/amd64 only; ARM64 installations must use an external MongoDB endpoint until an ARM64-compatible image is validated.

**Image Management:**

- `image.tag` defaults directly to the chart `appVersion` when empty. `appVersion` must always be the exact published Docker image tag, including any `v` prefix.
- Stable release metadata may therefore use `appVersion: "v0.8.3"` with chart version `0.8.3`; prerelease and branch packages use the exact tags actually published for those builds.
- Branch packages use the immutable Docker-published `sha-<seven-character-commit>` tag as `appVersion`; commit-suffixed chart versions are package identifiers, not image tags.
- `image.pullPolicy` auto-detects `Always` for mutable tags (`latest`, `main`, `develop`) and `IfNotPresent` for immutable tags.
- For production installs, pin `image.tag` to an immutable tag or digest.

**Storage and Private Endpoints:**

- S3 configuration is supplied through a Secret: set `storage.s3.existingSecret` to the name of a Secret containing `STORAGE_DRIVER`, `STORAGE_S3_*`, and other storage environment variables.
- Custom CA certificates are supplied through a Secret: set `customCA.existingSecret` to mount a CA certificate in distributed API and processor pods. `AWS_CA_BUNDLE` is set to the mounted `ca.crt` path unless overridden in the workload env or `extraEnv`.

**Extension Points:**

- Use `extraEnv` to add environment variables to all workload containers.
- Use `extraEnvFrom` to add ConfigMap or Secret references to all workload containers.
- Use `extraVolumes` and `extraVolumeMounts` to add volumes to all workload pods.

**Password Defaults:**

- The chart no longer ships with reusable password defaults in public values.
- For embedded dependencies, the bundled dependency charts generate random credentials and Observer workloads consume those dependency Secrets directly.
- For external dependencies, use `runtime.existingSecret` or external Secret-backed values; reusable credentials are never supplied by chart defaults.

**Exposure and Networking:**

- The chart does not render Ingress, Gateway API, or Certificate resources.
- Downstream infrastructure or deployment bundles provide ingress routing, DNS, and TLS.
- The chart exposes stable Services on documented ports for downstream use.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure for persisted dependencies or AIO state

## Artifact Locations

- OCI Registry: `oci://ghcr.io/stanterprise/observer/charts/observer`
- GitHub Pages Helm repository: `https://stanterprise.github.io/observer/`
- Source checkout: `charts/observer/`

## Versioning Policy

- Tagged releases publish clean semantic chart versions such as `0.7.0-beta` or `0.7.0`.
- Manual testing publishes internal prerelease artifacts in the form `<chart-version>-<sha7>`.

## Installation

### From Source

```bash
cd charts/observer
helm dependency update
cd ../..

helm install observer ./charts/observer
```

### From OCI Registry

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version <chart-version>
```

### From GitHub Pages Repository

If the release you want has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update
helm install observer observer/observer --version <chart-version>
```

## Deployment Modes

### Distributed Mode

Distributed mode is the default. It deploys separate ingestion, processor, API, and web workloads, and it can run with embedded or external PostgreSQL, MongoDB, and NATS.

```bash
helm install observer ./charts/observer \
  --set distributed.ingestion.auth.existingSecret=observer-ingestion-auth
```

Create `observer-ingestion-auth` through the organization's approved secret
manager before installation. It must contain a high-entropy
`INGESTION_AUTH_KEY`; the chart intentionally fails to render when no auth
Secret is configured. Reporters send it as gRPC metadata:
`authorization: Bearer <key>`. Use the TLS-protected public gateway for every
request. Share this deployment-wide key only with trusted reporters; rotation
requires updating the Secret and coordinating reporter configuration.

When `postgres.migration.enabled=true`, install and upgrade runs include a dedicated migration hook job. The API and processor workloads do not run migrations themselves.

### AIO Mode

AIO mode runs Observer in a single container for development and evaluation.
It intentionally runs in keyless local mode; do not expose it as a production
public ingestion service.

From source, use the shipped preset:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio.yaml
```

Or set the equivalent values explicitly:

```bash
helm install observer ./charts/observer \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set postgresql.enabled=false \
  --set mongodb.enabled=false \
  --set nats.enabled=false
```

### Legacy AIO Compatibility Preset

The `values-aio-gateway.yaml` file is retained for compatibility with existing automation names, but it no longer renders Gateway API resources:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Key Values

| Key                                  | Purpose                                                                      | Default       |
| ------------------------------------ | ---------------------------------------------------------------------------- | ------------- |
| `mode`                               | Deployment mode                                                              | `distributed` |
| `image.tag`                          | Image tag override. Empty means the chart `appVersion`.                      | `""`          |
| `image.pullPolicy`                   | Pull policy override. Empty means auto-detect based on the effective tag.    | auto          |
| `postgresql.enabled`                 | Deploy embedded PostgreSQL                                                   | `true`        |
| `postgres.host`                      | External PostgreSQL host when `postgresql.enabled=false` in distributed mode | `""`          |
| `mongodb.enabled`                    | Deploy embedded MongoDB                                                      | `true`        |
| `externalDatabase.host`              | External MongoDB host when `mongodb.enabled=false` in distributed mode       | `""`          |
| `nats.enabled`                       | Deploy embedded NATS                                                         | `true`        |
| `nats.config.merge.max_payload`      | Maximum message payload in bytes for bundled distributed NATS                | `16777216` (16 MiB) |
| `externalNats.url`                   | External NATS URL when `nats.enabled=false` in distributed mode              | `""`          |
| `runtime.existingSecret`             | Existing Secret with `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`           | `""`          |
| `postgres.migration.enabled`         | Run the dedicated migration hook job in distributed mode                     | `true`        |
| `distributed.ingestion.service.type` | Service type for the distributed gRPC ingestion workload                     | `ClusterIP`   |
| `distributed.api.service.type`       | Service type for the distributed API workload                                | `ClusterIP`   |
| `distributed.web.service.type`       | Service type for the distributed web workload                                | `ClusterIP`   |
| `aio.service.type`                   | Service type for the AIO service                                             | `ClusterIP`   |
| `podAnnotations`                     | Additional annotations for workload pods and the migration job               | `{}`          |
| `imagePullSecrets`                   | Image pull secrets applied to workload pods and the migration job            | `[]`          |

For shared or production environments, pin `image.tag` to an immutable image tag.

### NATS Payload Limit

The bundled distributed NATS server accepts messages up to 16 MiB by default.
Override the limit in bytes with `nats.config.merge.max_payload`, for example:

```bash
helm upgrade observer ./charts/observer \
  --set distributed.ingestion.auth.existingSecret=observer-ingestion-auth \
  --set nats.config.merge.max_payload=25165824
```

This value configures only the distributed NATS subchart. External NATS servers
must be configured separately. The AIO image uses the same 16 MiB default from
its bundled [NATS configuration](../../nats-server.conf); changing the AIO
limit requires mounting a replacement configuration. NATS warns that payloads
larger than 8 MiB may affect performance. JetStream stream and storage limits
are independent. The ingestion server's separate gRPC receive limit is 16 MiB
per serialized protobuf message in both deployment modes; the NATS Helm value
does not control it. NATS counts the JSON-encoded event including its envelope,
so encoding overhead can cause an accepted gRPC message to exceed the NATS limit.

## Service And Exposure Contract

The chart does not render Ingress or Gateway API resources. It exposes stable Services that downstream infrastructure can front with a load balancer, ingress controller, or gateway implementation.

Distributed mode service contract:

- `<release>-web`: HTTP on port `80`
- `<release>-api`: HTTP on port `8080`
- `<release>-ingestion`: gRPC on port `50051`

AIO service contract:

- `<release>-aio`: web `80`, API `8080`, gRPC `50051`, NATS `4222`, NATS monitor `8222`

Chart-level exposure knobs are limited to Service types and ports, for example `distributed.web.service.type`, `distributed.api.service.type`, `distributed.ingestion.service.type`, and `aio.service.type`.

## Runtime Secret Contract

Distributed workloads read `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` from a Secret.

- Recommended production path: create `observer-runtime-env` out of band and set `runtime.existingSecret=observer-runtime-env`.
- Generated-secret path: leave `runtime.existingSecret` empty and let the chart render the Secret for internal or testing installs.
- When using external dependencies, keep `postgres.*`, `externalDatabase.*`, and `externalNats.*` aligned with the services selected for the deployment.
- When `runtime.existingSecret` is set, create or update that Secret before `helm install` or `helm upgrade`.

Example production values:

```yaml
mode: distributed
image:
  tag: "<immutable-image-tag>"

distributed:
  enabled: true

runtime:
  existingSecret: observer-runtime-env

postgresql:
  enabled: false
postgres:
  host: postgres.example.com

mongodb:
  enabled: false
externalDatabase:
  host: mongo.example.com

nats:
  enabled: false
externalNats:
  url: nats://nats.example.com:4222
```

## Object Storage (S3) Configuration

Observer can persist test artifacts and attachments to S3-compatible storage.

### Using an S3 Secret

Create a Kubernetes Secret with S3 configuration:

```bash
kubectl create secret generic observer-s3-config \
  --from-literal=STORAGE_DRIVER=s3 \
  --from-literal=STORAGE_S3_ENDPOINT=https://s3.example.com \
  --from-literal=STORAGE_S3_REGION=us-east-1 \
  --from-literal=STORAGE_S3_BUCKET=observer-artifacts \
  --from-literal=STORAGE_S3_USE_PATH_STYLE=true \
  --from-literal=STORAGE_S3_ACCESS_KEY_ID=<access-key> \
  --from-literal=STORAGE_S3_SECRET_ACCESS_KEY=<secret-key>
```

Then reference it in the chart:

```yaml
storage:
  s3:
    existingSecret: observer-s3-config
```

## Custom CA Certificate Configuration

For private S3 endpoints or private image registries, supply a custom CA certificate:

### Using a Custom CA Secret

Create a Kubernetes Secret with the CA certificate:

```bash
kubectl create secret generic observer-custom-ca \
  --from-file=ca.crt=/path/to/ca.crt
```

Then reference it in the chart:

```yaml
customCA:
  existingSecret: observer-custom-ca
  mountPath: /etc/ssl/certs/custom-ca
```

The CA certificate is mounted in the distributed API and processor pods. The chart points the AWS SDK at the mounted certificate using `AWS_CA_BUNDLE`; an explicit `AWS_CA_BUNDLE` in `distributed.api.env`, `distributed.processor.env`, or `extraEnv` takes precedence.

## Extension Points

The chart provides stable extension points for adding custom environment variables, volumes, and mounts without modifying the chart templates.

### Extra Environment Variables

Add environment variables to all workload containers:

```yaml
extraEnv:
  LOG_LEVEL: debug
  CUSTOM_HEADER: custom-value
```

### Extra Environment References

Add ConfigMap or Secret references to all workload containers:

```yaml
extraEnvFrom:
  - configMapRef:
      name: app-config
  - secretRef:
      name: app-secrets
```

### Extra Volumes and Mounts

Add custom volumes to workload pods:

```yaml
extraVolumes:
  - name: custom-config
    configMap:
      name: my-config

extraVolumeMounts:
  - name: custom-config
    mountPath: /etc/custom-config
    readOnly: true
```

## Upgrading

```bash
helm upgrade observer ./charts/observer
```

Distributed upgrades run the migration hook job when `postgres.migration.enabled=true`.

If you use `runtime.existingSecret`, update the Secret before running `helm upgrade`.

The chart rejects legacy distributed connection overrides in `distributed.*.env`, so keep runtime connection settings in `runtime.existingSecret` or canonical dependency values.

## Uninstalling

```bash
helm uninstall observer
```

## Testing The Chart

Distributed rendering requires a Secret name. The name below is a test
placeholder; rendering does not require the Secret or its key to exist. For an
installation, use an operator-managed Secret as described under Distributed
Mode.

```bash
helm lint ./charts/observer --set-string distributed.ingestion.auth.existingSecret=chart-test-ingestion-auth
helm template observer ./charts/observer --set-string distributed.ingestion.auth.existingSecret=chart-test-ingestion-auth
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml --set-string distributed.ingestion.auth.existingSecret=chart-test-ingestion-auth
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

For the Make targets, pass the same flag with
`HELM_ARGS='--set-string distributed.ingestion.auth.existingSecret=chart-test-ingestion-auth'`.
For distributed install dry runs, supply the operator-managed Secret name;
Helm does not verify that the Secret or its key exists.

## Accessing The Application

Use the post-install notes for the selected mode:

```bash
helm status observer
```

For ClusterIP access in distributed mode:

```bash
kubectl port-forward svc/observer-web 8080:80
kubectl port-forward svc/observer-ingestion 50051:50051
kubectl port-forward svc/observer-api 8080:8080
```

## Support

For issues and questions, visit https://github.com/stanterprise/observer.

## License

See the main repository for license information.

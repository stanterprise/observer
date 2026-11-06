# Observer architecture and security review

This package describes the **current distributed Helm deployment** for an
engineering leader and an infrastructure or InfoSec reviewer considering an
internal installation. It is an implementation overview, not an assessment of
a running cluster, a penetration test, or a deployment approval.

| Document | Purpose |
| --- | --- |
| [Container architecture](architecture/container-architecture.drawio) | C4-style view of the applications, stores, external actors, and their communications. |
| [Kubernetes deployment](architecture/kubernetes-deployment.drawio) | Default Helm workloads, Services, storage, and operator-controlled boundaries. |
| [Deployment and security overview](deployment-security-overview.md) | Concise footprint, network, access, and data-handling notes with source links. |
| [Review checklist](review-checklist.md) | Practical questions for the deployment review. |

The diagrams are editable in draw.io/diagrams.net. Solid arrows describe a
normal connection; dashed arrows or boxes indicate optional or
operator-provided paths. Ports are defaults unless otherwise labeled.

The source baseline is repository revision `11b5cf7` (chart `0.8.3`); actual
cluster configuration, installed reporter versions, and organizational
controls have not been verified. In particular, a Kubernetes `ClusterIP` is
not by itself evidence of network isolation.

For installation details, see the [Helm chart](../../charts/observer/README.md)
and [chart values](../../charts/observer/values.yaml). The existing
[InfoSec package](../infosec/README.md) describes a separate proposed
target state; it is not evidence that those controls are implemented here.

The default resource inventory can be reproduced without supplying a real
credential:

```sh
helm template observer-review ./charts/observer \
  --set distributed.ingestion.auth.existingSecret=observer-ingestion-auth
```

This only renders a reference to a Secret; it does not create or validate its
contents, establish connectivity, or inspect a cluster. Rendered manifests
can contain Secret data; do not share them unredacted when using real values.

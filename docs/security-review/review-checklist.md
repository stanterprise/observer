# Minimal InfoSec review checklist

Use these questions with the [deployment and security overview](deployment-security-overview.md);
answers require the selected Helm values and cluster configuration.

- [ ] What Observer workloads, bundled or external dependencies, persistent
      claims, and optional object storage will this installation use?
- [ ] Which CI/test runners must reach gRPC ingestion, and is their selected
      route, transport protection, and exposure approved?
- [ ] How will engineers reach the dashboard, REST API, attachments, and
      optional WebSocket, and what restricts that access?
- [ ] What actually isolates NATS, databases, and internal Services from
      unintended clients, including access that bypasses an ingress?
- [ ] Are the required ServiceAccounts, token mounts, container users,
      capabilities, and resource settings acceptable in this cluster?
- [ ] Can the deployed reporters send the shared ingestion key, and is its
      deployment-wide scope understood?
- [ ] How are the ingestion key, database and messaging credentials, optional
      storage credentials, and their rotation handled, including NATS URLs in
      service logs?
- [ ] What sensitive data can tests send, where will results, logs, and
      attachments reside, and who can access them?
- [ ] Who owns capacity, backups, retention, deletion of external objects, and
      any remaining exposure or access decisions?

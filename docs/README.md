# Observer Documentation

This folder contains the maintained **public documentation set** for Observer.

## Start here

- Project overview and quick start: [../README.md](../README.md)
- Detailed quick start (local + Docker): [../QUICKSTART.md](../QUICKSTART.md)
- Deployment guide (Compose + Helm): [../DEPLOYMENT.md](../DEPLOYMENT.md)

## Architecture

- Overview: [architecture/00-overview.md](architecture/00-overview.md)
- Components: [architecture/01-components.md](architecture/01-components.md)
- Data flow: [architecture/02-dataflow.md](architecture/02-dataflow.md)
- Deployment modes (AIO vs distributed): [architecture/03-modes.md](architecture/03-modes.md)
- Docker Compose details: [architecture/04-docker-compose.md](architecture/04-docker-compose.md)
- Dockerfiles: [architecture/05-dockerfile.md](architecture/05-dockerfile.md)
- Database schema (MongoDB): [architecture/07-database-schema.md](architecture/07-database-schema.md)
- Roadmap / next steps: [architecture/10-next-steps.md](architecture/10-next-steps.md)

## Integrations

- Playwright reporter integration (practical guide): [PLAYWRIGHT_INTEGRATION.md](PLAYWRIGHT_INTEGRATION.md)
- Reporter integration (architecture view): [architecture/08-reporter-integration.md](architecture/08-reporter-integration.md)

## Real-time (WebSocket)

- WebSocket endpoint and event format: [WEBSOCKET_IMPLEMENTATION.md](WEBSOCKET_IMPLEMENTATION.md)

## Web UI

- Web UI overview: [../web/README.md](../web/README.md)
- Local web development (Vite + backend): [../web/README-LOCAL-DEV.md](../web/README-LOCAL-DEV.md)
- Manual testing guide: [WEB_UI_TESTING.md](WEB_UI_TESTING.md)

## Testing

- Test suite guide: [../tests/README.md](../tests/README.md)
- Test report (coverage + notes): [TEST_REPORT.md](TEST_REPORT.md)

## CI/CD and build

- Build optimization (BuildKit/buildx): [BUILD_OPTIMIZATION.md](BUILD_OPTIMIZATION.md)
- GitHub Actions workflows: [GITHUB_ACTIONS_BUILDS.md](GITHUB_ACTIONS_BUILDS.md)
- Quick reference commands: [../BUILD_QUICK_REF.md](../BUILD_QUICK_REF.md)

## Historical documents

Some older “status report” / “implementation summary” documents were generated during earlier Copilot-assisted work and are kept for archaeology only.

- Archive index: [archive/README.md](archive/README.md)

## Internal-only documents

The following files are **not** part of the public documentation set and should be treated as internal references:

- Root-level STEP*COMPONENT*\* guides
- Implementation plans and fix summaries in docs/ that are superseded by current guides
- Archived summaries are stored in archive/2026-release-cleanup/

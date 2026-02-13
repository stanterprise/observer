# Contributing

Thanks for your interest in contributing to Observer!

## Quick Start

1. Fork the repo and create a feature branch.
2. Follow the local setup in [QUICKSTART.md](QUICKSTART.md).
3. Make your changes with clear commit messages.
4. Run tests:
   - `make test`
5. Open a pull request with a concise description.

## Code Style

- Go: run `make fmt` and `make lint` when relevant.
- Web UI: follow existing lint and formatting rules in `web/`.

## Testing

- Unit tests: `make test`
- NATS integration tests: `make test-nats-integration` (requires NATS)

## Security

For vulnerabilities, please follow [SECURITY.md](SECURITY.md).

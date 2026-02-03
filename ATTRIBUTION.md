# Attribution Inventory

This file tracks third-party dependencies for Observer.

## Generated inventories

- Go modules list: specs/001-release-cleanup/artifacts/go-modules.json
- Web dependency licenses: specs/001-release-cleanup/artifacts/web-licenses.json

## Notes

- The Go module list currently does not include per-module license metadata. The Linux go-licenses report is still pending (specs/001-release-cleanup/artifacts/go-licenses.txt is empty due to tooling issues) and must be regenerated before release signoff.
- The web license inventory is derived from package-lock.json and includes license metadata when available.

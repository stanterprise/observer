# Attribution Inventory

This file tracks third-party dependencies for Observer.

## Generated inventories

- Go modules list: specs/001-release-cleanup/artifacts/go-modules.json
- Go licenses report: specs/001-release-cleanup/artifacts/go-licenses.txt
- Web dependency licenses: specs/001-release-cleanup/artifacts/web-licenses.json

## Notes

- The Go license report contains `Unknown` entries for github.com/stanterprise/proto-go modules. Confirm the license in that repository and update attribution before release signoff.
- The web license inventory is derived from package-lock.json and includes license metadata when available.

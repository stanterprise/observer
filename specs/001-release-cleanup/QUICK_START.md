# Release Cleanup - Quick Start Guide

**For**: Team members starting work on release cleanup  
**See**: [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for full details

## Overview

This guide helps you quickly get started with the release cleanup process. Follow these steps to contribute to making the Observer repository release-ready.

## Prerequisites

- Access to Observer repository
- Appropriate tools for your assigned phase (see below)
- GitHub account with repository access

## Quick Start by Role

### Release Manager

**Your Phases**: 1, 8  
**Key Documents**:
- [Implementation Plan](./IMPLEMENTATION_PLAN.md)
- [Readiness Checklist](./artifacts/readiness-checklist.md)
- [Spec](./spec.md)

**First Steps**:
1. Review the implementation plan thoroughly
2. Assign owners to each phase
3. Set up tracking board (GitHub Projects)
4. Schedule phase kickoff meetings
5. Start Phase 1 audit

### Technical Writer / Documentation Lead

**Your Phase**: 2  
**Duration**: 2-3 days  

**Setup**:
```bash
# Clone if not already
git clone https://github.com/stanterprise/observer
cd observer

# Review all docs
find docs/ -name "*.md" -type f
ls -la *.md
```

**Tasks Checklist**:
- [ ] Read all files in `docs/` directory
- [ ] Test quickstart instructions in clean environment
- [ ] Identify outdated or contradictory content
- [ ] Update documentation for accuracy
- [ ] Remove/archive implementation details
- [ ] Verify all links work
- [ ] Document your findings in `artifacts/docs-inventory.md`

**Key Files to Review**:
```
README.md
QUICKSTART.md
DEPLOYMENT.md
CODESPACES.md
docs/README.md
docs/architecture/*.md
docs/*.md (all files)
```

### DevOps Engineer

**Your Phases**: 3, 4, 7  
**Duration**: 5-8 days total

**Setup**:
```bash
# Install required tools
brew install gitleaks          # or appropriate for your OS
docker --version              # Ensure Docker installed
kubectl version --client      # Ensure kubectl installed
helm version                  # Ensure Helm installed

# Clone repo fresh
git clone https://github.com/stanterprise/observer
cd observer
```

**Phase 3 Tasks** (Repository Hygiene):
- [ ] Review for temporary files
- [ ] Update .gitignore
- [ ] Clean up directory structure
- [ ] Remove build artifacts

**Phase 4 Tasks** (Build Verification):
- [ ] Test build on Ubuntu 22.04
- [ ] Test build on macOS
- [ ] Test build on Windows/WSL2
- [ ] Test Docker builds
- [ ] Test Helm installation
- [ ] Document prerequisites

**Phase 7 Tasks** (CI/CD):
- [ ] Review GitHub Actions workflows
- [ ] Configure branch protection
- [ ] Document release process
- [ ] Create automated readiness checks

**Test Commands**:
```bash
# Build testing
make build-all
make docker-build-all
make docker-buildx-aio

# Deployment testing
docker compose --profile web-dev up -d
docker compose --profile dist up -d
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer

# Clean up
make mongodb-down
make nats-down
docker compose down -v
```

### Security Engineer

**Your Phase**: 5  
**Duration**: 2-3 days

**Setup**:
```bash
# Install scanning tools
brew install gitleaks          # macOS
# OR
apt-get install gitleaks      # Ubuntu
# OR
pip install trufflehog        # Python package manager

# Clone repo
git clone https://github.com/stanterprise/observer
cd observer
```

**Tasks Checklist**:
- [ ] Install gitleaks or similar tool
- [ ] Scan current files for secrets
- [ ] Scan full git history
- [ ] Review configuration files
- [ ] Document all findings
- [ ] Create remediation plan
- [ ] Execute remediation
- [ ] Create SECURITY.md

**Scan Commands**:
```bash
# Current files scan
gitleaks detect --source . --verbose \
  --report-path specs/001-release-cleanup/artifacts/gitleaks-report.json

# Full history scan (IMPORTANT)
gitleaks detect --source . --log-opts="--all" --verbose \
  --report-path specs/001-release-cleanup/artifacts/gitleaks-history-report.json

# Review findings
cat specs/001-release-cleanup/artifacts/gitleaks-report.json | jq .

# Check specific file types
find . -name "*.env" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" | \
  grep -v node_modules | grep -v vendor
```

**Files to Review Manually**:
- `.env.example`
- `docker-compose.yml`
- `.github/workflows/*.yml`
- Any config files with credentials

### Legal/Compliance Team

**Your Phase**: 6  
**Duration**: 2-3 days

**Setup**:
```bash
# Clone repo
git clone https://github.com/stanterprise/observer
cd observer

# Generate dependency lists
go list -m all > specs/001-release-cleanup/artifacts/dependencies.txt
go mod graph > specs/001-release-cleanup/artifacts/dependency-graph.txt

# Web dependencies
cd web
npm list --all > ../specs/001-release-cleanup/artifacts/web-dependencies.txt
cd ..
```

**Tasks Checklist**:
- [ ] Review all Go dependencies licenses
- [ ] Review all NPM dependencies licenses
- [ ] Check for GPL/AGPL conflicts
- [ ] Identify attribution requirements
- [ ] Choose appropriate license for project
- [ ] Create LICENSE file
- [ ] Create ATTRIBUTION.md if needed
- [ ] Document findings in `artifacts/license-inventory.md`

**Key Dependencies to Check**:
```bash
# Top-level Go dependencies
grep -E "^github.com|^go.mongodb.org|^google.golang.org" go.mod

# Key packages
# - MongoDB driver: go.mongodb.org/mongo-driver
# - NATS: github.com/nats-io/nats.go
# - gRPC: google.golang.org/grpc
# - Protobuf: github.com/stanterprise/proto-go
```

**License Resources**:
- [SPDX License List](https://spdx.org/licenses/)
- [Choose a License](https://choosealicense.com/)
- [TLDRLegal](https://tldrlegal.com/)

## Tool Installation

### gitleaks (Secrets Scanner)

```bash
# macOS
brew install gitleaks

# Ubuntu/Debian
wget https://github.com/gitleaks/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz
tar -xzf gitleaks_8.18.0_linux_x64.tar.gz
sudo mv gitleaks /usr/local/bin/

# Verify
gitleaks version
```

### Build Tools

```bash
# Go
# Download from https://go.dev/dl/
# Or use package manager
brew install go           # macOS
sudo apt install golang   # Ubuntu

# Make
brew install make         # macOS
sudo apt install make     # Ubuntu

# Docker
# Download from https://docker.com/

# Kubectl
brew install kubectl      # macOS
sudo apt install kubectl  # Ubuntu

# Helm
brew install helm         # macOS
sudo snap install helm    # Ubuntu
```

## Test Environments

### Option 1: Local VM
```bash
# Using Vagrant
vagrant init ubuntu/jammy64
vagrant up
vagrant ssh

# Or using Docker
docker run -it --rm ubuntu:22.04 bash
```

### Option 2: GitHub Codespaces
1. Go to repository on GitHub
2. Click "Code" → "Codespaces" → "Create codespace on main"
3. Wait 2-3 minutes for setup
4. Start working!

### Option 3: Cloud VM
- AWS EC2, GCP Compute Engine, Azure VM
- Use Ubuntu 22.04 LTS
- Install Docker and tools

## Communication

### Daily Standups
- What did you complete yesterday?
- What are you working on today?
- Any blockers?

### Phase Completion Reports
When completing a phase:
1. Update readiness checklist
2. Upload evidence artifacts
3. Notify release manager
4. Document any issues/risks

### Issue Reporting
If you find issues:
1. Document in appropriate artifact file
2. Add to risk register if significant
3. Create GitHub issue if needed
4. Tag release manager

## Artifact Submission

All artifacts go in: `specs/001-release-cleanup/artifacts/`

**Template**:
```markdown
# [Artifact Name]

**Phase**: [1-8]
**Owner**: [Your Name]
**Date**: [YYYY-MM-DD]
**Status**: [Draft/Review/Approved]

## Summary
[Brief overview of findings]

## Details
[Detailed information]

## Recommendations
[Action items or next steps]

## Evidence
[Links to files, screenshots, logs, etc.]
```

## Timeline Tracker

| Phase | Duration | Owner | Status | Completion Date |
|-------|----------|-------|--------|-----------------|
| 1 - Audit & Planning | 1-2 days | Release Manager | 🔄 | - |
| 2 - Documentation | 2-3 days | Tech Writer | ⏸️ | - |
| 3 - Repo Hygiene | 1-2 days | DevOps | ⏸️ | - |
| 4 - Build Verification | 2-3 days | DevOps | ⏸️ | - |
| 5 - Security Scan | 2-3 days | Security | ⏸️ | - |
| 6 - Licensing | 2-3 days | Legal | ⏸️ | - |
| 7 - CI/CD | 2-3 days | DevOps | ⏸️ | - |
| 8 - Final Validation | 1-2 days | Release Manager | ⏸️ | - |

**Legend**: ✅ Complete | 🔄 In Progress | ⏸️ Not Started | ❌ Blocked

## Quick Links

- [Implementation Plan](./IMPLEMENTATION_PLAN.md) - Full detailed plan
- [Spec](./spec.md) - Feature specification
- [Readiness Checklist](./artifacts/readiness-checklist.md) - Gate tracking
- [Observer README](../../README.md) - Repository overview
- [Artifacts](./artifacts/) - Evidence and reports

## Getting Help

- **Questions**: Ask in team chat or email release manager
- **Blockers**: Escalate immediately to release manager
- **Technical Issues**: Check existing documentation first
- **Security Concerns**: Contact security engineer immediately

## Success Criteria

We're done when:
- ✅ All 18 gates in readiness checklist show PASS or RISK (with approval)
- ✅ All artifacts collected and approved
- ✅ Stakeholders have signed off
- ✅ Release decision documented

---

**Created**: February 3, 2026  
**Last Updated**: February 3, 2026  
**Maintained By**: Release Manager  
**Questions**: Contact release manager

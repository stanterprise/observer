# Tasks: Release Readiness Cleanup

**Branch**: `001-release-cleanup` | **Date**: 2026-02-03 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Task Breakdown

### Phase 0: Research & Discovery

- [x] **Task 0.1**: Audit existing documentation
  - Review all markdown files in repository root
  - Identify outdated, incomplete, or conflicting documentation
  - Document findings in research.md

- [x] **Task 0.2**: Scan for secrets and sensitive data
  - Search codebase for hardcoded credentials, tokens, API keys
  - Check git history for accidentally committed secrets
  - Review environment variable usage patterns
  - Document findings in research.md

- [x] **Task 0.3**: Review build and deployment configurations
  - Analyze Dockerfiles, docker-compose.yml, Makefiles
  - Identify undocumented build steps or missing configurations
  - Check CI/CD pipeline status (if exists)
  - Document findings in research.md

- [x] **Task 0.4**: Licensing audit
  - Verify LICENSE file exists and is appropriate
  - Check dependency licenses for compatibility
  - Identify any missing copyright headers or attribution
  - Document findings in research.md

### Phase 1: Design & Contracts

- [x] **Task 1.1**: Design readiness checklist schema
  - Define JSON schema for readiness gates (docs, build, secrets, licensing, CI)
  - Create contracts/readiness-checklist.schema.json
  - Include validation rules and evidence requirements

- [x] **Task 1.2**: Design cleanup task schema
  - Define JSON schema for tracking remediation tasks
  - Create contracts/cleanup-task.schema.json
  - Include fields for ownership, priority, status, evidence

- [x] **Task 1.3**: Design risk register schema
  - Define JSON schema for documenting blockers and risks
  - Create contracts/risk-register.schema.json
  - Include mitigation strategies and impact assessment

- [x] **Task 1.4**: Create data model documentation
  - Document all schemas and their relationships
  - Define validation rules and constraints
  - Create data-model.md with clear examples

- [x] **Task 1.5**: Create quickstart guide
  - Document how to use the readiness checklist
  - Provide examples of filling out schemas
  - Create quickstart.md with step-by-step instructions

### Phase 2: Implementation

- [x] **Task 2.1**: Initialize readiness checklist
  - Create readiness-checklist.json following schema
  - Populate initial gate status based on Phase 0 findings
  - Mark all incomplete items with owners

- [x] **Task 2.2**: Create cleanup tasks registry
  - Create cleanup-tasks.json following schema
  - Break down each readiness gate failure into actionable tasks
  - Assign owners and priorities

- [x] **Task 2.3**: Initialize risk register
  - Create risk-register.json following schema
  - Document all blockers identified in Phase 0
  - Define mitigation strategies for high-priority risks

- [x] **Task 2.4**: Documentation remediation
  - Update outdated documentation identified in Task 0.1
  - Remove conflicting or redundant docs
  - Ensure README accurately reflects current state
  - Update QUICKSTART.md with current setup steps

- [x] **Task 2.5**: Secrets cleanup
  - Remove any secrets found in Task 0.2
  - Add .gitignore entries for sensitive files
  - Document required environment variables
  - Create .env.example template

- [x] **Task 2.6**: Build configuration cleanup
  - Ensure all build steps are documented
  - Verify Dockerfiles follow best practices
  - Update Makefile with missing targets
  - Document deployment process

- [ ] **Task 2.7**: Licensing compliance
  - Add LICENSE file if missing
  - Add copyright headers where needed
  - Create ATTRIBUTION.md for dependencies
  - Document license compatibility

### Phase 3: Validation

- [x] **Task 3.1**: Validate JSON schemas
  - Test all JSON artifacts against their schemas
  - Ensure validation passes with no errors
  - Fix any schema violations

- [x] **Task 3.2**: Build validation
  - Run `make build-all` and verify success
  - Run `go test ./tests` and verify pass
  - Run `cd web && npm run build` and verify success
  - Document any build issues in risk-register.json

- [x] **Task 3.3**: Documentation review
  - Verify all docs are internally consistent
  - Check that quickstart guide works end-to-end
  - Ensure no broken links or references
  - Get peer review on updated docs

- [ ] **Task 3.4**: Release readiness gate review
  - Review readiness-checklist.json for completeness
  - Verify all cleanup tasks are completed or tracked
  - Confirm all high-priority risks have mitigation plans
  - Get approval from stakeholders

### Phase 4: Delivery

- [ ] **Task 4.1**: Commit all changes
  - Stage all modified files
  - Write descriptive commit message
  - Push to 001-release-cleanup branch

- [ ] **Task 4.2**: Create pull request
  - Write PR description linking to spec.md
  - Reference readiness-checklist.json in PR
  - Request reviews from relevant stakeholders
  - Address review feedback

- [ ] **Task 4.3**: Merge and tag
  - Merge PR after approval
  - Tag release (if appropriate)
  - Update main branch documentation

- [ ] **Task 4.4**: Post-cleanup verification
  - Verify merged changes work in main branch
  - Ensure CI pipeline passes (if exists)
  - Archive spec artifacts for future reference
  - Document lessons learned

## Task Dependencies

```
Phase 0 (Research)
  ↓
Phase 1 (Design) — depends on Phase 0 findings
  ↓
Phase 2 (Implementation) — depends on Phase 1 schemas
  ↓
Phase 3 (Validation) — depends on Phase 2 artifacts
  ↓
Phase 4 (Delivery) — depends on Phase 3 approval
```

## Ownership & Timeline

- **Owner**: TBD (assign during task kickoff)
- **Estimated Duration**: 2-3 days
- **Target Completion**: TBD
- **Blocking Dependencies**: None (self-contained cleanup)

## Success Criteria

- [ ] All readiness gates documented with status and evidence
- [ ] All cleanup tasks tracked with owners and priorities
- [ ] All risks documented with mitigation strategies
- [ ] Documentation is accurate, consistent, and complete
- [ ] No secrets or sensitive data in repository
- [ ] Build process is documented and reproducible
- [ ] Licensing is compliant and documented

## Notes

- Tasks can be worked in parallel within each phase
- Phase transitions require review and approval
- Use readiness-checklist.json as the single source of truth for status
- Update risk-register.json immediately when blockers are discovered

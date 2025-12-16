# Observer Custom Agents

This document describes the specialized AI agent definitions for the Observer test observability system. These agents are designed to work with GitHub Copilot and provide domain-specific expertise for different aspects of the project.

## Agent Files Location

All agent definitions are located in `.github/agents/`:
- `architect.md` - System design and architecture
- `developer.md` - Go backend and React/TypeScript frontend implementation  
- `ux-designer.md` - UI/UX design and user experience
- `devops.md` - Infrastructure, deployment, and operations
- `testing.md` - Test strategy and implementation
- `documentation.md` - Technical writing and documentation

## Coding Guidelines for Agent Files

### File Size Management

To maintain cognitive load and readability:

**Target File Size**: 400-600 lines per agent file
- **Minimum**: 200 lines (ensures comprehensive coverage)
- **Maximum**: 1000 lines (prevents cognitive overload)
- **Sweet Spot**: 400-600 lines (detailed but manageable)

**Current Status**:
| Agent | Lines | Status |
|-------|-------|--------|
| architect.md | ~192 | ✅ Within range |
| developer.md | ~393 | ✅ Within range |
| ux-designer.md | ~451 | ✅ Optimal |
| devops.md | ~553 | ✅ Optimal |
| testing.md | ~612 | ⚠️ Above optimal (but acceptable) |
| documentation.md | ~931 | ⚠️ Consider splitting |

### Organization Principles

1. **Clear Structure**: Use consistent heading hierarchy (H1 → H2 → H3)
2. **Scannable Content**: Break large sections into subsections
3. **Concrete Examples**: Include 3-5 practical examples per major topic
4. **Progressive Disclosure**: Start with overview, then drill into details
5. **Cross-References**: Link to related agents and sections

### Content Guidelines

**Essential Sections** (every agent should have):
- Agent name and role (H1)
- Core Expertise (H2)
- Observer-Specific Knowledge (H2)
- Responsibilities (H2)
- Guidelines (H2)
- Collaboration (H2)
- Example Scenarios (H2)

**Optional Sections** (add as needed):
- Anti-Patterns
- Context Awareness  
- Output Format
- Tool/Technology-specific guidance

### When to Split an Agent

Consider splitting when:
- File exceeds 1000 lines
- Agent covers multiple distinct domains
- Sections become too deep (H4+)
- Examples dominate the content

**Example**: If Documentation agent grows beyond 1000 lines, consider:
- `documentation-api.md` - API documentation
- `documentation-architecture.md` - Architecture docs
- `documentation-user-guides.md` - User-facing docs

## Available Agents

### 🏗️ [Architect](./agents/architect.md)
**Expertise**: System design, distributed architecture, event-driven patterns

Use the Architect agent when you need help with:
- Designing new features and system components
- Reviewing and refactoring existing architecture
- Evaluating architectural decisions and trade-offs
- Planning system evolution and scalability
- Designing service boundaries and API contracts
- Reviewing PRs for architectural compliance

**Example prompts**:
- "Design a feature to support test artifact storage with MinIO"
- "Review the current NATS consumer architecture for scalability"
- "Help refactor ingestion service to remove dual-write pattern"
- "Design a caching strategy for the API service"

### 👨‍💻 [Developer](./agents/developer.md)
**Expertise**: Go backend, React/TypeScript frontend, implementation patterns

Use the Developer agent when you need help with:
- Implementing new features and bug fixes
- Writing idiomatic Go and TypeScript code
- Following established code patterns and conventions
- Implementing tests (unit, integration, E2E)
- Reviewing code quality and best practices
- Debugging and troubleshooting issues

**Example prompts**:
- "Implement a new gRPC method to delete test runs"
- "Fix the WebSocket connection lifecycle bug"
- "Add validation for test metadata fields"
- "Review this PR for code quality issues"
- "Implement the test detail view component"

### 🎨 [UX Designer](./agents/ux-designer.md)
**Expertise**: UI/UX design, React components, accessibility, Tailwind CSS

Use the UX Designer agent when you need help with:
- Designing user interfaces and interactions
- Creating React components with proper UX
- Ensuring accessibility (WCAG compliance)
- Implementing responsive designs
- Designing loading, error, and empty states
- Reviewing UI implementations

**Example prompts**:
- "Design a test detail view with step timeline"
- "Create a filtering UI for test runs"
- "Review the loading states in the dashboard"
- "Design real-time update indicators for test cards"
- "Improve the error handling UX"

### 🚀 [DevOps](./agents/devops.md)
**Expertise**: Docker, Kubernetes, Helm, CI/CD, infrastructure

Use the DevOps agent when you need help with:
- Optimizing Docker images and builds
- Designing Kubernetes deployments
- Creating and maintaining Helm charts
- Building CI/CD pipelines
- Implementing monitoring and observability
- Planning disaster recovery and high availability

**Example prompts**:
- "Optimize the AIO Docker image size"
- "Design Kubernetes deployment with autoscaling"
- "Add Prometheus metrics to all services"
- "Create a CI pipeline for multi-arch Docker builds"
- "Design a disaster recovery plan"

### 🧪 [Testing](./agents/testing.md)
**Expertise**: Test strategy, Go testing, React testing, integration tests

Use the Testing agent when you need help with:
- Designing test strategies and coverage
- Writing unit, integration, and E2E tests
- Implementing test infrastructure
- Fixing flaky or failing tests
- Performance and load testing
- Reviewing test quality and coverage

**Example prompts**:
- "Design test strategy for artifact storage feature"
- "Fix the flaky NATS integration test"
- "Add test coverage for error paths"
- "Create E2E tests for the web UI"
- "Implement performance benchmarks"

### 📝 [Documentation](./agents/documentation.md)
**Expertise**: Technical writing, API docs, architecture docs, user guides

Use the Documentation agent when you need help with:
- Writing technical documentation
- Creating API documentation
- Documenting architecture and design decisions
- Writing user guides and tutorials
- Creating troubleshooting guides
- Reviewing documentation quality

**Example prompts**:
- "Document the new artifact storage API"
- "Create a troubleshooting guide for NATS issues"
- "Update deployment guide with new configuration"
- "Write architecture documentation for WebSocket streaming"
- "Create a quick start guide for new contributors"

## How to Use Custom Agents

### GitHub Copilot Chat

When using GitHub Copilot Chat, you can reference these agents by mentioning them in your prompts:

```
@workspace Use the Architect agent to design a caching layer for the API service
```

```
@workspace Ask the Developer agent to implement the artifact upload endpoint
```

```
@workspace Have the UX Designer agent review the test detail view design
```

### Best Practices

1. **Be Specific**: Clearly describe what you need help with
2. **Provide Context**: Share relevant code, files, or background information
3. **Right Agent for the Job**: Choose the agent whose expertise matches your need
4. **Iterate**: Start with high-level design, then drill down to implementation
5. **Combine Agents**: Use multiple agents in sequence (Architect → Developer → Testing)

### Agent Collaboration Workflow

For complex features, use agents in sequence:

1. **Architect**: Design the feature architecture
   - System design, API contracts, data flow
2. **Developer**: Implement the feature
   - Write code following the architectural design
3. **Testing**: Add test coverage
   - Write tests to validate the implementation
4. **DevOps**: Deploy and monitor
   - Create deployment configs, add monitoring
5. **Documentation**: Document the feature
   - Write user guides, API docs, architecture docs

### Example Workflow: Adding Artifact Storage

**Step 1: Architecture**
```
@workspace Use the Architect agent to design a feature for storing test artifacts 
(screenshots, videos) with MinIO/S3 backend.
```

**Step 2: Implementation**
```
@workspace Use the Developer agent to implement the artifact storage API endpoints
based on the architectural design.
```

**Step 3: Testing**
```
@workspace Use the Testing agent to create test strategy and implement tests for 
artifact storage feature.
```

**Step 4: Infrastructure**
```
@workspace Use the DevOps agent to add MinIO to docker-compose and Helm chart 
for artifact storage.
```

**Step 5: UI Design**
```
@workspace Use the UX Designer agent to design an artifact viewer component 
in the web UI.
```

**Step 6: Documentation**
```
@workspace Use the Documentation agent to document the artifact storage feature
in the user guide and API documentation.
```

## Agent Guidelines

### What Agents Can Do
✅ Provide expert guidance and recommendations  
✅ Design architectures and features  
✅ Review code and designs  
✅ Generate implementation code  
✅ Suggest best practices and patterns  
✅ Explain trade-offs and alternatives  
✅ Help debug and troubleshoot issues  

### What Agents Cannot Do
❌ Make actual code changes directly (you must apply their suggestions)  
❌ Run tests or build code (you must execute commands)  
❌ Access external systems or APIs  
❌ Make decisions requiring business context  
❌ Guarantee correctness (always review and validate)  

## Maintaining Custom Agents

### Updating Agent Definitions

When the project evolves, update agents to reflect:
- New architectural patterns or decisions
- Updated technology versions
- New coding conventions
- Changed deployment strategies
- New features or components

### File Size Management

Monitor agent file sizes:
```bash
# Check current sizes
wc -l .github/agents/*.md

# Target: Keep files under 1000 lines
# Optimal: 400-600 lines for best readability
```

If an agent file grows beyond 1000 lines:
1. Review content for redundancy
2. Move detailed examples to separate docs
3. Consider splitting into sub-agents
4. Archive outdated content

### Adding New Agents

Consider adding specialized agents for:
- **Security**: Security reviews, vulnerability scanning, threat modeling
- **Performance**: Performance optimization, profiling, benchmarking
- **Database**: Schema design, migration strategies, query optimization
- **API Design**: REST/GraphQL API design, versioning strategies
- **Mobile**: If adding mobile test reporters or mobile UI

### Agent Quality Checklist

Good agent definitions should:
- [ ] Clearly define expertise and responsibilities
- [ ] Include Observer-specific context and patterns
- [ ] Provide concrete examples and scenarios
- [ ] Explain collaboration with other agents
- [ ] Include anti-patterns to avoid
- [ ] Use consistent formatting and structure
- [ ] Be regularly updated with project evolution
- [ ] Stay within 200-1000 line range
- [ ] Have 3+ practical examples per major topic

## Feedback and Improvements

These agents are living documents that should evolve with the project. If you find:
- Missing information or context
- Outdated patterns or conventions
- Unclear guidance or examples
- New areas requiring specialized expertise
- Files becoming too large or unwieldy

Please update the relevant agent file or create a new agent definition!

## Quick Reference

| Agent | Primary Use Case | Key Expertise | File Size |
|-------|------------------|---------------|-----------|
| **Architect** | System design & review | Distributed systems, event-driven, microservices | ~192 lines |
| **Developer** | Code implementation | Go, TypeScript, React, testing | ~393 lines |
| **UX Designer** | UI/UX design | React components, Tailwind, accessibility | ~451 lines |
| **DevOps** | Infrastructure & deployment | Docker, Kubernetes, CI/CD, monitoring | ~553 lines |
| **Testing** | Test strategy & implementation | Unit tests, integration, E2E, performance | ~612 lines |
| **Documentation** | Technical writing | User guides, API docs, architecture docs | ~931 lines |

---

**Last Updated**: December 2024  
**Project Version**: 0.3.0 (Phase 3+)  
**Maintained by**: Observer Development Team

# Observer Custom Agents

The `.github/agents/` directory contains custom GitHub Copilot agents tailored for the Observer test observability system. These agents follow the [GitHub custom agents configuration standards](https://docs.github.com/en/copilot/reference/custom-agents-configuration).

## Agent Files

All agent files use the `.agent.md` extension and include YAML frontmatter for configuration, followed by markdown instructions for the agent's expertise and behavior.

### Available Agents

| Agent | Purpose | Key Tools |
|-------|---------|-----------|
| **architect** | System architecture, design patterns, distributed systems | read, search, edit, grep, glob, bash, view, create |
| **developer** | Full-stack development (Go backend, React frontend) | read, search, edit, grep, glob, bash, view, create, gh-advisory-database, codeql_checker |
| **devops** | CI/CD, containerization, infrastructure automation | read, search, edit, grep, glob, bash, view, create, github-mcp-server-actions_* |
| **documentation** | Technical writing, API docs, user guides | read, search, edit, grep, glob, view, create, web_search |
| **testing** | Test strategy, test automation, quality assurance | read, search, edit, grep, glob, bash, view, create, codeql_checker, code_review |
| **ux-designer** | UI/UX design, component design, accessibility | read, search, edit, grep, glob, view, create, playwright-browser_* |

## Configuration Format

Each agent file follows this structure:

```yaml
---
name: agent-name
description: "Brief description of agent capabilities"
tools: [list, of, available, tools]
infer: true
metadata:
  owner: observer-team
  category: agent-category
  version: 1.0.0
---

# Agent Instructions (Markdown)
```

### YAML Properties

- **name**: Unique identifier for the agent
- **description**: Required. Clear explanation of agent capabilities
- **tools**: List of tools the agent can use (see Tool Recommendations below)
- **infer**: Whether Copilot can automatically invoke this agent (default: true)
- **metadata**: Optional key-value pairs for organizational purposes

## Tool Recommendations by Agent

### Architect Agent
**Purpose**: Design system architecture, review design decisions, plan technical solutions

**Recommended Tools**:
- `read`, `view` - Read existing architecture documentation and code
- `search`, `grep`, `glob` - Find architectural patterns and components
- `edit`, `create` - Document architectural decisions and designs
- `bash` - Execute commands to explore system behavior

**When to use**: System design, architecture reviews, technical planning, refactoring guidance

### Developer Agent
**Purpose**: Implement features, fix bugs, write code (Go backend + React frontend)

**Recommended Tools**:
- `read`, `view` - Read existing code and documentation
- `search`, `grep`, `glob` - Find code patterns and dependencies
- `edit`, `create` - Write and modify code
- `bash` - Run builds, tests, linters
- `gh-advisory-database` - Check dependencies for security vulnerabilities
- `codeql_checker` - Run security scans on code changes

**When to use**: Feature implementation, bug fixes, code reviews, refactoring

### DevOps Agent
**Purpose**: Manage deployments, CI/CD pipelines, infrastructure automation

**Recommended Tools**:
- `read`, `view`, `edit`, `create` - Work with Dockerfiles, Helm charts, workflows
- `search`, `grep`, `glob` - Find infrastructure configurations
- `bash` - Execute Docker, kubectl, helm commands
- `github-mcp-server-actions_list` - List GitHub Actions workflows
- `github-mcp-server-actions_get` - Get workflow details and logs
- `github-mcp-server-get_job_logs` - Retrieve CI/CD job logs for debugging

**When to use**: Deployment configuration, CI/CD issues, infrastructure setup, container optimization

### Documentation Agent
**Purpose**: Create and maintain technical documentation

**Recommended Tools**:
- `read`, `view` - Review existing documentation
- `search`, `grep`, `glob` - Find related documentation
- `edit`, `create` - Write and update documentation files
- `web_search` - Research best practices and external references

**When to use**: Writing docs, updating guides, API documentation, troubleshooting guides

### Testing Agent
**Purpose**: Design test strategies, implement tests, ensure quality

**Recommended Tools**:
- `read`, `view` - Review existing tests and code
- `search`, `grep`, `glob` - Find test patterns and coverage gaps
- `edit`, `create` - Write and modify tests
- `bash` - Run test suites, coverage reports, benchmarks
- `codeql_checker` - Run security scans
- `code_review` - Request automated code reviews

**When to use**: Test implementation, coverage analysis, quality reviews, test debugging

### UX Designer Agent
**Purpose**: Design user interfaces, improve UX, ensure accessibility

**Recommended Tools**:
- `read`, `view` - Review UI components and styles
- `search`, `grep`, `glob` - Find design patterns and components
- `edit`, `create` - Design and document UI components
- `playwright-browser_snapshot` - Capture accessibility snapshots
- `playwright-browser_take_screenshot` - Take UI screenshots for review

**When to use**: UI design, component creation, accessibility improvements, UX reviews

## Usage

### Invoking Agents Explicitly

In GitHub Copilot chat or CLI:
```
@architect help me design a new feature for artifact storage
@developer implement the artifact upload endpoint
@devops optimize the Docker image size
@documentation write API docs for the new endpoint
@testing create integration tests for artifact storage
@ux-designer design the artifact viewer component
```

### Automatic Invocation

With `infer: true`, Copilot can automatically select the appropriate agent based on your request context. For example:
- "Design a caching layer" → May invoke **architect**
- "Fix this bug in the API" → May invoke **developer**
- "Why is the CI failing?" → May invoke **devops**
- "Document this API" → May invoke **documentation**
- "Add tests for this" → May invoke **testing**
- "Improve this UI" → May invoke **ux-designer**

## Best Practices

1. **Agent Specialization**: Each agent has a specific domain. Use the right agent for the task.
2. **Tool Access**: Agents only have access to tools listed in their configuration.
3. **Context Provision**: Provide sufficient context when invoking agents (file paths, requirements, constraints).
4. **Iterative Design**: For complex tasks, break them into agent-specific subtasks.
5. **Cross-Agent Collaboration**: Agents can reference each other's work (e.g., developer implements architect's design).

## Agent Maintenance

- **Cognitive Load**: Keep agent instructions between 400-600 lines for optimal performance
- **Consistency**: Follow the Observer coding guidelines documented in each agent
- **Updates**: Update agent versions in metadata when making significant changes
- **Testing**: Verify agents work as expected by testing invocations

## References

- [GitHub Custom Agents Configuration](https://docs.github.com/en/copilot/reference/custom-agents-configuration)
- [GitHub Copilot CLI Custom Agents](https://deepwiki.com/github/copilot-cli/3.6-custom-agents)
- [Observer Custom Agents Guidelines](../CUSTOM_AGENTS.md)

## Contributing

When modifying agents:
1. Maintain YAML frontmatter structure
2. Follow markdown formatting standards
3. Keep tool lists relevant to agent purpose
4. Update version in metadata
5. Test agent invocations after changes

For questions or suggestions, contact the Observer team.

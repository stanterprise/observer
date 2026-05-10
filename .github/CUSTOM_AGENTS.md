# Custom Agents Policy

This repository intentionally keeps a small, role-based custom agent set.

## Active Set

The supported agents are:

- `developer`
- `architect`
- `testing`
- `devops`
- `documentation`
- `ux-designer`

## Design Rules

- Prefer updating an existing role agent over adding a new one.
- Only add a new agent when the workflow is distinct and frequently reused.
- Keep frontmatter valid and minimal: `name`, `description`, `tools`, `infer`.
- Keep instructions practical and Observer-specific.

## Anti-Sprawl Rules

- Do not keep duplicate prompt systems and agent systems for the same workflow.
- Remove stale agents that are not part of regular development work.
- If an agent has not been used in recent cycles, archive or delete it.

## Suggested Invocation Style

- "Use developer to implement ..."
- "Use testing to validate ..."
- "Use devops to debug CI ..."

## Maintenance Checklist

- Confirm each agent maps to a real recurring task.
- Confirm tool scope is necessary and no broader than needed.
- Confirm docs in `.github/AGENTS.md` match actual files in `.github/agents/`.
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

| Agent             | Primary Use Case               | Key Expertise                                    | File Size  |
| ----------------- | ------------------------------ | ------------------------------------------------ | ---------- |
| **Architect**     | System design & review         | Distributed systems, event-driven, microservices | ~192 lines |
| **Developer**     | Code implementation            | Go, TypeScript, React, testing                   | ~393 lines |
| **UX Designer**   | UI/UX design                   | React components, Tailwind, accessibility        | ~451 lines |
| **DevOps**        | Infrastructure & deployment    | Docker, Kubernetes, CI/CD, monitoring            | ~553 lines |
| **Testing**       | Test strategy & implementation | Unit tests, integration, E2E, performance        | ~612 lines |
| **Documentation** | Technical writing              | User guides, API docs, architecture docs         | ~931 lines |

---

**Last Updated**: December 2024  
**Project Version**: 0.3.0 (Phase 3+)  
**Maintained by**: Observer Development Team

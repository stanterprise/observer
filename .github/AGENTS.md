# Observer Agent Setup

This repository now uses a lean agent set focused on day-to-day work.

## Active Agents

- `developer`: Default for implementation, bug fixes, and refactors
- `architect`: System design and cross-service changes
- `testing`: Test strategy, flaky tests, and coverage gaps
- `devops`: CI/CD, Docker, Helm, and deployment workflows
- `documentation`: API docs, guides, and runbooks
- `ux-designer`: Frontend UX, accessibility, and UI polish

## Why This Was Simplified

- Removed workflow-heavy agent sprawl that was hard to use consistently
- Kept only role-based agents mapped to common Observer tasks
- Reduced duplicate docs and confusing overlap between prompts and agents

## Recommended Usage

Use natural prompts first. Mention an agent only when you need specialized focus.

Examples:

- "Use developer to implement test run filtering in the API"
- "Use testing to add coverage for websocket reconnect behavior"
- "Use devops to diagnose this GitHub Actions failure"

## Maintenance Rules

- Prefer adding capability to an existing role agent before creating a new agent
- Add a new agent only if it has a distinct, repeated workflow
- Keep agent descriptions and tool scopes short and specific

## References

- GitHub custom agent docs: https://docs.github.com/en/copilot/reference/custom-agents-configuration

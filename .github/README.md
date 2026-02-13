# GitHub Configuration Directory

This directory contains GitHub-specific configuration files and workflows.

## Contents

### Workflows
- `docker-publish.yml` - Automated Docker image builds and publishing to GitHub Container Registry

### AI Agent Configurations (`agents/`)

This project uses GitHub Copilot and custom AI agents for development workflow automation. The agent configuration files in the `agents/` directory define specialized agents for:

- **Architecture** (`architect.agent.md`) - System design and architecture decisions
- **Development** (`developer.agent.md`) - Code implementation and feature development  
- **DevOps** (`devops.agent.md`) - Infrastructure, deployment, and CI/CD
- **Documentation** (`documentation.agent.md`) - Technical writing and documentation
- **Testing** (`testing.agent.md`) - Test strategy and test automation
- **UX Design** (`ux-designer.agent.md`) - UI/UX design and frontend development
- **SpecKit** agents - Project specification and planning workflow tools

These configuration files are part of our internal development process and provide context to AI tools. They do not affect the runtime behavior of the Observer service.

**Note for contributors**: These files document our development workflow and conventions. While you don't need to use these tools to contribute, they provide valuable context about the project's architecture and best practices.

### Development Instructions (`copilot-instructions.md`)

Comprehensive AI agent instructions that document:
- Service architecture and component boundaries
- Database integration patterns and safety rules
- NATS messaging patterns
- Testing strategies
- Build and deployment workflows

This file serves as both an AI agent reference and documentation for developers working on the project.

## For Contributors

If you're contributing to Observer, you don't need to interact with these agent configurations. Follow the standard contribution guidelines in [CONTRIBUTING.md](../CONTRIBUTING.md).

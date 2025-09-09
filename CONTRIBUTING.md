# Contributing to Mycelian

Welcome! This project embraces AI-assisted development with guardrails to keep quality high while moving fast.

## Project Status

This is early-stage software under active development. APIs will change, docs will evolve, and the focus is on core functionality. Architecture feedback and real-world use-cases are as valuable as code.

## Getting Started

### Fork and Clone

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/mycelian-memory.git
   cd mycelian-memory
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/mycelian/mycelian-memory.git
   ```

### Understanding the Project Structure

Before diving in, review our [monorepo documentation](docs/monorepo.md) to understand how the project is organized.

**Key components:**
- **Go modules**: Server, Client SDK, MCP server, and CLI tools
- **Python benchmarker**: Performance testing and evaluation tools
- **Docker stack**: Local development environment with Postgres and Weaviate

### Prerequisites and Dependencies

#### Required Software

**Go 1.24.6+**
```bash
# macOS
brew install go

# Ubuntu/Debian
wget https://go.dev/dl/go1.24.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version
```

**Docker Desktop**
- [Download Docker Desktop](https://www.docker.com/products/docker-desktop)
- Ensure Docker Compose is included (comes with Desktop)

**Ollama with embedding model**
```bash
# macOS
brew install ollama

# Ubuntu/Debian
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama and pull embedding model
ollama serve &
ollama pull nomic-embed-text
```

#### Optional Dependencies

**Python 3.11+** *(optional, for benchmarker and performance testing)*
```bash
# macOS
brew install python@3.11

# Ubuntu/Debian
sudo apt update
sudo apt install python3.11 python3.11-pip python3.11-venv

# Verify installation
python3 --version
```

> **Python Environment Management**: We recommend using `pyenv` to manage Python versions and `venv` for project isolation. If you're working on multiple Python projects, consider installing `pyenv` first (`brew install pyenv` or `curl https://pyenv.run | bash`), then `pyenv install 3.11.9` and `pyenv local 3.11.9` in this project. Our `./setup-env.sh` script will create a local `venv` regardless of how you installed Python, keeping dependencies isolated from your system Python.

**Make** *(required for build automation)*
```bash
# macOS (usually pre-installed, but if needed)
xcode-select --install
# or
brew install make

# Ubuntu/Debian (usually pre-installed, but if needed)
sudo apt install build-essential
```

#### Optional Development Tools

**jq** *(optional, helpful for JSON processing in examples and debugging)*
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt install jq
```

### Initial Setup and Verification

1. **Verify everything compiles**:
   ```bash
   make build-check
   ```

   **Optional**: Run full quality gate (formatting, linting, tests, vulnerability scan):
   ```bash
   make quality-check
   ```

2. **Set up Python virtual environment** *(optional, for benchmarker and performance testing)*:
   ```bash
   # This creates a venv, installs Python dependencies, and builds CLI tools
   source ./setup-env.sh
   ```

   **For future sessions**, activate the environment:
   ```bash
   source venv/bin/activate
   ```

3. **Run full test suite** to verify everything works:
   ```bash
   make test-full-local-stack
   ```

This command will:
- Run server unit tests
- Start the backend stack (Postgres, Weaviate, Memory Service)
- Wait for services to be healthy
- Run integration and e2e tests
- Clean up automatically

If all tests pass, you're ready to contribute!

## Branch and PR Workflow

### Creating Feature Branches

**Important**: Never develop features directly on the `main` branch. Always create feature branches from an up-to-date main.

```bash
# Keep your fork up to date
git fetch upstream
git checkout main
git pull --rebase upstream main

# Create a feature branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
# or
git checkout -b docs/update-name
```

### Submitting Pull Requests

1. **Push your branch** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a PR** from your fork to the main repository

3. **Link to issues** if applicable using "Fixes #123" or "Closes #456"

**Note**: We're building reliable CI automation - for now, ensure your changes pass local tests before submitting.

## Issue and Discussion Workflow

### When to Use Issues vs Discussions

**Use GitHub Issues for:**
- Bug reports with reproducible steps
- Feature requests with clear requirements
- Documentation improvements

**Use GitHub Discussions for:**
- Architecture questions and design conversations
- General questions about usage or implementation
- Brainstorming new features or approaches

### Linking PRs to Issues

When submitting a PR, reference related issues:
- `Fixes #123` - automatically closes the issue when PR merges
- `Addresses #456` - links to the issue without auto-closing
- `Related to #789` - general reference

## AI-Assisted Development Philosophy

AI-generated code is welcome, but it must pass human review and follow our four-layer quality method:

1. ADRs – capture architectural decisions
2. Specifications – define behavior before coding
3. TDD – write tests first, then implement
4. Invariants – validate system properties

## Before You Contribute

### For Major Changes

**Architectural decisions** require documentation first:
- **ADRs** for irreversible technical choices (database, transport, etc.)
- **Design docs** for complex features, APIs, or system changes
- **Specifications** for new protocols, data formats, or interfaces

**Process:**
1. Submit design documentation PR first
2. Get review and approval from maintainers
3. Begin implementation only after design is accepted

### Templates

- **ADR template**: `docs/adrs/000-adr-template.md`
- **Design doc template**: `docs/designs/design-template.md`

## Development Workflow

### Simple Development Loop

```bash
# Start everything and run tests
make test-full-local-stack

# That's it! This command:
# - Runs server unit tests
# - Starts backend stack
# - Waits for health
# - Runs integration tests
# - Cleans up automatically
```

**For iterative development:**
```bash
# 1. Start backend once
make start-dev-mycelian-server

# 2. Write tests, implement, repeat
make server-test

# 3. When done, clean up
make backend-down
```

### Working with Different Components

**Server-side changes:**
```bash
# Quick server-only tests
make server-test

# Server + e2e against running stack
make server-e2e
```

**Client SDK changes:**
```bash
# Client unit tests
make client-test

# Client integration tests (requires backend running)
make client-test-integration
```

**MCP server development:**
```bash
# Start MCP server for Cursor integration
make start-mcp-streamable-server

# Restart after changes
make mcp-streamable-restart
```

### Code Quality Standards

#### Size and Scope
- Keep PRs focused: ~150 lines when possible
- One feature or fix per PR
- Break large refactors into logical steps

#### Test-Driven Development (Required)
1. Write failing tests first – define expected behavior
2. Implement minimal code to make tests pass
3. Include both happy path and error cases
4. Target ~90%+ coverage (checked locally, goal in CI)

#### Invariant Testing
- Black-box invariant tests validate system properties
- Run locally:
```bash
go test ./tools/invariants-checker/...
```

## Using AI Assistants Effectively

Good prompts reference standards and invariants:
```
"Following the shardqueue spec, implement ..."
"Per ADR-0008, maintain FIFO ordering while ..."
"Write tests for a retry mechanism that backs off exponentially ..."
"Ensure this doesn't violate the data immutability invariant ..."
```

Context management:
- Attach relevant specs to your session
- Reference specific ADRs when making changes
- Include test outputs in problem descriptions
- Provide invariant definitions for validation

Critical review is required:
- AI-generated code must align with ADRs and specs
- Verify implementation matches behavior
- Ensure all tests pass (existing and new)
- Confirm no invariant violations

## What We're Looking For

### High-Priority Contributions
1. Architecture feedback – review ADRs, propose improvements, identify scaling risks
2. Real-world use cases – features, gaps, performance requirements
3. Test coverage – add missing cases, invariants, and performance benchmarks
4. Documentation – clarify specs, add examples, fix ambiguities

### Code Contributions
- Performance optimizations (with benchmarks)
- Bug fixes (with reproducing tests)
- New features (follow ADR → Spec → TDD → Invariants)

## Submitting Changes

### PR Checklist
- [ ] For major changes: design documentation submitted and approved
- [ ] Tests written and passing (unit + integration as applicable)
- [ ] Race detector clean: `go test -race ./...` (from `server/` and relevant modules)
- [ ] Quality gate: `make quality-check`
- [ ] Invariant tests passing: `go test ./tools/invariants-checker/...`
- [ ] Failing tests are fixed, not disabled
- [ ] Implementation aligns with relevant ADRs/specs

### PR Description Template
```markdown
## What
[Brief description of changes]

## Why
[Link to ADR or issue]

## How
[Implementation approach]

## Testing
- [ ] Unit tests added
- [ ] Integration tests updated
- [ ] Invariants validated

## AI Assistant Used
[Which tool and how it helped]
```

## Communication

### Getting Help
- Architecture questions → review existing ADRs first
- Implementation questions → check specifications
- Bug reports → include a failing test case
- Feature requests → open a discussion issue first

### Discussions
- GitHub Issues: bug reports and feature requests
- GitHub Discussions: architecture and design conversations
- PRs: code review and implementation feedback

## Quick Start Examples

### Adding a New Feature
```bash
# 1) Discuss and get ADR approved (if needed)
# 2) Write or update a spec
# 3) Write failing tests
make server-test  # should fail

# 4) Implement with AI assistance as needed

# 5) Validate
make server-test             # tests pass
cd server && go test -cover ./...
go test ./tools/invariants-checker/...  # no violations

# 6) Submit PR (<~150 lines)
```

### Fixing a Bug
```bash
# 1) Write a test reproducing the bug
# 2) Fix the implementation
# 3) Ensure invariants still hold
# 4) Submit PR with test + fix
```

## Resources
- **ADR template**: `docs/adrs/000-adr-template.md`
- **Design doc template**: `docs/designs/design-template.md`
- **Spec examples**: `docs/specs/` (shardqueue, healthchecker)
- **Architecture docs**: `docs/designs/` (client-sdk, mcp-server, etc.)

## Security
Please refer to `SECURITY.md`.

## Conventional Commits
Use conventional commit messages:
- feat(server): add health probe timeouts
- fix(client): correct base URL handling
- docs(contributing): clarify Quickstart and CI gates

## License
By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Remember: keep changes small, test-first, and optimize for clarity and reliability.

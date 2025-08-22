# Contributing to Mycelian

Welcome! This project embraces AI-assisted development with guardrails to keep quality high while moving fast.

## Project Status

This is early-stage software under active development. APIs will change, docs will evolve, and the focus is on core functionality. Architecture feedback and real-world use-cases are as valuable as code.

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

### Initial Setup

```bash
# Fork and clone the repository
git clone https://github.com/yourusername/mycelian-memory.git
cd mycelian-memory

# Optional: bootstrap local environment
./setup-env.sh

# Verify everything compiles
make build-check
```

### Simple Development Loop

```bash
# Start everything and run tests
make test-all-postgres

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
make backend-postgres-up

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
make mcp-streamable-up

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
- [ ] Formatting and static analysis: `go fmt ./... && go vet ./... && golangci-lint run`
- [ ] Vulnerability scan: `govulncheck ./...`
- [ ] Coverage target (~90%) checked locally
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
Report vulnerabilities via `SECURITY.md` in this repository. When running locally, prefer Docker Compose targets (e.g., `make -C server run-postgres`) instead of launching ad-hoc host binaries to avoid duplicate endpoints and environment drift.

## Conventional Commits
Use conventional commit messages:
- feat(server): add health probe timeouts
- fix(client): correct base URL handling
- docs(contributing): clarify Quickstart and CI gates

## License
By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Remember: keep changes small, test-first, and optimize for clarity and reliability.



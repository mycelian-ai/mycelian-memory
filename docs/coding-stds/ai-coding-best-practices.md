# AI-Assisted Development Best Practices

**Author**: Development Team
**Type**: Development Methodology Guide

_This document targets developers working with AI coding assistants. Written following "On Writing Well" principles: clear, concise, factual, and free of unnecessary complexity._

## Objective

Define a systematic approach for maintaining code quality, architectural coherence, and system reliability when developing with AI assistance. AI tools accelerate development but require structured guardrails to prevent technical debt and ensure long-term maintainability.

## Problem

AI-assisted development introduces unique challenges:

- **Rapid iteration** can outpace architectural thinking
- **Context loss** between sessions requires explicit knowledge capture
- **Code generation** may lack domain-specific best practices
- **Quality assurance** becomes more critical at higher velocities
- **Documentation drift** occurs when implementation moves faster than specs

## The Four-Layer Methodology

This project employs a four-layer approach that builds quality into the development process:

```
1. ADRs (Capture Decisions)
   ↓
2. Specifications (Define Behavior)
   ↓
3. TDD (Validate Implementation)
   ↓
4. Invariants (Guard System Properties)
   ↓
Continuous Feedback Loop
   ↑__________________________|
```

## Layer 1: Architecture Decision Records (ADRs)

### Purpose
ADRs capture the **why** behind architectural choices, providing context that survives developer turnover and AI session resets.

### Format
```markdown
---
title: [Decision Title]
status: [Proposed|Accepted|Deprecated|Superseded]
date: YYYY-MM-DD
superseded_by: [ADR number] (if applicable)
---

# Context
[What forces led to this decision?]

# Decision
[What we decided to do]

# Consequences
[Positive and negative outcomes]

# Alternatives Considered
[What else we evaluated and why we rejected it]
```

### When to Write ADRs
- **Before** implementing significant architectural changes
- When choosing between **competing approaches**
- When establishing **system-wide patterns**
- When **deprecating** or **superseding** previous decisions

### Examples in This Codebase
- `0001-client-sdk.md` - Decision to adopt dedicated Go SDK
- `0008-concurrency-model.md` - Sharded worker pool design
- `0019-await-consistency.md` - API naming decisions

## Layer 2: Specification-Driven Development

### Purpose
Specifications define **what** the system should do before implementation begins, providing clear contracts for both humans and AI assistants.

### Specification Types

#### Component Specifications
Document internal system components with:
- **Architecture overview** - high-level design
- **API contracts** - interfaces and data structures
- **Configuration** - tuneable parameters
- **Error handling** - failure modes and recovery
- **Performance characteristics** - throughput, latency, resource usage

#### Design Documents
Cover feature development with:
- **Objective** - what problem we're solving
- **Requirements** - functional and non-functional constraints
- **High-level design** - system interactions
- **Low-level design** - implementation details
- **Testing strategy** - verification approach

### Example: ShardQueue Specification
The `shardqueue.md` specification demonstrates this approach:
- Clear architecture section with code examples
- Comprehensive API documentation
- Configuration parameters with ranges
- Error handling patterns
- Performance characteristics
- Observable metrics

### Benefits for AI Development
- **Context preservation** across sessions
- **Implementation guidance** for AI assistants
- **Consistency checking** - does implementation match spec?
- **Onboarding efficiency** for new team members

## Layer 3: Test-Driven Development (TDD)

### The TDD Cycle
1. **Red** - Write failing test that captures desired behavior
2. **Green** - Write minimal code to make test pass
3. **Refactor** - Improve code while keeping tests green

### TDD with AI Assistants

#### Test-First Prompting
Instead of: *"Implement a retry mechanism"*
Use: *"Write tests for a retry mechanism that backs off exponentially and stops after 3 attempts, then implement it"*

#### Incremental Development
- Start with **simple test cases**
- Add **edge cases** progressively
- Let tests **guide API design**
- Use tests to **validate AI-generated code**

### Test Categories in This Codebase

#### Unit Tests
- **Single component** isolation
- **Fast execution** (< 100ms each)
- **No external dependencies**
- Example: `shardqueue_test.go`

#### Integration Tests
- **Component interaction** validation
- **Real dependencies** (databases, services)
- **Separate test packages** for organization
- Example: `client/integration_test/`

#### End-to-End Tests
- **Full system behavior** validation
- **Production-like environment**
- **Business scenario coverage**
- Example: `server/dev_env_e2e_tests/`

## Layer 4: Invariant-Driven Final Guardrails

### Purpose
Invariants define **fundamental system properties** that must never be violated, regardless of implementation details. They serve as the final safety net in AI-assisted development.

### System Invariants Categories

#### Data Integrity
- **Immutability**: Entries never change after creation
- **Consistency**: Related data remains synchronized
- **Atomicity**: Operations complete fully or not at all

#### Security & Isolation
- **Access control**: Users only see their own data
- **Data boundaries**: Strict tenant isolation
- **Audit trails**: All changes are logged

#### Performance & Reliability
- **Response times**: APIs respond within SLA limits
- **Resource usage**: Memory and CPU stay within bounds
- **Availability**: System handles expected load

#### Business Logic
- **Ordering guarantees**: Per-memory FIFO semantics
- **State transitions**: Valid workflow progressions only
- **Data retention**: Soft deletes preserved as configured

### InvariantChecker Pattern

This codebase implements a black-box testing harness that validates invariants:

```go
type InvariantChecker struct {
    client *http.Client
    baseURL string
}

func (c *InvariantChecker) AssertDataImmutability(memoryID string) error {
    // 1. Create entry
    entry := c.AddEntry(memoryID, "test content")

    // 2. Attempt modification (should fail)
    err := c.ModifyEntry(entry.ID, "changed content")
    if err == nil {
        return errors.New("invariant violation: entry was modified")
    }

    // 3. Verify original content unchanged
    retrieved := c.GetEntry(entry.ID)
    if retrieved.Content != "test content" {
        return errors.New("invariant violation: content corrupted")
    }

    return nil
}
```

### Invariant Testing Workflow
1. **Define invariants** before implementing features
2. **Write invariant tests** as black-box checks
3. **Run continuously** in CI/CD pipeline
4. **Block deployment** if invariants fail

## Methodology Integration

### Development Workflow

#### 1. Decision Phase (ADR)
- **Identify** architectural decision point
- **Research** alternatives and trade-offs
- **Document** decision with context and consequences
- **Review** with team before implementation

#### 2. Specification Phase
- **Define** component behavior and contracts
- **Document** API surfaces and error conditions
- **Specify** performance and reliability requirements
- **Create** examples and usage patterns

#### 3. Implementation Phase (TDD)
- **Write failing tests** for each specification requirement
- **Generate code** with AI assistance to pass tests
- **Refactor** for clarity and performance
- **Validate** against specification

#### 4. Validation Phase (Invariants)
- **Run invariant tests** against implementation
- **Verify** no system properties violated
- **Test edge cases** and failure scenarios
- **Confirm** SLA compliance

### AI Assistant Integration

#### Prompt Engineering
- **Reference specifications** in prompts: *"Following the shardqueue spec..."*
- **Include test cases**: *"Make this test pass..."*
- **Cite ADRs**: *"Per ADR-0008, maintain FIFO ordering..."*
- **Request invariant checks**: *"Ensure this doesn't violate data immutability..."*

#### Context Management
- **Attach relevant specs** to coding sessions
- **Reference ADR decisions** when making changes
- **Include test outputs** in problem descriptions
- **Provide invariant definitions** for validation

#### Quality Gates
- **Code must pass** all existing tests
- **New features require** specification updates
- **Breaking changes need** ADR documentation
- **Invariant violations** block merge

## Continuous Feedback Loop

### Retrospective Questions
- Do our **ADRs** accurately reflect current architecture?
- Are **specifications** complete and up-to-date?
- Do **tests** cover the critical paths?
- Are **invariants** detecting real issues?

### Improvement Triggers
- **Failed invariants** → Review specification accuracy
- **Difficult testing** → Consider API design changes
- **Unclear decisions** → Write missing ADRs
- **Repeated bugs** → Add new invariant checks

## Implementation Guidelines

### For New Features
1. **Write ADR** if introducing new patterns
2. **Create specification** defining behavior
3. **Implement TDD** starting with tests
4. **Validate invariants** throughout development

### For Bug Fixes
1. **Add failing test** reproducing the issue
2. **Fix implementation** to pass test
3. **Check invariants** weren't violated
4. **Update specs** if behavior changed

### For Refactoring
1. **Ensure test coverage** before changes
2. **Maintain API contracts** per specifications
3. **Preserve invariants** throughout process
4. **Update documentation** if patterns change

## Benefits

### Development Velocity
- **Faster onboarding** with clear documentation
- **Reduced context switching** between sessions
- **Systematic debugging** with layered validation
- **Confident refactoring** with safety nets

### Code Quality
- **Architectural consistency** through ADRs
- **Interface clarity** through specifications
- **Behavioral correctness** through TDD
- **System reliability** through invariants

### Team Collaboration
- **Shared mental models** via documentation
- **Clear decision history** in ADRs
- **Testable contracts** in specifications
- **Objective quality measures** in invariants

## References

- **ADR Examples**: `docs-archive/client/adr/` and `docs-archive/server/adr/`
- **Specification Example**: `docs/specs/shardqueue.md`
- **Invariant Testing**: Server ADR 0005 - Invariant-Driven Development
- **Design Template**: `docs/design/design-template.md`

---

*This methodology has been successfully applied in the development of the Mycelian Memory system, enabling rapid iteration while maintaining system quality and architectural coherence.*

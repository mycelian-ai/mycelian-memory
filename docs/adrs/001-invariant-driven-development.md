# ADR-001: Invariant-Driven Development & Black-Box Testing

**Status**: Accepted
**Date**: 2025-07-20

## Context

System correctness is defined by high-level business invariants (immutability, isolation, consistency, etc.) rather than internal method behaviour. A reusable black-box test harness exercises the public API to assert these rules and catch regressions.

## Decision

1. **Primary correctness contract**: Four invariant categories documented in system invariants
2. **Black-box tests**: Integration tests use REST endpoints against running stack to validate invariants end-to-end
3. **Development cadence**: New features begin with invariant test, followed by implementation until test passes

## Consequences

### Positive Consequences
- Implementation can refactor freely as long as invariants stay green
- High CI confidence with minimal unit-test maintenance overhead
- Clear correctness criteria for all stakeholders
- Catches integration issues that unit tests miss

### Negative Consequences
- Requires running stack for test execution (slower than pure unit tests)
- Test failures can be harder to debug than focused unit tests
- Requires REST API even if other transports are primary

## Implementation Notes

### Invariant Categories
- **Data Integrity**: Immutability and consistency rules
- **Access Control**: Authorization and isolation guarantees
- **Operational**: Performance and availability requirements
- **Business Logic**: Domain-specific rules and constraints

### Test Structure
- Tests run against live service instance
- Use REST API as primary test interface
- Assert invariants hold under various operation sequences
- Include both positive and negative test cases

### Development Workflow
1. Define invariant for new feature
2. Write black-box test asserting invariant
3. Implement feature until test passes
4. Refactor implementation while keeping invariant green

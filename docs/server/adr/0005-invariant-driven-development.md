---
title: Invariant-Driven Development & Black-Box Test Harness
status: Accepted
date: 2025-07-20
---

# Context

Correctness in Synapse is defined by high-level business invariants (immutability, isolation, soft-delete, etc.) rather than by internal method behaviour.  A reusable black-box test harness (`InvariantChecker`) exercises the public API to assert these rules.

# Decision

1. **Primary correctness contract** = four invariant categories documented in `docs/system-invariants.md`.  
2. **Black-box tests**: Integration tests use REST endpoints against a running stack (emulator) to validate invariants end-to-end.  
3. **Development cadence**: New features begin with an invariant test, followed by implementation until the test passes.

# Consequences

• Implementation can refactor freely as long as invariants stay green.  
• REST remains required even after gRPC ships because it is the easiest surface for invariant tests.  
• CI confidence is high with minimal unit-test maintenance overhead. 
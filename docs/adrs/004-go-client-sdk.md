# ADR-004: Go Client SDK Adoption

**Status**: Accepted
**Date**: 2025-07-24

## Context

Legacy client code duplicated logic, lacked retry/shard support, and made handler code verbose. A dedicated client SDK was needed to consolidate memory service interactions.

## Decision

Implement a dedicated Go SDK in `client/` and migrate all MCP handlers and tools to use it.

## Consequences

### Positive Consequences
- Shared retry and sharding logic at one layer
- Cleaner handler code across applications
- Centralized error handling and logging

### Negative Consequences
- Must delete legacy client code after full migration
- Additional abstraction layer to maintain

## Implementation Notes

- SDK provides idiomatic Go API with context support
- Includes automatic retry logic with exponential backoff
- Maintains connection pooling and request routing
- Consolidates all memory service interactions through single interface

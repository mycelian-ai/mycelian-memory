# ADR-008: Await Consistency Primitive

**Status**: Accepted
**Date**: 2025-07-28

## Context

The system's local-first architecture writes immediately to local storage, then replicates to backend asynchronously. Early API naming used `await_commit` which proved confusing because:

- In local-first architecture, writes land immediately in local store; backend performs replication phase
- Term "commit" overlapped with database terminology and Git conventions
- Real purpose is providing **global read-your-write consistency** by blocking until replication completes

## Decision

Rename the primitive to `await_consistency` across all interfaces:

1. MCP tool (`await_consistency <memory-id>`)
2. Go SDK (`client.AwaitConsistency(ctx, memoryID, opts...)`)
3. CLI aliases (`await` maps to the same call for convenience)

Temporary alias `await_commit` kept for one release as deprecation shim with warning log.

## Consequences

### Positive Consequences
- Clear semantic meaning: waiting for consistency, not transaction commit
- Eliminates confusion with database and Git terminology
- Better describes the actual behavior to developers

### Negative Consequences
- API naming change requires documentation updates
- Temporary complexity during deprecation period
- Existing tooling needs updates to use new naming

## Implementation Notes

### Consistency Semantics
- Blocks until all prior writes for the memory are durably committed in the primary store
- Provides read-your-write consistency for the primary store only
- Does not wait for propagation to secondary systems (e.g., search/index via outbox)
- Essential for workflows that need strong read-after-write on the primary store

### Migration Strategy
- Maintain backward compatibility during transition period
- Log deprecation warnings for old naming
- Update all documentation to reference new naming

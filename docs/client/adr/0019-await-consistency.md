---
title: Rename await_commit to await_consistency
status: accepted
ADR: 0019
date: 2025-07-28
---
# Context

Early iterations of the SDK and MCP tools exposed a primitive named `await_commit`.  The term "commit" proved confusing because:

* In the local-first architecture, writes land immediately in the on-device WAL; the MCP backend performs the *second* phase (cloud replication).  The word *commit* therefore implied a two-phase commit which is not accurate.
* `commit` overlapped with database terminology (transaction commit) and Git conventions, leading to ambiguous documentation searches.

The primitive's real purpose is to give **global read-your-write consistency** by blocking until all prior writes for the memory are durably replicated in the cloud backend.

# Decision

Rename the primitive to `await_consistency` across:

1. MCP tool (`synapse await_consistency <memory-id>`)
2. Go SDK (`client.AwaitConsistency(ctx, memoryID, opts...)`)
3. HTTP API (`POST /v1/memories/{id}:await_consistency`)
4. CLI aliases (`synapse await` still maps to the same call for convenience)

A temporary alias `await_commit` will be kept **one minor release** as a deprecation shim that forwards to `await_consistency` and logs a warning.

# Consequences

* All design docs must reference `await_consistency` when describing strong consistency semantics.
* The term "commit" will remain only in internal replication logs, not in public APIs.
* Historical ADRs (≤ 0018) preserve the original wording for provenance; a banner will point readers to this ADR. 
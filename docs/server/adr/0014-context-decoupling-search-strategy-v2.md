---
description: Context Decoupling & Search Strategy v2
status: accepted
supercedes: "0013-context-search-strategy.md"
alwaysApply: false
---
# ADR-0014 – Context Decoupling & MemoryContext Search Strategy v2

## Status
✅ Accepted – 2025-07-23

## Context
Phase 1-4 introduced a dedicated `MemoryContexts` Spanner table and Waviate `MemoryContext` class, removing the embedded `context` JSON from `MemoryEntries`.  ADR-0013 captured an initial design but kept context retrieval coupled to entries.  Subsequent validation showed:

* Context payloads grow independently from entries and need separate retention policies.
* Entry-coupled context forced duplicate storage in Waviate and increased write amplification.
* Query relevance suffers when context and entry content compete inside the same vector space.

## Decision
1. **Context snapshots are first-class resources** stored in `MemoryContexts` (Spanner) & `MemoryContext` (Waviate).
2. **Indexer** scans both entries & contexts with independent watermarks; context batch size 1 guarantees freshness.
3. **Searcher API** exposes two context flavours:
   * `latestContext` – most recent snapshot (temporal relevance).
   * `bestContext` – hybrid search over `MemoryContext` with the same query vector (semantic relevance).
4. `/api/search` response schema updated (v3) to include `bestContext`, `bestContextTimestamp`, `bestContextScore` while maintaining v2 for backward compatibility.
5. ADR-0013 is marked ❌ *Superseded*.

## Consequences
* **Read Path** complexity increases slightly (two Waviate calls) but latencies remain within SLO (p95 < 200 ms, measured on dev stack).
* Storage costs decrease ~12 % due to eliminating duplicate context embeddings.
* Clients may migrate to v3 at their own pace; v2 remains stable.

## Alternatives Considered
* Keep context in entries – rejected due to write amplification.
* Serve latest context only – rejected; hybrid relevance adds >18 % precision@5 in A/B tests.

## Migration
No data migration required; future contexts will be stored separately. Existing `context` JSON in historical entries is preserved but ignored. 
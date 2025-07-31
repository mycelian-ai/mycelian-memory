---
title: Vector Search Integration Strategy
status: Superseded
superseded_by: "0017-vector-store-abstraction.md"
date: 2025-07-21
---

# Context

Semantic retrieval requires vector similarity search beyond Spanner's capabilities.  The design envisions an asynchronous `indexer` that projects memory entries into a managed vector store (e.g., Vertex AI Vector Search).

# Decision

1. **Indexer Service** subscribes to Spanner commit-stream or table change feed and writes embeddings + metadata to the vector store.  
2. Writes are **eventually consistent**; read APIs expose a `searchStaleness` field so callers know lag risk.  
3. Embedding model is versioned; index namespace includes model version to allow re-index migrations.

# Consequences

• Core write path remains low-latency; vector indexing is off the critical path.  
• Search API can evolve independently of transactional store.  
• Re-embedding with a new model is a background bulk job that writes to a new namespace, allowing zero-downtime cutover. 
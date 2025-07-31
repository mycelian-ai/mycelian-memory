---
title: Latest-Context Search Strategy (v1)
status: Accepted
date: 2025-07-22
---

# Context

The `context` field is currently duplicated in **every** `MemoryEntry` row.  When BM25 runs over this field the same tokens appear in every entry of a memory, reducing precision.

A full redesign (versioned `MemoryContext` class) is desirable but adds schema and ingestion complexity.  As an interim solution we want to:

* Continue storing context inside `MemoryEntry` only.
* Prevent it from dominating entry search.
* Still expose the *latest* context snapshot to the LLM so it can reason over memory-level background.

# Decision

1. **Hybrid search property filter** – All BM25 / hybrid queries must specify `properties:["summary","rawEntry","tags","metadata"]`, thereby excluding the `context` field from the sparse index.
2. **Return latest context snapshot** – After retrieving the top-k entries the API performs a lightweight SQL query:
   ```sql
   SELECT Context, CreationTime
   FROM   MemoryEntries
   WHERE  UserId=? AND MemoryId=? AND DeletionScheduledTime IS NULL
   ORDER  BY CreationTime DESC
   LIMIT  1;
   ```
   The decoded (and compacted) string is returned alongside the entry list.
3. **Response shape**
   ```json
   {
     "entries": [ {"entryId":"…","summary":"…","_score":0.94}, … ],
     "latestContext": "<decoded string>",
     "contextTimestamp": "2025-06-27T11:42:03Z"
   }
   ```
4. **Schema unchanged** – No new class/table is added; no backfill required.
5. **Benchmark gate** – If search benchmarks show insufficient precision/recall this ADR will be superseded by a follow-up ADR introducing a dedicated `MemoryContextVersion` class.

# Consequences

• Higher precision: duplicated context no longer fires the BM25 scorer.
• Recall unchanged: dense side still uses entry summary vectors.
• Minimal code delta: mainly query builder and one extra `SELECT`.
• Forward compatible with a future versioned-context design.

# Implementation Notes

* Query helper `executeHybridSearch` in `indexer_e2e_test.go` will be updated to pass `WithProperties([...])`.
* REST & CLI search endpoints adopt the new response JSON.
* Optional hardening: set `IndexSearchable:false` on the `context` property when the schema is recreated, making the filter unnecessary.

# Status

Accepted – Pending implementation in branch `feat/search-api` and initial benchmark run. 
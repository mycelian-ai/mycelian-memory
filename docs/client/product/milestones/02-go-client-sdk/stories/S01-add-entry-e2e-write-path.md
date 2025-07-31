---
id: S01
status: ⏳ in-progress
---
# Story S01 – AddEntry End-to-End Write Path (SDK → CLI → MCP)

Milestone: **02 – Go Client SDK & Concurrency**  
Last updated: 2025-06-23

## Problem / Value
Users need a fast, reliable way to write new memory entries from the Go SDK, CLI, and MCP tools. The AddEntry "steel-thread" demonstrates the entire pipeline – SDK → CLI → MCP → **Sharded Queue** → Cloud-Run + Spanner replication – and proves the new three-class concurrency model & header strategy in production-like flows (ADR-0020).

## Scope
* Go SDK `AddEntry()` client method (completed)
* Synapse CLI `create-entry` command (completed)
* MCP `add_entry` tool + handler (completed)
* Unit & integration tests (current focus)
* Documentation updates, ADR-0017 linkage

## Task Breakdown

| Order | Title                              | Brief Description                                                                                                                  | Status        |
| ----- | ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| 1     | SDK AddEntry unit tests            | Add `add_entry_test.go` with table-driven cases (201, 500, ctx cancel)                                                             | ✅             |
| 2     | ShardExecutor (SQ)                 | Implement `internal/shardqueue` package `ShardExecutor` with worker goroutines, per-memory FIFO, back-pressure, Prometheus metrics | ✅             |
| 3     | Context API – SDK layer            | Implement `PutContext` / `GetContext`, atomic JSON file writes, auto-init empty file                                               | ✅             |
| 4     | Context API – MCP tools            | Register `put_context` / `get_context` tools and handler; integrate with existing Client SDK                                       | ✅             |
| 5     | AddEntry embeds context snapshot   | Modify shard-worker job to read `context.json` and inject into `AddEntryRequest.Context`                                           | ✅             |
| 6     | Context round-trip integration     | Live test: put_context → await_consistency → add_entry → list_entries verify context field                                         | ✅             |
| 7     | Docs & ADR updates                 | Update ADR-0025, API reference, Memory Bank; ensure story docs reflect new state machine                                           | ✅             |
| 8     | Consistency options plumbing       | Add `Consistent=true` opt to `GetEntry`/`ListEntries`; ensure they enqueue through SQ                                              | ✅             |
| 9     | MCP prompt tools                   | Expose `get_summary_prompt` & `get_context_prompt` as read-only tools so remote agents can fetch canonical prompts                 | ✅             |
| 10    | MCP search tool & SDK helper       | Implement `search_entries` tool plus fallback substring search handler; add `Search` helper in Go SDK                              | ⏳ in-progress |
| 11    | DMR benchmark harness              | Run Deep-Memory-Retrieval ingestion + probe flow via MCP tools; collect accuracy / ROUGE metrics                                   | 🔜            |
| 12    | Benchmark results & docs           | Capture baseline numbers, update ADR-0020 appendix, add LongMemEval follow-up story link                                           | 🔜            |
| 13    | Header generation helpers          | Middleware to attach Idempotency-Key, Request-Id, traceparent to HTTP requests; unit tests                                         | 🔜            |
| 14    | CLI integration test: create-entry | Extend `cli_integration_test.go` to cover `create-entry`, `await_consistency`, and list verification                               | 🔜            |


## Definition of Done (DoD)
- [x] SDK method returns Entry with ID & timestamps
- [x] CLI command prints entryId
- [x] MCP tool registered & documented
- [ ] All tasks in table marked ✅
- [ ] CI green (`go vet`, `golangci-lint`, `go test -race`)
- [ ] Roadmap & progress files updated

## References
* `docs/design/client_concurrency_model.md`
* ADR-0020 Client Concurrency Model
* ADR-0017 Unified Identifier Strategy 
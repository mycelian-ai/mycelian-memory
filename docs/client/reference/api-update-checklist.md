# Backend API Update Checklist – June 2025 Design Alignment

This living checklist enumerates the changes required in the Memory Backend REST & MCP tool layers to align with the latest design and ADR decisions.

> Source of truth for changes: ADR-0011, 0012, 0013, 0014 + design docs.

| # | Area | Change | Priority | Notes |
|---|------|--------|----------|-------|
| 1 | **Entry Retrieval** | **REMOVE** `GET /users/{u}/memories/{m}/entries/top-k` (or `get_top_k` MCP tool) | High | Superseded by `GET /entries?limit=K` and `list_entries` tool (ADR-0013) |
| 2 | **Entry Retrieval** | Ensure `GET /entries` supports `limit` (≤50) and optional `before` / `after` RFC3339Nano cursors | High | Already in docs; verify implementation & tests |
| 3 | **Inflight APIs** | **DO NOT expose** `list_inflight_entries` / `get_inflight_entry` in public OpenAPI; gate behind `X-Synapse-Labs: inflight` header | Medium | Experimental per ADR-0014; metrics required |
| 4 | **Naming** | Replace residual "message" nomenclature with "entry" across Swagger descriptions, examples, and error payloads | Medium | Search & replace + review |
| 5 | **Error Codes** | Add `already_committed` 404 error for `get_inflight_entry` when row synced | Medium | Only inside Labs flag |
| 6 | **Health Docs** | Confirm `/api/health/sqlite` removed (mirror is internal) – keep `/health` & `/health/spanner` | Low | Reflects mirror internals hidden from clients |
| 7 | **Rate Limits** | Document and enforce per-user write QPS if decided (not yet in ADRs) | Low | TBD based on perf tests |
| 8 | **Idempotency** | Accept `Idempotency-Key` header on **all mutating endpoints**; deduplicate for 24 h; echo back in response | High | Per ADR-0015 – SDK auto-generates only if caller omits; agents encouraged to supply their own |
| 9 | **Request Tracing** | Generate/propagate `Request-Id` (or honour incoming) per HTTP attempt; echo in response | Medium | ADR-0015, ADR-0016 |
| 10 | **Payload Checksums** | For `
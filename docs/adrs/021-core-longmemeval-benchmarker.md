# ADR-021: Core LongMemEval Benchmarker (clean runner, context-only default, two-pass QA)

**Status**: Accepted
**Date**: 2025-09-08
**Supersedes**: Partially amends ADR-015; implements ADR-018 in runner scope

## Context

The LongMemEval benchmarker evolved from earlier LangGraph-based prototypes (ADR-013/014 → ADR-015). We now have a consolidated, production-ready runner under `longmemeval-benchmarker/` that:
- Uses a clean agent implementation with MCP tools to interact with Mycelian Memory
- Defaults to context-only ingestion for performance
- Adds a two-pass search strategy to improve QA retrieval
- Integrates with an orchestrator for resumability (ADR-022), while still supporting direct CLI runs

This ADR records the present, implemented decisions for the core runner and clarifies deviations from ADR-015 and details proposed in ADR-018.

## Decision

1. Runner entrypoint and responsibilities
   - The canonical runner is `src/benchmarker.py`.
   - Responsibilities: parse TOML config, run model healthchecks, load dataset, resolve vault, prepare output directories, delegate per-question execution to `SingleQuestionRunner`, and write `hypotheses.jsonl` in QA/full modes.
   - Non-responsibilities: per-message handling (owned by `SingleQuestionRunner`), orchestration (owned by Huey orchestrator per ADR-022).

2. Agent and ingestion mode
   - Use `MycelianMemoryAgent` built via `mycelian_memory_agent.build.build_agent_with_invoker`.
   - Default ingestion mode is context-only: the agent synthesizes and persists session-level context without per-message `add_entry` calls (see `CONTEXT_ONLY_AGENT.md`).
   - Sessions are processed sequentially per question to preserve chronology. The agent is reset across sessions; Mycelian is the durable store.

3. QA retrieval: two-pass search (default on)
   - `SingleQuestionRunner` implements a two-pass search (`_two_pass_search`) that:
     - Pass 1: hybrid search with broader `top_ke/top_kc` limits
     - LLM analysis to decide if refinement is needed
     - Pass 2: optional refined query with focused limits
   - Fallback: single-pass search using configured `top_k` (legacy) when two-pass is disabled.

4. Config and CLI interface
   - Config is TOML (`config.example.toml`). Required: `dataset_file_path`, `vault_title`, `[models].ingest`, `[models].qa`.
   - CLI (`src/benchmarker.py`):
     - `--config`, `--run-id`, `--workers` (for local WorkerManager only), `--mode [ingestion|qa|full]`.
     - Orchestrator integration flags: `--question-id`, `--start-session`, `--memory-id`, `--vault-id`.
   - The orchestrator (ADR-022) is the authoritative way to run at scale. Local `--workers` is for developer convenience only.

5. Progress and identifiers
   - The runner writes fine-grained progress (message and session counters, `vault_id`, `memory_id`) directly into `data/progress.db` for observability and safe resume.
   - `await_consistency` is invoked prior to QA search to ensure storage durability.

6. Memory naming
   - Standardize memory title template in runner config as `{question_id}__{run_id}` (double underscore separator). This is the preferred and documented format.
   - Orchestrator components MUST normalize to the same template when creating memories for consistency (see Note in Consequences).

## Implementation Details

- Runner: `longmemeval-benchmarker/src/benchmarker.py`
  - Loads TOML, runs model healthchecks, resolves vault with `MemoryManager`, delegates to `SingleQuestionRunner`.
- Single-question execution: `longmemeval-benchmarker/src/single_question_runner.py`
  - Builds the agent/invoker, runs sequential sessions per question, writes incremental progress to `data/progress.db` (ingested message counts, completed sessions, `vault_id`, `memory_id`).
  - Two-pass QA search and context assembly prior to answer generation.
- Dataset loader: `longmemeval-benchmarker/src/dataset_loader.py`
  - Normalizes input formats (`sessions` vs `haystack_sessions`).
- Memory utilities: `longmemeval-benchmarker/src/memory_manager.py`
  - Vault creation/lookup, memory creation (via MCP `create_memory_in_vault`), and `search_memories` invocation.
- Context-only agent notes: `longmemeval-benchmarker/CONTEXT_ONLY_AGENT.md`.

## Consequences

Positive:
- Faster ingestion by default via context-only updates; large reduction in MCP calls and DB writes.
- Better QA via two-pass search that recovers scattered facts with controlled expansion.
- Deterministic per-question logs and resumable progress written to SQLite.

Neutral/Tradeoffs:
- Context-only mode reduces per-message auditability. Most LongMemEval questions are unaffected; a minority requiring granular counts may regress slightly.
- Local `--workers` concurrency remains supported for convenience, but the orchestrator is the recommended path for scale.

Notes / Deviations to reconcile:
- Memory title template mismatch: some orchestrator code paths use `{run_id}_{question_id}`. Direction: converge on `{question_id}__{run_id}` everywhere.
- Legacy `top_k` mapping in `MemoryManager.search_memories` still caps `top_ke/top_kc` to old limits when `top_k` is used. Direction: prefer explicit `top_ke/top_kc` (see ADR-020) and remove legacy capping in a follow-up change.

## Alternatives Considered

- Keep per-message entries with periodic context: higher fidelity but significantly slower for benchmark throughput; rejected as default but still supported when `context_only=false`.
- Single-pass QA search only: simpler but lower recall for multi-fragment answers; two-pass offers a better balance with bounded extra tokens.

## References

- ADR-015: Minimal LangGraph-based LongMemEval Benchmarker
- ADR-018: LongMemEval Resumable Benchmarker (progress DB design)
- ADR-017, ADR-020: Search API parameter naming and range expansion
- Runner and components under `longmemeval-benchmarker/src/`

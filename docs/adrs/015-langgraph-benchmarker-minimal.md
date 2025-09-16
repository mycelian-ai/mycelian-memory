# ADR-015: Minimal LangGraph-based LongMemEval Benchmarker

**Status**: Accepted
**Date**: 2025-08-28
**Supersedes**: ADR-013, ADR-014

## Context
We need a simple, reproducible benchmarker for evaluating Mycelian’s long-memory behavior on LongMemEval-style datasets. Earlier harnesses grew complex with multiple custom agents and orchestration layers. We want to standardize on LangGraph’s prebuilt agent, keep the agent fully tool-enabled for ingestion, and drive QA/Eval from a small CLI runner. Mycelian provides durable memory (entries + contexts); the agent should remain stateless per session and use prompts to decide captures.

## Decision
- Use LangGraph’s prebuilt agent with a dynamic prompt; no custom action nodes.
- Keep the agent stateless per session (no checkpointer). Reset between sessions and bootstrap via `get_context` + `list_entries`.
- Expose Mycelian MCP tools directly to the agent: `add_entry`, `put_context`, `get_context`, `list_entries`, `await_consistency`, `search_memories`.
- Control runs via a TOML config (provider, models for agent/qa/eval, dataset path, vault title/ID, memory title template, top_k, safety caps).
- Orchestrate via a single CLI (`mycelian-longmemeval`) with three modes: ingestion, qa, eval. Default execution is per-question end-to-end.
- Split code into `dataset_loader`, `graph/agent builder`, and `runner/controller` modules under `longmemeval-benchmarker/`.

## Consequences
- Simpler implementation: minimal code, faster iteration, easier debugging.
- Deterministic runs: per-session resets, logged tool calls, fixed temperatures. Mycelian remains the durable source of truth.
- Extensible: swap QA/Eval models via TOML without changing orchestration. Optional parallelism can be added later.
- Tradeoff: No per-agent checkpointer means the runner must stream messages turn-by-turn and reset between sessions, which is intentional to reflect how Mycelian provides cross-session continuity.

## References
- Design: `docs/designs/langgraph_longmemeval_benchmarker.md`
- LangGraph prebuilt agent (memory & custom prompt): https://langchain-ai.github.io/langgraph/agents/agents/#5-add-memory

# ADR-0027 – Benchmark Harness (Anthropic-first)

Status: ✅ accepted  
Date: 2025-07-30

## Context

Early benchmarks (ADR-0021 DMR execution) verified the **read/​write pipeline** but still relied on an LLM agent running inside Cursor.  We now want an *automation-friendly* harness that we can run offline and compare Synapse against MemGPT and Zep baselines.

Key observations:
1. Synapse relies on **collaborative memory formation** – the LLM must call `mcp_synapse-memory_add_entry` and `update_context` while chatting.
2. Anthropic’s Messages API supports **tool calling** and has an official Python SDK.
3. Hugging-Face benchmark datasets (DMR, LongMemEval, LocoMoco) ship loader utilities in Python.
4. The Go codebase remains the source for server + SDK; we only need HTTP calls from the harness.

## Decision

1. **Language & location** – Create `benchmarks/python/` containing a Python 3.11 harness.  Keep it inside the mono-repo for traceability.
2. **Model support (v1)** – Target Anthropic Claude-3.5 Sonnet exclusively.  Future models will be pluggable via a small strategy interface.
3. **Tool schema** – Expose two MCP tools to the LLM on every call:
   * `mcp_synapse-memory_add_entry` – persist raw entry + summary.
   * `update_context` – write the context document.
4. **Ingestion flow**
   1. Cold boot / reconnect: read `get_context` + top-25 recent entries; build system prompt with
      `context_summary_rules.md`, `context_prompt.md`, context doc, recent entries.
   2. Replay user turns one-by-one; keep a running `messages` list so Claude retains in-session memory.
   3. Execute any emitted tool calls immediately against the Synapse HTTP API.
   4. Maintain a sliding window & running summary to bound prompt size.
5. **Evaluation flow** – After ingestion, use Synapse `search_memories` + Claude judge prompt to score answers; aggregate metrics identical to prior harness.
6. **Dependencies** – Track in `benchmarks/python/requirements.txt` (anthropic>=0.24, tqdm, datasets, pandas…).
7. **CI** – Add a lightweight `benchmark-smoke` job that stubs the Anthropic client and validates harness logic; full runs remain manual.

## Consequences

• Rapid prototyping: Python gives immediate access to Anthropic SDK and HF datasets.  
• Go server + SDK remain untouched; benchmark stays a thin HTTP client.  
• Model-specific behaviour (tool calling) captured in one place; adding OpenAI or Mistral later requires only a new strategy module.  
• CI complexity minimal – harness has its own venv and does not affect Go builds.

## Alternatives Considered

| Option | Notes | Why rejected |
|--------|-------|--------------|
| Port harness to Go | Single-language repo | No Anthropic SDK; high effort for proto + HF loaders |
| External repo | Clean separation | Loses change-tracking with server/SDK; harder to coordinate PRs |
| Reuse existing MemGPT harness | Already in Python | Does not exercise Synapse tool semantics; heavy dependency tree |

## Follow-up Work
1. Implement tasks captured in the TODO list (`task_client_extend` … `task_tests`).
2. Schedule weekly benchmark runs and track results in `docs/benchmarks/results.md`.
3. Evaluate adding OpenAI GPT-4o or Mistral once tool-calling parity is confirmed. 
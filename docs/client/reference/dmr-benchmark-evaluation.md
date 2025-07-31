> **DEPRECATION NOTICE (2025-07-07)**  
> This operator guide is **deprecated**. Synapse now uses the unified *Benchmark Harness* (`docs/design/benchmark_harness.md`) as the **authoritative** tool for evaluating long-term memory across all benchmarks (DMR, MSC-Self-Instruct, and future suites). The interactive "BEGIN DMR BENCHMARK" chat flow is kept only for ad-hoc smoke tests and will be removed in a future release.

# DMR Benchmark – Operator Guide

**Doc status:** draft (living reference)

---

## Purpose
This document explains *how* to reproduce the Deep-Memory-Retrieval (DMR) benchmark results for Synapse without writing or running any stand-alone scripts.  The entire flow is driven by an LLM agent connected via the MCP stdio server.

It is intended for developers who already have:

* a running `synapse-mcp-server` (built from the repo)  
* the MSC benchmark data checked into the workspace under `bench/dmr/`  
* a valid `OPENAI_API_KEY` (or Anthropic key) exported

## High-level execution model

1. **Agent as orchestrator** – The evaluator is the very same LLM that talks to the MCP server.  It calls tools (`create_user`, `create_memory`, `add_entry`, `await_consistency`, `search_entries`, `get_*_prompt`) to ingest data, query memory, and grade itself.
2. **On-the-fly summaries** – Every utterance is summarised in-context with the canonical summary prompt fetched from `get_summary_prompt`.
3. **Retrieval** – Probe questions are answered by first invoking `search_entries` and then applying the context prompt obtained from `get_context_prompt`.
4. **Scoring** – The agent uses the GPT-4 judge prompt (“CORRECT/WRONG”) to mark each probe; it also computes a ROUGE-L-R recall score.

## Files & paths

```
bench/dmr/
  ├── msc_5sessions.jsonl   # multi-session chat logs
  └── dmr_qa.jsonl          # 97 questions with gold answers
```

These are copied from MemGPT's paper repo; do **not** modify them.

## Step-by-step (operator)

1. **Start servers**
   ```bash
   export OPENAI_API_KEY=sk-…
   make run-mcp-server   # or go run cmd/synapse-mcp-server/main.go
   ```

2. **Open a shell with the agent**  
   In Cursor, create a new chat that points at the running MCP server.  Verify with:
   ```
   > list_tools
   get_summary_prompt, get_context_prompt, search_entries, …
   ```

3. **Kick off the benchmark**  
   In the agent chat, type exactly:
   ```
   BEGIN DMR BENCHMARK
   ```
   The agent will:
   * create a benchmark user (`u_dmr`)
   * iterate through `msc_5sessions.jsonl`, streaming every message via `add_entry`
   * block on `await_consistency` per conversation
   * loop through `dmr_qa.jsonl`, answer each question, and score

4. **Wait for the summary**  
   At the end the agent prints two numbers, e.g.:
   ```
   DMR completed: ACC=0.72  ROUGE_L_R=0.64
   ```

## Expected runtime
*(Apple M2, OpenAI GPT-3.5-turbo)*: ≈ 4 minutes (ingest 9 k messages + answer 97 probes).

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `get_summary_prompt` not found | Prompt handler not registered | Ensure `PromptHandler` is added in `main.go` |
| `ErrBackPressure` spam | Shard queue size too small | Increase `SQ_MAX_ATTEMPTS` / run with larger `SQ_MAX_INTERVAL` |
| Agent repeats "INSUFFICIENT INFORMATION" | `search_entries` returns empty list | Check search handler; confirm summaries contain the probe fact |

## Updating the baseline
After any change that could affect memory recall (prompt tweak, search backend swap, queue sizing), rerun the steps above and copy the new ACC/ROUGE numbers into *Story S01* task 10 and *ADR-0021* 'Consequences' section.

---

*End of guide* 
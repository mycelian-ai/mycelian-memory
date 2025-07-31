# ADR-0021 – Deep-Memory-Retrieval Benchmark Execution Strategy

Status: ✅ accepted  
Date: 2025-07-28

## Context

We need a reproducible way to evaluate Synapse's long-term-memory pipeline while we are still in a rapid-prototyping phase.  The MemGPT **Deep-Memory-Retrieval (DMR)** benchmark is lightweight (97 probes over 5-session chat logs) and exercises exactly the read/write path we have already implemented (`AddEntry`, ShardExecutor, AwaitConsistency).

Key constraints:

1.  Avoid bespoke ingestion scripts – the benchmark must run **through the same MCP tools** that external agents will use.
2.  Summaries must be generated on-the-fly with the canonical summary prompt.
3.  Retrieval must use the new `search_entries` MCP tool (initially naive substring, later Weaviate).
4.  We don't want any extra Go/Python harness checked into the repo – the evaluation loop will be executed by the LLM agent connected via MCP.

## Decision

1.  Add two read-only MCP tools (`get_summary_prompt`, `get_context_prompt`) so remote agents can fetch prompts programmatically.
2.  Add `search_entries` MCP tool + SDK helper; first implementation uses in-process substring scan, pluggable later.
3.  Benchmark flow (orchestrated by the agent):
   1. Create a single benchmark user and one memory per conversation.
   2. Stream each utterance through `add_entry`, generating summaries via the summary prompt.
   3. Call `await_consistency` after each conversation to drain the shard queue.
   4. For each DMR probe question call `search_entries`, assemble context using the context prompt, and let the LLM answer.
   5. Grade with GPT-4 judge prompt; report accuracy + ROUGE-L-R.
4.  Four implementation tasks were appended to **Story S01** (tasks 7-10) to track prompt/search tools, SDK helper, harness run, and documentation of baseline scores.

## Consequences

• DMR can be executed end-to-end today; future swaps (Weaviate search, richer prompts) require no interface changes.  
• LongMemEval integration will reuse the same infrastructure.  
• CI pipeline gains an optional `make bench-dmr` stage (non-blocking) once the OpenAI key is available.

## Alternatives Considered

| Option | Notes | Why rejected |
|--------|-------|--------------|
| Stand-alone Go or Python ingestion script | Simple to code | Bypasses Synapse concurrency & prompt logic; lower fidelity |
| Import full MemGPT harness | Heavy deps, mixes Python into Go repo | Same fidelity problem + tooling overhead |
| Wait for Weaviate search before running DMR | Cleaner retrieval | Delays feedback loop and blocks prompt-tool work | 
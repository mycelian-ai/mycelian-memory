## LongMemEval Benchmarker (LangGraph) for Mycelian

### Overview
This document specifies a lightweight, reproducible benchmarker to evaluate long‑memory behavior using LangGraph as the orchestrator and Mycelian as the memory system. It aligns with the log‑structured architecture in `docs/designs/001_mycelian_memory_architecture.md`: agents append immutable entries and periodically persist context; retrieval uses hybrid search with a tunable alpha to blend sparse and dense signals.

### Goals
- Evaluate end‑to‑end agent memory: capture (entries + summaries), context persistence, retrieval, and answer quality.
- Support retrieval‑only evaluation to isolate the index/search layer.
- Provide parameter sweeps (alpha, topK, context cadence), deterministic runs, and JSON results for regression tracking.
- Keep the orchestration simple: one tool‑enabled agent node, plus deterministic setup and evaluation nodes.

### Non‑Goals
- Building a complex multi‑agent system.
- Implementing custom retrievers beyond Mycelian’s `/v0/search` and Weaviate‑native hybrid search.
- Serving as a production agent runtime; this is a benchmark harness.

---

## Architecture

### Fit with Mycelian (001)
- **Data plane**: Agent writes per‑turn `entries` (raw text + concise summary) and persists `context` snapshots opportunistically. Both are durable, append‑only logs.
- **Retrieval**: New sessions reconstruct working context by loading the latest context and searching for top‑K entries. Hybrids blend BM25 and embeddings via alpha.
- **Observability**: Benchmark logs every decision (what was written, when persisted, how retrieved), timing each step.

### Interfaces
- **MCP tools (preferred path)** via `mycelian-mcp-server`:
  - `create_vault`, `list_vaults`
  - `create_memory_in_vault`, `get_memory`
  - `add_entry`, `list_entries`, `get_entry`
  - `put_context`, `get_context`
  - `await_consistency`
  - `search_memories` (returns entries + latestContext)
- **HTTP** (optional, for richer search payload):
  - `POST /v0/search` returns `{ entries, latestContext, bestContext, timestamps }`. Alpha is service‑wide (see below).
- Keep exactly one authoritative `memory-service` instance. See Operational Notes.

## Configuration (TOML)

Runs are controlled by a single TOML file passed to the CLI. Credentials are not stored in the file; they come from environment variables (e.g., `OPENAI_API_KEY`, AWS default chain for Bedrock).

### Minimal schema

```toml
# Required: path to local clone of the LongMemEval repo
dataset_repo_path = "/path/to/LongMemEval"

# Required: vault selection (prefer titles for portability)
vault_title = "longmemeval"   # if absent, use vault_id
# vault_id = "..."            # optional explicit UUID; overrides vault_title when provided

# Optional: user-supplied run identifier; if omitted the runner auto-generates one
# run_id = "2025-08-28_a"

[provider]
type = "openai"               # "openai" | "bedrock"

[provider.openai]
# Credentials via env: OPENAI_API_KEY
base_url = "https://api.openai.com/v1"   # optional override

[provider.bedrock]
# Credentials via standard AWS chain (env/profile/role). No secrets in file.
region = "us-west-2"

[models]
# Role-specific models; names must match the selected provider
# OpenAI example:
agent = "gpt-4o-mini"
qa    = "gpt-4o-mini"
# Bedrock example (Claude Haiku):
# agent = "anthropic.claude-3-haiku-20240307"
# qa    = "anthropic.claude-3-haiku-20240307"
# Use an LLM judge by default; set to an explicit model for your provider
eval  = "gpt-4o-mini"

[params]
# Keep defaults simple
top_k = 10
use_llm_judge = true
# Prevent runaway tool loops; if exceeded, the harness marks the turn failed and continues
max_tool_calls_per_turn = 5

# Memory naming convention per conversation (one LME question = one conversation)
# Placeholders: {question_id}, {run_id}
memory_title_template = "{question_id}__{run_id}"
```

### Semantics
- Vault: If `vault_id` is provided, use it. Otherwise, ensure a vault with `vault_title` exists (create when missing).
- Conversation mapping: One LongMemEval question maps to one Mycelian memory (which may span many sessions). The memory title is rendered using `memory_title_template`.
- Defaults: Keep retrieval on service defaults (alpha from `SEARCH_ALPHA`) and `top_k=10`. Use an LLM judge when `models.eval` is set; otherwise fall back to EM.
- Providers:
  - OpenAI: use `OPENAI_API_KEY` and optional `base_url`.
  - Bedrock: resolve credentials via AWS default chain; `region` required. Model IDs must be valid Bedrock model identifiers.
- CLI: The CLI accepts only the path to the TOML file and derives all behavior from it.

### CLI contract (high-level)
- Invocation: `mycelian-longmemeval run /path/to/run.toml`
- The runner reads the TOML, prepares the provider clients, sets up the vault, and executes the pipeline. No other CLI flags are required for a normal run.
- A minimal test config is provided at `tools/longmemeval-benchmarker/config.test.toml` (edit paths/models as needed).

---

## Graph Design (LangGraph)

### Simplicity first
- One setup step: create/get vault and create one memory per LME question.
- One tool‑enabled agent node (prebuilt LangGraph agent) that calls MCP tools directly.
- QA is a simple LLM call (no second agent); evaluation uses EM or an LLM judge.

### Agent node behavior (agentic, not a fixed state‑machine)
At each turn, the agent acts as an observer of a conversation between a User and an AI Assistant, tasked with recording accurate, high‑fidelity memories for this conversation. It:
- Bootstraps every new session by fetching `latestContext` and/or issuing a quick search for priming.
- Decides whether to `add_entry` with concise `summary`, or skip if content is phatic/redundant.
- Decides when to `put_context` based on semantic delta, topic boundary, elapsed time, or size thresholds.
- Uses `await_consistency` after bursts of writes to ensure strong read‑after‑write for subsequent reads.

All policies are prompt‑ and budget‑driven; no rigid orchestration branching required.

### Run modes
- ingestion: stream sessions/messages to the agent to persist entries/context only
- qa: run retrieval + answer only (assumes prior ingestion exists)
- eval: compute metrics (EM or LLM judge) on stored answers

---

## Orchestration

### Who runs what
- The CLI runner (`mycelian-longmemeval`) orchestrates all phases. The agent only decides captures (tool calls) during ingestion. QA (answering) and Eval (scoring) are runner‑driven LLM calls.

### Execution order
- Default (simple): per‑question end‑to‑end
  1) Create memory for the question
  2) Ingest all its sessions (reset agent between sessions)
  3) Run retrieval + QA for that question
  4) Evaluate (EM or LLM judge) and write results

### Components and responsibilities
- Dataset loader: reads LongMemEval from `dataset_repo_path`, yields
  `question → sessions → messages` for streaming.
- Graph builder: constructs the prebuilt LangGraph agent (dynamic system prompt, MCP tools). Stateless per session.
- Runner/controller: reads TOML, selects run mode, creates vault/memory names, streams messages to the agent (reset between sessions), enforces caps (`max_tool_calls_per_turn`), calls `await_consistency` as needed, runs `search_memories` + QA model, invokes the judge if configured, and writes JSONL results.

### Separation of code
- Keep these as small modules under `tools/longmemeval-benchmarker/`:
  - `dataset_loader.py` – parsing/splitting into question/session/message
  - `graph.py` (or `agent.py`) – builds the LangGraph agent
  - `runner.py` – orchestrates phases per TOML (`mode` or `phases`), holds main loop
  One CLI entrypoint imports these modules and coordinates them.

### State passing
- Maintain a map `{question_id → memory_id}` during a run to reuse memories across QA/Eval.
- Memory titles follow `memory_title_template` and include `run_id` for isolation.

### Concurrency (optional)
- Default serial execution for reproducibility. Add an optional `concurrency` setting to process multiple questions in parallel with rate limits.

---

## Agent State Management

This harness follows the proven pattern from `tools/benchmarker/` where the LLM is a tool‑enabled agent that maintains its own working state per turn, while durable memory is entrusted to Mycelian.

### Principles
- Use a dynamic prompt per LangGraph’s prebuilt agent pattern to insert system instructions.
- Single system prompt: load governance + chat prompts once; do not append them to rolling chat history.
- Full rolling transcript: each turn includes prior user/assistant messages from the dataset to maintain fidelity.
- Session resets: orchestrator resets the agent between sessions within the same conversation so it bootstraps via Mycelian (`get_context`/`list_entries`) from prior sessions.
- Tool‑first bootstrap: `get_context()` then `list_entries(limit=10)` before the first dataset turn.
- Agentic persistence: `add_entry` with concise summary; `put_context` on semantic deltas; `await_consistency` after bursts.

### Ephemeral state (in‑graph)
- `history`: prior dataset turns (user/assistant). Excludes system.
- `message_counter`: used to enforce simple cadence (e.g., flush after ≈6 stored messages).

### Durable state (in Mycelian)
- `entries`: append‑only raw + summary records per turn with trace tags `{run_id, dataset_id, question_id, turn_index}`.
- `contexts`: periodic snapshots persisted by the agent according to policy.

### Turn loop
1) Receive the next dataset message (a single user or assistant utterance) from the LongMemEval loader; append to `history`.
2) Call the model with: `system` (static prompts loaded at agent construction) + `messages` (full `history`) + `tools` (MCP schema).
3) Execute any tool calls emitted by the model in order (e.g., `get_context`, `add_entry`, `await_consistency`, `put_context`).
4) Record timings and decisions; update ephemeral counters and `last_context_*`.

### Replay and determinism
- Fix temperatures/seeds; log tool calls and emitted outputs.
- Replay mode can consume `decision_log` to bypass live policy and isolate retrieval.

### Provider differences
- Anthropic/OpenAI/Bedrock adapters normalize to a `messages.create`‑like API that supports `system`, `messages`, `tools`.
- Bedrock Anthropic models are invoked via an adapter that maps request/response blocks to the agent’s expectations (see `tools/benchmarker/model_provider.py`).

---

## LangGraph Agent Wiring (prebuilt)

Use LangGraph’s prebuilt agent with a dynamic prompt. Keep it stateless per session (no checkpointer) and reset between sessions. For reference on dynamic prompts and memory configuration, see the LangGraph prebuilt agent guide ([LangGraph guide](https://langchain-ai.github.io/langgraph/agents/agents/#5-add-memory)).

- Provide a `prompt(state, config)` function: return one system message (rules + chat prompts) plus `state["messages"]` (rolling transcript). Do not duplicate system content in the transcript.
- Register Mycelian tools (MCP wrappers) with the agent: `add_entry`, `put_context`, `get_context`, `list_entries`, `await_consistency`, `search_memories`.

Minimal sketch (illustrative):

```python
from langgraph.prebuilt import create_react_agent
from langchain.chat_models import init_chat_model

def build_system() -> str:
  # Load once: context_summary_rules + chat prompts (entry_capture, summary, context)
  return load_rules_and_prompts()

def prompt(state, config):
  return [{"role": "system", "content": build_system()}] + state["messages"]

agent = create_react_agent(
  model=init_chat_model(provider_model_id, temperature=0),
  tools=[add_entry, put_context, get_context, list_entries, await_consistency, search_memories],
  prompt=prompt,
)

# Per session within a conversation, stream messages turn-by-turn (reset agent between sessions)
for msg in session_messages:
  agent.invoke({"messages": [msg]})
```

This keeps the agent fully agentic (deciding when to write/persist) while avoiding custom orchestration. Short‑term chat state lives in the agent process per session; Mycelian remains the durable store for entries and contexts.

---

## Minimal Flow
1. Setup vault (by title) and create one memory per LongMemEval question using `memory_title_template`.
2. Agent ingest: for each conversation → for each session → stream each message to the agent (reset agent between sessions). The agent uses a dynamic system prompt (rules + chat prompts) and tools `get_context`, `list_entries`, `add_entry`, `put_context`, `await_consistency`.
3. Retrieval+QA: call `search_memories(top_k)`; build a compact context and answer with the QA model.
4. Metric: default to EM. Enable LLM judge only if `models.eval` is set.

---

## Operational Notes

- Use the provided Make targets for a single authoritative local stack:
  - `make start-dev-mycelian-server` (starts Postgres/backend dependencies)
  - `make start-mcp-streamable-server` (builds and runs the MCP server container)
  - `make wait-backend-health` (waits for `/v0/health` to report healthy)
  - Avoid host‑level debug binaries that create duplicate endpoints.
- `mycelian-mcp-server` should point at `MEMORY_SERVICE_URL` (default `http://localhost:11545`).
- To change hybrid alpha used by `/v0/search`, set `SEARCH_ALPHA` in the server’s environment.
- Prefer `search_memories` via MCP; call `/v0/search` directly only if you need `bestContext` in the response.

---



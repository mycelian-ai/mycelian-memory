# Prompt Management APIs

**Doc status:** draft (will be kept in sync with ADR-0022)

---

## Scope
This document describes how application code and LLM agents retrieve the three prompt templates used by Synapse memories:

| Prompt | Purpose | Namespace |
|--------|---------|-----------|
| `entry_capture_prompt` | Decide whether to store an incoming message | `prompts/<memory_type>/entry_capture_prompt.txt` |
| `summary_prompt` | Instructs how to compress a message into the `summary` field | `prompts/<memory_type>/summary_prompt.txt` |
| `context_prompt` | Guides how retrieved entries should be used when answering | `prompts/<memory_type>/context_prompt.txt` |

Each prompt must be **≤ 200 words (~150–180 tokens)**.

---

## Resolution Algorithm
```
if $PROMPT_OVERRIDE_DIR/<memory_type>/<name>_prompt.txt exists → return (source:"override")
else                                                         → return embedded default (source:"default")
```

Environment variable `PROMPT_OVERRIDE_DIR` defaults to `/var/lib/synapse/prompts`.

---

## Go Client SDK helpers

```go
prompt, src, err := sdk.GetEntryCapturePrompt(ctx, memoryType)  // src ∈ {"override","default"}
```

| Helper | Signature | Notes |
|--------|-----------|-------|
| `GetEntryCapturePrompt` | `func(ctx, memType string) (string, string, error)` | Reads file on every call |
| `GetSummaryPrompt` | same | – |
| `GetContextPrompt` | same | – |

All helpers do a fresh `os.ReadFile`; no caching layer is used.

---

## MCP tools (LLM-facing)
The MCP server simply calls the SDK helpers and wraps them as tools.

### `get_entry_capture_prompt`
```jsonc
{
  "tool": "get_entry_capture_prompt",
  "memory_id": "mem_123"
}
→
{
  "prompt": "<string>",
  "source": "override" // or "default"
}
```

Analogous tools:
* `get_summary_prompt`
* `get_context_prompt`

All three accept either `memory_id` **or** `memory_type`. If both are provided, `memory_type` is inferred from the memory.

---

## Example workflow (LLM agent)
1. Call `get_summary_prompt` once per memory at session start.
2. For each incoming user message:
   * Call `get_entry_capture_prompt` → decide to skip or add entry.
   * If storing, use returned `summary_prompt` to generate the summary.
3. When answering, call `get_context_prompt` and combine with retrieved entries.

---

## Compatibility & Future Work
* The public API shape (helpers + tools) will remain stable even if we later add a file-watcher cache.
* If prompt sizes ever exceed the 200-word guideline, revisit token-limit guidance in ADR-0022.

---

*Last updated: 2025-06-22* 
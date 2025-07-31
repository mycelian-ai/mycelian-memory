# MCP Tool Reference

## Tool: `get_default_prompts`

Returns the immutable default prompt templates compiled into the Synapse MCP server binary for a given `memoryType`.

### Arguments
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `memory_type` | `string` | ✅ | Memory type to fetch prompts for. Current enum: `chat`, `code`. Agents should treat the value as case-insensitive and fall back to lower-case. |

### Response (text)
The tool returns a JSON-encoded document (as plain text) with the following schema:

```jsonc
{
  "version": "v1",                      // Incremented whenever defaults change incompatibly
  "context_summary_rules": "...",       // Global rules (same for all memory types)
  "templates": {
    "context_prompt": "...",           // When to update context
    "entry_capture_prompt": "...",     // Decide when to persist an entry
    "summary_prompt": "..."            // How to create the summary field
  }
}
```

### Example Call / Result
```
> get_default_prompts(memory_type="chat")
{
  "version": "v1",
  "context_summary_rules": "# Context & Summary Rules\n...",
  "templates": {
    "context_prompt": "\nYou are ...",
    "entry_capture_prompt": "When you receive a message ...",
    "summary_prompt": "Summarise the message ..."
  }
}
```

### Notes
1. The tool is **read-only**; it never modifies server state.
2. The API is local to the MCP server – no round-trip to the Memory backend, ensuring sub-millisecond latency.
3. When the default prompt set is updated incompatibly, the server will bump `version` (e.g. `v2`). Agents should cache by `(version, memoryType)` pair and re-fetch when version changes. 
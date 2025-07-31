# ADR-0022 – Prompt Storage Revision (Filesystem Overrides)

Status: ✅ accepted  
Date: 2025-07-29  
Supersedes: [ADR-0005](0005-prompt-management.md)

---

## Context
ADR-0005 proposed keeping prompt overrides inside SQLite.  During milestone work we realised:

* The Go SDK never embeds SQLite; shipping it as part of the MCP binary added unwanted footprint.
* Operators typically package MCP as a Docker container; mounting a config volume is simpler than editing a DB.
* Prompts behave like configuration files (rarely mutated at runtime, edited by ops), not transactional data.

Therefore we switch to **filesystem-based overrides** while preserving the same read-only MCP tools.

## Decision

1. **Prompt resolution order**
   1. `PROMPT_OVERRIDE_DIR/<memory_type>/<name>_prompt.txt` if the file exists.
   2. Embedded default compiled into the binary (`prompts/default/...`).

2. **Prompt names (uniform)**
   * `entry_capture_prompt.txt` – when/if to store a message.
   * `summary_prompt.txt`      – how to summarise the message.
   * `context_prompt.txt`      – how to use retrieved memory when answering.

3. **Environment variable / flag**
   * `PROMPT_OVERRIDE_DIR` (default `/var/lib/synapse/prompts`).  Operators mount this path in Docker.

4. **MCP tools (unchanged)**
   * `get_entry_capture_prompt`
   * `get_summary_prompt`
   * `get_context_prompt`
   Each returns `{ prompt, source }` where `source ∈ { "override", "default" }`.

5. **Implementation notes**
   * New helper `prompt.Load(kind, memType)` checks the override directory first.
   * The helper performs a fresh `os.ReadFile` **every call**.  Prompt files are intentionally concise—**≤ 200 words (≈ 150-180 tokens)**—so the read cost is trivial and the token overhead for an LLM remains low.  No in-memory cache is kept: the disk-read latency is negligible and this enables hot-reload when operators edit a prompt.
   * If the override file is absent it falls back to the embedded default (`embed.FS`).
   * Future optimisation: if prompt request volume becomes significant we can wrap `prompt.Load` with an `fsnotify` watcher + cache without changing any public API.

## Consequences

+ Removes SQLite dependency from prompt flow; container image is smaller.  
+ Ops can tweak prompts by editing files on a mounted volume; no migrations needed.  
+ ADR-0005 is marked **❌ superseded** in `decisionLog.md`.

## Migration

*If* operators had already used the (never-released) SQLite override feature, they must export those prompts and place them in the override directory before upgrading.

---

_End of ADR_ 
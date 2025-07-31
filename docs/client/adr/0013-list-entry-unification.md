---
description: Deprecate top-k endpoint and CLI in favour of parameterised list_entries
---
# ADR-0013: Unify List & Top-K Entry Retrieval

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-26 | @core-team |

## Context

Early prototypes exposed two separate read patterns:

1. **`list_entries`** – cursor-based pagination with optional `limit`, `before`, `after`.
2. **`get_top_k` / CLI `top-entries`** – convenience wrapper returning the K most-recent rows.

Maintaining duplicate verbs increases surface area (docs, SDKs, tests) while offering no additional capability—`list_entries --limit K` already fulfils the same use-case.

## Decision

* **API**: Remove the dedicated `get_top_k` REST/MCP tool.
* **CLI**: Delete `top-entries`; users call `list-entries --limit K` instead.
* **Docs & SDKs**: All references to top-k have moved to this ADR; existing ADRs remain unchanged to preserve history.
* **Backward Compatibility**: A one-release grace period is provided where `top-entries` CLI alias still exists but will print a deprecation warning (implemented client-side). After the grace period the alias will be removed.

## Consequences

+ **Simplicity**: Single source of truth for multi-row retrieval.
+ **Discoverability**: Users learn one verb and rely on flags for variants.
+ **Maintenance**: Less redundant test and handler code.

## Alternatives Considered
| Option | Reason Rejected |
| ------ | --------------- |
| Keep both commands | Redundant, higher maintenance cost. |
| Rename list_entries to top_entries globally | Breaks existing integrations; list_* naming matches REST conventions. |

## Migration Guide

CLI:
```
# old
synapse top-entries --user-id U --memory-id M --limit 3
# new
synapse list-entries --user-id U --memory-id M --limit 3
```

MCP Tool:
```
# old
{"name":"get_top_k","arguments":{"user_id":"U","memory_id":"M","limit":"3"}}
# new
{"name":"list_entries","arguments":{"user_id":"U","memory_id":"M","limit":"3"}}
```

No server-side schema changes are required; the existing list endpoint already supports a `limit` parameter. 
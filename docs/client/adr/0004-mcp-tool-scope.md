---
adr: 0004-mcp-tool-scope
status: accepted
date: 2025-07-24
---
# Restrict MCP tool surface to Context & Entry operations; move user/memory creation to CLI

## Context
Early MCP implementation exposed `create_user`, `create_memory`, and related update tools directly to the LLM agent. This blurred the boundary between human-initiated persistent changes and agent-initiated workflow assistance, raising safety and audit concerns.

Feedback highlighted that:
* Users want full control of permanent resources (users, memories).
* The agent should remain read-only for those resources to avoid unintended writes.
* Memory creation is still required but fits better in a dedicated human CLI.

## Decision
1. **Remove** `create_user` and `create_memory` tools from the MCP server.
2. **Retain** underlying HTTP endpoints and Go SDK methods so a human-facing `synapse` CLI can manage users and memories.
3. **Limit** MCP tool surface (agent-facing) to:
   * Context APIs (future)
   * Entry APIs (add_entry, get_entries, etc.)
4. Enforce the policy in code:
   * Handlers no longer register the removed tools.
   * Agent tokens carry scopes only for read + `entry:create`.
5. Document the workflow: humans create memories via CLI, copy the `memory_id` into agent config.

## Consequences
+ Eliminates risk of silent user/memory creation by the LLM.
+ Simplifies threat model & audit trail: all permanent resources originate from human actions.
+ Requires a lightweight `synapse` CLI (already planned) for admin operations.
+ Minimal code impact—mainly handler pruning and scope tightening.

_Status: Accepted and implemented 2025-06-18_ 
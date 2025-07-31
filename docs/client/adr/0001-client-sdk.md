---
adr: 0001-client-sdk
status: accepted
date: 2025-07-24
---
# Adopt Go client SDK, deprecate pkg/memoryclient

## Context
Legacy `pkg/memoryclient` duplicated logic, lacked retry/shard support, and made handler code verbose.

## Decision
Implement a dedicated Go SDK in `client/` and migrate all MCP handlers and tools to use it.

## Consequences
+ Shared retry and sharding logic at one layer.
+ Cleaner handler code.
+ Must delete `pkg/memoryclient` after full migration. 
---
adr: 0002-zerolog
status: accepted
date: 2025-07-24
---
# Adopt zerolog v1.34 for structured logging

## Context
Early code used the standard library `log` which lacks levels and JSON output.

## Decision
Add `github.com/rs/zerolog` v1.34 as the sole logging dependency. Initialise once in `internal/config/logger.go`; inject logger via context.

## Consequences
+ Consistent structured logs (JSON/file, pretty/console).  
+ Low GC overhead.  
+ Replaced previous `log.*` calls. 
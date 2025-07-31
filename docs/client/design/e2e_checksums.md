# End-to-End Checksums

_Status: Planned – implementation scheduled for the Context APIs phase_

## Why?
Our data passes through many layers (CLI → SDK → MCP → DB). Bugs or refactors can silently mix up `user_id`, `memory_id`, or truncate an entry. We want a **cheap, always-on check** that these critical fields stay bound together end-to-end.

## Core Idea
1. **Critical fields only** – `user_id`, `memory_id`, `summary/body`, `timestamp`, and a per-request `nonce` (UUID).
2. **Envelope** – Canonical-JSON of those fields plus a CRC-32C checksum.
   ```json
   {
     "payload": { ... },
     "crc32c": 1234567890
   }
   ```
3. **Verify at TCB boundaries** – SDK seals, MCP verifies & reseals. Pure routing layers just pass the blob.
4. **Fail closed** – If checksum fails, request is rejected (5xx) and logged.

## Minimal Go API (internal/integrity)
```go
func Seal(payload any) (Envelope, error)   // client side
func Verify(e Envelope, dest any) error    // server side
```

CRC-32C is hardware-accelerated; overhead < 5 µs/op.

## Roll-out Plan (post-Context API phase)
1. Land `internal/integrity` package with CRC-32C helpers (zero external deps).  
2. Integrate sealing in SDK and verification middleware in MCP for **AddEntry** write path.  
3. Enable in staging behind flag `INTEGRITY_ENVELOPE=on`; observe error / latency metrics for one release cycle.  
4. Expand to other write APIs (delete, create_memory).  
5. Optional: add response binding for reads if benefit outweighs cost.

That's it—no crypto keys, no formal proofs for now; just a lightweight safety belt. 
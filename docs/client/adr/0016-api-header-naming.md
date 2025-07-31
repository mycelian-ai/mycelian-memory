---
description: Standardise custom HTTP header names (drop legacy X- prefix)
---
# ADR-0016: API Header Naming Convention

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-27 | @core-team |

## Context

The project introduces two custom headers:
* Idempotency token for deduplication.
* Per-attempt request identifier for tracing.

Historically many APIs used an `X-` prefix for non-standard headers (e.g., `X-Request-Id`). RFC 6648 (2012) discourages new `X-` prefixes in favour of plain names.

## Decision

Adopt modern, prefix-free names:

| Concern | Header Name | Example |
|---------|-------------|---------|
| Idempotency | **Idempotency-Key** | `Idempotency-Key: 6b5a3f2c-c63e-4d1e-a9a0-8e3b2cbf3ef4` |
| Per-request trace | **Request-Id** | `Request-Id: 3d2f85bb-8e12-4c91-930a-d3f9c539dc64` |

Rules:
1. Headers are case-insensitive; docs use start-case for readability.
2. SDKs accept/emit exactly these names. Aliases like `X-Request-Id` are **not** supported.
3. JSON/tool parameters use snake_case equivalents: `idempotency_key`, `request_id`.

## Consequences

+ Aligns with current best practice (Stripe, RFC 6648).  
+ Avoids confusion between legacy and modern header names.  
+ Requires updating ADR-0015 and docs to replace `X-Request-Id` with `Request-Id`.

## Migration

* Update SDK constants and middleware.  
* Search/replace docs for `X-Request-Id` → `Request-Id`.  
* Backend must echo `Request-Id` in every response.

## References
* RFC 6648 – Deprecating the "X-" prefix in HTTP header field names  
* Stripe API Idempotency docs (uses `Idempotency-Key`) 
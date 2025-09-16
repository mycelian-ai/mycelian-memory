# ADR-006: API Header Naming Convention

**Status**: Accepted
**Date**: 2025-07-27

## Context

The project introduces custom HTTP headers for:
- Idempotency token for deduplication
- Per-attempt request identifier for tracing

Historically APIs used `X-` prefix for non-standard headers (e.g., `X-Request-Id`). RFC 6648 (2012) discourages new `X-` prefixes in favour of plain names.

## Decision

Adopt modern, prefix-free names:

| Concern | Header Name | Example |
|---------|-------------|---------|
| Idempotency | **Idempotency-Key** | `Idempotency-Key: 6b5a3f2c-c63e-4d1e-a9a0-8e3b2cbf3ef4` |
| Per-request trace | **Request-Id** | `Request-Id: 3d2f85bb-8e12-4c91-930a-d3f9c539dc64` |

### Rules
1. Headers are case-insensitive; docs use start-case for readability
2. SDKs accept/emit exactly these names. Aliases like `X-Request-Id` are **not** supported
3. JSON/tool parameters use snake_case equivalents: `idempotency_key`, `request_id`

## Consequences

### Positive Consequences
- Aligns with current best practice (Stripe, RFC 6648)
- Avoids confusion between legacy and modern header names
- Clear, standardized naming across all interfaces

### Implementation Notes
- Update SDK constants and middleware
- Backend must echo `Request-Id` in every response
- Consistent naming between HTTP headers and JSON parameters

## References
- RFC 6648 – Deprecating the "X-" prefix in HTTP header field names
- Stripe API Idempotency docs (uses `Idempotency-Key`)

---
title: Structured Error Schema & Invariant Error Codes
status: Accepted
date: 2025-07-21
---

# Context

Ad-hoc error messages complicate client handling and invariants rely on specific failure modes.  A uniform error envelope enables automatic retry / user messaging and keeps REST & gRPC behaviour aligned.

# Decision

1. Every error response (REST and gRPC metadata) follows: `{error: string, code: string, details?: string}`.  
2. Reserved `code` values: `IMMUTABILITY_VIOLATION`, `CONTENT_IMMUTABLE`, `UNAUTHORIZED_ACCESS`, `ALREADY_EXISTS`, `NOT_FOUND`, `INTERNAL`.  
3. New invariant-driven codes must be added to this ADR and referenced in SDK enums before being used server-side.

# Consequences

• Client SDKs can switch on `code` field for control flow.  
• Test harness can assert invariant failures precisely.  
• gRPC Status details are mapped 1-to-1 with the JSON envelope to avoid divergence. 
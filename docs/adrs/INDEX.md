# ADR Index

A consolidated list of Architectural Decision Records (ADRs).

Policy: ADRs are immutable once Accepted. Future changes must be recorded in a new ADR that references and supersedes the prior one. The only allowed status transitions after Acceptance are keeping it as Accepted or marking it as Superseded (by ADR-XXX).

| #   | Title                               | Decision   | Status        | Date       |
|-----|-------------------------------------|----------|--------------|------------|
| 001 | Invariant-Driven Development        | Accepted | Implemented  | 2025-07-20 |
| 002 | Server-Side Entity ID Generation    | Accepted | Implemented  | 2025-07-20 |
| 003 | Environment-Driven Configuration    | Accepted | Implemented  | 2025-07-21 |
| 004 | Go Client SDK Adoption              | Accepted | Implemented  | 2025-07-24 |
| 005 | Concurrency & Replication Model     | Accepted | Implemented  | 2025-07-25 |
| 006 | API Header Naming Convention        | Accepted | Pending      | 2025-07-27 |
| 007 | Idempotency & Request-ID Semantics  | Accepted | Pending      | 2025-07-27 |
| 008 | Await Consistency Primitive         | Accepted | Implemented  | 2025-07-28 |
| 009 | PostgreSQL-Only Backend             | Accepted | Implemented  | 2025-08-09 |
| 010 | Consolidated PostgreSQL Docker Stack| Accepted | Implemented  | 2025-08-09 |
| 011 | Memory Scoping and Isolation        | Accepted | Pending      | 2025-08-10 |
| 012 | API Key-Based Authorization         | Accepted | In progress  | 2025-08-11 |
| 013 | LangGraph-Based LongMemEval Benchmarker | Accepted | Pending   | 2025-08-26 |

Notes
- ADRs are immutable artifacts once Accepted; do not edit historical decisions. Create a new ADR to change direction and reference the old one via Superseded by.
- ADR files communicate decisions only. Use the "State" column here to track implementation: Pending | In progress | Implemented.
- Numbers are chronological.
- Update this index whenever a new ADR is added or states change.

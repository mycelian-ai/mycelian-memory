---
description: Flag inflight entry APIs as experimental Labs feature, excluded from MVP
---
# ADR-0014: Experimental Gating of Inflight Entry APIs

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-27 | @core-team |

## Context

ADR-0011/0012 defined `list_inflight_entries` and `get_inflight_entry` for sub-millisecond, local-only reads of unsynced rows.  While technically valuable, these endpoints are *not required* for the SMB-focused MVP.

We still believe fast "read-your-local-write" may differentiate Synapse in edge / mobile scenarios, so we want to keep the code path but hide it from general availability until demand is validated.

## Decision

1. **Labs Flag**
   • Inflight APIs are behind feature flag `labs.inflight_read=true` at build-time or via `X-Synapse-Labs: inflight` HTTP header.
   • SDK & CLI will not document the flag; interested users must opt-in explicitly.

2. **MVP Exclusion**
   • Public docs, Swagger/OpenAPI, quick-starts, and pricing pages will *omit* these endpoints.
   • They do not influence the MVP acceptance criteria.

3. **Telemetry & Sunset Criteria**
   • Metrics `labs_inflight_read_calls` and per-org cardinality will be shipped.
   • After 90 days post-GA:
     – If <5 % of orgs use the flag, we will deprecate and remove.
     – Otherwise, promote to GA and update ADR-0012 accordingly (new superseding ADR).

## Consequences

+ **Focus**: Keeps the MVP API surface minimal for SMB onboarding.
+ **Flexibility**: Power users can experiment without blocking GA.
+ **Governance**: Documented path to graduate or sunset the feature.

## Alternatives Considered
| Option | Reason Rejected |
| ------ | --------------- |
| Ship inflight APIs as GA in MVP | Extra cognitive load; unclear demand. |
| Delete inflight code entirely | Hard to re-implement; gives up potential edge advantage. |

## References
* [ADR-0011](0011-inflight-messages.md) – Inflight entry API surface
* [ADR-0012](0012-read-consistency-model.md) – Read consistency model and inflight optimisation 
---
title: Custodian Service for Immutable Audit Log
status: Accepted
date: 2025-07-22
---

# Context

Regulatory and forensic requirements mandate an immutable history of all memory mutations.  Rather than pollute the transactional path, a dedicated **custodian** service tails Spanner commit logs and writes an append-only audit record.

# Decision

1. Custodian subscribes to the Spanner change stream (or periodic snapshot) and persists each mutation to an `AuditEvents` table keyed by commit timestamp.  
2. Delivery guarantee = **at least once**; downstream consumers must de-duplicate by commit ID.  
3. Custodian is versioned alongside other backend services and deployed as a stateless container with exponential back-off retries on failure.

# Consequences

• Core write latency is unaffected; audit durability is offloaded.  
• Security teams can replay or export the audit log at any time.  
• Future features (analytics, billing) can consume the same event stream without impacting OLTP traffic. 
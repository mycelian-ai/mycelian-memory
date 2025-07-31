---
title: Soft Deletion & Data Retention Policy
status: Accepted
date: 2025-07-21
---

# Context

Memory entries may need to disappear from standard queries (GDPR, user mistakes) while retaining an immutable audit trail.  Physical deletes would break referential integrity and impede replay analytics.

# Decision

1. DELETE operations set `DeletionScheduledTime` (UTC timestamp) but never remove the row physically.  
2. All read queries filter `WHERE DeletionScheduledTime IS NULL`.  
3. A future GC job may purge rows older than a configurable TTL, but only after snapshotting to cold storage.

# Consequences

• Users experience "deletion" instantly without losing historical truth.  
• Analytics and vector search can ignore deleted rows or archive them separately.  
• Complies with audit requirements while enabling future right-to-erasure workflows via encrypted tombstones. 
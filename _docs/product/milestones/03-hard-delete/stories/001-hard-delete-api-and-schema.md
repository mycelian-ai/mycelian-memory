## Hard delete: remove soft-deletion timestamp; explicit DELETE endpoints

Status: ⏳ in-progress • Owner: TBD • Target: TBD

### Problem
Simplify deletion semantics by removing soft-deletion (`DeletionScheduledTime`) and performing immediate hard deletes. This reduces complexity in read paths and client behavior. We need consistent DELETE APIs for entries and contexts, and to confirm memory/vault deletes are hard deletes.

### Task table
| Order | Title | Brief | Status |
|---|---|---|---|
| 1 | Docs: DELETE semantics | Update `docs/server/api-documentation.md`: remove `deletionScheduledTime`; add DELETE endpoints for entry/context; clarify memory/vault deletes | ✅ done |
| 2 | DB schema: drop soft-delete cols | Remove `DeletionScheduledTime` from `Memories` and `MemoryEntries` (Spanner + SQLite) | ✅ done (no such cols present) |
| 3 | Server: remove soft-delete filters | Remove `... DeletionScheduledTime IS NULL` filters from queries | ✅ done (not present) |
| 4 | Server: DELETE entry by ID | Add `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}`; storage delete by entryId | ✅ done |
| 5 | Server: DELETE context | Add `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}` | ✅ done |
| 6 | SDK: delete helpers | Add `DeleteEntry` (by entryId) and `DeleteContext` helpers; update tests | ✅ done |
| 7 | Indexer: delete propagation | Ensure vector index (Waviate) deletes corresponding objects on hard delete | 🔜 planned |

### Details per task

1) Docs: DELETE semantics
- Remove `deletionScheduledTime` from all response schemas
- Add endpoints:
  - `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}` → 204
  - `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}` → 204
- Clarify existing deletes:
  - `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}` → 204 (hard delete; cascades to entries/contexts via interleave)
  - `DELETE /api/users/{userId}/vaults/{vaultId}` → 204 (keep current invariant: vault must be empty)

2) DB schema: drop soft-delete columns
- Spanner DDL changes:
  - `ALTER TABLE MemoryEntries DROP COLUMN DeletionScheduledTime;`
  - `ALTER TABLE Memories DROP COLUMN DeletionScheduledTime;`
- SQLite: drop columns in migration, adjust scans/selects

3) Server: remove soft-delete filters
- Remove `AND DeletionScheduledTime IS NULL` from list/select queries
- Ensure list/search paths no longer reference the field or omit rows incorrectly

4) Server: DELETE entry by ID
- Route: `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}`
- Storage: implement `DeleteMemoryEntryByID(userID, vaultID, memoryID, entryID)` for Spanner + SQLite
- Return 204 on success; 404 when not found; 400 on invalid IDs

5) Server: DELETE context
- Route: `DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}`
- Storage: implement `DeleteMemoryContextByID(userID, vaultID, memoryID, contextID)`

6) SDK: delete helpers
- `DeleteEntry(ctx, userID, vaultID, memID, entryID)` → HTTP DELETE
- `DeleteContext(ctx, userID, vaultID, memID, contextID)` → HTTP DELETE
- Update integration/mock tests

7) Indexer: delete propagation
- Add delete propagation via background worker. For local dev, best-effort cleanup may lag; production path will use transactional outbox → worker to call Weaviate delete by `entryId`/`contextId`.

### Definition of Done
- For each task, run:
  - `go fmt ./... && go vet ./... && go test -race ./... && golangci-lint run && govulncheck ./...`
- Server + SDK build
- Docs updated to reflect actual behavior

### Conventional commits (suggested)
- docs(api): document hard delete; add entry/context DELETE endpoints
- db(schema): drop DeletionScheduledTime (Memories, MemoryEntries)
- feat(server): delete entry by entryId and delete context by contextId
- refactor(server): remove soft-delete filters in read paths
- feat(client): add DeleteEntry/DeleteContext helpers
- feat(search): propagate deletions to vector index

### Risk note
What can go wrong: irreversible deletes; accidental data loss; index inconsistency if vector delete fails.
Mitigations: careful CLI/UI confirmation for destructive actions; ensure index delete is best-effort with logs; add tests for 404/validation; keep vault-delete invariant (must be empty) to avoid surprises.



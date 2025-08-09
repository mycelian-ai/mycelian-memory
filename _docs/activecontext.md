# Active Context - SQLite/Spanner Deprecation

## Current Focus
Systematically removing SQLite and Spanner storage adapters and indexer-prototype after successful PostgreSQL migration completion. Working on branch `deprecate-sqlite-spanner`.

## Context
- ✅ PostgreSQL migration fully implemented per ADR 0014
- ✅ All tests passing with Postgres-only setup  
- ✅ Outbox worker integrated with Weaviate indexing
- ✅ Vault isolation and hard delete implemented
- 🎯 **Now ready to remove legacy storage backends**

## Current Status
**Phase 0 Complete**: PostgreSQL implementation validated
- PostgreSQL adapter implements full Storage interface
- Docker compose postgres stack working
- All vault isolation and hard delete features working
- Test suite passes with `make test-all`

**Next: Phase 1** - Inline embed provider and remove indexer-prototype

## Active Decisions

### Storage Architecture (FINAL)
- **Postgres-only**: Single storage backend reduces complexity
- **Hard deletes**: No expiration_time/TTL, immediate deletion
- **Vault isolation**: All operations scoped by vault_id
- **Outbox pattern**: Async Weaviate sync via outbox worker

### Removal Strategy
**Phased approach** to minimize risk:
1. **Phase 1**: Remove indexer-prototype (inline embed providers)
2. **Phase 2**: Remove SQLite adapter and config
3. **Phase 3**: Remove Spanner adapter and config  
4. **Phase 4**: Simplify config to postgres-only
5. **Phase 5**: Documentation and final cleanup

### Key Constraints
- Must maintain CI gates: `go fmt && go vet && go test -race && golangci-lint && govulncheck`
- No breaking changes to public APIs
- Preserve existing Postgres functionality
- Keep rollback points between phases

## Recent Learnings
- Indexer-prototype has embedding providers used by search service
- Need to inline `Embedder` interface and `OllamaProvider` before deletion
- Several tools depend on indexer-prototype: waviate-tool, e2e tests
- Compose files reference indexer-prototype service that needs removal

## Implementation Notes
- **Embedding providers**: Move from indexer-prototype to internal/search
- **Config simplification**: Remove build target complexity, default to postgres
- **Dependency cleanup**: Remove cloud.google.com/go/spanner, modernc.org/sqlite
- **Documentation**: Update ADRs, README, runbooks to reflect postgres-only

## Files to Monitor
- `server/internal/search/embed*.go` - embedding provider refactor
- `server/internal/config/config.go` - config simplification
- `server/internal/platform/factory/storage.go` - single backend
- `deployments/docker/docker-compose.postgres.yml` - final compose
- Root and server Makefiles - target cleanup

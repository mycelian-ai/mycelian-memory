# Task List - SQLite/Spanner/Indexer-Prototype Removal

## Overview
Systematic removal of legacy storage backends and indexer-prototype after successful PostgreSQL migration. All phases must maintain CI gates and preserve postgres functionality.

## Phase 1: Remove Indexer-Prototype ⏳

| Order | Task | Status | Notes |
|-------|------|--------|-------|
| 1.1 | Define Embedder interface in server/internal/search/embed.go | ⏳ | Replace type alias with direct interface definition |
| 1.2 | Implement OllamaProvider in server/internal/search/ | ⏳ | Move from indexer-prototype, maintain same API |
| 1.3 | Update embed_factory.go NewProvider implementation | ⏳ | Remove indexer-prototype import |
| 1.4 | Update server/cmd/waviate-tool/main.go imports | ⏳ | Change to server/internal/search |
| 1.5 | Update server/dev_env_e2e_tests/search_relevance_test.go | ⏳ | Use search.NewProvider |
| 1.6 | Delete server/internal/indexer-prototype/ directory | ⏳ | Remove all files |
| 1.7 | Delete server/cmd/indexer-prototype/ and Dockerfile | ⏳ | Remove binary and container |
| 1.8 | Remove indexer-prototype from compose files | ⏳ | Update references |
| 1.9 | Update DEVELOPER.md, docs, runbooks | ⏳ | Remove indexer-prototype mentions |
| 1.10 | Test and lint Phase 1 | ⏳ | Ensure CI gates pass |

## Phase 2: Remove SQLite 🔜

| Order | Task | Status | Notes |
|-------|------|--------|-------|
| 2.1 | Delete server/internal/storage/sqlite/ directory | 🔜 | adapter.go, conn.go, tests |
| 2.2 | Remove SQLite schema helper (localstate/schema.go) | 🔜 | Keep paths.go if still used |
| 2.3 | Update storage factory - remove sqlite case | 🔜 | Remove EnsureSQLiteSchema call |
| 2.4 | Remove SQLitePath from config.go | 🔜 | Remove field and validation |
| 2.5 | Remove DB_DRIVER=sqlite support | 🔜 | Update ResolveDefaults, allowed list |
| 2.6 | Delete docker-compose.sqlite.yml | 🔜 | Remove compose file |
| 2.7 | Remove sqlite targets from Makefiles | 🔜 | run-sqlite, clean-sqlite-data, etc |
| 2.8 | Remove modernc.org/sqlite dependency | 🔜 | go mod tidy |
| 2.9 | Test and lint Phase 2 | 🔜 | Ensure no sqlite references remain |

## Phase 3: Remove Spanner 🔜

| Order | Task | Status | Notes |
|-------|------|--------|-------|
| 3.1 | Delete server/internal/storage/spanner.go | 🔜 | Main adapter file |
| 3.2 | Delete spanner tests (spanner_test.go, real_spanner_integration_test.go) | 🔜 | Test files |
| 3.3 | Delete server/internal/platform/database/spanner.go | 🔜 | Platform integration |
| 3.4 | Remove spanner schema files if spanner-specific | 🔜 | Check schema.sql |
| 3.5 | Remove Spanner config fields from config.go | 🔜 | SpannerInstanceID, etc |
| 3.6 | Remove GetSpanner* helper methods | 🔜 | Config helper cleanup |
| 3.7 | Disallow DB_DRIVER=spanner-pg | 🔜 | Update validation |
| 3.8 | Delete docker-compose.spanner.yml | 🔜 | Remove compose file |
| 3.9 | Remove spanner targets from Makefiles | 🔜 | run-spanner, schema targets |
| 3.10 | Delete spanner emulator scripts | 🔜 | docker-setup scripts |
| 3.11 | Update tests using spanner admin client | 🔜 | api_test.go imports |
| 3.12 | Delete tools/schema-manager/ | 🔜 | Spanner-only utility |
| 3.13 | Remove cloud.google.com/go/spanner dependency | 🔜 | go mod tidy |
| 3.14 | Test and lint Phase 3 | 🔜 | Ensure no spanner references |

## Phase 4: Config Simplification 🔜

| Order | Task | Status | Notes |
|-------|------|--------|-------|
| 4.1 | Simplify BuildTarget DB derivation | 🔜 | Default postgres for all targets |
| 4.2 | Keep only POSTGRES_DSN config | 🔜 | Single storage config |
| 4.3 | Update storage factory postgres-only path | 🔜 | Clear errors if DSN missing |
| 4.4 | Verify docker-compose.postgres.yml wiring | 🔜 | Confirm env vars correct |
| 4.5 | Update Makefile help to show postgres targets only | 🔜 | Clean help output |
| 4.6 | Test and lint Phase 4 | 🔜 | Single backend validation |

## Phase 5: Documentation & Cleanup 🔜

| Order | Task | Status | Notes |
|-------|------|--------|-------|
| 5.1 | Mark ADR 0014 spanner removal completed | 🔜 | Update implementation status |
| 5.2 | Update README.md to postgres-only | 🔜 | Remove sqlite/spanner sections |
| 5.3 | Update DEVELOPER.md | 🔜 | Postgres-only development |
| 5.4 | Update runbooks and quickstarts | 🔜 | Remove legacy references |
| 5.5 | Final go mod tidy across all modules | 🔜 | Clean dependencies |
| 5.6 | Run govulncheck ./... | 🔜 | Security validation |
| 5.7 | Final CI validation | 🔜 | All tests pass |

## Success Criteria
- [ ] All CI gates pass: `go fmt && go vet && go test -race && golangci-lint && govulncheck`
- [ ] No references to sqlite, spanner, or indexer-prototype in codebase
- [ ] Only postgres compose and config remains
- [ ] Documentation reflects postgres-only architecture
- [ ] Dependencies cleaned up (no unused imports)

## Risk Mitigation
- Each phase is its own commit/PR for rollback capability
- CI gates enforced at each phase
- Grep validation for missed references
- Test coverage maintained throughout

## Notes
- Repository is on `deprecate-sqlite-spanner` branch
- PostgreSQL migration (ADR 0014) is complete and validated
- Current postgres setup includes vault isolation and hard deletes
- Outbox worker integration with Weaviate is working

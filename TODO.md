# TODO: Consolidate Applications Under Root-Level cmd Directory

## Overview
Move all applications from nested cmd directories to the root-level `cmd/` directory for better organization and maintainability.

## Current State
- Root `cmd/`: Empty directories (memoryctl, mycelian-memory-service, mycelian-mcp-server)
- Server applications: Located in `server/cmd/` with Dockerfiles
- Client applications: Located in `clients/go/cmd/`

## Applications to Move

### From `server/cmd/`:
- [] **memory-service** → `cmd/mycelian-memory-service/`
  - [] Move main.go
  - [] Move Dockerfile
  - [] Update Dockerfile build path from `./cmd/memory-service` to `./cmd/mycelian-memory-service`
  - [] Test Go build: `go build -o mycelian-memory-service ./cmd/mycelian-memory-service`
  - [] Test Docker build: `make backend-sqlite-up` ✅

- [] **memoryctl** → `tools/memoryctl/`
  - [] Move main.go and all .go files from cmd/memoryctl/
  - [] Remove Dockerfile (replaced with direct curl in bootstrap containers)
  - [] Update Docker Compose to use alpine/wget instead of custom container
  - [] Test Go build: `cd tools && GOWORK=off go build -o memoryctl ./memoryctl`

- [] **waviate-tool** → `tools/waviate-tool/`
  - [] Move main.go from server/cmd/waviate-tool/
  - [] Update import paths for tools module structure
  - [] Test Go build: `cd tools && GOWORK=off go build -o waviate-tool ./waviate-tool`

- [] **indexer-prototype** → `cmd/indexer-prototype/`
  - [] Move main.go
  - [] Move Dockerfile
  - [] Update Dockerfile build path (path remained the same: `./cmd/indexer-prototype`)
  - [] Test Go build: `go build -o indexer-prototype ./cmd/indexer-prototype`
  - [] Test Docker build: `make backend-sqlite-up` ✅

### From `clients/go/cmd/`:
- [] **mycelian** → `cmd/mycelian-mcp-server/`
  - [] Move application files to cmd/mycelian-mcp-server/
  - [] Move config package out of internal to avoid import restrictions
  - [] Update go.mod with replace directive for module dependency
  - [] Test Go build: `go build -o mycelian-mcp-server ./cmd/mycelian-mcp-server`
  - [] Test Docker build: `docker build -f cmd/mycelian-mcp-server/Dockerfile -t mycelian-mcp-server .` ✅

## Build System Updates

### Docker Compose Files
- [] Update `deployments/docker/docker-compose.streamable.yml`
  - [] Change build context from `../../clients/go` to `../../`
  - [] Update dockerfile path to `cmd/mycelian-mcp-server/Dockerfile`

- [ ] Update `deployments/docker/docker-compose.spanner.yml`
  - [ ] Update any references to old cmd paths

- [ ] Update `deployments/docker/docker-compose.sqlite.yml`
  - [ ] Update any references to old cmd paths

### Makefile Updates
- [ ] Update root `Makefile` targets to reference new cmd locations
- [ ] Update `server/Makefile` if it references old cmd paths
- [ ] Update `clients/go/Makefile` if it references old cmd paths

### CI/CD Scripts
- [ ] Update any CI/CD scripts that reference old cmd paths
- [ ] Update any GitHub Actions workflows
- [ ] Update any build scripts in `scripts/` directory

## Documentation Updates
- [ ] Update `README.md` files that reference old cmd paths
- [ ] Update `DEVELOPER.md` with new structure
- [ ] Update any documentation in `docs/` that references old paths
- [ ] Update runbooks and quickstart guides

## Testing & Validation
- [ ] Verify Docker containers start correctly from new paths
- [ ] Run integration tests to ensure functionality is preserved
- [ ] Test CLI tools work from new locations

## Cleanup
- [ ] Remove empty `server/cmd/` directory
- [] Remove empty `clients/go/cmd/` directory
- [ ] Update any import paths that reference old locations
- [ ] Remove any orphaned references to old paths

## Risk Mitigation
- [ ] Create backup branches before starting migration
- [ ] Move one application at a time and test
- [ ] Keep original directories until migration is complete
- [ ] Update all references to maintain consistency

## Benefits
- Unified structure: All applications under single `cmd/` directory
- Simplified navigation: Easier to find and manage applications
- Consistent organization: Follows Go project conventions
- Reduced complexity: Eliminates nested cmd directories

## Notes
- Dockerfiles will need updated build paths since they reference `./cmd/<app-name>`
- Multi-stage builds must work correctly with new paths
- All COPY commands and build contexts need updating
- CMD/ENTRYPOINT references must be correct

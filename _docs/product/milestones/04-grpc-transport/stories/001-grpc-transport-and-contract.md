## gRPC transport: single contract (proto), dual surfaces (gRPC + REST)

Status: 🔜 planned • Owner: TBD • Target: TBD

### Goal
Introduce a first-class gRPC API as the production surface while keeping REST for local dev/CI. Protobuf is the single source of truth; REST is generated/thin-wrapped from proto to prevent drift.

Refs: `docs/server/adr/0003-transport-and-monorepo-strategy.md`, `docs/server/adr/0004-public-api-contract.md`.

### Scope
- Services: Health, Users, Vaults, Memories, Entries, Contexts, Search
- Contract: `proto/synapse/memory/v1/*.proto` (Go package `synapse.memory.v1`)
- Codegen: gRPC server/client, grpc-gateway (optional), OpenAPI (optional)

### Task table
| Order | Title | Brief | Status |
|---|---|---|---|
| 1 | Proto module & tooling | Add Buf workspace (`buf.yaml`, `buf.gen.yaml`); standard plugin config for Go, grpc-gateway, OpenAPI | 🔜 planned |
| 2 | Define v1 protos | Specify services/messages for Users/Vaults/Memories/Entries/Contexts/Search; include annotations | 🔜 planned |
| 3 | Generate code | Wire `make protogen`; generate Go stubs, gateway, OpenAPI; commit generated code | 🔜 planned |
| 4 | Server: gRPC endpoints | Implement service handlers in `server/internal/api/grpc` calling domain services | 🔜 planned |
| 5 | Server: wiring | Start gRPC server (port 9090 by default), enable reflection and health | 🔜 planned |
| 6 | REST bridging (phase 2) | Option A: adopt grpc-gateway to serve `/api/*`; Option B: keep existing REST and validate parity | 🔜 planned |
| 7 | Client SDK: gRPC transport | Add transport selection (REST default for CI/local; gRPC for prod); implement gRPC client | 🔜 planned |
| 8 | Headers/metadata mapping | Map Idempotency-Key, Request-Id, traceparent ↔ gRPC metadata; update error mapping | 🔜 planned |
| 9 | Tests: parity & e2e | Add e2e tests for gRPC; assert parity with REST invariants; nightly gRPC job | 🔜 planned |
| 10 | Deployments | Expose gRPC port in docker-compose; health/readiness updates | 🔜 planned |

### Details per task

1) Proto module & tooling
- Add `buf.yaml` at repo root; module import path `github.com/mycelian/mycelian-memory`
- Add `buf.gen.yaml` to generate:
  - `protoc-gen-go`, `protoc-gen-go-grpc`
  - `protoc-gen-grpc-gateway` (optional in phase 2)
  - `protoc-gen-openapiv2` (optional)
- Make: `make protogen` target; CI step ensures clean generation

2) Define v1 protos
- Files: `proto/synapse/memory/v1/{health,users,vaults,memories,entries,contexts,search}.proto`
- Use resource-oriented RPCs: `CreateUser`, `GetUser`, `CreateVault`, `ListVaults`, `CreateMemory`, `ListMemories`, `GetMemory`, `DeleteMemory`, `AddEntry`, `ListEntries`, `GetEntry`, `UpdateEntryTags`, `PutContext`, `GetLatestContext`, `Search`.
- For gateway, annotate with `google.api.http` to mirror existing REST paths

3) Generate code
- Add `tools` or `Makefile` to pin plugin versions
- Commit generated Go stubs under `server/internal/proto` (or `gen/`)

4) Server: gRPC endpoints
- New package `server/internal/api/grpc` implementing v1 services; call existing domain services (`internal/core/*`)
- Interceptors: logging, tracing, validation, recovery

5) Server: wiring
- Start gRPC server alongside HTTP server (separate port); optional TLS flags
- Implement gRPC health via `grpc_health_v1`

6) REST bridging (phase 2)
- Option A (preferred): grpc-gateway serves `/api/*` paths from proto annotations, replacing Gorilla mux gradually
- Option B: keep current REST, assert parity in tests; plan migration later

7) Client SDK: gRPC transport
- Add a gRPC client in `client/internal/grpc/`; transport selectable via options/env
- Preserve the public client API; use gRPC under the hood when enabled

8) Headers/metadata mapping
- Map `Idempotency-Key`, `Request-Id`, `traceparent` to gRPC metadata; echo in responses where relevant
- Error mapping table: domain errors → `status.Code`; include details for structured errors

9) Tests: parity & e2e
- Unit tests for service handlers; e2e tests hitting gRPC; nightly job runs REST+gRPC parity suite
- Bench basic latency; validate streaming feasibility for future APIs

10) Deployments
- Update `deployments/docker/*.yml` to expose gRPC port; document `grpcurl`/health checks

### Definition of Done
- For each step: `go fmt ./... && go vet ./... && go test -race ./... && golangci-lint run && govulncheck ./...`
- Build server and client binaries; CI green
- REST behavior unchanged in phase 1; parity tests pass

### Conventional commits (suggested)
- build(proto): add buf workspace and codegen config
- feat(proto): define synapse.memory.v1 services and messages
- feat(server): implement gRPC services and start server on :9090
- feat(client): add gRPC transport option for SDK
- docs(api): generate OpenAPI from proto (optional) and document parity

### Risk note
What can go wrong: drift between REST and gRPC; header/metadata mismatch; partial parity. Mitigations: proto as the single contract; rest via gateway or thin wrappers; parity tests; interceptors unify headers/metadata; keep REST as CI default until parity proven.



Ran tool

Read file: internal/storage/real_spanner_integration_test.go

The repo contains four distinct "flavours" of tests; each is enabled in a different way.

────────────────────────────────────────
1.  Regular unit / fast-integration tests  
   • Location: almost every package.  
   • Build tags: none.  
   • Invocation:  
     ```bash
     # run & cache results
     go test ./...

     # force rebuild/rerun
     go test -count=1 ./...
     # or clear all cached results
     go clean -testcache
     ```
   • They use the Spanner *emulator* spun up by `TestMain` and a mocked Waviate client, so they run in <1 s per package.

────────────────────────────────────────
2.  Developer-environment end-to-end (E2E) tests  
   • Location: `dev_env_e2e_tests/` (single package).  
   • Build tag: `e2e` (`//go:build e2e`).  
   • Stack requirements:  
     – `docker compose up` → brings up `memory-service`, `indexer-prototype`, Weaviate, Spanner emulator, etc.  
     – Ollama container running locally with the specified embedding model pre-pulled.  
   • Invocation (local dev):  
     ```bash
     docker compose up -d              # ensure stack is healthy
     go test -tags=e2e ./dev_env_e2e_tests -v
     ```
   • The suite contains:  
     – Smoke tests (ingestion + search round-trip)  
     – Relevance tests (hybrid alpha sweep, tag filter, metadata filter)  
   • Runtime ~6-8 s on a warm dev stack; CI job normally finishes <1 min.

────────────────────────────────────────
3.  Invariant tests  
   • Directory: `internal/invariants/`  
   • Build tag: `invariants`.  
   • Purpose: assert runtime invariants against the Dockerised dev stack (schema, env, etc.).  
   • Run:  
     ```bash
     docker compose up -d
     go test -tags=invariants ./internal/invariants
     ```

────────────────────────────────────────
4.  "Real Spanner" cloud tests  
   • File: `internal/storage/real_spanner_integration_test.go`  
   • No build tag, but **skips unless** `REAL_SPANNER=true`.  
   • Needs a real Cloud Spanner instance + credentials:  
     ```bash
     REAL_SPANNER=true \
       GCP_PROJECT_ID=<proj> \
       SPANNER_INSTANCE_ID=<inst> \
       SPANNER_DATABASE_ID=<db> \
       go test -v ./internal/storage -run TestRealSpannerIntegration
     ```

────────────────────────────────────────
Quick cheat-sheet
────────────────────────────────────────
• Fast local check (what CI calls "unit"):  
  `go test ./...`  

• Full developer sweep (local stack running):  
  ```
  go clean -testcache          # optional
  go test ./...                # fast tests
  go test -tags=e2e ./dev_env_e2e_tests
  go test -tags=invariants ./internal/invariants
  ```

• CI matrix recommendation:  

| Job          | Command                                          | Notes                         |
| ------------ | ------------------------------------------------ | ----------------------------- |
| unit         | `go test -race ./...`                            | Runs on every PR              |
| e2e          | `go test -tags=e2e ./...`                        | Needs docker-compose services |
| invariants   | `go test -tags=invariants ./internal/invariants` | Fast; same stack as e2e       |
| spanner-real | optional                                         | Only in nightly build         |

That's the full landscape—you only run the flavours relevant to the change you're working on.
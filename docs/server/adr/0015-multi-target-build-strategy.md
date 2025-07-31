---
title: Multi-Target Build Strategy (Original)
status: Superseded
superseded_by: "0020-open-source-pivot.md"
date: 2025-07-23
---

# Context

Synapse Memory must run in three distinct environments:

| Build Target | Purpose | Relational DB | Vector Store | Orchestration |
|--------------|---------|---------------|--------------|--------------|
| cloud-dev    | CI, staging, manual QA | Cloud Spanner (Emulator) | Waviate (Docker) | docker-compose |
| cloud        | Production & pilot customers | Cloud Spanner (PG Interface) **or** PostgreSQL/Aurora | Waviate | Terraform/K8s |

# Decision

1. Introduce `BUILD_TARGET` env var (`local`, `cloud-dev`, `cloud`).
2. Config mapping table derives `DB_DRIVER` and `VECTOR_STORE` automatically; explicit overrides allowed.
3. CI matrix runs tests on `local` & `cloud-dev` to guarantee parity.
4. Factories in `internal/platform/factory/` select adapters by target.
5. The schema (`internal/storage/schema.sql`) remains single-source; adapters translate as needed.

# Consequences

• Developers can start backend with `make local` in <30 s, no Docker required.
• CI continues to use docker-compose to mirror current behaviour.
• Future cloud migrations (Aurora, AlloyDB) require only new adapter, not build-target change.

# Supersedes

None.

# References

* `memory-bank/activeContext.md` Phase 7 – Local Build Target. 
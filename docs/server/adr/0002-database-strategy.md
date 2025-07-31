---
title: Use Cloud Spanner with Emulator for Local Development
status: Superseded
superseded_by: "0016-relational-storage-abstraction.md"
date: 2025-07-19
---

# Context

Synapse Memory Service needs strong consistency and horizontal scalability. Google Cloud Spanner offers these but is costly for local development and CI. We currently use the **Spanner Emulator** in Docker to run the full stack locally.

# Decision

1. **Google Cloud Spanner** (regional) is the authoritative datastore for **dev, beta, gamma, and prod** stages.  
2. **Spanner Emulator (Docker)** is mandatory for **local development, unit tests, and CI**.  
3. The two environments are **complementary** and run identical DDL managed via the same migration files.  
4. Environment variables (`SPANNER_EMULATOR_HOST`, `ENV_STAGE`) switch targets.

# Consequences

• Local development cost is zero; onboarding requires only Docker.  
• Tests run fast and deterministically with no cloud dependency.  
• Production parity is high—same SQL dialect and gRPC API.  
• We accept that emulator lacks IAM/auth, so auth-related behaviour must be integration-tested separately.

# Open Questions

• Multi-region replication strategy for GA launch.  
• Backup & point-in-time-recovery procedure.

# References

* `docker-compose.yml` service `spanner-emulator`  
* `memory-bank/activeContext.md` – Docker architecture section  
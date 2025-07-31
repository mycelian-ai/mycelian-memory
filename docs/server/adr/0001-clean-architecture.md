---
title: Adopt Clean Architecture for Synapse Memory Service
status: Deprecated
superseded_by: 0003-transport-and-monorepo-strategy.md
date: 2025-07-19
---

> **Deprecated:** The project intentionally maintains a pragmatic layered design with REST (testing) and gRPC (production) in a monorepo. Clean-Architecture strict layering is **not** adopted. See ADR-0003.

# Context

Early iterations of the Synapse Memory Service mixed HTTP transport logic directly with business rules and Spanner data-access code.  The tight coupling made unit testing hard, introduced duplicated validation logic, and slowed feature additions.  The project now requires multiple transport layers (REST, gRPC) and a future switch from the Spanner emulator to managed Cloud Spanner.  A modular architecture is needed.

# Decision

Adopt **Clean Architecture** as the project's structural guideline:

* **API / Transport Layer** – thin handlers, request/response mapping, no business logic.
* **Domain Layer (core)** – pure Go, business rules, invariants, independent of frameworks.
* **Platform Layer** – infrastructure concerns (database, logging, config, HTTP helpers).
* **Cross-cutting abstractions** – interfaces defined in domain layer, implemented in platform layer.

Folder structure:
```
internal/
  api/       # transport (http, grpc)
  core/      # domain entities & services
  platform/  # database, logger, config, http helpers
```

# Consequences

• Unit tests can run without the HTTP stack or Spanner emulator (domain layer only).  
• New transports (e.g., gRPC) plug into the same domain services.  
• Infrastructure swaps (Spanner → Postgres) impact only platform implementations.  
• Developers follow a clear "dependencies point inwards" rule, reducing cyclic-import risk.

# References

* Decision logged in `memory-bank/decisionLog.md` on *2025-06-14*  
* `memory-bank/systemPatterns.md` – architecture diagram 
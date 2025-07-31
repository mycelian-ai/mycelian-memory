---
title: Monorepo with REST for Testing and gRPC for Production
status: Accepted
date: 2025-07-19
---

# Context

Synapse Memory Service serves both internal clients and an open-source SDK.  Fast iteration and determinism during invariant-driven development require an HTTP/REST surface that is easy to probe with `curl` and black-box tests.  In production we need stronger contract guarantees, schema evolution tooling, and efficient binary transport—hence gRPC.

The team also prefers a **single repository** so that API contracts, server, and test harnesses evolve together.

# Decision

1. **Monorepo**: Keep server, API contracts, test harnesses, and internal tooling in one Git repository.  
2. **Transport modes**  
   • **REST/JSON** – primary surface in local dev, CI, and invariant tests.  
   • **gRPC** – primary surface in dev/beta/gamma/prod deployments.  
3. Feature work must land in both transports, but REST handlers may thin-wrap gRPC stubs to minimise duplication.
4. Automated tests target REST first (speed, readability) and a nightly job validates the same invariants against gRPC.

# Consequences

• New contributors can explore the API with nothing but `curl`.  
• Production traffic benefits from gRPC performance, streaming, and strong contracts.  
• Keeping everything in a monorepo avoids version-drift between transports.  
• We accept slightly higher code surface (two transports) in exchange for clearer testing stories.

# Trade-offs Considered

*Separate repos* were rejected because sharing proto files, invariants, and test harnesses would become brittle.

# References

* Supersedes ADR-0001.  
* `memory-bank/activeContext.md` – REST endpoints listed under *Missing API Features*.  
* `proto/v1/*.proto` – initial gRPC service definitions. 
---
title: Public Memory API Contract & Versioning
status: Accepted
date: 2025-07-20
---

# Context

Synapse must expose a stable external interface usable by OSS SDKs and internal systems.  We maintain two transports—REST/JSON for local tests and gRPC for production—but they must share a single authoritative contract and clear evolution rules.

# Decision

1. **Single source of truth**: Protobuf (`memory-api` repo) defines all services and messages.  
2. **REST generation**: The REST/JSON surface is generated or thin-wrapped from the proto definitions to avoid drift.  
3. **Versioning**: Major version in the proto package path (e.g., `synapse.memory.v1`).  Additive changes may ship within a major; removals or breaking changes require a new major.  
4. **Change control**: Any contract change needs an ADR update plus changelog entry; server & SDK versions are bumped together.

# Consequences

• External developers can rely on SemVer-like stability.  
• Tooling (Buf, OpenAPI) can generate docs and client code automatically.  
• REST tests remain valid as they exercise the same underlying proto contract. 
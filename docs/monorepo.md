# Mycelian Monorepo Layout

## Why we structure it this way

- **Hide server code** – All backend code lives in `internal/`, so nothing outside the repo can import it. This lets us refactor without breaking users.
- **One repo, many modules** – The server and each language SDK live side-by-side in one repository and are linked with a `go.work` (for Go) or their native lock files.  Each SDK is versioned and released independently.
- **Clear layers** – Code is grouped by its job so each concern stays separate and easy to swap.
- **Tiny cmd/** – Binaries just wire things together; real work stays in libraries so tests run fast and releases stay small.

## Folder map

```text
mycelian/
  go.work

  server/                 # backend Go module
    go.mod
    cmd/
      mycelian-memory-service/   # main.go entry-point
      mycelian-service-tools/    # CLI tool (formerly memoryctl)
    internal/
      core/               # basic types and rules
      api/                # HTTP handlers and routes
      storage/            # database and persistence layer
      search/             # vector search and indexing

  clients/                # all language SDKs (one sub-dir per language)
    go/
      go.mod
      client/             # public Go client SDK
      mcp/                # MCP server functionality
        handlers/         # MCP tool handlers
        server.go         # MCP server implementation
      cmd/                # binary entry points
        mycelian-mcp-server/
      internal/           # helpers not exported
    python/
      pyproject.toml
      src/mycelian/
    ts/
      package.json
      src/
    rust/
      Cargo.toml
      src/

  api/                    # protobuf + OpenAPI contracts (single source of truth)
  deployments/            # helm charts, docker-compose, terraform
  tools/                  # scripts and helper CLIs
  docs/                   # ADRs, onboarding, this file
```

## Versioning

Each component follows **Semantic Versioning** (`vX.Y.Z`).
- Bump **Z** for backwards-compatible fixes.
- Bump **Y** for new features that keep the public contract intact.
- Bump **X** only when a change breaks compatibility.

⚠️ **Pre-1.0.0**: any **Y** bump may introduce breaking changes.

The server and every language SDK (`clients/go`, `clients/python`, `clients/ts`, `clients/rust`, …) are tagged and published **independently**, but every server release records the _minimum compatible_ SDK versions in `VERSIONS.md`.

CI blocks a server tag unless:
1. Updated stubs compile for every language.
2. All per-language SDK test suites pass.

This guarantees contract coherence across languages.

| Component                | Tag example            | Publication target        |
|--------------------------|------------------------|---------------------------|
| Go SDK                   | `clients/go/v0.4.0`    | pkg.go.dev module         |
| Python SDK               | `clients/python/v0.2.1`| PyPI (`mycelian-memory`)  |
| TypeScript SDK           | `clients/ts/v0.3.0`    | npm (`@mycelian/memory`)  |
| Rust SDK                 | `clients/rust/v0.2.0`  | crates.io (`mycelian`)    |
| Server                   | `server/v0.8.0`        | ghcr.io Docker image      |

> CI ensures that any change to files under `api/` regenerates stubs, runs each language’s test suite, and bumps the affected SDK’s version tag **before** the server can be released.

## Adding features

- Public API changes begin with updates to the contracts in `api/`.
- Generated SDK code lives under `clients/generated/<lang>` (if used) and is committed so CI can diff changes.
- Hand-written helpers or ergonomic layers live in the same language folder (e.g. `clients/go/client/`, `clients/python/src/mycelian/`).
- Promote shared functionality out of `internal/` only after a design review.

## Staying tidy

- Keep generated code in separate sub-directories and exclude them from manual edits.
- Update this document whenever the directory structure changes.

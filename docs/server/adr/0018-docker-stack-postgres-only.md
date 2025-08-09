---
title: Consolidate Docker stack to Postgres-only (remove Spanner/SQLite stacks)
status: Accepted
date: 2025-08-09
---

## Context

Following ADR 0017 (remove Spanner and SQLite backends), multiple Docker stacks and emulator setup scripts became obsolete. Keeping them increases cognitive load and maintenance cost for developers and CI. We standardize on a single Postgres-based stack across local/dev/CI to match production.

## Decision

- Consolidate to a single Postgres docker-compose stack for the backend and supporting services.
- Remove Spanner emulator and SQLite docker-compose files, scripts, and Makefile targets.

## Scope

- Delete `deployments/docker/docker-compose.spanner.yml` and `deployments/docker/docker-compose.sqlite.yml`.
- Remove `server/scripts/docker-setup/*` (Spanner emulator init/health scripts and runbook).
- Update Makefiles to use only Postgres targets (`run-postgres`, `backend-postgres-up`, etc.); drop `run-spanner`, `backend-sqlite-up`, and related help text.
- Update developer docs to reference Postgres-only flow (quickstarts, profiles, logs).

## Operational Changes

- Local development: `make backend-postgres-up` (root) or `make run-postgres` (server) brings up the stack.
- Health checks: use `/api/health` for memory-service and Weaviate `/v1/meta`. No emulator setup needed.

## Makefile Targets (final)

- Root: `backend-postgres-up`, `backend-status`, `backend-logs`, `backend-down`.
- Server: `run-postgres`, `docker-stop`, `docker-status`, `docker-logs`.

## Migration Notes

- Any private scripts referencing `docker-compose.sqlite.yml` or Spanner emulator must be removed or updated.
- CI should rely on the Postgres compose stack only.

## Status

- Compose and scripts removed; Makefiles updated; developer docs updated; test suite green.



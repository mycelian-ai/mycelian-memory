# ADR-010: Consolidated PostgreSQL Docker Stack

**Status**: Accepted
**Date**: 2025-08-09

## Context

Following ADR-009 (PostgreSQL-only backend), multiple Docker stacks and database setup scripts became obsolete. Keeping them increases cognitive load and maintenance cost for developers and CI. We standardize on a single PostgreSQL-based stack across local/dev/CI to match production.

## Decision

- Consolidate to single PostgreSQL docker-compose stack for backend and supporting services
- Remove legacy database docker-compose files, scripts, and Makefile targets

## Scope

- Delete legacy docker-compose files for other database systems
- Remove database-specific setup scripts and runbook references
- Update Makefiles to use only PostgreSQL targets; drop legacy database targets
- Update developer docs to reference PostgreSQL-only flow

## Operational Changes

### Local Development
- `make backend-up` brings up the PostgreSQL stack
- Health checks use `/api/health` for memory-service and vector search endpoints
- No additional database setup needed

### Makefile Targets (Final)
- Root: `backend-up`, `backend-status`, `backend-logs`, `backend-down`
- Server: `run-postgres`, `docker-stop`, `docker-status`, `docker-logs`

## Consequences

### Positive Consequences
- Simplified developer onboarding with single stack
- Reduced maintenance overhead for Docker configurations
- Consistent environment between local development and production
- Faster local setup and teardown

### Negative Consequences
- Less flexibility for developers preferring different local setups
- Single point of failure if PostgreSQL stack has issues

## Migration Notes

- Any scripts referencing legacy compose files must be updated
- CI should rely on the PostgreSQL compose stack only
- Developer documentation updated to reflect single-stack workflow

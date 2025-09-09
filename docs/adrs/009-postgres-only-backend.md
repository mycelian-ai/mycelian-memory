# ADR-009: PostgreSQL-Only Backend

**Status**: Accepted
**Date**: 2025-08-09

## Context

Previous architecture supported multiple storage backends (Spanner/SQLite/PostgreSQL) to provide flexibility. After migration to PostgreSQL locally and in production, maintaining multiple storage paths created code drift, higher CI costs, and complicated operational runbooks.

## Decision

- Remove all non-PostgreSQL storage codepaths and tooling from the repository
- Make PostgreSQL the only supported storage backend for all environments

## Scope

- Delete legacy database adapters, tests, and schema files
- Remove associated docker-compose stacks and setup scripts
- Simplify configuration: default and only supported `DB_DRIVER=postgres`
- Update factory to error on unsupported drivers
- Adjust tests to run on PostgreSQL only

## Consequences

### Positive Consequences
- Reduced complexity in code, tests, and operations
- Single SQL dialect (PostgreSQL) simplifies development and maintenance
- Eliminates configuration drift between environments
- Faster CI pipeline with fewer test matrix combinations

### Negative Consequences
- Any future multi-backend support will require new ADR(s) and isolated modules
- Less flexibility for users with different database preferences

## Implementation Summary

### Server Changes
- Removed legacy database adapters and associated test files
- Updated configuration to PostgreSQL-only; removed legacy config options
- Updated factory to return error for unsupported drivers
- Skipped/removed tests dependent on legacy database harnesses
- Maintained search/outbox functionality (PostgreSQL as source of truth)

### Tooling & Compose Changes
- Deleted legacy docker-compose files and setup scripts
- Simplified Makefiles to PostgreSQL-only targets
- Updated developer documentation to reflect single-stack approach

## Rollback

Reintroducing other database backends would require new adapters, config keys, test harnesses, and compose stacks. This ADR documents the removal and rationale for future reference.

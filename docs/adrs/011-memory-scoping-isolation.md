# ADR-011: Memory Scoping and Isolation

**Status**: Accepted
**Date**: 2025-08-10

## Context

As the system scales to support multiple organizations and collaborative workflows, we need a hierarchical scoping model for memory management. The current flat structure with direct user-memory relationships doesn't support organizational boundaries, project-based collaboration, or granular access control needed for enterprise deployments.

Current limitations:
- No organizational boundaries between different customer deployments
- No project-level grouping for collaborative work
- Limited access control granularity
- `user_id` field assumes only human users, not AI agents or automated scripts
- Insufficient audit trail for multi-actor scenarios

## Decision

### 1. Hierarchical Scoping Model

Implement three-tier organizational hierarchy:
- **Organization** (`org_id`): Top-level tenant boundary
- **Project** (`project_id`): Collaborative workspace within organization
- **Vault** (`vault_id`): Memory container within project

### 2. Actor-Based Identity Model

Replace `user_id` with `actor_id` to support diverse entity types:
- **Human users**: Interactive operators with authentication
- **AI agents**: Autonomous systems performing memory operations
- **Scripts/Services**: Automated processes writing or reading memories

### 3. Project-Scoped Access Control

- Actors are assigned to projects with specific permissions
- Each actor receives project-specific API keys
- Permission levels: `read-only` or `read-write`
- Projects implicitly own all contained memory vaults

### 4. Primary Key Structure

Update entity relationships:
```
memories: (org_id, project_id, vault_id, memory_id)
entries:  (org_id, project_id, vault_id, memory_id, entry_id)
contexts: (org_id, project_id, vault_id, memory_id, context_id)
```

### 5. API Key Scoping

- API keys scoped to `(org_id, project_id, actor_id)`
- Keys grant access only to memories within assigned project
- Cross-project access requires multiple API keys

## Consequences

### Positive Consequences
- **Clear tenant isolation**: Organizations cannot access each other's data
- **Project collaboration**: Multiple actors can work within shared project scope
- **Granular permissions**: Read-only vs read-write access per actor per project
- **Enhanced auditability**: Actor-based logging supports compliance and debugging
- **Future-proof identity**: Supports humans, AI agents, and automated systems
- **Enterprise readiness**: Multi-tenant architecture suitable for SaaS deployment

### Negative Consequences
- **Migration complexity**: Existing single-tenant data needs migration to new schema
- **API key management**: More complex key provisioning and rotation
- **Increased query complexity**: All operations must validate org/project scope
- **Storage overhead**: Additional ID fields in every table

## Implementation Notes

### Schema Migration
- Add `org_id`, `project_id` columns to all memory-related tables
- Migrate existing data to default organization/project
- Update all foreign key constraints to include new composite keys
- Rename `user_id` → `actor_id` across codebase

### API Changes
- All endpoints require org/project context in URL paths
- API key validation checks org/project scope
- Response filtering enforces project boundaries
- Audit logs capture actor identity for all operations

### Permission Model
```json
{
  "actor_id": "agent-summarizer-v2",
  "org_id": "mycelian-corp",
  "project_id": "customer-support-kb",
  "permissions": ["memory:read", "memory:write", "entry:create"],
  "api_key": "mk_proj_abc123..."
}
```

### Backward Compatibility
- Existing single-tenant deployments map to default org/project
- Legacy `user_id` references preserved during transition period
- API versioning supports gradual client migration

## Alternatives Considered

### Alternative 1: Flat Multi-Tenancy
**Description**: Single org_id field without project hierarchy
**Why rejected**: Insufficient granularity for project-based collaboration within organizations

### Alternative 2: Role-Based Access Control (RBAC)
**Description**: Complex role hierarchy instead of simple read/write permissions
**Why rejected**: Over-engineering for current requirements; can be added later

### Alternative 3: External Authorization Service
**Description**: Delegate all access control to external service
**Why rejected**: Adds complexity and latency; prefer built-in project scoping

## Migration Plan

1. **Schema Update**: Add new columns with default values
2. **Data Migration**: Populate org/project IDs for existing records
3. **API Versioning**: Deploy v2 endpoints with new scoping
4. **Client Updates**: Migrate SDKs and tools to new API structure
5. **Legacy Cleanup**: Remove old endpoints after transition period

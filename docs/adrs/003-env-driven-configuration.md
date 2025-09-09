# ADR-003: Environment-Driven Configuration (12-Factor)

**Status**: Accepted
**Date**: 2025-07-21

## Context

Services run in containers across local Docker, CI, and production environments. Static config files introduce state drift and secrets leakage; environment variables fit the 12-factor model and integrate with cloud secret management systems.

## Decision

1. All runtime configuration provided via environment variables parsed with `kelseyhightower/envconfig`
2. Required keys include: `DB_CONNECTION_STRING`, `PORT`, `ENV_STAGE`, `VECTOR_SEARCH_URL`
3. Defaults exist only for local development (e.g., `PORT=8080`); CI ensures required keys are set for other stages

## Consequences

### Positive Consequences
- Containers are stateless and portable across environments
- Secret injection (cloud-specific) is seamless
- Configuration changes require no image rebuild—only env-var updates in deployment
- Eliminates config file sync issues between environments

### Negative Consequences
- Configuration scattered across deployment files rather than centralized
- Environment-specific debugging requires access to deployment environment
- No compile-time validation of configuration structure

## Implementation Notes

### Configuration Structure
- Use struct tags for environment variable mapping
- Provide reasonable defaults for development
- Validate required fields at startup
- Log configuration (with secrets redacted) at startup

### Environment Categories
- **Local Development**: Minimal required config with sensible defaults
- **CI/Testing**: All required fields explicitly set
- **Production**: All config via encrypted environment variables

### Secret Management
- Use cloud provider secret management for sensitive values
- Never commit secrets to version control
- Rotate secrets according to security policy
- Audit secret access and usage

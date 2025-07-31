# Docker Setup Scripts

This directory contains Docker-related setup scripts following Docker best practices.

## Scripts

### `docker-setup-schema.sh`
- **Purpose**: Initializes Spanner emulator database and schema
- **Usage**: Called automatically by `schema-setup` service in docker-compose.yml
- **Functions**:
  - Waits for Spanner emulator readiness
  - Configures gcloud for emulator connection
  - Creates Spanner instance and database
  - Applies DDL schema from `internal/storage/schema.sql`
  - Validates schema creation

### `docker-health-check.sh`
- **Purpose**: Performs deep SQL health checks on the database
- **Usage**: Called automatically by `db-health-check` service in docker-compose.yml
- **Functions**:
  - Tests table existence and structure
  - Validates table relationships (interleaving)
  - Checks column schema accuracy
  - Confirms data access with SELECT queries

## Docker Integration

These scripts are mounted as read-only volumes in docker-compose.yml:

```yaml
schema-setup:
  volumes:
    - ./scripts/docker-setup/docker-setup-schema.sh:/scripts/setup-schema.sh:ro
  command: ["sh", "/scripts/setup-schema.sh"]

db-health-check:
  volumes:
    - ./scripts/docker-setup/docker-health-check.sh:/scripts/health-check.sh:ro
  command: ["sh", "/scripts/health-check.sh"]
```

## Benefits

- **Clean Separation**: Docker Compose focuses on orchestration, scripts handle logic
- **Maintainable**: Easy to modify script behavior without touching compose files
- **Testable**: Scripts can be tested independently of Docker containers
- **Readable**: Clear shell script structure with proper error handling 
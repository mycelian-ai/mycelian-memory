# Synapse Memory Backend - Tools

This directory contains operational tools for the Synapse Memory Backend project.

## Schema Manager

The schema manager tool helps manage Spanner database schema operations.

### Installation

```bash
cd tools/schema-manager
go build -o schema-manager .
```

### Usage

#### Create Tables
Create tables from the schema file:

```bash
# For Spanner emulator (local development)
./schema-manager \
  -project test-project \
  -instance test-instance \
  -database memory-backend \
  -emulator localhost:9010 \
  -operation create-tables

# For real Spanner (production/staging)
./schema-manager \
  -project your-gcp-project \
  -instance your-spanner-instance \
  -database memory-backend \
  -credentials /path/to/service-account.json \
  -operation create-tables
```

#### Validate Schema
Check if the current database schema matches the expected schema:

```bash
./schema-manager \
  -project test-project \
  -instance test-instance \
  -database memory-backend \
  -emulator localhost:9010 \
  -operation validate-schema
```

#### Drop Tables (⚠️ DANGEROUS)
Drop all tables (requires manual confirmation):

```bash
./schema-manager \
  -project test-project \
  -instance test-instance \
  -database memory-backend \
  -emulator localhost:9010 \
  -operation drop-tables
```

**WARNING**: This operation is irreversible and will delete all data!

### Flags

- `-project`: Google Cloud Project ID (required)
- `-instance`: Spanner Instance ID (required)  
- `-database`: Spanner Database ID (required)
- `-credentials`: Path to service account credentials file (optional for emulator)
- `-emulator`: Spanner emulator host (e.g., localhost:9010)
- `-schema`: Path to schema file (default: `internal/storage/schema.sql`)
- `-operation`: Operation to perform (`create-tables`, `drop-tables`, `validate-schema`)

### Operations

#### create-tables
- Reads DDL statements from the schema file
- Applies them to the target database
- Idempotent - safe to run multiple times
- Creates tables, indexes, and constraints

#### validate-schema  
- Compares current database schema with expected schema file
- Reports missing tables
- Helps verify schema migrations

#### drop-tables
- **MANUAL CONFIRMATION REQUIRED**
- Lists all tables to be dropped
- Requires typing "DELETE ALL TABLES" to confirm
- Drops tables in reverse dependency order
- **IRREVERSIBLE OPERATION**

### Safety Features

1. **Manual Confirmation**: Drop operations require explicit confirmation
2. **Emulator Support**: Safe testing with local Spanner emulator
3. **Detailed Logging**: Clear output showing what operations are performed
4. **Error Handling**: Proper error messages and rollback on failures

### Development

When adding new tables or modifying schema:

1. Update `internal/storage/schema.sql`
2. Test with emulator: `make start-emulator` (if available)
3. Validate: `./schema-manager -operation validate-schema ...`  
4. Apply: `./schema-manager -operation create-tables ...`

### Future Enhancements

- Schema migrations and versioning
- Backup before destructive operations
- Rollback capabilities
- Production deployment safety checks 
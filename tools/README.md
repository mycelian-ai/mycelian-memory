# Mycelian Memory Backend - Tools

This directory contains operational tools for the Mycelian Memory Backend project.

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

## Vault Memory Deleter

A Python utility to safely delete all memories within a vault by vault ID.

### Installation

No installation required - uses standard Python libraries.

### Usage

#### Preview Mode (Safe)
Preview what will be deleted without making changes:

```bash
python delete_vault_memories.py <vault_id>
```

#### Delete Memories (Keep Vault)
Delete all memories but keep the vault structure:

```bash
python delete_vault_memories.py <vault_id> --yes
```

#### Delete Everything (Including Vault)
Delete all memories and the vault itself:

```bash
python delete_vault_memories.py <vault_id> --delete-vault --yes
```

#### Custom Database Path
Use a different SQLite database file:

```bash
python delete_vault_memories.py <vault_id> --db-path /path/to/memory.db
```

### Examples

```bash
# Preview deletion (safe)
python tools/delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de

# Delete memories with confirmation prompt
python tools/delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de

# Delete everything including vault, skip confirmation
python tools/delete_vault_memories.py 97db1a27-695b-4bf3-bbd1-a00c6d4501de --delete-vault --yes
```

### Features

- **Safe Preview**: Shows exactly what will be deleted before doing anything
- **Confirmation Required**: Interactive confirmation unless `--yes` flag is used
- **Proper Order**: Deletes in correct order to maintain referential integrity
- **Detailed Output**: Shows counts and descriptions of what was deleted
- **Error Handling**: Clear error messages and graceful failure handling

### Safety Features

1. **Preview Mode**: Default behavior shows what would be deleted
2. **Manual Confirmation**: Requires typing "DELETE" to confirm
3. **Vault Protection**: By default keeps the vault, only deletes with `--delete-vault`
4. **Clear Output**: Shows exactly what will be affected
5. **Database Validation**: Checks if database file exists before proceeding 
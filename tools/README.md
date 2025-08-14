# Mycelian Memory Backend - Tools

This directory contains operational tools for the Mycelian Memory Backend project.

## Schema Manager

Spanner schema manager has been removed. Postgres is the supported backend; the server applies its schema from `server/internal/storage/postgres/schema.sql` during startup. Any former Spanner emulator instructions no longer apply.

### Safety Features

1. **Manual Confirmation**: Drop operations require explicit confirmation
2. **Emulator Support**: Safe testing with local Spanner emulator
3. **Detailed Logging**: Clear output showing what operations are performed
4. **Error Handling**: Proper error messages and rollback on failures

### Development

When adding new tables or modifying schema:

1. Update `server/internal/storage/postgres/schema.sql`
2. Launch the stack: `make -C server run-postgres`
3. The server will apply schema on startup.

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
# Spanner Emulator Runbook

Quick commands for fast iteration loops.

## **Essential Commands**

### Start
```bash
docker-compose up -d
```

### Health Check
```bash
docker-compose run --rm --no-deps db-health-check
```

### Pause/Resume
```bash
docker-compose pause    # Pause
docker-compose unpause  # Resume
```

### Stop/Start
```bash
docker-compose stop     # Stop
docker-compose start    # Start
```

### Delete
```bash
docker-compose down
```

### Clean Restart
```bash
docker-compose down && docker-compose up -d
```

## **Status & Logs**

### Status
```bash
docker-compose ps
```

### Logs
```bash
docker-compose logs db-health-check
docker-compose logs schema-setup
docker-compose logs spanner-emulator
```

## **Raw Queries with gcloud CLI**

### Prerequisites
```bash
# Install gcloud CLI if not already installed
# https://cloud.google.com/sdk/docs/install

# Verify installation
gcloud --version
```

### Configure for Emulator
```bash
# Set emulator endpoint (run this in each new terminal session)
export SPANNER_EMULATOR_HOST=localhost:9010

# Verify emulator is running
docker-compose ps spanner-emulator
```

### Basic Query Commands
```bash
# List instances
gcloud spanner instances list --project=synapse-memory

# List databases  
gcloud spanner databases list --instance=synapse-instance --project=synapse-memory

# Execute SQL query
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT * FROM Users LIMIT 5"
```

### Example Queries

#### **User Data**
```bash
# Count all users
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT COUNT(*) as user_count FROM Users"

# List all users with details
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT UserID, Email, DisplayName, Status, CreationTime FROM Users ORDER BY CreationTime DESC"

# Find specific user by email
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT * FROM Users WHERE Email = 'test@example.com'"
```

#### **Memory Data**
```bash
# Count memories per user
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT UserID, COUNT(*) as memory_count FROM Memories WHERE DeletionScheduledTime IS NULL GROUP BY UserID"

# List recent memories
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT MemoryID, UserID, Title, MemoryType, CreationTime FROM Memories WHERE DeletionScheduledTime IS NULL ORDER BY CreationTime DESC LIMIT 10"
```

#### **Entry Data**
```bash
# Count entries per memory
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT MemoryID, COUNT(*) as entry_count FROM MemoryEntries WHERE DeletionScheduledTime IS NULL GROUP BY MemoryID"

# List recent entries with content preview
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT EntryID, MemoryID, SUBSTR(RawEntry, 1, 50) as content_preview, CreationTime FROM MemoryEntries WHERE DeletionScheduledTime IS NULL ORDER BY CreationTime DESC LIMIT 10"

# Find corrected entries
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT EntryID, MemoryID, CorrectionTime, CorrectionReason FROM MemoryEntries WHERE CorrectionTime IS NOT NULL ORDER BY CorrectionTime DESC"
```

#### **Schema Inspection**
```bash
# List all tables
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT table_name FROM information_schema.tables WHERE table_catalog = '' AND table_schema = ''"

# Describe Users table structure
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT column_name, spanner_type, is_nullable FROM information_schema.columns WHERE table_name = 'Users' ORDER BY ordinal_position"

# Describe Memories table structure  
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT column_name, spanner_type, is_nullable FROM information_schema.columns WHERE table_name = 'Memories' ORDER BY ordinal_position"

# Describe MemoryEntries table structure
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT column_name, spanner_type, is_nullable FROM information_schema.columns WHERE table_name = 'MemoryEntries' ORDER BY ordinal_position"
```

### Advanced Queries

#### **Data Integrity Checks**
```bash
# Check for orphaned memories (users that don't exist)
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT m.MemoryID, m.UserID FROM Memories m LEFT JOIN Users u ON m.UserID = u.UserID WHERE u.UserID IS NULL"

# Check for orphaned entries (memories that don't exist)
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT e.EntryID, e.MemoryID FROM MemoryEntries e LEFT JOIN Memories m ON e.MemoryID = m.MemoryID WHERE m.MemoryID IS NULL"

# Check for entries with invalid correction references
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT e1.EntryID, e1.OriginalEntryCreationTime FROM MemoryEntries e1 LEFT JOIN MemoryEntries e2 ON e1.OriginalEntryCreationTime = e2.CreationTime AND e1.MemoryID = e2.MemoryID WHERE e1.OriginalEntryCreationTime IS NOT NULL AND e2.EntryID IS NULL"
```

#### **Performance Analysis**
```bash
# Find largest entries by content size
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT EntryID, MemoryID, LENGTH(RawEntry) as content_size FROM MemoryEntries ORDER BY content_size DESC LIMIT 10"

# Find users with most memories
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT u.UserID, u.Email, COUNT(m.MemoryID) as memory_count FROM Users u LEFT JOIN Memories m ON u.UserID = m.UserID WHERE m.DeletionScheduledTime IS NULL GROUP BY u.UserID, u.Email ORDER BY memory_count DESC"
```

### Query Shortcuts

#### **Create alias for common command**
```bash
# Add to your ~/.bashrc or ~/.zshrc
alias spanner-query='gcloud spanner databases execute-sql synapse-memory --instance=synapse-instance --project=synapse-memory --sql'

# Usage after reloading shell:
spanner-query "SELECT COUNT(*) FROM Users"
```

#### **Multi-line queries**
```bash
# For complex queries, use heredoc
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="$(cat << 'EOF'
SELECT 
  u.Email,
  COUNT(DISTINCT m.MemoryID) as memories,
  COUNT(DISTINCT e.EntryID) as entries,
  MAX(e.CreationTime) as last_entry
FROM Users u
LEFT JOIN Memories m ON u.UserID = m.UserID AND m.DeletionScheduledTime IS NULL
LEFT JOIN MemoryEntries e ON m.MemoryID = e.MemoryID AND e.DeletionScheduledTime IS NULL
GROUP BY u.UserID, u.Email
ORDER BY last_entry DESC
EOF
)"
```

### Troubleshooting gcloud Queries

#### **Common Issues**
```bash
# If you get "connection refused" error:
export SPANNER_EMULATOR_HOST=localhost:9010
docker-compose ps spanner-emulator  # Verify emulator is running

# If you get "project not found" error:
# The emulator doesn't require real GCP authentication, but the project/instance/database names must match exactly

# If queries hang:
# Check if emulator is healthy
docker-compose logs spanner-emulator

# Reset emulator if needed
docker-compose restart spanner-emulator
```

#### **Query Output Formatting**
```bash
# JSON output
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --format=json \
  --sql="SELECT * FROM Users LIMIT 1"

# CSV output  
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --format="csv" \
  --sql="SELECT UserID, Email FROM Users"

# Table output (default)
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --format="table" \
  --sql="SELECT COUNT(*) as total_users FROM Users"
```

## **Fast Iteration Loop**

```bash
# 1. Start
docker-compose up -d

# 2. Test changes
docker-compose run --rm --no-deps db-health-check

# 3. Query data directly
export SPANNER_EMULATOR_HOST=localhost:9010
gcloud spanner databases execute-sql synapse-memory \
  --instance=synapse-instance \
  --project=synapse-memory \
  --sql="SELECT COUNT(*) FROM Users"

# 4. Clean restart if needed
docker-compose down && docker-compose up -d

# 5. Stop when done
docker-compose down
```

## **Troubleshooting**

### Force Clean Everything
```bash
docker-compose down --rmi all --volumes --remove-orphans
```

### Nuclear Reset
```bash
docker-compose down --rmi all --volumes
docker system prune -f
``` 
#!/bin/bash
set -e

echo "🔍 Running deep SQL health checks..."

# Install netcat for any connection testing if needed
apk add --no-cache netcat-openbsd

# Configure gcloud for emulator
echo "🔧 Configuring gcloud for health checks..."
gcloud config set api_endpoint_overrides/spanner http://spanner-emulator:9020/
gcloud config set auth/disable_credentials true
gcloud config set project synapse-memory

# Deep health checks with actual SQL queries
echo "📊 Checking table existence..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT table_name FROM information_schema.tables WHERE table_catalog = '' AND table_schema = '' ORDER BY table_name"

echo "🧪 Testing Users table..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT COUNT(*) as user_count FROM Users"

echo "🧪 Testing Memories table..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT COUNT(*) as memory_count FROM Memories"

echo "🧪 Testing MemoryEntries table..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT COUNT(*) as entry_count FROM MemoryEntries"

echo "🧪 Testing table relationships..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT t.table_name, t.parent_table_name FROM information_schema.tables t WHERE t.table_catalog = '' AND t.table_schema = '' ORDER BY t.table_name"

echo "🧪 Testing Users table schema..."
gcloud spanner databases execute-sql synapse-memory \
    --instance=synapse-instance \
    --sql="SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'Users' ORDER BY ordinal_position"

echo "✅ All deep health checks passed! Database is fully operational." 
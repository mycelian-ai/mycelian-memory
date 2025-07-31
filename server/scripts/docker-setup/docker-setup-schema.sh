#!/bin/bash
set -e

echo "🚀 Starting Spanner emulator schema setup..."

# Install netcat for connection testing
apk add --no-cache netcat-openbsd

# Wait for emulator to be ready with timeout
echo "⏳ Waiting for emulator to be ready..."
timeout=60
count=0
while [ $count -lt $timeout ]; do
  if nc -z spanner-emulator 9020 2>/dev/null; then
    echo "✅ Emulator is ready!"
    break
  fi
  sleep 2
  count=$((count + 2))
  echo "⏳ Still waiting... ($count/$timeout seconds)"
done

if [ $count -ge $timeout ]; then
  echo "❌ Emulator failed to start within $timeout seconds"
  exit 1
fi

# Configure gcloud for emulator
echo "🔧 Configuring gcloud for emulator..."
gcloud config set api_endpoint_overrides/spanner http://spanner-emulator:9020/
gcloud config set auth/disable_credentials true
gcloud config set project synapse-memory

# Create instance and database
echo "🏗️ Creating Spanner instance..."
gcloud spanner instances create synapse-instance --config=emulator-config --description="Synapse Memory Instance" --nodes=1

echo "🗄️ Creating database..."
gcloud spanner databases create synapse-memory --instance=synapse-instance --ddl-file=/schema/schema.sql

# Validate schema with SQL query
echo "🧪 Validating schema with SQL query..."
gcloud spanner databases execute-sql synapse-memory --instance=synapse-instance --sql="SELECT table_name FROM information_schema.tables WHERE table_catalog = '' AND table_schema = '' ORDER BY table_name"

echo "✅ Database and schema setup complete!" 
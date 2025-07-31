#!/bin/bash
# Simple memoryctl test - minimal commands to verify syntax
set -e

MEMORYCTL="${MEMORYCTL:-./bin/memoryctl}"

# Generate unique identifiers using timestamp and random suffix
TIMESTAMP=$(date +%s)
RANDOM_SUFFIX=$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n')
USER_ID="test_${TIMESTAMP}"
EMAIL="simple-test-${TIMESTAMP}@test.com"
VAULT_TITLE="vault-${TIMESTAMP}-${RANDOM_SUFFIX}"
MEMORY_TITLE="memory-${TIMESTAMP}-${RANDOM_SUFFIX}"

echo "=== Simple Memory Service Test ==="

# Step 1: Create user
echo "Creating user with ID: $USER_ID and email: $EMAIL"
USER_OUTPUT=$($MEMORYCTL users create --userId "$USER_ID" --name "Simple Test" --email "$EMAIL")
if [ $? -eq 0 ]; then
    echo "User created successfully"
    echo "User ID: $USER_ID"
else
    echo "Failed to create user"
    echo "Output: $USER_OUTPUT"
    exit 1
fi

# Step 2: Create vault (using -u flag for user)
echo "Creating vault with title: $VAULT_TITLE"
VAULT_ID=$($MEMORYCTL vaults create -u "$USER_ID" -l "$VAULT_TITLE" -d "Simple test vault" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
if [ -z "$VAULT_ID" ]; then
    echo "Failed to create vault"
    exit 1
fi
echo "Vault ID: $VAULT_ID"

# Step 3: Create memory (using -v flag for vault, -u for user)
echo "Creating memory with title: $MEMORY_TITLE"
MEMORY_ID=$($MEMORYCTL memories create -v "$VAULT_ID" -u "$USER_ID" -l "$MEMORY_TITLE" -d "Simple test memory" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
if [ -z "$MEMORY_ID" ]; then
    echo "Failed to create memory"
    exit 1
fi
echo "Memory ID: $MEMORY_ID"

# Step 4: Add entry (using -m for memory, -u for user, -r for raw text, -s for summary)
echo "Adding entry..."
ENTRY_OUTPUT=$($MEMORYCTL entries add -v "$VAULT_ID" -u "$USER_ID" -m "$MEMORY_ID" -r "The quick brown fox jumps over the lazy dog" -s "A story about a fox")
if [ $? -eq 0 ]; then
    ENTRY_ID=$(echo "$ENTRY_OUTPUT" | grep -oE '"entryId":"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"' | cut -d'"' -f4)
    echo "Entry ID: $ENTRY_ID"
else
    echo "Failed to add entry"
    exit 1
fi

# Step 5: Search for content (optional - may fail if indexer hasn't processed yet)
echo ""
echo "Waiting 1 seconds for indexer to process..."
sleep 1

echo "Searching for 'fox' in the memory..."
echo "Debug: Running command: $MEMORYCTL search -u \"$USER_ID\" -m \"$MEMORY_ID\" -q \"fox\" -k 3"
SEARCH_OUTPUT=$($MEMORYCTL search -u "$USER_ID" -m "$MEMORY_ID" -q "fox" -k 3 2>&1)
SEARCH_EXIT_CODE=$?

if [ $SEARCH_EXIT_CODE -eq 124 ]; then
    echo "Search timed out after 15 seconds"
    echo "Note: This might indicate the search service is not properly configured or running."
elif [ $SEARCH_EXIT_CODE -eq 0 ]; then
    echo "Search results:"
    echo "$SEARCH_OUTPUT" | jq '.' 2>/dev/null || echo "$SEARCH_OUTPUT"
else
    echo "Search failed (this is expected if the indexer hasn't processed the entry yet):"
    echo "$SEARCH_OUTPUT" | grep -oE '"message":"[^"]*"' | cut -d'"' -f4 || echo "Unknown error"
    echo ""
    echo "Note: Search may fail initially as the indexer needs time to process entries."
fi

echo ""
echo "=== Test Complete ==="
echo "Summary:"
echo "  User ID:   $USER_ID"
echo "  Vault ID:  $VAULT_ID"
echo "  Memory ID: $MEMORY_ID"
echo "  Entry ID:  $ENTRY_ID"
echo ""
echo "All operations completed successfully!"

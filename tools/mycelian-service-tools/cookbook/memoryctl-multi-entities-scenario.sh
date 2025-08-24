#!/bin/bash
# Service tools E2E Cookbook
# This script demonstrates a complete workflow from user creation to search
# Run with: ./scripts/memoryctl-e2e-cookbook.sh

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MEMORYCTL="${MEMORYCTL:-./bin/mycelian-service-tools}"
API_ENDPOINT="${API_ENDPOINT:-http://localhost:11545}"

echo -e "${BLUE}=== Memory Service E2E Test Cookbook ===${NC}"
echo -e "${YELLOW}Using API endpoint: ${API_ENDPOINT}${NC}"
echo -e "${YELLOW}Using service tools: ${MEMORYCTL}${NC}"
echo ""

# Generate unique identifiers using timestamp and random suffix
TIMESTAMP=$(date +%s)
RANDOM_SUFFIX=$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n')
USER_ID="e2e_${TIMESTAMP}"
EMAIL="e2e-test-${TIMESTAMP}@test.com"

# Step 1: Create a user
echo -e "${GREEN}Step 1: Creating user...${NC}"
USER_OUTPUT=$($MEMORYCTL users create --userId "$USER_ID" --name "Test User" --email "$EMAIL")
if [ $? -eq 0 ]; then
    echo "User created successfully"
    echo -e "${YELLOW}Created user ID: ${USER_ID}${NC}"
else
    echo "Failed to create user"
    echo "Output: $USER_OUTPUT"
    exit 1
fi
echo ""

# Step 2: Create vaults
echo -e "${GREEN}Step 2: Creating vaults...${NC}"

# Personal vault
echo -e "${BLUE}Creating personal vault...${NC}"
VAULT_PERSONAL_TITLE="personal-${TIMESTAMP}-${RANDOM_SUFFIX}"
VAULT_PERSONAL_OUTPUT=$($MEMORYCTL vaults create -u "$USER_ID" -l "$VAULT_PERSONAL_TITLE" -d "Personal memories and notes")
echo "$VAULT_PERSONAL_OUTPUT"
VAULT_PERSONAL_ID=$(echo "$VAULT_PERSONAL_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created personal vault ID: ${VAULT_PERSONAL_ID}${NC}"

# Work vault
echo -e "${BLUE}Creating work vault...${NC}"
VAULT_WORK_TITLE="work-${TIMESTAMP}-${RANDOM_SUFFIX}"
VAULT_WORK_OUTPUT=$($MEMORYCTL vaults create -u "$USER_ID" -l "$VAULT_WORK_TITLE" -d "Work-related memories and projects")
echo "$VAULT_WORK_OUTPUT"
VAULT_WORK_ID=$(echo "$VAULT_WORK_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created work vault ID: ${VAULT_WORK_ID}${NC}"
echo ""

# Step 3: Create memories in personal vault
echo -e "${GREEN}Step 3: Creating memories in personal vault...${NC}"

MEMORY_PERSONAL_1_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -l "vacation-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Amazing trip to Japan with family")
MEMORY_PERSONAL_1_ID=$(echo "$MEMORY_PERSONAL_1_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Summer Vacation 2024 (${MEMORY_PERSONAL_1_ID})${NC}"

MEMORY_PERSONAL_2_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -l "recipes-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Collection of favorite recipes and cooking tips")
MEMORY_PERSONAL_2_ID=$(echo "$MEMORY_PERSONAL_2_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Cooking Recipes (${MEMORY_PERSONAL_2_ID})${NC}"

MEMORY_PERSONAL_3_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -l "books-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Notes and highlights from books I've read")
MEMORY_PERSONAL_3_ID=$(echo "$MEMORY_PERSONAL_3_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Book Notes (${MEMORY_PERSONAL_3_ID})${NC}"
echo ""

# Step 4: Create memories in work vault
echo -e "${GREEN}Step 4: Creating memories in work vault...${NC}"

MEMORY_WORK_1_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -l "project-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Technical documentation and architecture decisions for Project Alpha")
MEMORY_WORK_1_ID=$(echo "$MEMORY_WORK_1_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Project Alpha Documentation (${MEMORY_WORK_1_ID})${NC}"

MEMORY_WORK_2_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -l "meetings-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Important meeting notes and action items")
MEMORY_WORK_2_ID=$(echo "$MEMORY_WORK_2_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Meeting Notes (${MEMORY_WORK_2_ID})${NC}"

MEMORY_WORK_3_OUTPUT=$($MEMORYCTL memories create \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -l "code-${TIMESTAMP}-${RANDOM_SUFFIX}" \
  -d "Useful code snippets and programming patterns")
MEMORY_WORK_3_ID=$(echo "$MEMORY_WORK_3_OUTPUT" | grep -oE '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | tail -1)
echo -e "${YELLOW}Created memory: Code Snippets (${MEMORY_WORK_3_ID})${NC}"
echo ""

# Step 5: Add entries to personal memories
echo -e "${GREEN}Step 5: Adding entries to personal memories...${NC}"

# Entries for Summer Vacation
echo -e "${BLUE}Adding entries to Summer Vacation memory...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_1_ID" \
  -r "Day 1: Arrived in Tokyo. The city is incredible! Visited Shibuya crossing and had amazing ramen at a local shop." \
  -s "Tokyo arrival and first impressions"

$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_1_ID" \
  -r "Day 3: Took the bullet train to Kyoto. Visited Fushimi Inari shrine - thousands of red torii gates. Absolutely breathtaking!" \
  -s "Kyoto visit and Fushimi Inari shrine"

$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_1_ID" \
  -r "Day 5: Mount Fuji day trip. The weather was perfect and we could see the summit clearly. Had traditional onsen experience." \
  -s "Mount Fuji trip and onsen"

# Entries for Cooking Recipes
echo -e "${BLUE}Adding entries to Cooking Recipes memory...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_2_ID" \
  -r "Grandma's Chocolate Chip Cookies: 2 cups flour, 1 cup butter, 3/4 cup sugar, 2 eggs, 1 tsp vanilla, chocolate chips. Bake at 375F for 10-12 minutes." \
  -s "Grandma's chocolate chip cookie recipe"

$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_2_ID" \
  -r "Perfect Pasta Carbonara: Use guanciale not bacon, pecorino romano cheese, fresh eggs, black pepper. No cream! The key is tempering the eggs." \
  -s "Authentic pasta carbonara recipe"

# Entries for Book Notes
echo -e "${BLUE}Adding entries to Book Notes memory...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_3_ID" \
  -r "Atomic Habits by James Clear: Small changes compound over time. Focus on systems not goals. Make good habits obvious, attractive, easy, and satisfying." \
  -s "Atomic Habits key takeaways"

$MEMORYCTL entries add \
  -v "$VAULT_PERSONAL_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_3_ID" \
  -r "Deep Work by Cal Newport: Shallow work keeps us busy but doesn't create value. Schedule deep work blocks. Embrace boredom to train focus." \
  -s "Deep Work principles"

echo ""

# Step 6: Add entries to work memories
echo -e "${GREEN}Step 6: Adding entries to work memories...${NC}"

# Entries for Project Alpha
echo -e "${BLUE}Adding entries to Project Alpha Documentation...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_1_ID" \
  -r "Architecture Decision: Chose microservices over monolith for better scalability. Using Kubernetes for orchestration and gRPC for inter-service communication." \
  -s "Microservices architecture decision"

$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_1_ID" \
  -r "API Design: RESTful endpoints with OpenAPI 3.0 specification. Implemented versioning strategy using URL path (v1, v2). Rate limiting at 1000 requests per minute." \
  -s "API design and versioning strategy"

# Entries for Meeting Notes
echo -e "${BLUE}Adding entries to Meeting Notes memory...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_2_ID" \
  -r "Q1 Planning Meeting: Priorities include migrating to cloud, improving CI/CD pipeline, and reducing technical debt. Timeline: 3 months." \
  -s "Q1 planning priorities"

$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_2_ID" \
  -r "Client Review: Positive feedback on new features. Requested improvements to dashboard performance. Action: Profile and optimize database queries." \
  -s "Client review feedback and actions"

# Entries for Code Snippets
echo -e "${BLUE}Adding entries to Code Snippets memory...${NC}"
$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_3_ID" \
  -r "Go error handling pattern: if err != nil { return fmt.Errorf(\"failed to process: %w\", err) }. Always wrap errors with context." \
  -s "Go error handling best practice"

$MEMORYCTL entries add \
  -v "$VAULT_WORK_ID" \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_3_ID" \
  -r "Docker multi-stage build: Use alpine for smaller images. COPY --from=builder pattern. Remember to use non-root user in final stage." \
  -s "Docker multi-stage build pattern"

echo ""

# Step 7: Wait for indexer and search across memories
echo -e "${GREEN}Step 7: Waiting for indexer to process entries...${NC}"
echo "Waiting 3 seconds for indexer to process..."
sleep 3

echo -e "${GREEN}Performing searches...${NC}"

# Search in specific memories
echo -e "${BLUE}Searching in Summer Vacation memory for 'Tokyo'...${NC}"
$MEMORYCTL search \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_1_ID" \
  -q "Tokyo" \
  -k 5

echo ""
echo -e "${BLUE}Searching in Cooking Recipes memory for 'chocolate'...${NC}"
$MEMORYCTL search \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_2_ID" \
  -q "chocolate" \
  -k 5

echo ""
echo -e "${BLUE}Searching in Project Alpha Documentation for 'microservices'...${NC}"
$MEMORYCTL search \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_1_ID" \
  -q "microservices" \
  -k 5

echo ""
echo -e "${BLUE}Searching in Meeting Notes for 'performance'...${NC}"
$MEMORYCTL search \
  -u "$USER_ID" \
  -m "$MEMORY_WORK_2_ID" \
  -q "performance" \
  -k 5

echo ""
echo -e "${BLUE}Searching in Book Notes for 'habits'...${NC}"
$MEMORYCTL search \
  -u "$USER_ID" \
  -m "$MEMORY_PERSONAL_3_ID" \
  -q "habits" \
  -k 5

echo ""
echo -e "${GREEN}=== E2E Test Complete! ===${NC}"
echo ""
echo -e "${YELLOW}Summary of created resources:${NC}"
echo "User ID: $USER_ID"
echo "Personal Vault ID: $VAULT_PERSONAL_ID"
echo "Work Vault ID: $VAULT_WORK_ID"
echo ""
echo -e "${YELLOW}Personal Memories:${NC}"
echo "  - Summer Vacation 2024: $MEMORY_PERSONAL_1_ID"
echo "  - Cooking Recipes: $MEMORY_PERSONAL_2_ID"
echo "  - Book Notes: $MEMORY_PERSONAL_3_ID"
echo ""
echo -e "${YELLOW}Work Memories:${NC}"
echo "  - Project Alpha Documentation: $MEMORY_WORK_1_ID"
echo "  - Meeting Notes: $MEMORY_WORK_2_ID"
echo "  - Code Snippets: $MEMORY_WORK_3_ID"
echo ""

# Optional: Export variables for further testing
echo -e "${BLUE}Exporting variables for further testing...${NC}"
echo "export USER_ID=$USER_ID"
echo "export VAULT_PERSONAL_ID=$VAULT_PERSONAL_ID"
echo "export VAULT_WORK_ID=$VAULT_WORK_ID"
echo "export MEMORY_PERSONAL_1_ID=$MEMORY_PERSONAL_1_ID"
echo "export MEMORY_WORK_1_ID=$MEMORY_WORK_1_ID"

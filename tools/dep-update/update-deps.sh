#!/bin/bash
set -e

echo "🔍 Memory Backend - Direct Dependency Update Script"
echo "==================================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to run command and show output
run_cmd() {
    echo -e "${BLUE}▶ $1${NC}"
    eval $1
    echo
}

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}⚠️  Warning: You have uncommitted changes. Consider committing first.${NC}"
    echo
fi

echo -e "${GREEN}📊 Checking current direct dependency status...${NC}"
echo

# Check outdated DIRECT dependencies in main module
echo -e "${BLUE}=== Main Module Direct Dependencies ===${NC}"
main_outdated=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}}{{end}}{{end}}' all | wc -l | tr -d ' ')
if [ "$main_outdated" -eq 0 ]; then
    echo -e "${GREEN}✅ All direct dependencies are up to date!${NC}"
    go list -m -f '{{if not .Indirect}}  ✅ {{.Path}} {{.Version}}{{end}}' all
else
    echo -e "${YELLOW}📦 $main_outdated direct dependencies to update:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if .Update}}  ⚠️  {{.Path}} {{.Version}} → {{.Update.Version}}{{end}}{{end}}' all
fi

echo

# Check outdated DIRECT dependencies in tools module
echo -e "${BLUE}=== Tools Module Direct Dependencies ===${NC}"
cd tools
tools_outdated=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}}{{end}}{{end}}' all | wc -l | tr -d ' ')
if [ "$tools_outdated" -eq 0 ]; then
    echo -e "${GREEN}✅ All direct dependencies are up to date!${NC}"
    go list -m -f '{{if not .Indirect}}  ✅ {{.Path}} {{.Version}}{{end}}' all
else
    echo -e "${YELLOW}🔨 $tools_outdated direct dependencies to update:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if .Update}}  ⚠️  {{.Path}} {{.Version}} → {{.Update.Version}}{{end}}{{end}}' all
fi
cd ..

total_updates=$((main_outdated + tools_outdated))

if [ "$total_updates" -eq 0 ]; then
    echo -e "${GREEN}🎉 All direct dependencies are already up to date! Nothing to do.${NC}"
    exit 0
fi

echo
echo -e "${GREEN}🔧 Updating $total_updates direct dependencies...${NC}"
echo

# Update main module - only direct dependencies
if [ "$main_outdated" -gt 0 ]; then
    echo -e "${BLUE}=== Updating Main Module Direct Dependencies ===${NC}"

    # Get list of outdated direct dependencies and update them specifically
    outdated_deps=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}}@{{.Update.Version}}{{end}}{{end}}' all)
    if [ -n "$outdated_deps" ]; then
        for dep in $outdated_deps; do
            run_cmd "go get $dep"
        done
    fi
    run_cmd "go mod tidy"
else
    echo -e "${GREEN}✅ Main module direct dependencies already up to date${NC}"
fi

echo

# Update tools module - only direct dependencies
if [ "$tools_outdated" -gt 0 ]; then
    echo -e "${BLUE}=== Updating Tools Module Direct Dependencies ===${NC}"
    cd tools

    # Get list of outdated direct dependencies and update them specifically
    outdated_deps=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}}@{{.Update.Version}}{{end}}{{end}}' all)
    if [ -n "$outdated_deps" ]; then
        for dep in $outdated_deps; do
            run_cmd "go get $dep"
        done
    fi
    run_cmd "go mod tidy"
    cd ..
else
    echo -e "${GREEN}✅ Tools module direct dependencies already up to date${NC}"
fi

echo -e "${GREEN}🧪 Running tests to verify updates...${NC}"
echo

# Test main module
echo -e "${BLUE}=== Testing Main Module ===${NC}"
run_cmd "go test ./..."

# Test tools compilation
echo -e "${BLUE}=== Testing Tools Module ===${NC}"
cd tools
run_cmd "go build ./..."
cd ..

echo -e "${GREEN}✅ Direct dependency updates completed successfully!${NC}"
echo

# Show final status
echo -e "${BLUE}=== Final Status ===${NC}"
main_direct=$(go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$" | wc -l | tr -d ' ')
tools_direct=$(cd tools && go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$" | wc -l | tr -d ' ')
echo "Main module: $main_direct direct dependencies (all current)"
echo "Tools module: $tools_direct direct dependencies (all current)"

echo
echo -e "${GREEN}🎉 Ready to commit your direct dependency updates!${NC}"
echo -e "${YELLOW}💡 Suggested commit message:${NC}"
echo "   chore: update direct dependencies to latest versions"
echo "   "
echo "   - Updated $total_updates direct dependencies"
echo "   - All tests passing"
echo "   - Transitive dependencies updated automatically"

echo
echo -e "${CYAN}💡 Note: Only direct dependencies were updated.${NC}"
echo -e "${CYAN}   Go modules automatically manages transitive dependencies.${NC}"

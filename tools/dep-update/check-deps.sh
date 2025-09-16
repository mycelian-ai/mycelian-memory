#!/bin/bash

echo "🔍 Memory Backend - Direct Dependency Health Check"
echo "================================================="
echo

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}📊 Main Module Status:${NC}"
echo -e "${CYAN}Current Go version: $(go version | cut -d' ' -f3)${NC}"
echo

# Check main module DIRECT dependencies only
echo -e "${BLUE}Direct dependencies in main module:${NC}"
outdated_main=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}} {{.Version}} [{{.Update.Version}}]{{end}}{{end}}' all | wc -l | tr -d ' ')

if [ "$outdated_main" -eq 0 ]; then
    echo -e "${GREEN}✅ All direct dependencies are up to date!${NC}"
    go list -m -f '{{if not .Indirect}}  ✅ {{.Path}} {{.Version}}{{end}}' all
else
    echo -e "${YELLOW}📦 $outdated_main outdated direct dependencies:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if .Update}}  ⚠️  {{.Path}} {{.Version}} → {{.Update.Version}}{{end}}{{end}}' all
    echo
    echo -e "${GREEN}Up to date direct dependencies:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if not .Update}}  ✅ {{.Path}} {{.Version}}{{end}}{{end}}' all
fi

echo

# Check tools module DIRECT dependencies only
echo -e "${BLUE}🔧 Tools Module Status:${NC}"
cd tools
outdated_tools=$(go list -u -m -f '{{if not .Indirect}}{{if .Update}}{{.Path}} {{.Version}} [{{.Update.Version}}]{{end}}{{end}}' all | wc -l | tr -d ' ')

if [ "$outdated_tools" -eq 0 ]; then
    echo -e "${GREEN}✅ All direct dependencies are up to date!${NC}"
    go list -m -f '{{if not .Indirect}}  ✅ {{.Path}} {{.Version}}{{end}}' all
else
    echo -e "${YELLOW}🔨 $outdated_tools outdated direct dependencies:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if .Update}}  ⚠️  {{.Path}} {{.Version}} → {{.Update.Version}}{{end}}{{end}}' all
    echo
    echo -e "${GREEN}Up to date direct dependencies:${NC}"
    go list -u -m -f '{{if not .Indirect}}{{if not .Update}}  ✅ {{.Path}} {{.Version}}{{end}}{{end}}' all
fi
cd ..

echo

# Summary
total_outdated=$((outdated_main + outdated_tools))
if [ "$total_outdated" -eq 0 ]; then
    echo -e "${GREEN}🎉 All direct dependencies are current! Great job!${NC}"
else
    echo -e "${YELLOW}💡 Summary: $total_outdated direct dependencies need updates${NC}"
    echo "   Run: ./scripts/update-deps.sh to update direct dependencies"
fi

echo
echo -e "${BLUE}📈 Direct Dependency Counts:${NC}"
main_direct=$(go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$" | wc -l | tr -d ' ')
tools_direct=$(cd tools && go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$" | wc -l | tr -d ' ')
echo "   Main module: $main_direct direct dependencies"
echo "   Tools module: $tools_direct direct dependencies"

echo
echo -e "${CYAN}💡 Note: Only checking direct dependencies you actually import.${NC}"
echo -e "${CYAN}   Transitive dependencies are managed automatically by Go modules.${NC}"

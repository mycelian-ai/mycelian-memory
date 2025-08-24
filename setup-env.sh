#!/bin/bash
# This script should be sourced, not executed: source setup-env.sh

# Get the project root (where this script is located)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "Project root: $PROJECT_ROOT"
cd "$PROJECT_ROOT"

echo "Setting up workspace-wide Python virtual environment..."

# Create venv if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python -m venv venv
fi

# Activate the virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# Install dependencies
echo "Installing dependencies..."
pip install -r requirements.txt

# Build mycelianCli binary
echo "Building mycelianCli binary..."
cd "$PROJECT_ROOT"
if make build-mycelian-cli; then
    echo "✅ Built mycelianCli binary at $PROJECT_ROOT/bin/mycelianCli"
else
    echo "❌ Failed to build mycelianCli binary"
    exit 1
fi

# Return to project root
cd "$PROJECT_ROOT"

echo "✅ Workspace environment ready and activated!"
echo "Current directory: $(pwd)"
echo ""
echo "Available tools:"
echo "  • Benchmarker: cd tools/benchmarker && python benchmark_runner.py --help"
echo "  • MycelianCli: ./bin/mycelianCli --help"
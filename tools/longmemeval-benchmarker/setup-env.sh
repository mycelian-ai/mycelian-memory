#!/bin/bash
# This script should be sourced, not executed: source setup-env.sh

# Get the longmemeval-benchmarker directory (where this script is located)
LONGMEMEVAL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "LongMemEval benchmarker directory: $LONGMEMEVAL_DIR"
cd "$LONGMEMEVAL_DIR"

echo "Setting up LongMemEval benchmarker-specific Python virtual environment..."

# Create venv if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python -m venv venv
fi

# Activate the virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# Install dependencies
if [ -f "requirements.txt" ]; then
    echo "Installing dependencies..."
    pip install -r requirements.txt
else
    echo "No requirements.txt found, skipping dependency installation"
fi

# Add Go binaries to PATH for MCP server access
export PATH="$(go env GOBIN 2>/dev/null || go env GOPATH)/bin:$PATH"

# Set PYTHONPATH for src/ directory structure
export PYTHONPATH=src

echo "✅ LongMemEval benchmarker environment ready and activated!"
echo "Current directory: $(pwd)"
echo ""
echo "Available commands:"
echo "  • Run benchmark: python -m runner config.real.toml --workers 2"
echo "  • Quick test: python -m runner config.quick.toml"
echo "  • Debug mode: python -m runner config.real.toml --workers 2 --debug"
echo "  • Create subset: python create_subset.py"
echo "  • Run tests: python -m pytest tests/"

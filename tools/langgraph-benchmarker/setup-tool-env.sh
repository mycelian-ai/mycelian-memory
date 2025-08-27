#!/bin/bash
# This script should be sourced, not executed: source setup-tool-env.sh

# Get the tool directory (where this script is located)
TOOL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TOOL_DIR/../.." && pwd)"

echo "Setting up LangGraph Benchmarker environment..."
echo "Tool directory: $TOOL_DIR"
echo "Project root: $PROJECT_ROOT"

cd "$TOOL_DIR"

# Create tool-specific venv if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment for LangGraph Benchmarker..."
    python -m venv venv
fi

# Activate the virtual environment
echo "Activating LangGraph Benchmarker virtual environment..."
source venv/bin/activate

# Install dependencies
echo "Installing LangGraph Benchmarker dependencies..."
pip install -r requirements.txt

echo "✅ LangGraph Benchmarker environment ready and activated!"
echo "Current directory: $(pwd)"
echo ""
echo "Usage:"
echo "  • Quick test: python -m src.main --config config.toml --mode quick-test"
echo "  • Full benchmark: python -m src.main --config config.toml --mode full"
echo ""
echo "Configuration:"
echo "  • Copy config.example.toml to config.toml and customize"
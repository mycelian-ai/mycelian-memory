#!/bin/bash
set -e

echo "🚀 Setting up LongMemEval Benchmarker environment..."

# Detect OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
else
    echo "❌ Unsupported OS: $OSTYPE"
    exit 1
fi

echo "📍 Detected OS: $OS"

# Install pyenv if not already installed
if ! command -v pyenv &> /dev/null; then
    echo "📦 Installing pyenv..."
    if [[ "$OS" == "macos" ]]; then
        if ! command -v brew &> /dev/null; then
            echo "❌ Homebrew not found. Please install Homebrew first:"
            echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            exit 1
        fi
        brew install pyenv
    else
        curl https://pyenv.run | bash
        # Add pyenv to PATH for this session
        export PATH="$HOME/.pyenv/bin:$PATH"
        eval "$(pyenv init -)"
        eval "$(pyenv virtualenv-init -)"
    fi
else
    echo "✅ pyenv already installed"
fi

# Install Poetry if not already installed
if ! command -v poetry &> /dev/null; then
    echo "📦 Installing Poetry..."
    curl -sSL https://install.python-poetry.org | python3 -
    # Add Poetry to PATH for this session
    export PATH="$HOME/.local/bin:$PATH"
else
    echo "✅ Poetry already installed"
fi

# Install Python 3.11.9 if not already installed
echo "🐍 Setting up Python 3.11.9..."
if ! pyenv versions | grep -q "3.11.9"; then
    echo "📦 Installing Python 3.11.9..."
    pyenv install 3.11.9
else
    echo "✅ Python 3.11.9 already installed"
fi

# Set local Python version
echo "🔧 Setting Python 3.11.9 as local version..."
pyenv local 3.11.9

# Install dependencies with Poetry
echo "📦 Installing dependencies with Poetry..."
poetry install

echo ""
echo "✅ Setup complete!"
echo ""
echo "🎯 To activate the environment:"
echo "   poetry shell"
echo ""
echo "🚀 To run the benchmarker:"
echo "   poetry shell"
echo "   python -m src.orchestrator config.toml --auto --workers 3"
echo ""

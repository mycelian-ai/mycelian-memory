# LongMemEval Integration

A clean, bottom-up integration of LongMemEval benchmarking with Mycelian Memory. Starting fresh with minimal complexity.

## Philosophy

- **Start Simple**: Begin with one working conversation before scaling
- **Incremental**: Add features one at a time, testing each step
- **Clean Architecture**: Avoid over-engineering from the start
- **MCP First**: Use Mycelian Memory's MCP interface directly

## Current Status

🚧 **Starting Fresh** - All previous complex code has been removed.

## Quick Start

```bash
# Install dependencies
pip install -e .

# Set environment variables
export OPENAI_API_KEY="your-key-here"

# Run a simple test (when implemented)
python -m src.main --test
```

## Project Structure

```
longmemeval-integration/
├── src/                    # Main implementation
├── test/data/              # Test datasets
├── docs/                   # Documentation
├── requirements.txt        # Dependencies
└── README.md              # This file
```

## Next Steps

1. **Create basic MCP client connection**
2. **Implement single conversation processing**
3. **Add memory building for one session**
4. **Add question answering**
5. **Add basic evaluation**

## Development

This is a fresh start - no legacy code to maintain or refactor. Build exactly what you need, step by step.

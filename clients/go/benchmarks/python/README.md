# Synapse Memory Client Benchmarks

This directory contains benchmarks and tests for the Synapse Memory Client, focusing on testing the integration with Claude and the Synapse memory system.

## Overview

The benchmark suite includes:

- **SessionSimulator**: Simulates conversations with Claude, handling context management and tool calls.
- **MSC Dataset Loader**: Loads and processes the MSC (Multi-Session Chat) dataset for testing.
- **System Prompt Builder**: Constructs system prompts for the Claude model.
- **Test Suite**: Comprehensive tests for all components.
- **Note-Taker Semantics**: See [docs/design/msc_dataset_note_taker.md](../../docs/design/msc_dataset_note_taker.md) for the canonical mapping between dataset speaker labels and Synapse roles, and for the ingestion rules followed by the benchmark *note-taker* agent.

## Key Features

### Context Management

The `SessionSimulator` includes an advanced context management system that:

- Updates context every 5 messages
- Handles critical updates after important tool calls
- Ensures context is saved at the end of each session
- Maintains conversation history and state

For detailed documentation, see [docs/context_management.md](docs/context_management.md).

## Running Tests

### Prerequisites

- Python 3.8+
- Dependencies from `requirements.txt`
- Anthropic API key in `ANTHROPIC_API_KEY` environment variable

### Running All Tests

```bash
# Install dependencies
pip install -r requirements.txt

# Run all tests
pytest benchmarks/python/tests/
```

### Running Specific Tests

```bash
# Run context management tests
pytest benchmarks/python/tests/test_session_simulator_context.py -v

# Run helper method tests
pytest benchmarks/python/tests/test_session_simulator_helpers.py -v

# Run integration tests
pytest benchmarks/python/tests/test_session_simulator.py -v
```

```
source venv/bin/activate && export PATH="$(go env GOBIN 2>/dev/null || go env GOPATH)/bin:$PATH" 
```

```
python -m benchmarks.python.benchmark_runner \                                                
  --synapse-url http://localhost:8080 \
  --conversations 1 \
  --tracker-file tracker.json \
  --model-name claude-sonnet-4-20250514\
  ingest > benchmark.log 2>&1 &
```

```
./.venv/bin/python -m benchmarks.python.benchmark_runner --synapse-url http://localhost:8080 validate-tools
```

```
python -m benchmarks.python.benchmark_runner --validate-prompts --synapse-url http://localhost:8080 
```
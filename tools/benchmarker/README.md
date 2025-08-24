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

## Quick Setup

### Environment Setup (Recommended)

Use the provided setup script to automatically configure your environment:

```bash
# Run this from the PROJECT ROOT (/path/to/mycelian-memory/)
source setup-env.sh
```

This script will:
- Create and activate a workspace-wide Python virtual environment in the project root
- Install all dependencies from `requirements.txt`
- Build the latest `synapse` CLI binary
- Configure `PATH` to prioritize the local synapse binary
- Keep you in the project root with access to all tools

### Prerequisites

- Python 3.8+
- Go 1.19+ (for building the synapse CLI)
- Anthropic API key in `ANTHROPIC_API_KEY` environment variable
- Synapse Memory Service running (e.g., `http://localhost:8080`)

### User Management

The benchmark client automatically tries to use the default `local_user` created by the backend. If `local_user` is not available, it falls back to creating a fresh benchmark user via the CLI. This provides better performance while maintaining test isolation when needed.

## Running Tests

**Note**: Always run the environment setup first from the project root:
```bash
# From PROJECT ROOT (/path/to/mycelian-memory/)
source setup-env.sh
```

### Running All Tests

```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there
pytest tests/
```

### Running Specific Tests

```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

# Run context management tests
pytest tests/test_session_simulator_context.py -v

# Run helper method tests
pytest tests/test_session_simulator_helpers.py -v

# Run integration tests
pytest tests/test_session_simulator.py -v
```

## Running Benchmarks

**Note**: All benchmark commands run from the `tools/benchmarker/` directory after setup.

### Quick Tool Validation

```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

# List available Synapse CLI commands (fast)
synapse --help

# Get MCP tools schema
synapse get-tools-schema

# Run integration test (slow - tests all tools end-to-end)
python benchmark_runner.py --synapse-url http://localhost:8080 validate-tools
```

### Full Benchmark Suite

#### Ingest Phase (Load MSC dataset)
```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

python benchmark_runner.py \
  --synapse-url http://localhost:8080 \
  --conversations 1 \
  --tracker-file tracker.json \
  --model-name claude-3-5-sonnet-20241022 \
  ingest
```

#### Evaluation Phase (Test recall)
```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

python benchmark_runner.py \
  --synapse-url http://localhost:8080 \
  --tracker-file tracker.json \
  --model-name claude-3-5-sonnet-20241022 \
  eval
```

#### Both Phases
```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

python benchmark_runner.py \
  --synapse-url http://localhost:8080 \
  --conversations 1 \
  --tracker-file tracker.json \
  --model-name claude-3-5-sonnet-20241022 \
  both
```

### Prompt Validation

```bash
# From tools/benchmarker/ directory
cd tools/benchmarker/  # if not already there

python benchmark_runner.py --validate-prompts --synapse-url http://localhost:8080
```
# LongMemEval Benchmarker

The benchmarker allows us to run the LongMemEval benchmark using Mycelian Memory as the backend. Please refer to the benchmarker [design doc](../../docs/designs/langgraph_longmemeval_benchmarker.md) to learn more.

LongMemEval is designed to test not just the performance of the memory system but also its scale. The large version contains 500 questions. Each question is composed of 500 sessions. Each session contains turns between a user and an assistant. To evaluate real world performance, we must store memories as a production agent will. This creates scaling challenges. Answering a single question of smaller dataset with ~50 sessions requires ~60 mins with Claude Haiku 3.5. Hence, the benchmarker supports sharding by questions and processing each question in parallel. **IMPORTANT**: we DO NOT parallelize sessions inside a question as the Memory Agent must build chronological context.

Note: Running many questions in parallel can hit provider rate limits. The sampler script allows extracting a representative subset from `longmemeval_{s/m}.json` files.

## ⚠️ IMPORTANT: Use the Orchestrator

**The Huey Orchestrator is the authoritative and recommended way to run the benchmark.** It provides:
- Task queue management with automatic retries
- Progress tracking and resumability
- Parallel processing with multiple workers
- Real-time monitoring with Rich UI
- Crash recovery and session-level resume

All other runner scripts are deprecated and should not be used.

## Quick Setup

**One-command setup:**
```bash
cd longmemeval-benchmarker
./setup.sh
```

This script will:
- Install pyenv (Python version management)
- Install Poetry (dependency management)
- Install Python 3.11.9 for this project (requires Python 3.10+)
- Install all dependencies
- Set up the environment

**Then run the benchmarker:**
```bash
# All commands must be run from the longmemeval-benchmarker directory
poetry run python -m src.orchestrator config/config.smoke.toml --auto -w 5
```

## Prerequisites

1. **Start Mycelian services** (from repo root):
   ```bash
   # Start the backend memory service
   make start-dev-mycelian-server

   # Start the MCP server for benchmarker communication
   make start-mcp-streamable-server
   ```

2. **Provider credentials**: See Model Provider Setup section below for detailed instructions.

## Model Provider Setup

The benchmarker supports multiple LLM providers. Choose one or more providers and configure authentication:

### OpenAI
```bash
# Add to ~/.zshrc or ~/.bashrc
export OPENAI_API_KEY="sk-proj-..."
```
Get your API key from: https://platform.openai.com/api-keys

### Vertex AI (Google Cloud)
```bash
# 1. Login to Google Cloud
gcloud auth login

# 2. Set your project
gcloud config set project YOUR_PROJECT_ID

# 3. Set up Application Default Credentials
gcloud auth application-default login

# 4. Add to ~/.zshrc or ~/.bashrc
export VERTEX_AI_PROJECT_ID="your-project-id"
```

### OpenRouter
```bash
# Add to ~/.zshrc or ~/.bashrc
export OPENROUTER_API_KEY="sk-or-v1-..."
```
Get your API key from: https://openrouter.ai/keys

### Model Configuration Format

In your config.toml, specify models using the format `provider:model_name`:

```toml
[models]
# OpenAI models (default provider if no prefix)
agent = "gpt-5-nano-2025-08-07"           # Defaults to OpenAI
agent = "openai:gpt-5-nano-2025-08-07"    # Explicit OpenAI

# Vertex AI models (requires prefix)
agent = "vertex-ai:gemini-2.5-flash-lite"
agent = "vertex-ai:gemini-1.5-pro"

# OpenRouter models (requires prefix)
agent = "openrouter:openai/gpt-5-nano-2025-08-07"
agent = "openrouter:google/gemini-pro-1.5"

# Mix providers for agent and QA
agent = "vertex-ai:gemini-2.5-flash-lite"
qa = "openrouter:openai/gpt-5-nano-2025-08-07"
```

## Dataset Setup

You can either use the full LongMemEval dataset or work with the included sample:

### Option 1: Use included sample dataset
The benchmarker includes a sample dataset file `longmemeval_0a995998.json` for testing.

### Option 2: Full LongMemEval dataset
```bash
# Clone LongMemEval and place datasets under data/
git clone https://github.com/xiaowu0162/LongMemEval.git
cd LongMemEval
mkdir -p data
# Ensure one of these exists after you download/unpack their release:
#   data/longmemeval_s.json
#   data/longmemeval_m.json
#   data/longmemeval_oracle.json
```

Then set `dataset_file_path` in your TOML config to point to the dataset file.

## Creating a sample dataset

For testing, you can create a smaller subset from the full LongMemEval dataset:

```bash
cd longmemeval-benchmarker

# Option 1: Point to directory (auto-finds longmemeval_s.json, longmemeval_m.json, etc.)
python -m src.lme_sampler /path/to/LongMemEval --num-questions 10 --copy-as-s

# Option 2: Point directly to specific file
python -m src.lme_sampler /path/to/LongMemEval/data/longmemeval_s.json --num-questions 10 --copy-as-s
python -m src.lme_sampler /path/to/LongMemEval/data/longmemeval_m.json --num-questions 10 --copy-as-s
```

This creates `longmemeval_s_10.json` with 10 questions (one from each question type) and copies it as `longmemeval_s.json` for the loader. The sampler systematically selects questions to cover all core abilities: single-session-user, multi-session, knowledge-update, temporal-reasoning, single-session-preference, single-session-assistant, and abstention questions.

## Quick Start

1. **Create a config file**:
   ```bash
   cd longmemeval-benchmarker
   cp config.example.toml run.toml
   # Edit run.toml to set dataset_file_path, provider/models, vault_title
   ```

2. **Run the benchmark with the Orchestrator**:
   ```bash
   # Make sure you're in the longmemeval-benchmarker directory
   cd longmemeval-benchmarker

   # Auto mode (recommended): Start, monitor, and shutdown automatically
   poetry run python -m src.orchestrator config/config.smoke.toml --auto --workers 3

   # Or run with custom run ID
   poetry run python -m src.orchestrator config/config.smoke.toml --auto --workers 3 --run-id my_benchmark_run

   # Process only first N questions
   poetry run python -m src.orchestrator config/config.smoke.toml --auto --workers 3 --num-questions 10
   ```

3. **Resume a failed/incomplete run**:
   ```bash
   # Resume from where it left off (default behavior)
   poetry run python -m src.orchestrator config/config.smoke.toml --resume --run-id my_benchmark_run

   # Force retry failed questions
   poetry run python -m src.orchestrator config/config.smoke.toml --resume --run-id my_benchmark_run --force

   # Restart questions from beginning (clears memory)
   poetry run python -m src.orchestrator config/config.smoke.toml --resume --run-id my_benchmark_run \
     --resume-mode restart-from-first-session
   ```

4. **Monitor progress** (separate terminal):
   ```bash
   poetry run python -m src.orchestrator config/config.smoke.toml --monitor --run-id my_benchmark_run
   ```

5. **Check results**:
   ```bash
   cat out/run_<RUN_ID>/hypotheses.jsonl
   ls out/run_<RUN_ID>/logs/  # Per-question logs
   ```

6. **Evaluate with LongMemEval's official QA evaluator** (if using full dataset):
   ```bash
   cd /path/to/LongMemEval/src/evaluation
   python3 evaluate_qa.py gpt-4o \
     /path/to/mycelian-memory/longmemeval-benchmarker/out/run_<RUN_ID>/hypotheses.jsonl \
     ../../data/longmemeval_oracle.json
   ```

## Project Structure

```
longmemeval-benchmarker/
├── src/
│   ├── orchestrator/               # 🎯 MAIN: Orchestration layer (USE THIS)
│   │   ├── orchestrator.py         # Main CLI for running benchmarks
│   │   ├── tasks.py                # Task definitions
│   │   ├── worker.py               # Worker process
│   │   ├── progress_tracker.py     # SQLite progress tracking
│   │   └── orchestrator_config.py  # Queue configuration
│   ├── single_question_runner.py   # Core logic for processing questions
│   ├── dataset_loader.py           # Load and parse LongMemEval JSON files
│   ├── mycelian_memory_agent/      # Agent implementation
│   │   ├── agent.py                # LangGraph agent with MCP tools
│   │   ├── build.py                # Agent factory
│   │   └── mcp_utils.py            # MCP client utilities
│   ├── memory_manager.py           # Vault/memory management via MCP
│   └── model_providers.py          # LLM provider abstractions
├── config/                         # Configuration files
│   ├── config.example.toml         # Example configuration
│   ├── config.smoke.toml           # Smoke test config
│   └── config.5s.toml              # 5-question test config
├── data/                           # Runtime databases
│   ├── progress.db                 # Progress tracking
│   └── orchestrator.db             # Task queue (Huey)
├── debug_qa.py                     # QA debugging tool
├── requirements.txt                # Python dependencies
└── README.md                       # This file
```

## Configuration

The benchmarker uses TOML configuration files. Key settings:

- `dataset_file_path`: Path to LongMemEval JSON file
- `vault_title`: Vault name for storing memories
- `models.agent`: Model for the memory agent (supports OpenAI, Vertex AI, OpenRouter)
- `models.qa`: Model for question answering (defaults to agent model)

### Orchestrator CLI Options

- `--auto`: Automatically start worker, monitor progress, and shut down on completion
- `--workers N`: Number of worker processes to use
- `--num-questions N`: Process only first N questions
- `--run-id ID`: Specify custom run ID (auto-generated if not provided)
- `--resume`: Resume an existing run
- `--resume-mode`: How to resume (`restart-from-first-session` or `resume-from-next-session`)
- `--force`: Force retry failed questions during resume
- `--monitor`: Monitor mode - show progress without enqueueing
- `--stop`: Stop all running benchmarker processes
- `--clear-state`: Delete all orchestrator state (databases)

## How It Works

For each question in the dataset, the benchmarker:

1. **Memory Setup**: Creates/binds a memory using `memory_title_template`
2. **Ingestion**: Streams all sessions/turns chronologically via MCP tools:
   - `add_entry`: Store individual messages
   - `put_context`: Save consolidated context
   - `await_consistency`: Ensure writes are committed
3. **Question Answering**:
   - `search_memories`: Retrieve relevant context
   - Call QA model with retrieved context
   - Append `{question_id, hypothesis}` to `out/run_<RUN_ID>/hypotheses.jsonl`

## Troubleshooting

### Import Errors
If you see `ModuleNotFoundError`, ensure you're using the correct Python path:
```bash
PYTHONPATH=src python -m src.benchmarker config.toml
```

### Connection Issues
Ensure the Mycelian services are running:
```bash
# Check backend service status
make backend-status

# Check MCP server logs
docker compose -f deployments/docker/docker-compose.streamable.yml logs -f
```

The benchmarker connects to `http://localhost:11546/mcp` by default. You can override with:
```bash
export MYCELIAN_MCP_URL="http://localhost:11546/mcp"
```

### Rate Limits
If you hit OpenAI rate limits, reduce `workers` or add delays in the retry logic.

## Development Notes

- **No Go PATH needed**: The benchmarker is pure Python and connects via HTTP/MCP
- **Chronological ordering**: Sessions within a question are processed sequentially to maintain temporal context
- **Parallel processing**: Only questions can be parallelized, not sessions within a question

## ⚠️ Deprecated Methods (DO NOT USE)

The following runner scripts are deprecated and should not be used:

### Deprecated Scripts
- `python -m src.benchmarker` - Old direct benchmarker module
- `./run_benchmark.py` - Legacy benchmark runner script
- `run_qa_phase.py` - Standalone QA runner with hardcoded values
- `simple_qa_test.py` - Simple test script
- `src/single_question_runner_cli.py` - Direct single question CLI

These scripts lack the robustness, monitoring, and resumability features of the orchestrator. They are kept only for historical reference and will be removed in a future version.

### Still Supported Tools
- `debug_qa.py` - Useful for debugging QA on specific memories
- `validate_dataset.py` - Dataset validation utility

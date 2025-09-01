# LongMemEval Benchmarker

The benchmarker allows us to run the LongMemEval benchmark using Mycelian Memory as the backend. Please refer to the benchmarker [design doc](../../docs/designs/langgraph_longmemeval_benchmarker.md) to learn more. 

LongMemEval is designed to test not just the performance of the memory system but also its scale. The large version contains 500 questions. Each question is composed of 500 sessions. Each session contains turns between a user and an assistant. To evaluate real world performance, we must store memories as a production agent will. This creates scaling challenges. Answering a single question of smaller dataset with ~50 sessions requires ~60 mins with Claude Haiku 3.5. Hence, the benchmarker supports sharding by questions and processing each question in parallel. **IMPORTANT**: we DO NOT parallelize sessions inside a question as the Memory Agent must build chronological context. 

Note: Running many questions in parallel can hit provider rate limits. The sampler script allows extracting a representative subset from `longmemeval_{s/m}.json` files.

## Prerequisites

1. **Start Mycelian services** (from repo root):
   ```bash
   # Start the backend memory service
   make start-dev-mycelian-server
   
   # Start the MCP server for benchmarker communication
   make start-mcp-streamable-server
   ```

2. **Set up Python environment**:
   ```bash
   cd tools/longmemeval-benchmarker
   python -m venv venv
   source venv/bin/activate
   pip install -r requirements.txt
   ```

3. **Provider credentials** (OpenAI only):
   ```bash
   export OPENAI_API_KEY=your-openai-key
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
cd tools/longmemeval-benchmarker

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
   cd tools/longmemeval-benchmarker
   cp config.example.toml run.toml
   # Edit run.toml to set dataset_file_path, provider/models, vault_title
   ```

2. **Run the benchmarker**:
   ```bash
   source venv/bin/activate
   PYTHONPATH=src python -m src.benchmarker run.toml --num-questions 1
   ```

3. **Check results**:
   ```bash
   cat out/run_<RUN_ID>/hypotheses.jsonl
   ls out/run_<RUN_ID>/logs/  # Per-question logs
   ```

4. **Evaluate with LongMemEval's official QA evaluator** (if using full dataset):
   ```bash
   cd /path/to/LongMemEval/src/evaluation
   python3 evaluate_qa.py gpt-4o \
     /path/to/mycelian-memory/tools/longmemeval-benchmarker/out/run_<RUN_ID>/hypotheses.jsonl \
     ../../data/longmemeval_oracle.json
   ```

## Project Structure

```
tools/longmemeval-benchmarker/
├── src/
│   ├── benchmarker.py              # Main entrypoint: config/dataset/vault/dirs → workers
│   ├── dataset_loader.py           # Load and parse LongMemEval JSON files
│   ├── mycelian_memory_agent.py    # LangGraph agent with MCP tools for memory operations
│   ├── memory_manager.py           # Vault/memory management via MCP
│   ├── single_question_runner.py   # Process one question end-to-end (ingestion + QA)
│   ├── worker_manager.py           # Sequential/parallel orchestration + logging
│   ├── tenacious_agent_invoker.py  # Retry logic for agent calls
│   └── lme_sampler.py              # Extract subsets from full datasets
├── config.example.toml             # Example configuration
├── config.1s.toml                  # Single question test config
├── longmemeval_0a995998.json       # Sample dataset for testing
├── requirements.txt                # Python dependencies
└── README.md                       # This file
```

## Configuration

The benchmarker uses TOML configuration files. Key settings:

- `dataset_file_path`: Path to LongMemEval JSON file
- `vault_title`: Vault name for storing memories
- `provider.type`: "openai" (currently supported)
- `models.agent`: Model for the memory agent
- `models.qa`: Model for question answering
- `params.question_limit`: Number of questions to process
- `params.workers`: Parallel workers (keep at 1 for chronological consistency)

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

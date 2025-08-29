# LongMemEval Integration

A clean, bottom-up integration of LongMemEval benchmarking with Mycelian Memory. Starting fresh with minimal complexity.

## Philosophy

- **Start Simple**: Begin with one working conversation before scaling
- **Incremental**: Add features one at a time, testing each step
- **Clean Architecture**: Avoid over-engineering from the start
- **MCP First**: Use Mycelian Memory's MCP interface directly

## Current Status

🚧 **Starting Fresh** - All previous complex code has been removed.

## Quick Start (Simple Flow)

```bash
# Install deps
pip install -r requirements.txt

# Environment
export OPENAI_API_KEY="your-openai-key"          # for provider=openai
export AWS_REGION="us-west-2"                    # for provider=bedrock

# Create a config file (see config.example.toml)
cp config.example.toml run.toml
vi run.toml  # set dataset_repo_path, provider/models, vault_title

# Ingest the first N questions from the dataset and write hypotheses.jsonl
PYTHONPATH=src python -m runner run.toml --num-questions 10

# Evaluate with LongMemEval's official QA evaluator
cd /Users/deesam/workspace/LongMemEval/src/evaluation
python3 evaluate_qa.py gpt-4o \
  /Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker/out/run_<RUN_ID>/hypotheses.jsonl \
  ../../data/longmemeval_oracle.json
```

## Project Structure

```
tools/longmemeval-benchmarker/
├── src/
│   ├── dataset_loader.py     # question → sessions → messages
│   ├── mycelian_memory_agent.py  # wraps MCP tools for ingestion/search
│   ├── runner.py             # ingest N questions and write hypotheses.jsonl
│   └── eval.py               # (optional) local EM/judge helpers
├── config.example.toml       # starter config
├── requirements.txt          # deps
└── README.md                 # this file
```

## Flow

- Single run: For each of the first N questions, the runner:
  - Creates/binds a memory
  - Streams all sessions/turns via MCP tools
  - Searches memory, builds a compact QA context, calls the QA model
  - Appends `{question_id, hypothesis}` to `out/run_<RUN_ID>/hypotheses.jsonl`

## Development

This is a fresh start - no legacy code to maintain or refactor. Build exactly what you need, step by step.

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
# Install deps
pip install -r requirements.txt

# Environment
export OPENAI_API_KEY="your-openai-key"          # for provider=openai
export AWS_REGION="us-west-2"                    # for provider=bedrock

# Create a config file (see config.example.toml)
cp config.example.toml run.toml
vi run.toml  # set dataset_repo_path, provider/models, vault_title

# Run ingestion → qa → eval per-question (default)
python -m longmemeval_benchmarker.runner run.toml

# Or run a single phase
python -m longmemeval_benchmarker.runner run.toml --mode ingestion
python -m longmemeval_benchmarker.runner run.toml --mode qa
python -m longmemeval_benchmarker.runner run.toml --mode eval
```

## Project Structure

```
tools/longmemeval-benchmarker/
├── longmemeval_benchmarker/
│   ├── dataset_loader.py     # question → sessions → messages
│   ├── agent.py              # builds LangGraph prebuilt agent
│   ├── runner.py             # orchestrates ingestion/qa/eval per TOML
│   └── mcp_client.py         # MCP client wrappers (if needed)
├── config.example.toml       # starter config
├── requirements.txt          # deps
└── README.md                 # this file
```

## Modes

- ingestion: stream sessions/messages to the agent to persist entries/context only
- qa: run retrieval + answer only (assumes prior ingestion exists)
- eval: compute metrics (EM or LLM judge) on stored answers

## Development

This is a fresh start - no legacy code to maintain or refactor. Build exactly what you need, step by step.

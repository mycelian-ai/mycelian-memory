# LongMemEval Benchmarker

The benchmarker allows us to run the LongMemEval benchmark using Mycelian Memory as the backend. Please refer to the benchmarker [design doc](../../docs/designs/langgraph_longmemeval_benchmarker.md) to learn more. 

LongMemEval is designed to test not just the performance of the memory system but also it's scale. The large version contains 500 questions. Each question is composed of 500 sessions. Each session contains turns between a user and an assistant. To evaluate real world performance, we must store memories as a production agent will. This creates scaling challenges. Answering a single question of smaller dataset with ~50 sessions requires ~60 mins with Claude Haiku 3.5. Hence, the benchmarker supports sharding by questions and processing each question in parallel. ITS IMPORTANT that we DO NOT parallelize sessions inside a question as the Memory Agent must build chronological context. 

Note: I got heavily throttled by both AWS Bedrock and OpenAI while trying to run more than one question in parallel, so did a small scale smoke test with 5 questions one of each type. The sampler script allows to extract these questions from `longmemeval_{s/m}.json` files. 

Please use following steps to run the benchmarker yourself.

## Environment setup

```bash
# Python deps (from repo root)
pip install -r tools/longmemeval-benchmarker/requirements.txt

# Start Mycelian MCP server (single authoritative instance)
docker compose -f deployments/docker/docker-compose.streamable.yml up -d

# Provider credentials
export OPENAI_API_KEY=...      # if provider=openai
export AWS_REGION=us-west-2    # if provider=bedrock
```

## Dependency on LongMemEval repo

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

Then set `dataset_repo_path` in your TOML to that LongMemEval directory.

## Creating a sample dataset

For testing, you can create a smaller subset from the full LongMemEval dataset:

```bash
cd tools/longmemeval-benchmarker

# Option 1: Point to directory (auto-finds longmemeval_s.json, longmemeval_m.json, etc.)
python lme_sampler.py /path/to/LongMemEval --num-questions 10 --copy-as-s

# Option 2: Point directly to specific file
python lme_sampler.py /path/to/LongMemEval/data/longmemeval_s.json --num-questions 10 --copy-as-s
python lme_sampler.py /path/to/LongMemEval/data/longmemeval_m.json --num-questions 10 --copy-as-s
```

This creates `longmemeval_s_10.json` with 10 questions (one from each question type) and copies it as `longmemeval_s.json` for the loader. The sampler systematically selects questions to cover all core abilities: single-session-user, multi-session, knowledge-update, temporal-reasoning, single-session-preference, single-session-assistant, and abstention questions.

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
cd tools/longmemeval-benchmarker
python -m src.runner run.toml --num-questions 10

# Evaluate with LongMemEval's official QA evaluator
cd /path/to/LongMemEval/src/evaluation
python3 evaluate_qa.py gpt-4o \
  /path/to/mycelian-memory/tools/longmemeval-benchmarker/out/run_<RUN_ID>/hypotheses.jsonl \
  ../../data/longmemeval_oracle.json
```

## Project Structure

```
tools/longmemeval-benchmarker/
├── src/
│   ├── dataset_loader.py         # question → sessions → messages
│   ├── mycelian_memory_agent.py  # wraps MCP tools for ingestion/search
│   ├── runner.py                 # ingest N questions and write hypotheses.jsonl
│   └── eval.py                   # (optional) local EM/judge helpers
├── config.example.toml       # starter config
├── requirements.txt          # deps
└── README.md                 # this file
```

## Generating a small sample

Run one question end-to-end to sanity check output:

```bash
cd tools/longmemeval-benchmarker
python -m src.runner run.toml --num-questions 1
cat out/run_<RUN_ID>/hypotheses.jsonl
```

## Running ingestion + QA to produce hypotheses

- Single run (no extra modes). For each of the first N questions, the runner:
  - Creates/binds a memory (uses `memory_title_template`)
  - Streams all sessions/turns via MCP tools (`add_entry`, `put_context`, `get_context`, `await_consistency`, `search_memories`)
  - Builds a compact QA context from search results and calls the QA model
  - Appends `{question_id, hypothesis}` to `out/run_<RUN_ID>/hypotheses.jsonl`

## Use LongMemEval evaluator to generate accuracy metrics

```bash
cd /path/to/LongMemEval/src/evaluation
python3 evaluate_qa.py gpt-4o \
  /path/to/mycelian-memory/tools/longmemeval-benchmarker/out/run_<RUN_ID>/hypotheses.jsonl \
  ../../data/longmemeval_oracle.json
```

The evaluator prints per-question logs (with `autoeval_label`) and overall accuracy. You can further aggregate with `print_qa_metrics.py` from the LongMemEval repo.

## Development

This is a fresh start - no legacy code to maintain or refactor. Build exactly what you need, step by step.

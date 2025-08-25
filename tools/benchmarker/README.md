# Mycelian Memory Benchmarker

**What it does**: Tests the Mycelian memory system by simulating conversations with Claude, ingesting MSC dataset conversations, and evaluating memory recall performance.

## TLDR - Quick Commands

### Setup
```bash
# From project root
source setup-env.sh
cd tools/benchmarker/
```

### Run Benchmarks (Anthropic)
Provider setup (Anthropic):
- Set environment variable `ANTHROPIC_API_KEY` to your Claude API key.
- Default provider is `anthropic`; default model is `claude-3-haiku-20240307`.

```bash
# Validate tools (Anthropic)
python benchmark_runner.py --mycelian-url http://localhost:11545 validate-tools
```

```bash
# Ingest MSC dataset conversations
python benchmark_runner.py --mycelian-url http://localhost:11545 --conversations 5 --tracker-file tracker.json --vault-title benchmarker ingest

# Evaluate memory recall
python benchmark_runner.py --mycelian-url http://localhost:11545 --tracker-file tracker.json eval

# Both phases
python benchmark_runner.py --mycelian-url http://localhost:11545 --conversations 5 --tracker-file tracker.json --vault-title benchmarker both
```

### Run Benchmarks (AWS Bedrock)
Provider setup (Bedrock):
- Configure AWS credentials with an IAM user/role that has Bedrock InvokeModel access.
  - `aws configure` (or environment variables) for Access Key, Secret, and default region.
  - Ensure Bedrock model access is granted in the target region (e.g., `us-west-2`) for the model you select.
- Use `--provider bedrock --aws-region <region>` and a valid Bedrock model ID (e.g., `anthropic.claude-3-5-haiku-20241022-v1:0`).

```bash
# Validate tools (Bedrock)
python benchmark_runner.py \
  --provider bedrock \
  --aws-region us-west-2 \
  --model-name anthropic.claude-3-5-haiku-20241022-v1:0 \
  --mycelian-url http://localhost:11545 \
  validate-tools

# Ingest (Bedrock)
python benchmark_runner.py \
  --provider bedrock \
  --aws-region us-west-2 \
  --model-name anthropic.claude-3-5-haiku-20241022-v1:0 \
  --mycelian-url http://localhost:11545 \
  --conversations 5 \
  --tracker-file tracker.json \
  --vault-title benchmarker \
  ingest

# Evaluate (Bedrock)
python benchmark_runner.py \
  --provider bedrock \
  --aws-region us-west-2 \
  --model-name anthropic.claude-3-5-haiku-20241022-v1:0 \
  --mycelian-url http://localhost:11545 \
  --tracker-file tracker.json \
  eval

# Both (Bedrock)
python benchmark_runner.py \
  --provider bedrock \
  --aws-region us-west-2 \
  --model-name anthropic.claude-3-5-haiku-20241022-v1:0 \
  --mycelian-url http://localhost:11545 \
  --conversations 5 \
  --tracker-file tracker.json \
  --vault-title benchmarker \
  both
```

Notes:
- Anthropic: export `ANTHROPIC_API_KEY`.
- Bedrock: ensure AWS credentials are configured and Bedrock model access is granted in the specified region.
- The runner reuses a persistent vault (`--vault-title`, default `benchmarker`) and creates a new memory per run inside that vault.

## How It Works

### Components
- **SessionSimulator**: Drives conversations with Claude, handles MCP tool calls and context management
- **MycelianMemoryClient**: CLI-based client that interfaces with the memory service
- **MSC Dataset Loader**: Processes Multi-Session Chat dataset for realistic conversation testing
- **BenchmarkRunner**: Orchestrates ingestion and evaluation phases

### Architecture
1. **Ingestion Phase**: 
   - Loads MSC dataset conversations
   - Claude acts as a "note-taker" agent, persisting each message via `add_entry` tool calls
   - Updates context documents via `put_context` at session end
   - Tracks conversation metadata for evaluation

2. **Evaluation Phase**:
   - Asks Claude questions about previously ingested conversations
   - Tests memory recall using `search_memories` and `get_context` tools
   - Measures accuracy of retrieved information

3. **Context Management**:
   - Updates context every 5 messages or after critical tool calls
   - Maintains conversation history and state
   - Ensures context persistence at session end

### Model Providers
- **Anthropic API**: Direct integration with Claude models
- **AWS Bedrock**: Claude models via Bedrock runtime (requires AWS credentials)

### Bootstrap & Prompts
- The model first calls `get_tools_schema`, then `get_default_prompts` with `memory_type='chat'` to load:
  - `context_summary_rules`
  - `templates.context_prompt`, `templates.entry_capture_prompt`, `templates.summary_prompt`
- Static `get_asset` calls are no longer required.

### Vault & Memory Policy
- Persistent vault (title via `--vault-title`, default `benchmarker`) is reused across runs.
- A new memory is created per run inside that vault; the model must use the provided `vault_id`/`memory_id` (it must not create vaults/memories).

### Rate Limiting
- Default: 1 second between requests (~60 RPM)
- Configurable via `SESSION_SIMULATOR_RATE_LIMIT_INTERVAL` environment variable
- Includes exponential backoff for rate limit (429) and overload (529) errors

## Prerequisites
- Python 3.8+
- Go 1.19+ (for mycelianCli binary)
- Mycelian Memory Service running (default: `http://localhost:11545`)
- API credentials:
  - Anthropic: `ANTHROPIC_API_KEY` environment variable
  - Bedrock: AWS credentials configured

## Dev Mode
The benchmarker operates in **dev mode** - no user management required. It connects directly to the memory service using a hardcoded development API key.

## Testing
```bash
# Run all tests
pytest tests/

# Specific test suites
pytest tests/test_session_simulator.py -v
pytest tests/test_session_simulator_context.py -v
```

## Configuration
- `--conversations N`: Limit ingestion to N conversations
- `--questions N`: Limit evaluation questions per conversation  
- `--max-messages N`: Truncate sessions to N messages
- `--model-name`: Specify Claude model (e.g., `claude-3-5-sonnet-20241022`)
- `--provider`: Choose `anthropic` or `bedrock`
- `--aws-region`: AWS region for Bedrock (e.g., `us-west-2`)

For detailed design documentation, see [docs/design/msc_dataset_note_taker.md](../../docs/design/msc_dataset_note_taker.md).
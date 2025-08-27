# LangGraph-Based LongMemEval Benchmarker

A LangGraph-powered benchmarker for evaluating Mycelian Memory against the LongMemEval academic benchmark. This implementation replaces the complex SessionSimulator architecture with a clean, maintainable LangGraph workflow.

## Architecture

- **End-to-End LangGraph Workflow**: Orchestrates three sub-workflows (Ingestion, QA, Evaluation)
- **Mycelian Memory Agent**: Observer pattern agent following `context_summary_rules.md` protocol
- **Frugal Search Strategy**: Enhanced search guidance for optimal memory quality vs efficiency
- **Native MCP Integration**: Seamless tool orchestration via LangGraph's MCP support

## Quick Start

### 1. Environment Setup

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -e .
```

### 2. Configuration

```bash
# Copy environment template
cp .env.example .env

# Edit configuration values
vim .env
```

Required environment variables:
- `OPENAI_API_KEY`: For GPT-4o evaluation judge
- `MYCELIAN_MCP_URL`: Mycelian MCP server endpoint (default: http://localhost:11546/mcp)
- `LONGMEMEVAL_DATA_PATH`: Path to LongMemEval dataset

### 3. Run Benchmark

```bash
# Basic benchmark run
python -m src.main --dataset longmemeval_s.json

# Full benchmark with custom configuration
python -m src.main \
  --dataset longmemeval_m.json \
  --vault-name my-benchmark-run \
  --output-dir ./results/run-$(date +%Y%m%d)
```

## Project Structure

```
langgraph-benchmarker/
├── src/
│   ├── __init__.py
│   ├── main.py                    # Entry point and CLI
│   ├── config.py                  # Configuration management
│   ├── workflows/
│   │   ├── __init__.py
│   │   ├── end_to_end.py          # End-to-end LangGraph workflow
│   │   ├── ingestion.py           # Ingestion sub-workflow
│   │   ├── qa.py                  # QA sub-workflow
│   │   └── evaluation.py          # Evaluation sub-workflow
│   ├── agents/
│   │   ├── __init__.py
│   │   ├── memory_agent.py        # Mycelian Memory Agent
│   │   └── qa_agent.py            # QA Agent for questions
│   ├── nodes/
│   │   ├── __init__.py
│   │   ├── conversation_processor.py  # Stateless conversation processing
│   │   ├── evaluation_node.py         # GPT-4o judge evaluation
│   │   ├── setup_node.py             # Memory initialization
│   │   └── results_node.py           # Results aggregation
│   ├── utils/
│   │   ├── __init__.py
│   │   ├── dataset_loader.py      # LongMemEval dataset loading
│   │   ├── mcp_client.py          # MCP client configuration
│   │   └── metrics.py             # Performance and quality metrics
│   └── types.py                   # Type definitions and state models
├── tests/
│   ├── __init__.py
│   ├── unit/                      # Unit tests for individual components
│   ├── integration/               # Integration tests with MCP server
│   └── e2e/                       # End-to-end benchmark tests
├── results/                       # Benchmark output directory
├── docs/                          # Additional documentation
└── README.md
```

## Key Components

### Mycelian Memory Agent
- **Pattern**: Observer - watches conversation turns objectively
- **Protocol**: Strict adherence to `context_summary_rules.md` 
- **Search Strategy**: Frugal search with 4 triggers (contradictions, past references, direct queries, old context shards)

### Workflow Architecture
1. **Setup**: Initialize vault and memory for conversation
2. **Ingestion**: Process haystack sessions through Memory Agent
3. **QA**: Answer questions using stored memories
4. **Evaluation**: GPT-4o judge scoring against expected answers
5. **Results**: Aggregate and analyze benchmark metrics

## Development

### Running Tests

```bash
# Unit tests
pytest tests/unit/

# Integration tests (requires running Mycelian MCP server)
pytest tests/integration/

# End-to-end tests
pytest tests/e2e/
```

### Code Quality

```bash
# Format code
black src/ tests/
isort src/ tests/

# Type checking
mypy src/

# Linting
flake8 src/ tests/
```

## Performance

Expected performance improvements over SessionSimulator:
- **Code Complexity**: ~80% reduction (from 2000+ lines to ~500 lines)
- **State Management**: Native LangGraph handling vs manual flags
- **Tool Orchestration**: Built-in MCP support vs custom dispatch
- **Maintainability**: Clear separation of concerns with agents/nodes pattern

## Results

Benchmark results are saved to `results/` directory with:
- Individual conversation scores
- Aggregate metrics by question type
- Memory quality analysis (precision/recall)
- Search pattern effectiveness stats

## References

- [ADR-013: LangGraph-Based LongMemEval Benchmarker](../../docs/adrs/013-langgraph-longmemeval-benchmarker.md)
- [Design Document](../../docs/designs/langgraph_longmemeval_benchmarker.md)
- [LongMemEval Paper](https://arxiv.org/pdf/2410.10813.pdf)
- [LongMemEval Dataset](https://huggingface.co/datasets/xiaowu0162/longmemeval)
# Huey Orchestrator for LongMemEval Benchmarker

This directory contains the Huey-based orchestration layer for the LongMemEval benchmarker. It provides task queue management, progress tracking, and resumability without modifying the existing benchmarker code.

## Architecture

```
orchestrator.py (CLI)
    ↓ enqueues tasks
Huey Queue (SqliteHuey)
    ↓ workers pull tasks
worker.py (processes)
    ↓ executes
tasks.py (wrappers)
    ↓ calls
existing single_question_runner.py (unchanged)
    ↓ stores to
Mycelian Memory Service
```

## Components

- **huey_config.py** - Huey configuration with SqliteHuey backend
- **progress_tracker.py** - SQLite-based progress tracking
- **tasks.py** - Huey task definitions that wrap existing runners
- **orchestrator.py** - Main CLI for enqueueing and monitoring
- **worker.py** - Worker process that executes tasks

## Usage

### Start a new benchmark run:
```bash
python -m huey_orchestrator.orchestrator \
    ../longmemeval-datasets/longmemeval_s.json \
    ../config.toml \
    --num-questions 10 \
    --workers 3
```

### Resume an interrupted run:
```bash
python -m huey_orchestrator.orchestrator \
    ../longmemeval-datasets/longmemeval_s.json \
    ../config.toml \
    --resume \
    --run-id run_1234567890
```

### Start workers (in separate terminals):
```bash
python -m huey_orchestrator.worker
```

### Monitor progress:
```bash
python -m huey_orchestrator.orchestrator \
    ../longmemeval-datasets/longmemeval_s.json \
    ../config.toml \
    --monitor \
    --run-id run_1234567890
```

## Database Schema

Progress is tracked in SQLite with the following schema:
- `run_id` - Unique identifier for the benchmark run
- `question_id` - Question identifier from dataset
- `vault_id` - Mycelian vault UUID
- `memory_id` - Mycelian memory UUID (for QA phase)
- `total_sessions` - Number of sessions in question
- `completed_sessions` - Sessions processed so far
- `status` - Overall status (pending/in_progress/completed/failed)

## Features

- **Resumable** - Tracks progress at session level
- **Parallel** - Multiple workers can process different questions
- **Isolated** - Runs existing code as subprocess (no modifications needed)
- **Monitored** - Real-time progress tracking
- **Robust** - Automatic retries on failure

## Testing

### Integration Test
Run the comprehensive integration test that uses the mock runner:
```bash
python test_orchestrator_integration.py
```

This test validates:
- Progress tracking initialization
- Task processing with mock runner
- Resume capability for partial completions
- Concurrent task claiming prevention
- Statistics tracking accuracy

### Mock vs Real Runner
- **Production**: `tasks.py` always uses the real `single_question_runner_cli.py`
- **Testing**: `tasks_mock.py` uses `mock_runner.py` for fast testing
- **Integration Test**: Uses `tasks_mock.py` to avoid Mycelian dependencies
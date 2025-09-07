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

### Orchestrator CLI

Start a new run (enqueue questions only):
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --num-questions 10 \
    --workers 3
```

Resume a run (default: resume-from-next-session):
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --resume \
    --run-id run_1234567890
```

Resume with explicit mode:
```bash
# Restart from first session (clears memory_id, resets counters)
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --resume --run-id run_1234567890 \
    --resume-mode restart-from-first-session

# Resume from next session (keep memory_id, continue from completed_sessions)
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --resume --run-id run_1234567890 \
    --resume-mode resume-from-next-session
```

Force retry failed questions during resume:
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --resume --run-id run_1234567890 --force
```

Monitor progress (no enqueue):
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --monitor --run-id run_1234567890
```

Auto mode (enqueue → start worker → monitor → shutdown):
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator \
    /absolute/path/to/config.toml \
    --auto --workers 3
```

Stop workers and orchestrators, optionally clear state:
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator --stop [--force] [--clear-state]
```

Clear orchestrator state (task DB and progress DB) only:
```bash
PYTHONPATH=.. python -m huey_orchestrator.orchestrator --clear-state
```

### Worker

Start workers (separate terminal):
```bash
PYTHONPATH=.. HUEY_QUEUE_NAME=huey-<run_id> HUEY_RUN_ID=<run_id> python -m huey_orchestrator.worker --workers 3
```

### Single-question CLI (advanced)

Ingestion only (supports --start-session for manual resume):
```bash
python -m src.single_question_runner_cli \
    --mode ingest \
    --question-json /abs/path/to/question.json \
    --config /abs/path/to/config.toml \
    --start-session 2
```

QA only (requires vault_id and memory_id):
```bash
python -m src.single_question_runner_cli \
    --mode qa \
    --question-json /abs/path/to/question.json \
    --config /abs/path/to/config.toml \
    --vault-id <vault_uuid> \
    --memory-id <memory_uuid>
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

### Resume semantics

- restart-from-first-session (clears `memory_id`, resets counters/status) – deterministic restart.
- resume-from-next-session (default) continues from `completed_sessions` using existing `memory_id`.
- For QA-stuck with completed ingestion: resume mode keeps ingestion and re-enqueues QA.

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

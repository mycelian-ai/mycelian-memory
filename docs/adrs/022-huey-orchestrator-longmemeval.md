# ADR-022: Huey Orchestrator for LongMemEval

**Status**: Accepted
**Date**: 2025-09-08
**Supersedes**: Implements orchestration aspects of ADR-018

## Context

A full LongMemEval run can span many hours across hundreds of questions. We need a resilient orchestration layer that queues work, tracks progress, supports resume, and provides live monitoring without modifying the core runner.

## Decision

1. Use Huey with `SqliteHuey` (async-only) for task queueing and retries, with per-run isolation by setting `HUEY_QUEUE_NAME=huey-{run_id}` and `HUEY_RUN_ID={run_id}`.
2. Persist progress in SQLite at `longmemeval-benchmarker/data/progress.db` (WAL mode). Tables: `runs`, `question_progress` (includes `vault_id`, `memory_id`, counters, timestamps, `last_progress_at`).
3. Resume strategy: on resume, gather `pending`, `resumable`, stuck-in-progress, and QA-stuck rows. Depending on `resume-mode`, restart from session 0 (clearing `memory_id`, preserving `vault_id`) or continue from `completed_sessions`. `--force` re-enqueues previously failed items.
4. Tasks wrap existing runner:
   - `process_question(run_id, question_id)`: loads question/config from DB, computes start index, runs `SingleQuestionRunner` ingestion in-process, persists IDs, schedules QA if done.
   - `run_qa(run_id, question_id)`: loads IDs, runs QA in-process, appends to `out/run_<run_id>/hypotheses.jsonl`, merges per-question result JSON.
   - `check_run_health(run_id)`: returns aggregate stats for monitoring.
5. CLI: `python -m src.orchestrator` provides `--num-questions`, `--resume`, `--run-id`, `--workers`, `--resume-mode`, `--monitor`, `--auto`, `--clear-state`, `--stop`, `--force`. Preflight checks validate absolute dataset path and DB accessibility. `--auto` starts a worker subprocess bound to the run and monitors until completion.
6. Logging: per-run orchestrator log at `logs/{run_id}/orchestrator.log` (rotating). Runner produces per-question logs and results.

## Consequences

Positive:
- Robust resumability with session/message counters and stuck detection via `last_progress_at`.
- Per-run queue isolation prevents cross-run interference; live dashboard provides real-time visibility.
- Minimal coupling: tasks wrap the runner; no runner code changes required.

Tradeoffs / Cleanups:
- Env-based queue/log binding is acceptable; future work could remove env reliance.
- Memory title formatting should converge on runner template `{question_id}__{run_id}` consistently.

## Alternatives Considered

- Redis-backed Huey (more complex deployment).
- Immediate/sync mode (rejected; async-only enforced).
- Custom thread pool (would reimplement retries/backoff/queueing).

## References

- Code: `src/orchestrator/__main__.py`, `orchestrator_config.py`, `tasks.py`, `progress_tracker.py`, `worker.py`
- Runner: `src/benchmarker.py`, `src/single_question_runner.py`
- ADR-018: Resumable benchmarker (DB intent)

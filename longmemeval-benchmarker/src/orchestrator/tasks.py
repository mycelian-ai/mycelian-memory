#!/usr/bin/env python3
"""
Huey task definitions for LongMemEval benchmarker.
These tasks wrap the existing single_question_runner functionality.
"""

import os
import sys
import json
import logging
import subprocess
import tempfile
from pathlib import Path
from datetime import datetime
import time
import tomllib
from typing import Dict, Optional, Tuple
import fcntl
# Add parent directory to path to import existing modules BEFORE importing runner modules
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from src.orchestrator.progress_tracker import ProgressTracker
from src.single_question_runner import SingleQuestionRunner
from src.config_parser import parse_config
from src.memory_manager import MemoryManager
from src.mycelian_memory_agent import create_mcp_client

from src.orchestrator.orchestrator_config import (
    huey,
    DEFAULT_TASK_RETRIES,
    DEFAULT_RETRY_DELAY,
    QUESTION_PROCESSING_TIMEOUT,
    QA_TIMEOUT_SEC,
    LOGS_DIR,
)
from src.paths import resolve_under_root

logger = logging.getLogger('orchestrator.tasks')


def _log_startup_config():
    """Log orchestrator configuration once at startup."""
    try:
        logger.info(
            "Orchestrator config: retries=%d retry_delay=%ds ingest_timeout=%ds qa_timeout=%ds logs_dir=%s",
            DEFAULT_TASK_RETRIES,
            DEFAULT_RETRY_DELAY,
            QUESTION_PROCESSING_TIMEOUT,
            QA_TIMEOUT_SEC,
            LOGS_DIR,
        )
    except Exception:
        pass


_SRC_DIR = Path(__file__).parent.parent / "src"


def _atomic_write_json(target: Path, payload: Dict) -> None:
    """Atomically write JSON by writing to a temp file and renaming."""
    try:
        target.parent.mkdir(parents=True, exist_ok=True)
        tmp = target.with_suffix(target.suffix + ".tmp")
        with open(tmp, 'w', encoding='utf-8') as f:
            json.dump(payload, f, ensure_ascii=False)
        os.replace(tmp, target)
    except Exception:
        # Best effort; avoid crashing the task for artifact write issues
        pass


def _merge_result_json(target: Path, updates: Dict) -> None:
    """Merge updates into existing result.json and write atomically."""
    existing: Dict = {}
    try:
        if target.exists():
            existing = json.loads(target.read_text(encoding='utf-8') or '{}')
    except Exception:
        existing = {}
    existing.update(updates)
    existing['updated_at'] = datetime.utcnow().isoformat() + 'Z'
    _atomic_write_json(target, existing)


def _append_jsonl_atomic(target: Path, record: Dict) -> None:
    """Append a single JSON record as a line to a JSONL file with best-effort locking."""
    try:
        target.parent.mkdir(parents=True, exist_ok=True)
        with open(target, 'a', encoding='utf-8') as f:
            try:
                fcntl.flock(f.fileno(), fcntl.LOCK_EX)
            except Exception:
                # If flock is unavailable, proceed without locking
                pass
            f.write(json.dumps(record, ensure_ascii=False) + '\n')
            try:
                f.flush()
                os.fsync(f.fileno())
            except Exception:
                pass
            try:
                fcntl.flock(f.fileno(), fcntl.LOCK_UN)
            except Exception:
                pass
    except Exception:
        # Do not fail the task due to artifact write issues
        pass


@huey.task(retries=DEFAULT_TASK_RETRIES, retry_delay=DEFAULT_RETRY_DELAY)
def process_question(
    run_id: str,
    question_id: str,
    worker_id: Optional[str] = None
) -> Dict:
    """
    Process a single question by calling the existing single_question_runner.

    Args:
        run_id: Unique identifier for this benchmark run
        question_data: Complete question data from dataset
        config_path: Path to configuration TOML file
        start_session_index: Session index to start from (for resume)
        worker_id: Identifier for this worker process

    Returns:
        Dict with processing results and statistics
    """
    _log_startup_config()
    logger.info(f"Worker {worker_id}: Processing question {question_id}")
    tracker = ProgressTracker()

    try:
        # Load config path and question JSON from DB
        run_cfg = tracker.get_run_config(run_id) or {}
        config_path = run_cfg.get('config_path')
        question_data = tracker.get_question_json(run_id, question_id)

        if not question_data or not config_path:
            raise RuntimeError(f"Missing question/config for {run_id}/{question_id}")

        # No-op if already completed
        details = tracker.get_question_details(run_id, question_id)
        if details and details.get('ingestion_status') == 'completed':
            logger.info(f"Question {question_id} already ingested; skipping ingestion")
            # If QA isn't completed yet (pending or failed), enqueue QA
            qa_status = (details.get('qa_status') or '').strip()
            if qa_status in ('pending', 'failed'):
                logger.info(f"Question {question_id}: scheduling QA (status={qa_status})")
                # Enqueue QA task immediately
                run_qa(run_id, question_id)
            else:
                logger.info(f"Question {question_id}: QA status is {qa_status or 'unknown'}; not scheduling")
            return {
                'vault_id': details.get('vault_id'),
                'memory_id': details.get('memory_id'),
                'sessions_completed': details.get('completed_sessions', 0),
                'status': 'success'
            }

        # Mark in-progress
        tracker.mark_ingestion_in_progress(run_id, question_id, worker_id)

        # Compute resume start index from DB
        start_session_index = int(details.get('completed_sessions', 0)) if details else 0

        # Create memory name and run in-process runner
        memory_title = f"{run_id}_{question_id}"
        # Pass vault_id and memory_id if resuming
        result = _run_single_question_inprocess(
            question_data=question_data,
            config_path=config_path,
            memory_title=memory_title,
            start_session_index=start_session_index,
            run_id=run_id,
            vault_id=details.get('vault_id') if details else None,
            memory_id=details.get('memory_id') if details else None
        )

        # Extract vault_id and memory_id from result; fallback to DB
        vault_id = result.get('vault_id')
        memory_id = result.get('memory_id')
        details_after = tracker.get_question_details(run_id, question_id) or {}
        if not vault_id:
            vault_id = details_after.get('vault_id')
        if not memory_id:
            memory_id = details_after.get('memory_id')
        if vault_id and memory_id:
            tracker.update_vault_memory(run_id, question_id, vault_id, memory_id)

        # Update sessions from result or DB
        sessions_completed = result.get('sessions_completed')
        if sessions_completed is None:
            sessions_completed = int(details_after.get('completed_sessions') or 0)
        tracker.update_session_progress(run_id, question_id, sessions_completed)

        # Mark ingestion as complete when completed sessions match expected total
        total_sessions = (
            (details.get('total_sessions') if details else None)
        )
        if total_sessions is None:
            total_sessions = len(
                (question_data.get('sessions') or question_data.get('haystack_sessions') or [])
            )
        if sessions_completed == total_sessions:
            tracker.mark_ingestion_complete(run_id, question_id)
            logger.info(f"Question {question_id}: Ingestion completed ({sessions_completed} sessions)")

            # Enqueue QA task by ID (task will load from DB)
            run_qa(run_id, question_id)
        else:
            logger.warning(f"Question {question_id}: Partial completion ({sessions_completed}/{len(question_data.get('haystack_sessions', []))} sessions)")

        return result

    except Exception as e:
        logger.error(f"Failed to process question {question_id}: {e}")
        tracker.mark_failed(run_id, question_id, str(e))
        raise


@huey.task(retries=2, retry_delay=30)
def run_qa(
    run_id: str,
    question_id: str
) -> Dict:
    """
    Run QA phase for a completed question.

    Args:
        run_id: Unique identifier for this benchmark run
        question_id: Question identifier
        vault_id: Mycelian vault UUID
        memory_id: Mycelian memory UUID
        config_path: Path to configuration TOML file

    Returns:
        Dict with QA results
    """
    _log_startup_config()
    logger.info(f"Running QA for question {question_id}")
    tracker = ProgressTracker()

    try:
        # Load config, question, and ids from DB
        run_cfg = tracker.get_run_config(run_id) or {}
        config_path = run_cfg.get('config_path')
        details = tracker.get_question_details(run_id, question_id) or {}
        vault_id = details.get('vault_id')
        memory_id = details.get('memory_id')
        if not config_path or not vault_id or not memory_id:
            raise RuntimeError(f"Missing QA prerequisites for {run_id}/{question_id}")

        # Always run QA when requested (QA-only reruns supported)

        tracker.mark_qa_in_progress(run_id, question_id)

        # Load full question JSON for QA context
        question_data = tracker.get_question_json(run_id, question_id) or {'question_id': question_id}

        # Call QA runner in-process
        result = _run_qa_inprocess(
            question_data=question_data,
            vault_id=vault_id,
            memory_id=memory_id,
            config_path=config_path,
            run_id=run_id
        )

        # Mark QA as complete
        tracker.mark_qa_complete(run_id, question_id)
        logger.info(f"Question {question_id}: QA completed")

        return result

    except Exception as e:
        logger.error(f"QA failed for question {question_id}: {e}")
        tracker.mark_failed(run_id, question_id, f"QA failed: {str(e)}")
        raise


def _run_single_question_inprocess(
    question_data: Dict,
    config_path: str,
    memory_title: str,
    start_session_index: int = 0,
    run_id: str = "",
    vault_id: Optional[str] = None,
    memory_id: Optional[str] = None
) -> Dict:
    """Run the single-question pipeline in-process inside the worker."""
    # Prepare directories: logs/<run_id>/ for logs & artifacts, out/run_<run_id>/ for aggregated files
    target_run = run_id or "default"
    out_dir = Path("out") / (target_run if target_run.startswith("run_") else f"run_{target_run}")
    logs_dir = Path(LOGS_DIR) / (run_id or "default")
    logs_dir.mkdir(parents=True, exist_ok=True)
    log_path = logs_dir / f"{memory_title}.log"

    # Load config and build cfg object (resolve relative path under bench root)
    cfg_file = resolve_under_root(config_path)
    with open(cfg_file, 'rb') as f:
        cfg_dict = tomllib.load(f)
    cfg = parse_config(cfg_dict)
    cfg.run_id = run_id

    # Apply start-session slicing if resuming
    q = dict(question_data)
    # Normalize haystack_sessions -> sessions for runner ingestion
    if not isinstance(q.get('sessions'), list):
        hs = q.get('haystack_sessions')
        if isinstance(hs, list):
            q['sessions'] = [ {'messages': s} if isinstance(s, list) else s for s in hs ]
    if start_session_index and start_session_index > 0:
        sessions = q.get('sessions')
        if isinstance(sessions, list):
            q['sessions'] = sessions[start_session_index:]
        else:
            hs = q.get('haystack_sessions') or []
            if isinstance(hs, list):
                q['haystack_sessions'] = hs[start_session_index:]

    # Invoke runner with per-question log
    with open(log_path, 'a', encoding='utf-8') as qlog:
        # Ensure MCP client and vault exist; MemoryManager will create/get vault
        mcp_client = create_mcp_client()
        mm = MemoryManager(mcp_client, debug=False)
        # Resolve a valid vault_id from vault_title if not provided
        if not vault_id:
            vault_id = mm.ensure_vault(cfg.vault_title, getattr(cfg, 'vault_id', None))
        runner = SingleQuestionRunner(cfg, mcp_client=mcp_client, mode="ingestion")
        result = runner.run_question(q, vault_id=vault_id, run_id=run_id, log=qlog, memory_id=memory_id)

    # Read identifiers and counters from DB to ensure consistency
    tracker = ProgressTracker()
    qid = q.get("question_id")
    details = tracker.get_question_details(run_id, qid) or {}
    artifact = {
        'vault_id': details.get('vault_id') or result.get('vault_id'),
        'memory_id': details.get('memory_id') or result.get('memory_id'),
        'sessions_completed': int(details.get('completed_sessions') or result.get('sessions_completed') or 0),
        'messages_processed': int(details.get('ingested_messages') or result.get('messages_processed') or 0),
        'status': 'success',
        'error': None,
    }
    # Write per-question result.json
    result_path = logs_dir / f"{qid}.result.json"
    _merge_result_json(result_path, {
        'run_id': run_id,
        'question_id': qid,
        **artifact,
    })
    return artifact


def _run_qa_inprocess(
    question_data: Dict,
    vault_id: str,
    memory_id: str,
    config_path: str,
    run_id: str
) -> Dict:
    """Run the QA phase in-process using SingleQuestionRunner."""
    target_run = run_id or "default"
    out_dir = Path("out") / (target_run if target_run.startswith("run_") else f"run_{target_run}")
    logs_dir = Path(LOGS_DIR) / (run_id or "default")
    logs_dir.mkdir(parents=True, exist_ok=True)
    log_path = logs_dir / f"{question_data.get('question_id', 'unknown')}_qa.log"

    cfg_file = resolve_under_root(config_path)
    with open(cfg_file, 'rb') as f:
        cfg_dict = tomllib.load(f)
    cfg = parse_config(cfg_dict)
    cfg.run_id = run_id

    with open(log_path, 'a', encoding='utf-8') as qlog:
        mcp_client = create_mcp_client()
        # Ensure pending writes for this memory are visible before QA
        try:
            mm = MemoryManager(mcp_client, debug=False)
            qlog.write(f"AWAIT_CONSISTENCY (orchestrator) memory_id={memory_id}\n")
            mm._call_tool("await_consistency", {"memory_id": memory_id})
            qlog.write("SLEEP_BEFORE_QA seconds=5\n")
            qlog.flush()
            time.sleep(5)
        except Exception:
            # Best-effort; continue even if await_consistency is unavailable
            pass
        # Vault is provided; build runner in QA mode
        runner = SingleQuestionRunner(cfg, mcp_client=mcp_client, mode="qa")
        result = runner.run_question(question_data, vault_id=vault_id, run_id=run_id, log=qlog, memory_id=memory_id)

    # Return minimal structure; DB is the source of truth
    artifact = {
        'question_id': question_data.get('question_id'),
        'status': 'success'
    }
    # Include hypothesis in per-question result.json and append to aggregated hypotheses.jsonl
    hypothesis = (result.get('hypothesis') if isinstance(result, dict) else None)
    result_path = logs_dir / f"{question_data.get('question_id','unknown')}.result.json"
    _merge_result_json(result_path, {
        'run_id': run_id,
        **artifact,
        'vault_id': vault_id,
        'memory_id': memory_id,
        'hypothesis': hypothesis,
    })
    # Append to out/run_<run_id>/hypotheses.jsonl
    jsonl_path = out_dir / 'hypotheses.jsonl'
    _append_jsonl_atomic(jsonl_path, {
        'run_id': run_id,
        'question_id': question_data.get('question_id'),
        'vault_id': vault_id,
        'memory_id': memory_id,
        'model': getattr(cfg.models, 'qa', None),
        'hypothesis': hypothesis,
        'created_at': datetime.utcnow().isoformat() + 'Z',
    })
    return artifact


@huey.task()
def check_run_health(run_id: str) -> Dict:
    """
    Check health and progress of a benchmark run.

    Returns current statistics and identifies any stuck tasks.
    """
    tracker = ProgressTracker()
    stats = tracker.get_run_stats(run_id)

    # Add percentage completion
    if stats['total_sessions_expected'] > 0:
        stats['session_completion_pct'] = (
            stats['total_sessions_completed'] / stats['total_sessions_expected'] * 100
        )

    if stats['total_questions'] > 0:
        stats['question_completion_pct'] = (
            stats['completed'] / stats['total_questions'] * 100
        )

    logger.info(f"Run {run_id}: {stats['completed']}/{stats['total_questions']} questions complete")

    return stats

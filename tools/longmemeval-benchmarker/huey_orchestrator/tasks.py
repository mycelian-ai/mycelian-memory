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
from typing import Dict, Optional, Tuple

# Add parent directory to path to import existing modules
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from .huey_config import (
    huey,
    DEFAULT_TASK_RETRIES,
    DEFAULT_RETRY_DELAY,
    QUESTION_PROCESSING_TIMEOUT,
    QA_TIMEOUT_SEC,
    LOGS_DIR,
)
from .progress_tracker import ProgressTracker

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
_BENCH_SCRIPT = _SRC_DIR / "benchmarker.py"


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
            logger.info(f"Question {question_id} already ingested; skipping")
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

        # Create memory name and run subprocess
        memory_title = f"{run_id}_{question_id}"
        # Pass vault_id and memory_id if resuming
        result = _run_single_question_subprocess(
            question_data=question_data,
            config_path=config_path,
            memory_title=memory_title,
            start_session_index=start_session_index,
            run_id=run_id,
            vault_id=details.get('vault_id') if details else None,
            memory_id=details.get('memory_id') if details else None
        )
        
        # Extract vault_id and memory_id from result
        vault_id = result.get('vault_id')
        memory_id = result.get('memory_id')
        
        if vault_id and memory_id:
            tracker.update_vault_memory(run_id, question_id, vault_id, memory_id)
        
        # Update progress
        sessions_completed = result.get('sessions_completed', 0)
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

        # Skip if already completed
        if details.get('qa_status') == 'completed':
            logger.info(f"Question {question_id} already QA-complete; skipping")
            return {'status': 'success', 'question_id': question_id}

        tracker.mark_qa_in_progress(run_id, question_id)

        # Call QA runner subprocess
        result = _run_qa_subprocess(
            question_id=question_id,
            vault_id=vault_id,
            memory_id=memory_id,
            config_path=config_path,
            output_dir=f"out/{run_id}"
        )
        
        # Mark QA as complete
        tracker.mark_qa_complete(run_id, question_id)
        logger.info(f"Question {question_id}: QA completed")
        
        return result
        
    except Exception as e:
        logger.error(f"QA failed for question {question_id}: {e}")
        tracker.mark_failed(run_id, question_id, f"QA failed: {str(e)}")
        raise


def _run_single_question_subprocess(
    question_data: Dict,
    config_path: str,
    memory_title: str,
    start_session_index: int = 0,
    run_id: str = "",
    vault_id: Optional[str] = None,
    memory_id: Optional[str] = None
) -> Dict:
    """
    Run single_question_runner as a subprocess.
    
    This function calls the real runner CLI or mock runner based on environment.
    """
    import tempfile
    
    # Save question data to temp file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(question_data, f)
        question_json_path = f.name
    
    try:
        # Use benchmarker single-question mode with all options
        cmd = [
            sys.executable,
            str(_BENCH_SCRIPT),
            "--config", config_path,
            "--run-id", run_id,
            "--mode", "ingestion",
            "--question-id", question_data.get("question_id"),
            "--start-session", str(start_session_index),
            "--workers", "1"  # always single-threaded from subprocess
        ]
        
        # Add optional vault_id and memory_id if provided (for resume)
        if vault_id:
            cmd.extend(["--vault-id", vault_id])
        if memory_id:
            cmd.extend(["--memory-id", memory_id])
        
        logger.info(f"Running subprocess: {' '.join(cmd[:3])}...")
        
        # Run subprocess
        # Prepare per-question log file under logs/{run_id}
        target_run = run_id or "default"
        logs_dir = Path(LOGS_DIR) / target_run
        logs_dir.mkdir(parents=True, exist_ok=True)
        log_file = logs_dir / f"{memory_title}.log"

        # Pass full environment to subprocess
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=QUESTION_PROCESSING_TIMEOUT,
            env=os.environ.copy(),
        )
        
        if result.returncode != 0:
            # Write stdout/stderr to log file for triage
            try:
                log_file.write_text((result.stdout or '') + "\n" + (result.stderr or ''))
            except Exception:
                pass
            raise RuntimeError(f"Ingestion failed (see {log_file}): {result.stderr.strip()}")
        
        # Parse JSON output
        output_lines = result.stdout.strip().split('\n')
        # Find the JSON line (last line that starts with '{')
        json_output = None
        for line in reversed(output_lines):
            if line.strip().startswith('{'):
                json_output = line
                break
        
        if not json_output:
            raise ValueError(f"No JSON output found in subprocess output")
        
        result_data = json.loads(json_output)
        
        # Map to expected format
        return {
            'vault_id': result_data.get('vault_id'),
            'memory_id': result_data.get('memory_id'),
            'sessions_completed': result_data.get('sessions_completed', 0),
            'messages_processed': result_data.get('messages_processed', 0),
            'status': result_data.get('status', 'unknown'),
            'error': result_data.get('error')
        }
        
    finally:
        # Clean up temp file
        os.unlink(question_json_path)


def _run_qa_subprocess(
    question_id: str,
    vault_id: str,
    memory_id: str,
    config_path: str,
    output_dir: str
) -> Dict:
    """
    Run QA evaluation as a subprocess.
    
    This calls the mock QA runner (or real runner when ready).
    """
    import tempfile
    
    # Need to get question data for QA
    # In real implementation, this would be passed or loaded from dataset
    # For now, create minimal question data
    question_data = {
        'question_id': question_id,
        'question': 'Mock question',
        'answer': 'Mock answer'
    }
    
    # Save question data to temp file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(question_data, f)
        question_json_path = f.name
    
    try:
        # Get run_id from output_dir pattern (out/{run_id})
        run_id = output_dir.split('/')[-1] if '/' in output_dir else output_dir.replace('out/', '')
        
        cmd = [
            sys.executable,
            str(_BENCH_SCRIPT),
            "--config", config_path,
            "--run-id", run_id,
            "--mode", "qa",
            "--question-id", question_id,
            "--vault-id", vault_id,
            "--memory-id", memory_id,
            "--workers", "1"
        ]
        
        logger.info(f"Running QA subprocess for question {question_id}")
        
        # Pass full environment to subprocess
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=QA_TIMEOUT_SEC,
            env=os.environ.copy(),
        )
        
        if result.returncode != 0:
            raise RuntimeError(f"QA failed: {result.stderr.strip()}")
        
        # Parse JSON output
        output_lines = result.stdout.strip().split('\n')
        json_output = None
        for line in reversed(output_lines):
            if line.strip().startswith('{'):
                json_output = line
                break
        
        if not json_output:
            raise ValueError(f"No JSON output found in QA subprocess output")
        
        result_data = json.loads(json_output)
        
        # Handle hypothesis as string or dict
        hypothesis = result_data.get('hypothesis', '')
        confidence = 0.0
        if isinstance(hypothesis, dict):
            confidence = hypothesis.get('confidence', 0.0)
        
        return {
            'question_id': question_id,
            'answer_correct': result_data.get('is_correct', False),
            'confidence': confidence,
            'status': result_data.get('status', 'unknown')
        }
        
    finally:
        # Clean up temp file
        os.unlink(question_json_path)


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
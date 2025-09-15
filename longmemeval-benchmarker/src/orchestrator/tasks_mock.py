#!/usr/bin/env python3
"""
Mock task implementations for testing the orchestrator.
These use the mock_runner instead of the real benchmarker.
"""

import os
import sys
import json
import logging
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, Optional

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from orchestrator_config import huey, DEFAULT_TASK_RETRIES, DEFAULT_RETRY_DELAY
from progress_tracker import ProgressTracker

logger = logging.getLogger('orchestrator.tasks.mock')


@huey.task(retries=DEFAULT_TASK_RETRIES, retry_delay=DEFAULT_RETRY_DELAY)
def process_question_mock(
    run_id: str,
    question_data: Dict,
    config_path: str,
    start_session_index: int = 0,
    worker_id: Optional[str] = None
) -> Dict:
    """
    Process a question using the mock runner for testing.

    Args:
        run_id: Unique identifier for this benchmark run
        question_data: Complete question data from dataset
        config_path: Path to configuration TOML file
        start_session_index: Session index to start from (for resume)
        worker_id: Identifier for this worker process

    Returns:
        Dict with processing results and statistics
    """
    question_id = question_data['question_id']
    logger.info(f"[MOCK] Worker {worker_id}: Processing question {question_id} from session {start_session_index}")

    # Initialize progress tracker
    tracker = ProgressTracker()

    try:
        # Create memory name
        memory_title = f"{run_id}_{question_id}"

        # Call the mock runner
        result = _run_mock_subprocess(
            question_data=question_data,
            config_path=config_path,
            memory_title=memory_title,
            start_session_index=start_session_index
        )

        # Extract vault_id and memory_id from result
        vault_id = result.get('vault_id')
        memory_id = result.get('memory_id')

        if vault_id and memory_id:
            tracker.update_vault_memory(run_id, question_id, vault_id, memory_id)

        # Update progress
        sessions_completed = result.get('sessions_completed', 0)
        tracker.update_session_progress(run_id, question_id, sessions_completed)

        # Mark ingestion as complete if all sessions done
        if sessions_completed == len(question_data.get('haystack_sessions', [])):
            tracker.mark_ingestion_complete(run_id, question_id)
            logger.info(f"[MOCK] Question {question_id}: Ingestion completed ({sessions_completed} sessions)")
        else:
            logger.warning(f"[MOCK] Question {question_id}: Partial completion ({sessions_completed}/{len(question_data.get('haystack_sessions', []))} sessions)")

        return result

    except Exception as e:
        logger.error(f"[MOCK] Failed to process question {question_id}: {e}")
        tracker.mark_failed(run_id, question_id, str(e))
        raise


def _run_mock_subprocess(
    question_data: Dict,
    config_path: str,
    memory_title: str,
    start_session_index: int = 0
) -> Dict:
    """
    Run the mock runner as a subprocess for testing.
    """
    # Save question data to temp file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(question_data, f)
        question_json_path = f.name

    try:
        # Build command for mock runner
        cmd = [
            sys.executable,
            str(Path(__file__).parent / "mock_runner.py"),
            "--mode", "ingest",
            "--question-json", question_json_path,
            "--config", config_path,
            "--memory-title", memory_title,
            "--start-session", str(start_session_index)
        ]

        logger.info(f"[MOCK] Running subprocess: {' '.join(cmd[:3])}...")

        # Run subprocess
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30  # Shorter timeout for mock
        )

        if result.returncode != 0:
            raise RuntimeError(f"Mock subprocess failed: {result.stderr}")

        # Parse JSON output
        output_lines = result.stdout.strip().split('\n')
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

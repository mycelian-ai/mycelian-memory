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

from huey_config import huey, DEFAULT_TASK_RETRIES, DEFAULT_RETRY_DELAY
from progress_tracker import ProgressTracker

logger = logging.getLogger('orchestrator.tasks')


@huey.task(retries=DEFAULT_TASK_RETRIES, retry_delay=DEFAULT_RETRY_DELAY)
def process_question(
    run_id: str,
    question_data: Dict,
    config_path: str,
    start_session_index: int = 0,
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
    question_id = question_data['question_id']
    logger.info(f"Worker {worker_id}: Processing question {question_id} from session {start_session_index}")
    
    # Initialize progress tracker
    tracker = ProgressTracker()
    
    try:
        # Create memory name
        memory_title = f"{run_id}_{question_id}"
        
        # Call the existing single_question_runner as a subprocess
        # This keeps it isolated and allows the existing code to work unchanged
        result = _run_single_question_subprocess(
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
        
        # Mark ingestion as complete
        if sessions_completed == len(question_data.get('haystack_sessions', [])):
            tracker.mark_ingestion_complete(run_id, question_id)
            logger.info(f"Question {question_id}: Ingestion completed ({sessions_completed} sessions)")
            
            # Trigger QA task
            # In immediate mode, this will run synchronously
            # In normal mode, it will be queued
            run_qa(run_id, question_id, vault_id, memory_id, config_path)
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
    question_id: str, 
    vault_id: str,
    memory_id: str,
    config_path: str
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
    logger.info(f"Running QA for question {question_id} (memory: {memory_id})")
    
    tracker = ProgressTracker()
    
    try:
        # Call QA runner subprocess
        result = _run_qa_subprocess(
            question_id=question_id,
            vault_id=vault_id,
            memory_id=memory_id,
            config_path=config_path,
            output_dir=f"out/run_{run_id}"
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
    start_session_index: int = 0
) -> Dict:
    """
    Run single_question_runner as a subprocess.
    
    This function calls the mock runner (or real runner when ready).
    """
    import tempfile
    
    # Save question data to temp file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(question_data, f)
        question_json_path = f.name
    
    try:
        # Build command
        cmd = [
            sys.executable,
            str(Path(__file__).parent / "mock_runner.py"),
            "--mode", "ingest",
            "--question-json", question_json_path,
            "--config", config_path,
            "--memory-title", memory_title,
            "--start-session", str(start_session_index)
        ]
        
        logger.info(f"Running subprocess: {' '.join(cmd[:3])}...")
        
        # Run subprocess
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300  # 5 minute timeout
        )
        
        if result.returncode != 0:
            raise RuntimeError(f"Subprocess failed: {result.stderr}")
        
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
        import os
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
        # Build command
        cmd = [
            sys.executable,
            str(Path(__file__).parent / "mock_runner.py"),
            "--mode", "qa",
            "--question-json", question_json_path,
            "--config", config_path,
            "--vault-id", vault_id,
            "--memory-id", memory_id,
            "--output-dir", output_dir
        ]
        
        logger.info(f"Running QA subprocess for question {question_id}")
        
        # Run subprocess
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=60  # 1 minute timeout for QA
        )
        
        if result.returncode != 0:
            raise RuntimeError(f"QA subprocess failed: {result.stderr}")
        
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
        
        return {
            'question_id': question_id,
            'answer_correct': result_data.get('is_correct', False),
            'confidence': result_data.get('hypothesis', {}).get('confidence', 0.0),
            'status': result_data.get('status', 'unknown')
        }
        
    finally:
        # Clean up temp file
        import os
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
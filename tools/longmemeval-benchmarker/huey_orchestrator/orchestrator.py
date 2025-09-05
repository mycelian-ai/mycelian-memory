#!/usr/bin/env python3
"""
Main orchestrator script for Huey-based LongMemEval benchmarker.
Enqueues questions and monitors progress.
"""

import os
import sys
import json
import time
import click
import logging
from pathlib import Path
from datetime import datetime
from typing import List, Dict, Optional

from huey_config import huey
from progress_tracker import ProgressTracker
from tasks import process_question, check_run_health

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger('orchestrator.main')


def load_dataset(dataset_path: str) -> List[Dict]:
    """Load LongMemEval dataset from JSON file."""
    path = Path(dataset_path)
    if not path.exists():
        raise FileNotFoundError(f"Dataset not found: {dataset_path}")
    
    with open(path, 'r') as f:
        data = json.load(f)
    
    logger.info(f"Loaded {len(data)} questions from {dataset_path}")
    return data


def generate_run_id() -> str:
    """Generate a unique run ID based on timestamp."""
    return f"run_{int(time.time())}"


@click.command()
@click.argument('dataset_path', type=click.Path(exists=True))
@click.argument('config_path', type=click.Path(exists=True))
@click.option('--num-questions', '-n', default=None, type=int,
              help='Number of questions to process (default: all)')
@click.option('--resume', '-r', is_flag=True,
              help='Resume an existing run')
@click.option('--run-id', default=None,
              help='Specify run ID (for resume or custom ID)')
@click.option('--workers', '-w', default=1, type=int,
              help='Number of worker processes to use')
@click.option('--monitor', '-m', is_flag=True,
              help='Monitor mode: show progress without enqueueing')
def main(dataset_path: str, config_path: str, num_questions: Optional[int],
         resume: bool, run_id: Optional[str], workers: int, monitor: bool):
    """
    Orchestrate LongMemEval benchmark execution using Huey.
    
    DATASET_PATH: Path to LongMemEval dataset JSON file
    CONFIG_PATH: Path to configuration TOML file
    """
    
    # Initialize progress tracker
    tracker = ProgressTracker()
    
    if monitor:
        # Monitor mode - just show progress
        if not run_id:
            click.echo("Error: --run-id required for monitor mode")
            return 1
        
        monitor_progress(tracker, run_id)
        return 0
    
    # Load dataset
    dataset = load_dataset(dataset_path)
    
    if num_questions:
        dataset = dataset[:num_questions]
        logger.info(f"Limited to {len(dataset)} questions")
    
    # Determine run ID
    if resume:
        if not run_id:
            click.echo("Error: --run-id required for resume")
            return 1
        logger.info(f"Resuming run: {run_id}")
    else:
        if not run_id:
            run_id = generate_run_id()
        logger.info(f"Starting new run: {run_id}")
        
        # Initialize run in database
        tracker.init_run(run_id, dataset)
    
    # Get questions to process
    if resume:
        # Get pending and resumable questions
        pending = tracker.get_pending_questions(run_id)
        resumable = tracker.get_resumable_questions(run_id)
        
        logger.info(f"Found {len(pending)} pending questions")
        logger.info(f"Found {len(resumable)} resumable questions")
        
        # Enqueue resumable questions with their progress
        for q_progress in resumable:
            question_id = q_progress['question_id']
            # Find question data in dataset
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if question_data:
                start_session = q_progress['completed_sessions']
                logger.info(f"Resuming {question_id} from session {start_session}")
                enqueue_question(question_data, run_id, config_path, start_session)
        
        # Enqueue pending questions
        for q_progress in pending:
            question_id = q_progress['question_id']
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if question_data:
                enqueue_question(question_data, run_id, config_path, 0)
    else:
        # Enqueue all questions
        logger.info(f"Enqueueing {len(dataset)} questions...")
        for question in dataset:
            enqueue_question(question, run_id, config_path, 0)
    
    # Show instructions for starting workers
    click.echo("\n" + "="*60)
    click.echo("Questions enqueued successfully!")
    click.echo("="*60)
    click.echo(f"\nRun ID: {run_id}")
    click.echo(f"\nTo start processing, run workers in separate terminals:")
    click.echo(f"  python -m huey_orchestrator.worker")
    click.echo(f"\nYou can start up to {workers} workers for parallel processing.")
    click.echo(f"\nTo monitor progress:")
    click.echo(f"  python -m huey_orchestrator.orchestrator {dataset_path} {config_path} --monitor --run-id {run_id}")
    
    # If only one worker requested, offer to run in-process
    if workers == 1:
        if click.confirm("\nRun worker in this process?"):
            logger.info("Starting in-process worker...")
            from worker import run_worker
            run_worker()
    
    return 0


def enqueue_question(question_data: Dict, run_id: str, config_path: str, 
                     start_session_index: int = 0):
    """Enqueue a question for processing."""
    question_id = question_data['question_id']
    
    # Generate worker ID (could be more sophisticated)
    import random
    worker_id = f"worker-{random.randint(1000, 9999)}"
    
    # Enqueue the task
    result = process_question(
        run_id=run_id,
        question_data=question_data,
        config_path=config_path,
        start_session_index=start_session_index,
        worker_id=worker_id
    )
    
    logger.debug(f"Enqueued question {question_id} (start from session {start_session_index})")


def monitor_progress(tracker: ProgressTracker, run_id: str):
    """Monitor progress of a benchmark run."""
    click.clear()
    
    try:
        while True:
            stats = tracker.get_run_stats(run_id)
            
            # Clear screen and show header
            click.clear()
            click.echo("="*60)
            click.echo(f"LongMemEval Benchmark Monitor - Run: {run_id}")
            click.echo("="*60)
            click.echo(f"Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
            click.echo()
            
            # Show progress
            total = stats['total_questions']
            completed = stats['completed']
            in_progress = stats['in_progress']
            failed = stats['failed']
            pending = stats['pending']
            
            click.echo(f"Questions: {completed}/{total} completed")
            click.echo(f"  - In Progress: {in_progress}")
            click.echo(f"  - Pending: {pending}")
            click.echo(f"  - Failed: {failed}")
            click.echo()
            
            # Session progress
            sessions_done = stats['total_sessions_completed']
            sessions_total = stats['total_sessions_expected']
            if sessions_total > 0:
                pct = sessions_done / sessions_total * 100
                click.echo(f"Sessions: {sessions_done}/{sessions_total} ({pct:.1f}%)")
            
            # Progress bar
            if total > 0:
                pct_complete = completed / total * 100
                bar_width = 40
                filled = int(bar_width * completed / total)
                bar = '█' * filled + '░' * (bar_width - filled)
                click.echo(f"\nProgress: [{bar}] {pct_complete:.1f}%")
            
            # Exit if complete
            if completed == total:
                click.echo("\n✅ Benchmark complete!")
                break
            
            # Refresh every 5 seconds
            time.sleep(5)
            
    except KeyboardInterrupt:
        click.echo("\n\nMonitoring stopped.")


if __name__ == '__main__':
    sys.exit(main())
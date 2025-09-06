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

try:
    import tomllib  # Python 3.11+
except ModuleNotFoundError:
    import tomli as tomllib

from .huey_config import huey
from .progress_tracker import ProgressTracker
from .tasks import process_question, check_run_health

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
@click.option('--auto', is_flag=True,
              help='Automatically start worker, monitor progress, and shut down on completion')
def main(config_path: str, num_questions: Optional[int],
         resume: bool, run_id: Optional[str], workers: int, monitor: bool, auto: bool):
    """
    Orchestrate LongMemEval benchmark execution using Huey.
    
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
    
    # Parse config to get dataset
    with open(config_path, 'rb') as f:
        cfg_dict = tomllib.load(f)

    dataset_path = cfg_dict.get('dataset_file_path')
    if not dataset_path:
        click.echo("Error: dataset_file_path missing in config TOML")
        return 1

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
        
        # Initialize run in database with metadata
        tracker.init_run(run_id, dataset, dataset_path=dataset_path, config_path=config_path)
    
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
    
    if auto:
        # Spawn worker, monitor progress, then terminate worker.
        import subprocess
        import signal
        click.echo("\n" + "="*60)
        click.echo("Auto mode: starting worker and monitoring run")
        click.echo("="*60)
        # Use same interpreter to ensure venv consistency
        worker_cmd = [sys.executable, '-m', 'huey_orchestrator.worker', '--workers', str(max(1, workers))]
        worker_proc = subprocess.Popen(worker_cmd)
        try:
            monitor_progress(tracker, run_id)
        finally:
            try:
                worker_proc.send_signal(signal.SIGINT)
                try:
                    worker_proc.wait(timeout=10)
                except Exception:
                    worker_proc.kill()
            except Exception:
                pass
        return 0
    else:
        # Show instructions for starting workers (async-only)
        click.echo("\n" + "="*60)
        click.echo("Questions enqueued successfully!")
        click.echo("="*60)
        click.echo(f"\nRun ID: {run_id}")
        click.echo(f"\nStart workers to process tasks:")
        click.echo(f"  python -m tools.longmemeval-benchmarker.huey_orchestrator.worker --workers {workers}")
        click.echo(f"\nMonitor progress:")
        click.echo(f"  python -m tools.longmemeval-benchmarker.huey_orchestrator.orchestrator {config_path} --monitor --run-id {run_id}")
        
        return 0


def enqueue_question(question_data: Dict, run_id: str, config_path: str, 
                     start_session_index: int = 0):
    """Enqueue a question for processing (by ID only)."""
    question_id = question_data['question_id']
    
    # Generate worker ID (could be more sophisticated)
    import random
    worker_id = f"worker-{random.randint(1000, 9999)}"
    
    # Enqueue the task with IDs only (task loads from DB)
    process_question(
        run_id=run_id,
        question_id=question_id,
        worker_id=worker_id
    )
    
    logger.debug(f"Enqueued question {question_id}")


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
            ingested = stats.get('ingested', 0)
            qa_done = stats.get('qa_done', 0)
            
            click.echo(f"Questions: {completed}/{total} completed")
            click.echo(f"  - In Progress: {in_progress}")
            click.echo(f"  - Pending: {pending}")
            click.echo(f"  - Failed: {failed}")
            click.echo()
            
            # Session progress
            sessions_done = stats['total_sessions_completed']
            sessions_total = stats['total_sessions_expected']
            msgs_done = stats.get('total_messages_ingested', 0)
            msgs_total = stats.get('total_messages_expected', 0)
            if sessions_total > 0:
                pct = sessions_done / sessions_total * 100
                click.echo(f"Sessions: {sessions_done}/{sessions_total} ({pct:.1f}%)")
            if msgs_total > 0:
                mpct = msgs_done / msgs_total * 100
                click.echo(f"Messages: {msgs_done}/{msgs_total} ({mpct:.1f}%)")
            
            # Ingestion progress
            if total > 0:
                bar_width = 30
                ingest_pct = ingested / total * 100
                filled_ing = int(bar_width * ingested / total)
                bar_ing = '█' * filled_ing + '░' * (bar_width - filled_ing)
                click.echo(f"Ingested : [{bar_ing}] {ingest_pct:.1f}%")

                qa_pct = qa_done / total * 100
                filled_qa = int(bar_width * qa_done / total)
                bar_qa = '█' * filled_qa + '░' * (bar_width - filled_qa)
                click.echo(f"QA done  : [{bar_qa}] {qa_pct:.1f}%")

                click.echo()

            # Overall completion bar
            if total > 0:
                pct_complete = completed / total * 100
                bar_width = 40
                filled = int(bar_width * completed / total)
                bar = '█' * filled + '░' * (bar_width - filled)
                click.echo(f"Overall  : [{bar}] {pct_complete:.1f}%")

            # In-progress details (ordered by recent activity)
            details = tracker.get_in_progress_details(run_id, limit=10)
            if details:
                click.echo("In-progress details:")
                for d in details:
                    qid = d['question_id']
                    s_done, s_total = d['s_done'], d['s_total']
                    m_done, m_total = d['m_done'], d['m_total']
                    istatus, qstatus = (d.get('ingestion_status') or ''), (d.get('qa_status') or '')
                    phase = 'INGEST' if istatus == 'in_progress' else ('QA' if qstatus == 'in_progress' else istatus.upper() or 'PENDING')
                    # Bars
                    def _bar(done, tot, width=20):
                        if tot and tot > 0:
                            filled = int(width * done / tot)
                            return '█' * filled + '░' * (width - filled)
                        return '░' * width
                    s_bar = _bar(s_done, s_total)
                    m_bar = _bar(m_done, m_total)
                    # Show QA only after ingestion complete
                    if istatus != 'completed':
                        click.echo(f"  {qid} [{phase}] Sessions {s_done}/{s_total} [{s_bar}]  Messages {m_done}/{m_total} [{m_bar}]  worker {d.get('worker_id') or '-'}")
                    else:
                        # QA phase: show simple status
                        if qstatus == 'in_progress':
                            click.echo(f"  {qid} [QA] running…  worker {d.get('worker_id') or '-'}")
                        elif qstatus == 'completed':
                            click.echo(f"  {qid} [QA] done")
                        else:
                            click.echo(f"  {qid} [QA] waiting for ingestion")

            # Stuck detection (after showing details)
            stuck = tracker.get_stuck_questions(run_id)
            if stuck:
                click.echo(f"\n⚠️  {len(stuck)} task(s) possibly stuck (>30m): {', '.join(stuck[:5])}{'…' if len(stuck) > 5 else ''}")
            
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
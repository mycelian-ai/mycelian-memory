#!/usr/bin/env python3
"""
Unified benchmark runner that handles both task enqueueing and worker processes.
This ensures both components use the same database and environment.
"""

import os
import sys
import time
import signal
import subprocess
from pathlib import Path
import multiprocessing
import click

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent))

def run_orchestrator(config_path: str, run_id: str, num_questions: int = None):
    """Run the orchestrator to enqueue tasks."""
    cmd = [
        sys.executable, '-m', 'src.orchestrator',
        config_path
    ]

    if run_id:
        cmd.extend(['--run-id', run_id])

    if num_questions:
        cmd.extend(['--num-questions', str(num_questions)])

    print(f"Enqueueing tasks with run_id: {run_id}")
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Failed to enqueue tasks: {result.stderr}")
        return False

    print(result.stdout)
    return True

def run_worker(num_workers: int = 1):
    """Run the Huey worker process."""
    cmd = [
        sys.executable, '-m', 'src.orchestrator.tasks',
        '--workers', str(num_workers)
    ]

    print(f"\nStarting {num_workers} worker(s)...")
    # This will run until interrupted
    subprocess.run(cmd)

def run_monitor(config_path: str, run_id: str):
    """Run the monitor in a separate process."""
    cmd = [
        sys.executable, '-m', 'src.orchestrator',
        config_path,
        '--monitor',
        '--run-id', run_id
    ]

    subprocess.Popen(cmd)

@click.command()
@click.argument('config_path', type=click.Path(exists=True))
@click.option('--workers', '-w', default=1, type=int,
              help='Number of worker processes')
@click.option('--run-id', default=None,
              help='Custom run ID (default: auto-generated)')
@click.option('--num-questions', '-n', default=None, type=int,
              help='Number of questions to process (default: all)')
@click.option('--monitor/--no-monitor', default=True,
              help='Show progress monitor (default: True)')
def main(config_path: str, workers: int, run_id: str, num_questions: int, monitor: bool):
    """
    Run LongMemEval benchmark with integrated orchestrator and workers.

    This script:
    1. Enqueues all tasks
    2. Starts worker processes
    3. Optionally shows progress monitor
    """

    # Generate run ID if not provided
    if not run_id:
        run_id = f"run_{int(time.time())}"

    print("="*60)
    print("LongMemEval Benchmark Runner")
    print("="*60)
    print(f"Config: {config_path}")
    print(f"Run ID: {run_id}")
    print(f"Workers: {workers}")
    print()

    # Step 1: Enqueue tasks
    if not run_orchestrator(config_path, run_id, num_questions):
        print("Failed to enqueue tasks. Exiting.")
        return 1

    # Step 2: Start monitor in background if requested
    monitor_proc = None
    if monitor:
        print("\nStarting progress monitor...")
        run_monitor(config_path, run_id)
        time.sleep(2)  # Give monitor time to start

    # Step 3: Start workers (this blocks until interrupted)
    try:
        run_worker(workers)
    except KeyboardInterrupt:
        print("\n\nShutting down workers...")
        return 0

    return 0

if __name__ == '__main__':
    # Ensure we're in the right directory
    os.chdir(Path(__file__).parent)

    # Handle Ctrl+C gracefully
    signal.signal(signal.SIGINT, lambda s, f: sys.exit(0))

    sys.exit(main())

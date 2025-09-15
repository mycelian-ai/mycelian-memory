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
import atexit
import subprocess
import signal
from pathlib import Path
import sqlite3
from datetime import datetime
from typing import List, Dict, Optional

import tomllib  # Python 3.11+

from src.orchestrator.progress_tracker import ProgressTracker
from src.paths import resolve_under_root

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger('orchestrator.main')


BENCHMARKER_ROOT = Path(__file__).resolve().parents[2]  # Go up 2 levels: orchestrator -> src -> benchmarker
DATA_DIR = BENCHMARKER_ROOT / 'data'
HUEY_DB_PATH = str(DATA_DIR / 'orchestrator.db')

# Lazily imported tasks module bound to the per-run queue
_TASKS_MOD = None

# Optional Rich UI
try:
    from rich.console import Console
    from rich.table import Table
    from rich.panel import Panel
    from rich.live import Live
    from rich.align import Align
    _RICH_AVAILABLE = True
    _console = Console()
except Exception:
    _RICH_AVAILABLE = False
    _console = None


def _start_worker_subprocess(workers: int, queue_name: str) -> subprocess.Popen:
    """Start the Huey worker in its own process group and return the Popen."""
    env = os.environ.copy()
    existing = env.get('PYTHONPATH', '')
    # Ensure the benchmarker root is importable as a top-level package
    env['PYTHONPATH'] = f"{str(BENCHMARKER_ROOT)}{(':' + existing) if existing else ''}"
    # Ensure worker binds to the intended per-run queue and log file
    if queue_name:
        env['HUEY_QUEUE_NAME'] = queue_name
        # Extract run_id from queue_name (format: "huey-{run_id}")
        if queue_name.startswith('huey-'):
            run_id = queue_name[5:]  # Remove "huey-" prefix
            env['HUEY_RUN_ID'] = run_id
    cmd = [sys.executable, '-m', 'src.orchestrator.worker', '--workers', str(max(1, workers))]
    # Run from the benchmarker root so relative paths and package imports work
    proc = subprocess.Popen(
        cmd,
        cwd=str(BENCHMARKER_ROOT),
        env=env,
        preexec_fn=os.setsid  # create a new process group for clean teardown
    )
    return proc


def _stop_worker_subprocess(proc: Optional[subprocess.Popen]) -> None:
    """Terminate the worker process group gracefully; force-kill if needed."""
    if not proc:
        return
    try:
        if proc.poll() is None:
            pgid = os.getpgid(proc.pid)
            os.killpg(pgid, signal.SIGTERM)
            try:
                proc.wait(timeout=10)
            except Exception:
                os.killpg(pgid, signal.SIGKILL)
    except Exception:
        # Best-effort cleanup; ignore errors on shutdown
        pass


_KILL_PATTERNS = (
    "python -m src.orchestrator.worker",
    "python -m src.orchestrator",
)


def _ps_list() -> List[tuple[int, int, str]]:
    """Return list of (pid, pgid, cmd)."""
    out = subprocess.check_output(["ps", "-eo", "pid,pgid,command"], text=True)
    rows: List[tuple[int, int, str]] = []
    for line in out.splitlines()[1:]:
        line = line.strip()
        if not line:
            continue
        try:
            parts = line.split(None, 2)
            if len(parts) < 3:
                continue
            pid = int(parts[0])
            pgid = int(parts[1])
            cmd = parts[2]
            rows.append((pid, pgid, cmd))
        except Exception:
            continue
    return rows


def _find_benchmarker_procs(exclude_pids: Optional[set[int]] = None) -> tuple[List[tuple[int, int, str]], List[int]]:
    rows = _ps_list()
    if exclude_pids is None:
        exclude_pids = set()
    # Match patterns and exclude any explicitly excluded PIDs (e.g., this process)
    targets = [r for r in rows if (r[0] not in exclude_pids) and any(pat in r[2] for pat in _KILL_PATTERNS)]
    pgids = sorted({r[1] for r in targets})
    return targets, pgids


def _pgid_alive(pgid: int) -> bool:
    try:
        return any(r[1] == pgid for r in _ps_list())
    except Exception:
        return False


def _kill_group(pgid: int, timeout: float = 5.0) -> str:
    try:
        os.killpg(pgid, signal.SIGTERM)
    except ProcessLookupError:
        return f"pgid {pgid}: already gone"
    except Exception as e:
        return f"pgid {pgid}: SIGTERM failed: {e}"

    deadline = time.time() + timeout
    while time.time() < deadline:
        if not _pgid_alive(pgid):
            return f"pgid {pgid}: terminated"
        time.sleep(0.2)

    try:
        os.killpg(pgid, signal.SIGKILL)
    except ProcessLookupError:
        return f"pgid {pgid}: terminated after SIGTERM"
    except Exception as e:
        return f"pgid {pgid}: SIGKILL failed: {e}"
    time.sleep(0.2)
    return f"pgid {pgid}: killed" if not _pgid_alive(pgid) else f"pgid {pgid}: WARNING still alive"


def _state_paths() -> List[Path]:
    """Return list of orchestrator state files to delete."""
    data_dir = BENCHMARKER_ROOT / 'data'
    return [
        Path(HUEY_DB_PATH),
        data_dir / 'orchestrator.db-shm',
        data_dir / 'orchestrator.db-wal',
        data_dir / 'progress.db',
        data_dir / 'progress.db-shm',
        data_dir / 'progress.db-wal',
    ]


def _clear_orchestrator_state() -> None:
    """Delete Huey task DB and progress DB (with WAL/SHM)."""
    for p in _state_paths():
        try:
            if p.exists():
                p.unlink()
        except Exception:
            # best-effort; ignore failures
            pass


def _run_preflight_checks(config_path: str, dataset_abs_path: str) -> bool:
    """Preflight validation of environment and inputs before enqueueing tasks."""
    ok = True
    click.echo("\nPreflight checks:")
    # Dataset path must exist
    if not Path(dataset_abs_path).exists():
        click.echo(f"  [FAIL] dataset file not found: {dataset_abs_path}")
        ok = False
    else:
        click.echo("  [OK] dataset path valid")

    # Venv check: recommend using project venv if present
    venv_python = BENCHMARKER_ROOT / 'venv' / 'bin' / 'python'
    if venv_python.exists():
        if Path(sys.executable) != venv_python:
            click.echo(f"  [WARN] using interpreter {sys.executable}; recommended: {venv_python}")
        else:
            click.echo("  [OK] using project venv interpreter")
    else:
        click.echo("  [INFO] no project venv found at venv/bin/python")

    # PYTHONPATH hint for child worker (we set it, but surface info)
    py_path = os.environ.get('PYTHONPATH', '')
    if str(BENCHMARKER_ROOT) in py_path:
        click.echo("  [OK] PYTHONPATH includes benchmarker root")
    else:
        click.echo("  [INFO] PYTHONPATH will be set for worker automatically")

    # DB readiness: ensure data directory and progress DB are usable
    data_dir = BENCHMARKER_ROOT / 'data'
    try:
        data_dir.mkdir(parents=True, exist_ok=True)
        db_path = data_dir / 'progress.db'
        with sqlite3.connect(db_path) as conn:
            conn.execute('CREATE TABLE IF NOT EXISTS runs (run_id TEXT PRIMARY KEY, dataset_path TEXT, config_path TEXT, created_at TEXT)')
            conn.execute('''CREATE TABLE IF NOT EXISTS question_progress (
                run_id TEXT, question_id TEXT, status TEXT,
                completed_sessions INTEGER DEFAULT 0,
                total_sessions INTEGER DEFAULT 0,
                ingestion_status TEXT, qa_status TEXT,
                PRIMARY KEY (run_id, question_id)
            )''')
        click.echo("  [OK] progress DB accessible")
    except Exception as e:
        click.echo(f"  [FAIL] progress DB error: {e}")
        ok = False

    return ok


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
@click.argument('config_path', required=False)
@click.option('--num-questions', '-n', default=None, type=int,
              help='Number of questions to process (default: all)')
@click.option('--resume', '-r', is_flag=True,
              help='Resume an existing run')
@click.option('--run-id', default=None,
              help='Specify run ID (for resume or custom ID)')
@click.option('--workers', '-w', default=1, type=int,
              help='Number of worker processes to use')
@click.option('--resume-mode', type=click.Choice(['restart-from-first-session', 'resume-from-next-session']), default='resume-from-next-session',
              help='On resume, either restart from the first session or resume from the next session after the last successfully ingested session')
@click.option('--monitor', '-m', is_flag=True,
              help='Monitor mode: show progress without enqueueing')
@click.option('--auto', is_flag=True,
              help='Automatically start worker, monitor progress, and shut down on completion')
@click.option('--clear-state', is_flag=True,
              help='Delete all orchestrator state (orchestrator.db*, progress.db*) and exit')
@click.option('--stop', is_flag=True,
              help='Stop all running benchmarker processes (workers and orchestrators) and exit')
@click.option('--force', is_flag=True,
              help='When used with --stop: do not prompt. With --resume: force retry all failed questions.')
@click.option('--qa-only', is_flag=True,
              help='Run only QA phase for an existing run (requires --run-id). Always re-runs QA for questions with completed ingestion.')
def main(config_path: str, num_questions: Optional[int],
         resume: bool, run_id: Optional[str], workers: int, resume_mode: str, monitor: bool, auto: bool, clear_state: bool,
         stop: bool, force: bool, qa_only: bool):
    """
    Orchestrate LongMemEval benchmark execution using Huey.

    CONFIG_PATH: Path to configuration TOML file
    """

    # Stop processes early if requested
    if stop:
        # Exclude this orchestrator (so we don't kill ourselves)
        mypid = os.getpid()
        targets, pgids = _find_benchmarker_procs(exclude_pids={mypid})
        click.echo("Found processes:" if targets else "No benchmarker processes found.")
        for pid, pgid, cmd in targets:
            click.echo(f"  pid={pid} pgid={pgid} cmd={cmd}")
        if targets:
            proceed = True
            if not force:
                proceed = click.confirm("Terminate these process groups?", default=False)
            if proceed:
                for g in pgids:
                    click.echo(_kill_group(g))
            else:
                click.echo("Aborted.")
                return 1
        if clear_state:
            click.echo("\nClearing orchestrator state files…")
            _clear_orchestrator_state()
        return 0

    # Clear state if requested (no config required)
    if clear_state and not auto and not monitor and not resume and not run_id and not config_path:
        click.echo("You are about to DELETE all orchestrator state (Huey queue and progress DB).\n")
        click.echo("Files to be removed:")
        for p in _state_paths():
            click.echo(f"  - {p}")
        click.echo("")
        if not click.confirm("Proceed?", default=False, show_default=True):
            click.echo("Aborted. No changes made.")
            return 1
        _clear_orchestrator_state()
        click.echo("State cleared: orchestrator.db* and progress.db* removed.")
        return 0

    # Initialize progress tracker
    tracker = ProgressTracker()

    # Handle QA-only mode
    if qa_only:
        if not run_id:
            click.echo("Error: --run-id is required for --qa-only mode")
            return 1

        logger.info(f"QA-only mode for run: {run_id}")

        # Bind per-run queue
        queue_name = f"huey-{run_id}"
        os.environ['HUEY_QUEUE_NAME'] = queue_name
        os.environ['HUEY_RUN_ID'] = run_id

        # Import tasks
        from src.orchestrator import tasks as _tasks

        # Get all completed-ingestion questions from tracker
        questions = tracker.get_questions_for_qa(run_id, include_completed=True)

        if not questions:
            click.echo(f"No completed questions found for run {run_id}")
            return 1

        click.echo(f"Found {len(questions)} questions to run QA for")

        # Enqueue QA tasks
        for row in questions:
            qid = row.get('question_id') if isinstance(row, dict) else (row[0] if row else None)
            if not qid:
                continue
            _tasks.run_qa(run_id, qid)

        click.echo(f"{len(questions)} QA tasks enqueued")

        if auto:
            click.echo("\nAuto mode: starting worker and monitoring")
            worker_proc = _start_worker_subprocess(workers, queue_name)
            atexit.register(lambda: _stop_worker_subprocess(worker_proc))

            try:
                monitor_progress(tracker, run_id)
            except KeyboardInterrupt:
                click.echo("\nStopping worker...")
            finally:
                _stop_worker_subprocess(worker_proc)
        else:
            click.echo(f"\nStart worker to process QA tasks:")
            click.echo(f"  HUEY_QUEUE_NAME={queue_name} python -m src.orchestrator.worker --workers {workers}")

        return 0

    if monitor:
        # Monitor mode - just show progress
        if not run_id:
            click.echo("Error: --run-id required for monitor mode")
            return 1

        monitor_progress(tracker, run_id)
        return 0

    # Parse config to get dataset
    if not config_path:
        click.echo("Error: CONFIG_PATH is required unless using --clear-state")
        return 1

    # Normalize config path early to an absolute path
    config_path = str(resolve_under_root(config_path))
    with open(config_path, 'rb') as f:
        cfg_dict = tomllib.load(f)

    dataset_path = cfg_dict.get('dataset_file_path')
    if not dataset_path:
        click.echo("Error: dataset_file_path missing in config TOML")
        return 1

    # Resolve dataset path (relative to current working directory)
    ds_path_obj = Path(dataset_path)
    if not ds_path_obj.is_absolute():
        # Resolve relative to current working directory (benchmarker root)
        ds_path_obj = Path.cwd() / ds_path_obj

    # Convert to absolute path for consistency
    dataset_abs_path = str(ds_path_obj.resolve())

    # Preflight checks
    if not _run_preflight_checks(config_path, str(ds_path_obj)):
        return 1

    dataset = load_dataset(str(ds_path_obj))

    if num_questions:
        dataset = dataset[:num_questions]
        logger.info(f"Limited to {len(dataset)} questions")

    if len(dataset) == 0:
        click.echo("Error: dataset contains 0 questions (nothing to enqueue)")
        return 1

    # Determine run ID
    if resume:
        if not run_id:
            click.echo("Error: --run-id required for resume")
            return 1
        logger.info(f"Resuming run: {run_id} (mode={resume_mode})")
    else:
        if not run_id:
            run_id = generate_run_id()
        logger.info(f"Starting new run: {run_id}")

        # Initialize run in database with metadata (persist absolute config path)
        tracker.init_run(run_id, dataset, dataset_path=dataset_path, config_path=str(config_path))

    # Bind per-run queue and log file before importing tasks
    queue_name = f"huey-{run_id}"
    os.environ['HUEY_QUEUE_NAME'] = queue_name
    os.environ['HUEY_RUN_ID'] = run_id
    # Import tasks now so they see the queue name and log file path
    global _TASKS_MOD
    from src.orchestrator import tasks as _tasks  # type: ignore
    _TASKS_MOD = _tasks

    # Get questions to process
    if resume:
        # Get pending and resumable questions
        pending = tracker.get_pending_questions(run_id)
        resumable = tracker.get_resumable_questions(run_id)
        # If forcing, include failed questions for retry regardless of previous error
        failed_to_retry = []
        if force:
            try:
                failed_to_retry = tracker.get_failed_details(run_id, limit=10_000)
            except Exception:
                failed_to_retry = []
        # Additionally, detect obviously stuck states and hard-reset them
        stuck_unstarted = tracker.get_inprogress_unstarted(run_id)
        stuck_qa = tracker.get_qa_inprogress_after_ingest(run_id)

        logger.info(f"Found {len(pending)} pending questions")
        logger.info(f"Found {len(resumable)} resumable questions")
        if force:
            logger.info(f"Force mode: will retry {len(failed_to_retry)} failed questions")
        if stuck_unstarted:
            logger.info(f"Found {len(stuck_unstarted)} in_progress with 0 sessions; hard-resetting from session 0")
        if stuck_qa:
            logger.info(f"Found {len(stuck_qa)} QA-in-progress after ingestion; action depends on resume-mode")

        # Track which question_ids we will (re)enqueue to avoid duplicates
        enqueued_ids = set()

        # For resumable questions, either restart from 0 or continue from last completed session
        for q_progress in resumable:
            question_id = q_progress['question_id']
            # Find question data in dataset
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if question_data:
                if resume_mode == 'restart-from-first-session':
                    logger.info(f"Restarting {question_id} from session 0 (clearing memory_id; preserving vault_id)")
                    tracker.reset_for_restart(run_id, question_id)
                    enqueue_question(question_data, run_id, config_path, 0)
                else:
                    done = int(q_progress.get('completed_sessions') or 0)
                    logger.info(f"Continuing {question_id} from session {done} using existing memory_id")
                    enqueue_question(question_data, run_id, config_path, 0)
                enqueued_ids.add(question_id)

        # Hard-reset stuck-in-progress with 0 sessions
        for q_progress in stuck_unstarted:
            question_id = q_progress['question_id']
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if question_data:
                logger.info(f"Hard-resetting stuck question {question_id} (0 sessions done) from session 0")
                tracker.reset_for_restart(run_id, question_id)
                enqueue_question(question_data, run_id, config_path, 0)
                enqueued_ids.add(question_id)

        # Handle QA-in-progress-after-ingest
        for q_progress in stuck_qa:
            question_id = q_progress['question_id']
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if question_data:
                if resume_mode == 'restart-from-first-session':
                    logger.info(f"Hard-resetting QA-stuck question {question_id} to re-ingest from session 0")
                    tracker.reset_for_restart(run_id, question_id)
                    enqueue_question(question_data, run_id, config_path, 0)
                else:
                    logger.info(f"Re-enqueuing QA-stuck question {question_id} without reset; QA will be scheduled")
                    enqueue_question(question_data, run_id, config_path, 0)
                enqueued_ids.add(question_id)

        # Force retry previously failed questions (regardless of retry_count)
        if force and failed_to_retry:
            for q_progress in failed_to_retry:
                question_id = q_progress['question_id']
                if question_id in enqueued_ids:
                    continue
                question_data = next((q for q in dataset if q['question_id'] == question_id), None)
                if question_data:
                    if resume_mode == 'restart-from-first-session':
                        logger.info(f"Force retrying failed question {question_id} from session 0")
                        tracker.reset_for_restart(run_id, question_id)
                        enqueue_question(question_data, run_id, config_path, 0)
                    else:
                        done = int((q_progress.get('completed_sessions') or 0))
                        logger.info(f"Force retrying failed question {question_id} continuing from session {done}")
                        enqueue_question(question_data, run_id, config_path, 0)
                    enqueued_ids.add(question_id)

        # Enqueue pending questions
        for q_progress in pending:
            question_id = q_progress['question_id']
            if question_id in enqueued_ids:
                continue
            question_data = next((q for q in dataset if q['question_id'] == question_id), None)
            if not question_data:
                continue
            status = (q_progress.get('status') or '').strip()
            if status == 'failed' and resume_mode == 'restart-from-first-session':
                logger.info(f"Resetting failed question {question_id} for full retry")
                tracker.reset_for_restart(run_id, question_id)
            elif status == 'failed' and resume_mode == 'resume-from-next-session':
                done = int(q_progress.get('completed_sessions') or 0)
                logger.info(f"Retrying failed question {question_id} from session {done}")
            enqueue_question(question_data, run_id, config_path, 0)
    else:
        # Enqueue all questions
        logger.info(f"Enqueueing {len(dataset)} questions...")
        for question in dataset:
            enqueue_question(question, run_id, config_path, 0)

    if auto:
        # Spawn worker in own process-group, monitor progress, tear down on exit/CTRL-C
        click.echo("\n" + "="*60)
        click.echo("Auto mode: starting worker and monitoring run")
        click.echo("="*60)
        worker_proc = _start_worker_subprocess(workers, queue_name)

        # Ensure cleanup on abnormal exits as well
        atexit.register(lambda: _stop_worker_subprocess(worker_proc))

        try:
            monitor_progress(tracker, run_id)
        except KeyboardInterrupt:
            click.echo("\nStopping worker...")
        finally:
            _stop_worker_subprocess(worker_proc)
        return 0
    else:
        # Show instructions for starting workers (async-only)
        click.echo("\n" + "="*60)
        click.echo("Questions enqueued successfully!")
        click.echo("="*60)
        click.echo(f"\nRun ID: {run_id}")
        click.echo(f"\nStart workers to process tasks:")
        click.echo(f"  HUEY_QUEUE_NAME={queue_name} HUEY_RUN_ID={run_id} PYTHONPATH={BENCHMARKER_ROOT} python -m src.orchestrator.worker --workers {workers}")
        click.echo(f"\nMonitor progress:")
        click.echo(f"  PYTHONPATH={BENCHMARKER_ROOT} python -m src.orchestrator {config_path} --monitor --run-id {run_id}")

        return 0


def enqueue_question(question_data: Dict, run_id: str, config_path: str,
                     start_session_index: int = 0):
    """Enqueue a question for processing (by ID only)."""
    question_id = question_data['question_id']

    # Generate worker ID (could be more sophisticated)
    import random
    worker_id = f"worker-{random.randint(1000, 9999)}"

    # Enqueue the task with IDs only (task loads from DB)
    global _TASKS_MOD
    if _TASKS_MOD is None:
        # Fallback: import with current environment
        from src.orchestrator import tasks as _tasks  # type: ignore
        _TASKS_MOD = _tasks
    _TASKS_MOD.process_question(
        run_id=run_id,
        question_id=question_id,
        worker_id=worker_id
    )

    logger.debug(f"Enqueued question {question_id}")


def monitor_progress(tracker: ProgressTracker, run_id: str):
    """Monitor progress of a benchmark run."""
    def _bar(done: int, tot: int, width: int = 30) -> str:
        if tot and tot > 0:
            filled = int(width * (done / tot))
            return '█' * filled + '░' * (width - filled)
        return '░' * width

    def _build_rich_view(stats: Dict, details: List[Dict]) -> Panel:
        total = stats.get('total_questions', 0) or 0
        completed = stats.get('completed', 0) or 0
        in_progress = stats.get('in_progress', 0) or 0
        failed = stats.get('failed', 0) or 0
        pending = stats.get('pending', 0) or 0
        ingested = stats.get('ingested', 0) or 0
        qa_done = stats.get('qa_done', 0) or 0
        sessions_done = stats.get('total_sessions_completed', 0) or 0
        sessions_total = stats.get('total_sessions_expected', 0) or 0
        msgs_done = stats.get('total_messages_ingested', 0) or 0
        msgs_total = stats.get('total_messages_expected', 0) or 0

        header = Table.grid(expand=True)
        header.add_row(f"[bold]LongMemEval Benchmark Monitor - Run: {run_id}[/bold]")
        header.add_row(datetime.now().strftime('%Y-%m-%d %H:%M:%S'))

        summary = Table.grid(padding=(0,1))
        summary.add_row(f"Questions: [bold]{completed}/{total}[/bold]", f"In Progress: {in_progress}", f"Pending: {pending}", f"Failed: {failed}")
        if sessions_total > 0:
            pct = sessions_done / sessions_total * 100
            summary.add_row(f"Sessions: {sessions_done}/{sessions_total} ({pct:.1f}%)")
        if msgs_total > 0:
            mpct = msgs_done / msgs_total * 100
            summary.add_row(f"Messages: {msgs_done}/{msgs_total} ({mpct:.1f}%)")
        if total > 0:
            summary.add_row(f"Ingested: [{_bar(ingested, total, 30)}] {ingested/total*100:.1f}%")
            summary.add_row(f"QA done : [{_bar(qa_done, total, 30)}] {qa_done/total*100:.1f}%")
            pct_complete = completed / total * 100 if total else 0
            summary.add_row(f"Overall : [{_bar(completed, total, 40)}] {pct_complete:.1f}%")

        details_table = Table(show_header=True, header_style="bold", expand=True)
        details_table.add_column("Question")
        details_table.add_column("Phase")
        details_table.add_column("Sessions")
        details_table.add_column("Messages")
        # Keep row positions stable across refreshes
        details_sorted = sorted(details or [], key=lambda d: (d.get('question_id') or ''))
        for d in details_sorted:
            qid = d['question_id']
            s_done, s_total = d['s_done'], d['s_total']
            m_done, m_total = d['m_done'], d['m_total']
            istatus, qstatus = (d.get('ingestion_status') or ''), (d.get('qa_status') or '')
            phase = 'INGEST' if istatus == 'in_progress' else ('QA' if qstatus == 'in_progress' else istatus.upper() or 'PENDING')
            s_bar = _bar(s_done, s_total, 20)
            m_bar = _bar(m_done, m_total, 20)
            if istatus != 'completed':
                details_table.add_row(qid, phase, f"{s_done}/{s_total} [{s_bar}]", f"{m_done}/{m_total} [{m_bar}]")
            else:
                label = 'running…' if qstatus == 'in_progress' else ('done' if qstatus == 'completed' else 'waiting')
                details_table.add_row(qid, f"QA {label}", f"{s_done}/{s_total}", f"{m_done}/{m_total}")

        grid = Table.grid(expand=True)
        grid.add_row(header)
        grid.add_row(summary)
        # Add failed details if present
        failed_rows = tracker.get_failed_details(run_id, limit=10)
        if failed_rows:
            failed_table = Table(show_header=True, header_style="bold red", expand=True)
            failed_table.add_column("Failed Question")
            failed_table.add_column("Step")
            failed_table.add_column("Error (truncated)")
            for f in failed_rows:
                step = 'QA' if (f.get('qa_status') == 'failed') else ('INGESTION' if f.get('ingestion_status') == 'failed' else '-')
                msg = (f.get('error_message') or '')
                msg = msg if len(msg) <= 120 else msg[:117] + '...'
                failed_table.add_row(f.get('question_id') or '-', step, msg)
            grid.add_row(failed_table)
        grid.add_row(details_table)
        return Panel(Align.left(grid))

    # Use Rich live dashboard if available and output is a TTY
    if _RICH_AVAILABLE and sys.stdout.isatty():
        try:
            with Live(refresh_per_second=4, console=_console) as live:
                while True:
                    stats = tracker.get_run_stats(run_id)
                    details = tracker.get_in_progress_details(run_id, limit=10)
                    live.update(_build_rich_view(stats, details))
                    total = stats.get('total_questions', 0) or 0
                    completed = stats.get('completed', 0) or 0
                    if total > 0 and completed == total:
                        break
                    time.sleep(1)
            _console.print("\n✅ Benchmark complete!")
        except KeyboardInterrupt:
            _console.print("\nMonitoring stopped.")
        return

    # Plain-text fallback
    click.clear()
    try:
        while True:
            stats = tracker.get_run_stats(run_id)
            click.clear()
            click.echo("="*60)
            click.echo(f"LongMemEval Benchmark Monitor - Run: {run_id}")
            click.echo("="*60)
            click.echo(f"Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
            click.echo()
            total = stats.get('total_questions', 0) or 0
            completed = stats.get('completed', 0) or 0
            in_progress = stats.get('in_progress', 0) or 0
            failed = stats.get('failed', 0) or 0
            pending = stats.get('pending', 0) or 0
            ingested = stats.get('ingested', 0) or 0
            qa_done = stats.get('qa_done', 0) or 0
            click.echo(f"Questions: {completed}/{total} completed")
            click.echo(f"  - In Progress: {in_progress}")
            click.echo(f"  - Pending: {pending}")
            click.echo(f"  - Failed: {failed}")
            click.echo()
            sessions_done = stats.get('total_sessions_completed', 0) or 0
            sessions_total = stats.get('total_sessions_expected', 0) or 0
            msgs_done = stats.get('total_messages_ingested', 0) or 0
            msgs_total = stats.get('total_messages_expected', 0) or 0
            if sessions_total > 0:
                pct = sessions_done / sessions_total * 100
                click.echo(f"Sessions: {sessions_done}/{sessions_total} ({pct:.1f}%)")
            if msgs_total > 0:
                mpct = msgs_done / msgs_total * 100
                click.echo(f"Messages: {msgs_done}/{msgs_total} ({mpct:.1f}%)")
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
            if total > 0:
                pct_complete = completed / total * 100
                bar_width = 40
                filled = int(bar_width * completed / total)
                bar = '█' * filled + '░' * (bar_width - filled)
                click.echo(f"Overall  : [{bar}] {pct_complete:.1f}%")
            details = tracker.get_in_progress_details(run_id, limit=10)
            details_sorted = sorted(details or [], key=lambda d: (d.get('question_id') or ''))
            if details_sorted:
                click.echo("In-progress details:")
                for d in details_sorted:
                    qid = d['question_id']
                    s_done, s_total = d['s_done'], d['s_total']
                    m_done, m_total = d['m_done'], d['m_total']
                    istatus, qstatus = (d.get('ingestion_status') or ''), (d.get('qa_status') or '')
                    phase = 'INGEST' if istatus == 'in_progress' else ('QA' if qstatus == 'in_progress' else istatus.upper() or 'PENDING')
                    s_bar = _bar(s_done, s_total)
                    m_bar = _bar(m_done, m_total)
                    if istatus != 'completed':
                        click.echo(f"  {qid} [{phase}] Sessions {s_done}/{s_total} [{s_bar}]  Messages {m_done}/{m_total} [{m_bar}]  worker {d.get('worker_id') or '-'}")
                    else:
                        if qstatus == 'in_progress':
                            click.echo(f"  {qid} [QA] running…  worker {d.get('worker_id') or '-'}")
                        elif qstatus == 'completed':
                            click.echo(f"  {qid} [QA] done")
                        else:
                            click.echo(f"  {qid} [QA] waiting for ingestion")
            # Failed details
            failed_rows = tracker.get_failed_details(run_id, limit=10)
            if failed_rows:
                click.echo("\nFailed:")
                for f in failed_rows:
                    step = 'QA' if (f.get('qa_status') == 'failed') else ('INGESTION' if f.get('ingestion_status') == 'failed' else '-')
                    msg = (f.get('error_message') or '')
                    msg = msg if len(msg) <= 120 else msg[:117] + '...'
                    click.echo(f"  {f.get('question_id')}: [{step}] {msg}")

            stuck = tracker.get_stuck_questions(run_id)
            if stuck:
                click.echo(f"\n⚠️  {len(stuck)} task(s) possibly stuck (>30m): {', '.join(stuck[:5])}{'…' if len(stuck) > 5 else ''}")
            if completed == total:
                click.echo("\n✅ Benchmark complete!")
                break
            time.sleep(5)
    except KeyboardInterrupt:
        click.echo("\n\nMonitoring stopped.")


if __name__ == '__main__':
    sys.exit(main())

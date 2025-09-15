#!/usr/bin/env python3
"""
Progress tracking for LongMemEval benchmarker using SQLite.
Tracks question-level and session-level progress for resumability.
"""

import sqlite3
import json
import logging
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from contextlib import contextmanager

logger = logging.getLogger('orchestrator.progress')


class ProgressTracker:
    """Track benchmark progress in SQLite database."""

    def __init__(self, db_path: str = "progress.db"):
        # Anchor default DB under benchmarker root data/ to avoid CWD mismatches
        if not db_path or db_path == "progress.db":
            data_dir = Path(__file__).resolve().parents[2] / 'data'
            data_dir.mkdir(parents=True, exist_ok=True)
            self.db_path = str(data_dir / "progress.db")
        else:
            self.db_path = db_path
        self._init_database()

    @contextmanager
    def _get_connection(self):
        """Context manager for database connections."""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA journal_mode=WAL")  # Enable WAL mode for concurrency
        try:
            yield conn
            conn.commit()
        except Exception:
            conn.rollback()
            raise
        finally:
            conn.close()

    def _init_database(self):
        """Initialize the progress tracking database."""
        with self._get_connection() as conn:
            # Runs table stores high-level run metadata and paths
            conn.execute("""
                CREATE TABLE IF NOT EXISTS runs (
                    run_id TEXT PRIMARY KEY,
                    dataset_path TEXT,
                    config_path TEXT,
                    created_at TIMESTAMP
                )
            """)

            conn.execute("""
                CREATE TABLE IF NOT EXISTS question_progress (
                    run_id TEXT NOT NULL,
                    question_id TEXT NOT NULL,
                    question_type TEXT,
                    question_json TEXT,
                    vault_id TEXT,
                    memory_id TEXT,
                    memory_title TEXT,  -- For debugging: {run_id}_{question_id}
                    total_sessions INTEGER,
                    completed_sessions INTEGER DEFAULT 0,
                    total_messages INTEGER DEFAULT 0,
                    ingested_messages INTEGER DEFAULT 0,
                    status TEXT DEFAULT 'pending',  -- pending/in_progress/completed/failed
                    ingestion_status TEXT DEFAULT 'pending',
                    qa_status TEXT DEFAULT 'pending',
                    ingestion_started_at TIMESTAMP,
                    ingestion_completed_at TIMESTAMP,
                    qa_started_at TIMESTAMP,
                    qa_completed_at TIMESTAMP,
                    last_progress_at TIMESTAMP,
                    worker_id TEXT,
                    error_message TEXT,
                    retry_count INTEGER DEFAULT 0,
                    PRIMARY KEY (run_id, question_id)
                )
            """)

            conn.execute("""
                CREATE INDEX IF NOT EXISTS idx_status
                ON question_progress(run_id, status)
            """)

            conn.execute("""
                CREATE INDEX IF NOT EXISTS idx_ingestion_status
                ON question_progress(run_id, ingestion_status)
            """)

            conn.execute("""
                CREATE INDEX IF NOT EXISTS idx_qa_status
                ON question_progress(run_id, qa_status)
            """)

            # Backfill schema changes: ensure new columns exist for older DBs
            cols = conn.execute("PRAGMA table_info(question_progress)").fetchall()
            col_names = {row[1] for row in cols}
            if 'question_json' not in col_names:
                conn.execute("ALTER TABLE question_progress ADD COLUMN question_json TEXT")
            if 'total_messages' not in col_names:
                conn.execute("ALTER TABLE question_progress ADD COLUMN total_messages INTEGER DEFAULT 0")
            if 'ingested_messages' not in col_names:
                conn.execute("ALTER TABLE question_progress ADD COLUMN ingested_messages INTEGER DEFAULT 0")
            if 'last_progress_at' not in col_names:
                conn.execute("ALTER TABLE question_progress ADD COLUMN last_progress_at TIMESTAMP")

    def init_run(self, run_id: str, questions: List[Dict], dataset_path: Optional[str] = None, config_path: Optional[str] = None) -> None:
        """Initialize a new benchmark run with all questions and persist question JSON."""
        with self._get_connection() as conn:
            # Optionally record run metadata
            if dataset_path is not None or config_path is not None:
                conn.execute("""
                    INSERT OR IGNORE INTO runs (run_id, dataset_path, config_path, created_at)
                    VALUES (?, ?, ?, ?)
                """, (run_id, dataset_path, config_path, datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')))

            for question in questions:
                # Compute sessions and total_messages from either canonical 'sessions' or 'haystack_sessions'
                sessions = question.get('sessions')
                if not isinstance(sessions, list):
                    sessions = question.get('haystack_sessions', [])
                try:
                    total_messages = 0
                    if isinstance(sessions, list):
                        for sess in sessions:
                            # haystack_sessions: list of messages; sessions: dict with 'messages'
                            if isinstance(sess, dict):
                                msgs = sess.get('messages') or []
                            else:
                                msgs = sess if isinstance(sess, list) else []
                            total_messages += sum(1 for m in msgs if isinstance(m, dict) and isinstance(m.get('content'), str) and (m.get('role') in ('user','assistant')) and m.get('content').strip())
                except Exception:
                    total_messages = 0

                conn.execute("""
                    INSERT OR IGNORE INTO question_progress
                    (run_id, question_id, question_type, question_json, total_sessions, memory_title, total_messages)
                    VALUES (?, ?, ?, ?, ?, ?, ?)
                """, (
                    run_id,
                    question['question_id'],
                    question.get('question_type', 'unknown'),
                    json.dumps(question),
                    len(sessions) if isinstance(sessions, list) else 0,
                    f"{run_id}_{question['question_id']}",
                    total_messages
                ))
        logger.info(f"Initialized run {run_id} with {len(questions)} questions")

    def get_run_config(self, run_id: str) -> Optional[Dict]:
        """Return dataset_path and config_path for a run, if recorded."""
        with self._get_connection() as conn:
            row = conn.execute(
                "SELECT dataset_path, config_path, created_at FROM runs WHERE run_id = ?",
                (run_id,)
            ).fetchone()
            return dict(row) if row else None

    def get_question_json(self, run_id: str, question_id: str) -> Optional[Dict]:
        """Return the stored question JSON for a given question in a run."""
        with self._get_connection() as conn:
            row = conn.execute(
                """
                SELECT question_json FROM question_progress
                WHERE run_id = ? AND question_id = ?
                """,
                (run_id, question_id)
            ).fetchone()
            if not row:
                return None
            try:
                return json.loads(row[0]) if row[0] else None
            except Exception:
                logger.warning("Failed to parse question_json for %s/%s", run_id, question_id)
                return None

    # Pull-model claim removed for push-model async orchestrator

    def update_vault_memory(self, run_id: str, question_id: str,
                           vault_id: str, memory_id: str) -> None:
        """Update vault and memory IDs after creation."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET vault_id = ?, memory_id = ?
                WHERE run_id = ? AND question_id = ?
            """, (vault_id, memory_id, run_id, question_id))

    def update_session_progress(self, run_id: str, question_id: str,
                               completed_sessions: int) -> None:
        """Update the number of completed sessions for a question."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET completed_sessions = ?
                WHERE run_id = ? AND question_id = ?
            """, (completed_sessions, run_id, question_id))

    def mark_ingestion_complete(self, run_id: str, question_id: str) -> None:
        """Mark ingestion as complete for a question."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET ingestion_status = 'completed',
                    ingestion_completed_at = ?,
                    status = CASE WHEN qa_status = 'completed' THEN 'completed' ELSE status END
                WHERE run_id = ? AND question_id = ?
            """, (datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), run_id, question_id))

    def mark_qa_complete(self, run_id: str, question_id: str) -> None:
        """Mark QA as complete for a question."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET qa_status = 'completed',
                    qa_completed_at = ?,
                    status = 'completed'
                WHERE run_id = ? AND question_id = ?
            """, (datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), run_id, question_id))

    def mark_ingestion_in_progress(self, run_id: str, question_id: str, worker_id: Optional[str]) -> None:
        """Mark ingestion as in progress and set worker/timestamp."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET status = 'in_progress',
                    ingestion_status = 'in_progress',
                    ingestion_started_at = ?,
                    worker_id = ?
                WHERE run_id = ? AND question_id = ?
            """, (datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), worker_id, run_id, question_id))

    def mark_qa_in_progress(self, run_id: str, question_id: str) -> None:
        """Mark QA as in progress and set timestamp."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET qa_status = 'in_progress',
                    qa_started_at = ?
                WHERE run_id = ? AND question_id = ?
            """, (datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), run_id, question_id))

    def mark_failed(self, run_id: str, question_id: str, error_message: str) -> None:
        """Mark a question as failed with error message."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET status = 'failed',
                    error_message = ?,
                    retry_count = retry_count + 1
                WHERE run_id = ? AND question_id = ?
            """, (error_message, run_id, question_id))

    # --- New helpers -------------------------------------------------

    def update_latest_progress_time(self, run_id: str, question_id: str) -> None:
        """Update last_progress_at to UTC now for a question."""
        with self._get_connection() as conn:
            conn.execute(
                """
                UPDATE question_progress
                SET last_progress_at = ?
                WHERE run_id = ? AND question_id = ?
                """,
                (datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), run_id, question_id),
            )

    def increment_ingested_messages(self, run_id: str, question_id: str, increment: int) -> None:
        """Increment ingested_messages and bump last_progress_at."""
        if increment <= 0:
            return
        with self._get_connection() as conn:
            conn.execute(
                """
                UPDATE question_progress
                SET ingested_messages = COALESCE(ingested_messages, 0) + ?,
                    last_progress_at = ?
                WHERE run_id = ? AND question_id = ?
                """,
                (increment, datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S'), run_id, question_id),
            )

    def mark_failed_ingestion(self, run_id: str, question_id: str, message: str) -> None:
        """Failure during ingestion step."""
        with self._get_connection() as conn:
            conn.execute(
                """
                UPDATE question_progress
                SET status = 'failed',
                    ingestion_status = 'failed',
                    error_message = ?,
                    retry_count = retry_count + 1
                WHERE run_id = ? AND question_id = ?
                """,
                (message, run_id, question_id),
            )

    def mark_failed_qa(self, run_id: str, question_id: str, message: str) -> None:
        """Failure during QA step."""
        with self._get_connection() as conn:
            conn.execute(
                """
                UPDATE question_progress
                SET status = 'failed',
                    qa_status = 'failed',
                    error_message = ?,
                    retry_count = retry_count + 1
                WHERE run_id = ? AND question_id = ?
                """,
                (message, run_id, question_id),
            )

    # ---------------------------------------------------------------

    def reset_for_restart(self, run_id: str, question_id: str) -> None:
        """Reset a question to restart ingestion from session 0.

        Preserves vault_id but clears memory_id and resets statuses/counters.
        """
        with self._get_connection() as conn:
            conn.execute(
                """
                UPDATE question_progress
                SET memory_id = NULL,
                    completed_sessions = 0,
                    ingested_messages = 0,
                    status = 'pending',
                    ingestion_status = 'pending',
                    qa_status = 'pending',
                    ingestion_started_at = NULL,
                    ingestion_completed_at = NULL,
                    qa_started_at = NULL,
                    qa_completed_at = NULL,
                    last_progress_at = NULL,
                    worker_id = NULL,
                    error_message = NULL
                WHERE run_id = ? AND question_id = ?
                """,
                (run_id, question_id),
            )

    def get_pending_questions(self, run_id: str) -> List[Dict]:
        """Get all pending questions for a run."""
        with self._get_connection() as conn:
            result = conn.execute("""
                SELECT * FROM question_progress
                WHERE run_id = ?
                  AND (status = 'pending' OR
                       (status = 'failed' AND retry_count < 3))
                ORDER BY question_id
            """, (run_id,))
            return [dict(row) for row in result.fetchall()]

    def get_resumable_questions(self, run_id: str) -> List[Dict]:
        """Get questions that need resuming (partially completed)."""
        with self._get_connection() as conn:
            result = conn.execute("""
                SELECT * FROM question_progress
                WHERE run_id = ?
                  AND status = 'in_progress'
                  AND completed_sessions > 0
                  AND completed_sessions < total_sessions
                ORDER BY question_id
            """, (run_id,))
            return [dict(row) for row in result.fetchall()]

    def get_inprogress_unstarted(self, run_id: str) -> List[Dict]:
        """Get questions marked in_progress but with zero sessions completed.

        These likely got stuck before the first session finished and should be
        safe to hard-reset on resume.
        """
        with self._get_connection() as conn:
            result = conn.execute(
                """
                SELECT * FROM question_progress
                WHERE run_id = ?
                  AND status = 'in_progress'
                  AND COALESCE(completed_sessions, 0) = 0
                ORDER BY question_id
                """,
                (run_id,),
            )
            return [dict(row) for row in result.fetchall()]

    def get_qa_inprogress_after_ingest(self, run_id: str) -> List[Dict]:
        """Get questions where ingestion completed but QA is stuck in_progress.

        On resume, we can either requeue QA or hard reset to re-ingest. For
        simplicity and determinism, callers may choose to hard-reset.
        """
        with self._get_connection() as conn:
            result = conn.execute(
                """
                SELECT * FROM question_progress
                WHERE run_id = ?
                  AND ingestion_status = 'completed'
                  AND qa_status = 'in_progress'
                ORDER BY question_id
                """,
                (run_id,),
            )
            return [dict(row) for row in result.fetchall()]

    def get_run_stats(self, run_id: str) -> Dict:
        """Get overall statistics for a run."""
        with self._get_connection() as conn:
            result = conn.execute("""
                SELECT
                    COUNT(*) as total_questions,
                    COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed,
                    COALESCE(SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END), 0) as in_progress,
                    COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
                    COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
                    COALESCE(SUM(completed_sessions), 0) as total_sessions_completed,
                    COALESCE(SUM(total_sessions), 0) as total_sessions_expected,
                    COALESCE(SUM(CASE WHEN ingestion_status = 'completed' THEN 1 ELSE 0 END), 0) as ingested,
                    COALESCE(SUM(CASE WHEN qa_status = 'completed' THEN 1 ELSE 0 END), 0) as qa_done,
                    COALESCE(SUM(COALESCE(ingested_messages, 0)), 0) as total_messages_ingested,
                    COALESCE(SUM(COALESCE(total_messages, 0)), 0) as total_messages_expected
                FROM question_progress
                WHERE run_id = ?
            """, (run_id,))
            return dict(result.fetchone())

    def get_question_details(self, run_id: str, question_id: str) -> Optional[Dict]:
        """Get detailed progress for a specific question."""
        with self._get_connection() as conn:
            result = conn.execute("""
                SELECT * FROM question_progress
                WHERE run_id = ? AND question_id = ?
            """, (run_id, question_id))
            row = result.fetchone()
            return dict(row) if row else None

    def get_stuck_questions(self, run_id: str, minutes: int = 30) -> List[str]:
        """Return question_ids that have been in_progress without update for > minutes."""
        with self._get_connection() as conn:
            result = conn.execute(
                """
                SELECT question_id FROM question_progress
                WHERE run_id = ? AND status = 'in_progress'
                  AND COALESCE(last_progress_at, ingestion_started_at) IS NOT NULL
                  AND (julianday('now') - julianday(COALESCE(last_progress_at, ingestion_started_at))) * 24 * 60 > ?
                LIMIT 20
                """,
                (run_id, minutes),
            )
            return [row[0] for row in result.fetchall()]

    def get_in_progress_details(self, run_id: str, limit: int = 10) -> List[Dict]:
        """Return compact details for in-progress questions ordered by latest progress."""
        with self._get_connection() as conn:
            result = conn.execute(
                """
                SELECT question_id, worker_id,
                       COALESCE(completed_sessions, 0) AS s_done,
                       COALESCE(total_sessions, 0)    AS s_total,
                       COALESCE(ingested_messages, 0) AS m_done,
                       COALESCE(total_messages, 0)    AS m_total,
                       ingestion_status, qa_status,
                       COALESCE(last_progress_at, ingestion_started_at) AS last_update
                FROM question_progress
                WHERE run_id = ? AND status = 'in_progress'
                ORDER BY COALESCE(last_progress_at, ingestion_started_at) DESC
                LIMIT ?
                """,
                (run_id, limit),
            )
            return [dict(row) for row in result.fetchall()]

    def get_failed_details(self, run_id: str, limit: int = 10) -> List[Dict]:
        """Return recent failed questions and their error messages."""
        with self._get_connection() as conn:
            result = conn.execute(
                """
                SELECT question_id,
                       COALESCE(error_message, '') AS error_message,
                       ingestion_status, qa_status,
                       COALESCE(last_progress_at, ingestion_started_at) AS last_update
                FROM question_progress
                WHERE run_id = ? AND status = 'failed'
                ORDER BY COALESCE(last_progress_at, ingestion_started_at) DESC
                LIMIT ?
                """,
                (run_id, limit),
            )
            return [dict(row) for row in result.fetchall()]

    def get_questions_for_qa(self, run_id: str, include_completed: bool = False) -> List[Dict]:
        """Return questions eligible for QA (ingestion completed).

        Args:
            run_id: Target run identifier
            include_completed: If True, include questions whose QA is already completed

        Returns:
            List of rows with at least question_id, ingestion_status, qa_status
        """
        with self._get_connection() as conn:
            if include_completed:
                result = conn.execute(
                    """
                    SELECT question_id, ingestion_status, qa_status
                    FROM question_progress
                    WHERE run_id = ? AND ingestion_status = 'completed'
                    ORDER BY question_id
                    """,
                    (run_id,),
                )
            else:
                result = conn.execute(
                    """
                    SELECT question_id, ingestion_status, qa_status
                    FROM question_progress
                    WHERE run_id = ? AND ingestion_status = 'completed' AND qa_status != 'completed'
                    ORDER BY question_id
                    """,
                    (run_id,),
                )
            return [dict(row) for row in result.fetchall()]

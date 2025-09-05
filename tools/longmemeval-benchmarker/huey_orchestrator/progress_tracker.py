#!/usr/bin/env python3
"""
Progress tracking for LongMemEval benchmarker using SQLite.
Tracks question-level and session-level progress for resumability.
"""

import sqlite3
import logging
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from contextlib import contextmanager

logger = logging.getLogger('orchestrator.progress')


class ProgressTracker:
    """Track benchmark progress in SQLite database."""
    
    def __init__(self, db_path: str = "progress.db"):
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
            conn.execute("""
                CREATE TABLE IF NOT EXISTS question_progress (
                    run_id TEXT NOT NULL,
                    question_id TEXT NOT NULL,
                    question_type TEXT,
                    vault_id TEXT,
                    memory_id TEXT,
                    memory_title TEXT,  -- For debugging: {run_id}_{question_id}
                    total_sessions INTEGER,
                    completed_sessions INTEGER DEFAULT 0,
                    status TEXT DEFAULT 'pending',  -- pending/in_progress/completed/failed
                    ingestion_status TEXT DEFAULT 'pending',
                    qa_status TEXT DEFAULT 'pending',
                    ingestion_started_at TIMESTAMP,
                    ingestion_completed_at TIMESTAMP,
                    qa_started_at TIMESTAMP,
                    qa_completed_at TIMESTAMP,
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
    
    def init_run(self, run_id: str, questions: List[Dict]) -> None:
        """Initialize a new benchmark run with all questions."""
        with self._get_connection() as conn:
            for question in questions:
                conn.execute("""
                    INSERT OR IGNORE INTO question_progress 
                    (run_id, question_id, question_type, total_sessions, memory_title)
                    VALUES (?, ?, ?, ?, ?)
                """, (
                    run_id,
                    question['question_id'],
                    question.get('question_type', 'unknown'),
                    len(question.get('haystack_sessions', [])),
                    f"{run_id}_{question['question_id']}"
                ))
        logger.info(f"Initialized run {run_id} with {len(questions)} questions")
    
    def claim_question(self, run_id: str, worker_id: str) -> Optional[Dict]:
        """
        Atomically claim a pending question for processing.
        Returns question details if claimed, None if no questions available.
        """
        with self._get_connection() as conn:
            # Use a transaction to atomically claim a question
            result = conn.execute("""
                UPDATE question_progress
                SET status = 'in_progress',
                    worker_id = ?,
                    ingestion_started_at = ?
                WHERE run_id = ?
                  AND status = 'pending'
                  AND rowid = (
                    SELECT rowid FROM question_progress
                    WHERE run_id = ? AND status = 'pending'
                    LIMIT 1
                  )
                RETURNING question_id, question_type, total_sessions, memory_title
            """, (worker_id, datetime.now().isoformat(), run_id, run_id))
            
            row = result.fetchone()
            if row:
                return dict(row)
            return None
    
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
                    ingestion_completed_at = ?
                WHERE run_id = ? AND question_id = ?
            """, (datetime.now().isoformat(), run_id, question_id))
    
    def mark_qa_complete(self, run_id: str, question_id: str) -> None:
        """Mark QA as complete for a question."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE question_progress
                SET qa_status = 'completed',
                    qa_completed_at = ?,
                    status = 'completed'
                WHERE run_id = ? AND question_id = ?
            """, (datetime.now().isoformat(), run_id, question_id))
    
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
    
    def get_run_stats(self, run_id: str) -> Dict:
        """Get overall statistics for a run."""
        with self._get_connection() as conn:
            result = conn.execute("""
                SELECT 
                    COUNT(*) as total_questions,
                    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
                    SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress,
                    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
                    SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
                    SUM(completed_sessions) as total_sessions_completed,
                    SUM(total_sessions) as total_sessions_expected
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
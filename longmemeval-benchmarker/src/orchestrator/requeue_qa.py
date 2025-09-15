#!/usr/bin/env python3
"""
Re-enqueue QA tasks for questions that have completed ingestion but failed QA.
"""

import sys
import sqlite3
from pathlib import Path

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent))

from tasks import run_qa

def main():
    if len(sys.argv) < 3:
        print("Usage: python requeue_qa.py <run_id> <config_path>")
        sys.exit(1)

    run_id = sys.argv[1]
    config_path = sys.argv[2]

    # Connect to progress database
    db_path = Path(__file__).resolve().parents[2] / 'data' / 'progress.db'  # Go up 2 levels now
    conn = sqlite3.connect(str(db_path))
    cursor = conn.cursor()

    # Find questions that need QA re-run
    cursor.execute("""
        SELECT question_id, vault_id, memory_id
        FROM question_progress
        WHERE run_id = ?
        AND ingestion_status = 'completed'
        AND qa_status = 'pending'
        AND vault_id IS NOT NULL
        AND memory_id IS NOT NULL
    """, (run_id,))

    questions = cursor.fetchall()
    conn.close()

    if not questions:
        print(f"No questions found needing QA for run {run_id}")
        return

    # Enqueue QA tasks
    for question_id, vault_id, memory_id in questions:
        print(f"Enqueueing QA task for question {question_id}")
        print(f"  vault_id: {vault_id}")
        print(f"  memory_id: {memory_id}")

        # This will add the task to Huey queue
        task = run_qa(run_id, question_id, vault_id, memory_id, config_path)
        print(f"  Task enqueued: {task}")

    print(f"\n{len(questions)} QA tasks enqueued successfully!")
    print("Workers should pick them up shortly.")

if __name__ == "__main__":
    main()

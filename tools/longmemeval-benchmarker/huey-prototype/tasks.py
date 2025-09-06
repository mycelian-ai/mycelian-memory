"""
Huey task definitions for processing LongMemEval dataset.
Each task processes one question and counts sessions/messages.
"""

import json
import time
import sqlite3
from datetime import datetime
from huey import SqliteHuey
from pathlib import Path

# Initialize Huey with SQLite backend anchored to benchmarker data/
_DATA_DIR = Path(__file__).resolve().parents[1] / 'data'
_DATA_DIR.mkdir(parents=True, exist_ok=True)
huey = SqliteHuey(str(_DATA_DIR / 'huey_tasks.db'), immediate=False)

# Progress tracking database (separate from Huey's queue)
PROGRESS_DB = str(_DATA_DIR / 'progress.db')


def init_progress_db():
    """Initialize the progress tracking database."""
    with sqlite3.connect(PROGRESS_DB) as conn:
        conn.execute('''
            CREATE TABLE IF NOT EXISTS question_progress (
                question_id TEXT PRIMARY KEY,
                question_type TEXT,
                total_sessions INTEGER,
                sessions_processed INTEGER DEFAULT 0,
                total_messages INTEGER DEFAULT 0,
                status TEXT DEFAULT 'pending',
                started_at TIMESTAMP,
                completed_at TIMESTAMP,
                worker_pid INTEGER,
                error_message TEXT
            )
        ''')


@huey.task(retries=2, retry_delay=5)
def process_question(question_data):
    """
    Process a single question to count sessions and messages.
    Demonstrates Huey's task processing with progress tracking.
    """
    import os
    
    question_id = question_data['question_id']
    question_type = question_data.get('question_type', 'unknown')
    
    print(f"[Worker {os.getpid()}] Processing question {question_id}")
    
    # Update progress: mark as in-progress
    with sqlite3.connect(PROGRESS_DB) as conn:
        conn.execute('''
            UPDATE question_progress 
            SET status = 'processing',
                started_at = ?,
                worker_pid = ?
            WHERE question_id = ?
        ''', (datetime.now(), os.getpid(), question_id))
    
    try:
        # Count sessions
        sessions = question_data.get('haystack_sessions', [])
        total_sessions = len(sessions)
        
        # Count messages per session
        session_message_counts = []
        total_messages = 0
        
        for session_idx, session in enumerate(sessions):
            # Simulate processing time (would be actual work in real scenario)
            time.sleep(0.01)  # Small delay to simulate work
            
            # Count messages in this session
            if isinstance(session, list):
                message_count = len(session)
            else:
                # Handle if session structure is different
                message_count = 0
            
            session_message_counts.append({
                'session_index': session_idx,
                'message_count': message_count
            })
            total_messages += message_count
            
            # Update progress periodically (every 10 sessions)
            if session_idx % 10 == 0:
                with sqlite3.connect(PROGRESS_DB) as conn:
                    conn.execute('''
                        UPDATE question_progress 
                        SET sessions_processed = ?
                        WHERE question_id = ?
                    ''', (session_idx + 1, question_id))
                print(f"  [{question_id}] Processed {session_idx + 1}/{total_sessions} sessions")
        
        # Prepare result
        result = {
            'question_id': question_id,
            'question_type': question_type,
            'total_sessions': total_sessions,
            'total_messages': total_messages,
            'average_messages_per_session': total_messages / total_sessions if total_sessions > 0 else 0,
            'session_details': session_message_counts[:5]  # First 5 sessions as sample
        }
        
        # Mark as completed
        with sqlite3.connect(PROGRESS_DB) as conn:
            conn.execute('''
                UPDATE question_progress 
                SET status = 'completed',
                    sessions_processed = ?,
                    total_messages = ?,
                    completed_at = ?
                WHERE question_id = ?
            ''', (total_sessions, total_messages, datetime.now(), question_id))
        
        print(f"[Worker {os.getpid()}] Completed {question_id}: {total_sessions} sessions, {total_messages} messages")
        
        # Write result to JSONL file
        with open('results.jsonl', 'a') as f:
            json.dump(result, f)
            f.write('\n')
        
        return result
        
    except Exception as e:
        # Mark as failed
        with sqlite3.connect(PROGRESS_DB) as conn:
            conn.execute('''
                UPDATE question_progress 
                SET status = 'failed',
                    error_message = ?
                WHERE question_id = ?
            ''', (str(e), question_id))
        raise  # Re-raise for Huey to handle retries


@huey.task()
def process_question_with_interruption(question_data):
    """
    Simulates processing that can be interrupted and resumed.
    Shows how to handle session-level resume with Huey.
    """
    import os
    import random
    
    question_id = question_data['question_id']
    
    # Check if we need to resume
    with sqlite3.connect(PROGRESS_DB) as conn:
        cursor = conn.execute('''
            SELECT sessions_processed, total_sessions 
            FROM question_progress 
            WHERE question_id = ?
        ''', (question_id,))
        row = cursor.fetchone()
        
        if row:
            start_session, total_sessions = row
            if start_session >= total_sessions:
                print(f"[{question_id}] Already completed")
                return
        else:
            start_session = 0
            total_sessions = len(question_data.get('haystack_sessions', []))
    
    print(f"[Worker {os.getpid()}] Resuming {question_id} from session {start_session}")
    
    sessions = question_data.get('haystack_sessions', [])
    
    for session_idx in range(start_session, len(sessions)):
        # Simulate failure at random session (to demonstrate resume)
        if random.random() < 0.05:  # 5% chance of "failure"
            print(f"[{question_id}] Simulated interruption at session {session_idx}")
            raise Exception(f"Simulated failure at session {session_idx}")
        
        # Process session
        time.sleep(0.01)
        
        # Update progress
        with sqlite3.connect(PROGRESS_DB) as conn:
            conn.execute('''
                UPDATE question_progress 
                SET sessions_processed = ?
                WHERE question_id = ?
            ''', (session_idx + 1, question_id))
        
        if session_idx % 50 == 0:
            print(f"  [{question_id}] Progress: {session_idx}/{len(sessions)}")
    
    print(f"[{question_id}] Completed all sessions")


@huey.task()
def simple_count_task(dataset_path):
    """
    Simple task to count total questions and sessions in dataset.
    Shows basic Huey task usage.
    """
    import os
    
    print(f"[Worker {os.getpid()}] Counting dataset statistics for {dataset_path}")
    
    with open(dataset_path, 'r') as f:
        data = json.load(f)
    
    total_questions = len(data)
    total_sessions = sum(len(q.get('haystack_sessions', [])) for q in data)
    
    result = {
        'dataset': dataset_path,
        'total_questions': total_questions,
        'total_sessions': total_sessions,
        'average_sessions_per_question': total_sessions / total_questions if total_questions > 0 else 0
    }
    
    print(f"Dataset stats: {result}")
    return result
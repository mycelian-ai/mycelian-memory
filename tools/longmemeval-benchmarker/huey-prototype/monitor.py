#!/usr/bin/env python
"""
Monitor script to track progress of Huey tasks and our custom progress tracking.
Shows the combination of Huey's queue management with session-level progress.
"""

import sqlite3
import time
import sys
import os
from datetime import datetime
from pathlib import Path

_DATA_DIR = Path(__file__).resolve().parents[1] / 'data'
HUEY_DB_PATH = str(_DATA_DIR / 'huey_tasks.db')
PROGRESS_DB_PATH = str(_DATA_DIR / 'progress.db')

def clear_screen():
    """Clear the terminal screen."""
    os.system('clear' if os.name == 'posix' else 'cls')

def get_huey_stats():
    """Get statistics from Huey's task queue."""
    try:
        with sqlite3.connect(HUEY_DB_PATH) as conn:
            # Pending tasks
            cursor = conn.execute('''
                SELECT COUNT(*) FROM huey_task WHERE is_complete = 0
            ''')
            pending = cursor.fetchone()[0]
            
            # Completed tasks
            cursor = conn.execute('''
                SELECT COUNT(*) FROM huey_task WHERE is_complete = 1
            ''')
            completed = cursor.fetchone()[0]
            
            # Failed tasks (those with retries > 0)
            cursor = conn.execute('''
                SELECT COUNT(*) FROM huey_task WHERE retries > 0
            ''')
            retried = cursor.fetchone()[0]
            
            return {
                'pending': pending,
                'completed': completed,
                'retried': retried
            }
    except Exception as e:
        return {'error': str(e)}

def get_progress_stats():
    """Get statistics from our custom progress tracking."""
    try:
        with sqlite3.connect(PROGRESS_DB_PATH) as conn:
            # Overall stats
            cursor = conn.execute('''
                SELECT 
                    status,
                    COUNT(*) as count,
                    SUM(sessions_processed) as sessions_done,
                    SUM(total_sessions) as total_sessions
                FROM question_progress 
                GROUP BY status
            ''')
            
            stats = {}
            for row in cursor:
                status, count, sessions_done, total_sessions = row
                stats[status] = {
                    'count': count,
                    'sessions_done': sessions_done or 0,
                    'total_sessions': total_sessions or 0
                }
            
            # Currently processing
            cursor = conn.execute('''
                SELECT 
                    question_id,
                    worker_pid,
                    sessions_processed,
                    total_sessions,
                    started_at
                FROM question_progress 
                WHERE status = 'processing'
                ORDER BY started_at DESC
                LIMIT 5
            ''')
            
            active = []
            for row in cursor:
                question_id, worker_pid, sessions_done, total_sessions, started_at = row
                active.append({
                    'question_id': question_id,
                    'worker_pid': worker_pid,
                    'progress': f"{sessions_done}/{total_sessions}",
                    'started_at': started_at
                })
            
            return {'stats': stats, 'active': active}
    except Exception as e:
        return {'error': str(e)}

def get_recent_completions():
    """Get recently completed questions."""
    try:
        with sqlite3.connect(PROGRESS_DB_PATH) as conn:
            cursor = conn.execute('''
                SELECT 
                    question_id,
                    total_sessions,
                    total_messages,
                    completed_at
                FROM question_progress 
                WHERE status = 'completed'
                ORDER BY completed_at DESC
                LIMIT 5
            ''')
            
            completions = []
            for row in cursor:
                question_id, sessions, messages, completed_at = row
                completions.append({
                    'question_id': question_id,
                    'sessions': sessions,
                    'messages': messages,
                    'completed_at': completed_at
                })
            
            return completions
    except Exception as e:
        return []

def display_dashboard():
    """Display a dashboard of current progress."""
    clear_screen()
    
    print("=" * 60)
    print("  HUEY TASK QUEUE MONITOR".center(60))
    print("=" * 60)
    print(f"  Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)
    
    # Huey Queue Stats
    huey_stats = get_huey_stats()
    if 'error' not in huey_stats:
        print("\n📋 HUEY QUEUE STATUS:")
        print(f"  ⏳ Pending:   {huey_stats['pending']}")
        print(f"  ✅ Completed: {huey_stats['completed']}")
        print(f"  🔄 Retried:   {huey_stats['retried']}")
    
    # Progress Stats
    progress_data = get_progress_stats()
    if 'error' not in progress_data:
        stats = progress_data['stats']
        
        print("\n📊 QUESTION PROGRESS:")
        total_questions = sum(s['count'] for s in stats.values())
        print(f"  Total Questions: {total_questions}")
        
        for status, data in stats.items():
            if data['count'] > 0:
                if data['total_sessions'] > 0:
                    pct = (data['sessions_done'] / data['total_sessions']) * 100
                    print(f"  {status.upper()}: {data['count']} questions, {data['sessions_done']}/{data['total_sessions']} sessions ({pct:.1f}%)")
                else:
                    print(f"  {status.upper()}: {data['count']} questions")
        
        # Active processing
        if progress_data['active']:
            print("\n🔄 CURRENTLY PROCESSING:")
            for item in progress_data['active']:
                print(f"  [{item['worker_pid']}] {item['question_id']}: {item['progress']} sessions")
    
    # Recent completions
    completions = get_recent_completions()
    if completions:
        print("\n✅ RECENT COMPLETIONS:")
        for comp in completions[:3]:
            print(f"  {comp['question_id']}: {comp['sessions']} sessions, {comp['messages']} messages")
    
    # Results file size
    if os.path.exists('results.jsonl'):
        size = os.path.getsize('results.jsonl')
        with open('results.jsonl', 'r') as f:
            lines = sum(1 for _ in f)
        print(f"\n📄 RESULTS FILE: {lines} entries, {size:,} bytes")
    
    print("\n" + "-" * 60)
    print("  Press Ctrl+C to exit | Refreshing every 2 seconds")

def main():
    """Main monitoring loop."""
    try:
        while True:
            display_dashboard()
            time.sleep(2)
    except KeyboardInterrupt:
        print("\n\nMonitor stopped.")
        sys.exit(0)

if __name__ == "__main__":
    main()
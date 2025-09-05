#!/usr/bin/env python
"""
Script to enqueue LongMemEval questions for processing.
This demonstrates how to add tasks to Huey's queue.
"""

import json
import sqlite3
import sys
import os
from datetime import datetime
from tasks import huey, init_progress_db, process_question, simple_count_task

def load_dataset(dataset_path):
    """Load the LongMemEval dataset."""
    print(f"Loading dataset from {dataset_path}")
    with open(dataset_path, 'r') as f:
        data = json.load(f)
    print(f"Loaded {len(data)} questions")
    return data


def initialize_progress_tracking(dataset):
    """Initialize progress tracking for all questions."""
    init_progress_db()
    
    with sqlite3.connect('progress.db') as conn:
        for question in dataset:
            question_id = question['question_id']
            question_type = question.get('question_type', 'unknown')
            total_sessions = len(question.get('haystack_sessions', []))
            
            # Insert or ignore if already exists (for resume capability)
            conn.execute('''
                INSERT OR IGNORE INTO question_progress 
                (question_id, question_type, total_sessions, status)
                VALUES (?, ?, ?, 'pending')
            ''', (question_id, question_type, total_sessions))
        
        conn.commit()
    
    print("Initialized progress tracking")


def enqueue_all_questions(dataset, limit=None):
    """Enqueue all questions for processing."""
    count = 0
    
    for question in dataset[:limit] if limit else dataset:
        # Check if already completed
        with sqlite3.connect('progress.db') as conn:
            cursor = conn.execute('''
                SELECT status FROM question_progress 
                WHERE question_id = ?
            ''', (question['question_id'],))
            row = cursor.fetchone()
            
            if row and row[0] == 'completed':
                print(f"Skipping completed question: {question['question_id']}")
                continue
        
        # Enqueue the task
        process_question(question)
        count += 1
        print(f"Enqueued: {question['question_id']}")
    
    print(f"\nEnqueued {count} questions for processing")
    return count


def show_queue_status():
    """Show current queue status."""
    # Query Huey's task queue (if it exists)
    try:
        with sqlite3.connect('huey_tasks.db') as conn:
            # Check if table exists
            cursor = conn.execute('''
                SELECT name FROM sqlite_master 
                WHERE type='table' AND name='huey_task'
            ''')
            if not cursor.fetchone():
                print(f"\nQueue Status: Tasks enqueued (database will be created when worker starts)")
                return
            
            cursor = conn.execute('''
                SELECT COUNT(*) FROM huey_task WHERE is_complete = 0
            ''')
            pending = cursor.fetchone()[0]
            
            cursor = conn.execute('''
                SELECT COUNT(*) FROM huey_task WHERE is_complete = 1
            ''')
            completed = cursor.fetchone()[0]
        
        print(f"\nQueue Status:")
        print(f"  Pending tasks: {pending}")
        print(f"  Completed tasks: {completed}")
    except Exception as e:
        print(f"\nQueue Status: Tasks enqueued (waiting for worker to start)")
    
    # Show progress status
    with sqlite3.connect('progress.db') as conn:
        cursor = conn.execute('''
            SELECT 
                status, 
                COUNT(*) as count
            FROM question_progress 
            GROUP BY status
        ''')
        
        print(f"\nProgress Status:")
        for row in cursor:
            print(f"  {row[0]}: {row[1]}")


def main():
    if len(sys.argv) < 2:
        print("Usage: python enqueue_work.py <dataset_path> [limit]")
        print("Example: python enqueue_work.py ../longmemeval-datasets/longmemeval_s.json 5")
        sys.exit(1)
    
    dataset_path = sys.argv[1]
    limit = int(sys.argv[2]) if len(sys.argv) > 2 else None
    
    if not os.path.exists(dataset_path):
        print(f"Dataset not found: {dataset_path}")
        sys.exit(1)
    
    # First, enqueue a simple count task
    print("Enqueueing dataset statistics task...")
    simple_count_task(dataset_path)
    
    # Load dataset and enqueue questions
    dataset = load_dataset(dataset_path)
    initialize_progress_tracking(dataset)
    
    if limit:
        print(f"\nProcessing first {limit} questions only")
    
    enqueue_all_questions(dataset, limit)
    show_queue_status()
    
    print("\n" + "="*50)
    print("Tasks enqueued! Now run workers with:")
    print("  python run_worker.py")
    print("\nTo run multiple workers:")
    print("  python run_worker.py &  # Terminal 1")
    print("  python run_worker.py &  # Terminal 2")
    print("  python run_worker.py &  # Terminal 3")


if __name__ == "__main__":
    main()
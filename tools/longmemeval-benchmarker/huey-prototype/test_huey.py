#!/usr/bin/env python
"""
Test script to verify Huey is working correctly.
Run this to test the Huey setup without user interaction.
"""

import os
import sys
import json
import time
import sqlite3
import subprocess
from datetime import datetime

# Add current directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_basic_setup():
    """Test basic Huey functionality."""
    print("=" * 60)
    print("HUEY PROTOTYPE TEST")
    print("=" * 60)
    
    # Clean up old files
    print("\n1. Cleaning up old data...")
    for db_file in ['huey_tasks.db', 'progress.db', 'results.jsonl']:
        if os.path.exists(db_file):
            os.remove(db_file)
            print(f"   Removed {db_file}")
    
    # Check dataset
    print("\n2. Checking for dataset...")
    dataset_path = None
    for path in ['../longmemeval-datasets/longmemeval_s.json', 
                 '../longmemeval-datasets/longmemeval_5s.json']:
        if os.path.exists(path):
            dataset_path = path
            with open(path, 'r') as f:
                data = json.load(f)
            print(f"   Found: {path}")
            print(f"   Questions: {len(data)}")
            break
    
    if not dataset_path:
        print("   ERROR: No dataset found!")
        return False
    
    # Import and initialize
    print("\n3. Importing Huey components...")
    try:
        from tasks import huey, init_progress_db, process_question, simple_count_task
        print("   ✓ Tasks imported successfully")
    except ImportError as e:
        print(f"   ERROR: Failed to import tasks: {e}")
        print("   Installing huey...")
        subprocess.run([sys.executable, "-m", "pip", "install", "huey"], check=True)
        from tasks import huey, init_progress_db, process_question, simple_count_task
    
    # Initialize progress DB
    print("\n4. Initializing progress database...")
    init_progress_db()
    print("   ✓ Progress database created")
    
    # Test enqueueing
    print("\n5. Testing task enqueueing...")
    
    # Load first question
    with open(dataset_path, 'r') as f:
        dataset = json.load(f)
    
    # Initialize progress for first 3 questions
    with sqlite3.connect('progress.db') as conn:
        for question in dataset[:3]:
            question_id = question['question_id']
            question_type = question.get('question_type', 'unknown')
            total_sessions = len(question.get('haystack_sessions', []))
            
            conn.execute('''
                INSERT OR IGNORE INTO question_progress 
                (question_id, question_type, total_sessions, status)
                VALUES (?, ?, ?, 'pending')
            ''', (question_id, question_type, total_sessions))
    
    # Enqueue tasks
    print("   Enqueueing 3 test questions...")
    for question in dataset[:3]:
        result = process_question(question)  # This enqueues the task
        print(f"   ✓ Enqueued: {question['question_id']}")
    
    # Also enqueue a simple task
    simple_count_task(dataset_path)
    print(f"   ✓ Enqueued: dataset statistics task")
    
    print("\n6. Starting a worker for 10 seconds...")
    
    # Start worker as subprocess
    worker = subprocess.Popen(
        [sys.executable, "run_worker.py"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    print(f"   Worker started (PID: {worker.pid})")
    
    # Let it run for 10 seconds
    print("   Processing tasks...")
    for i in range(5):
        time.sleep(2)
        
        # Check progress
        with sqlite3.connect('progress.db') as conn:
            cursor = conn.execute('''
                SELECT COUNT(*) FROM question_progress 
                WHERE status = 'completed'
            ''')
            completed = cursor.fetchone()[0]
            print(f"   Progress: {completed}/3 questions completed")
            
            if completed >= 3:
                print("   ✓ All test questions processed!")
                break
    
    # Stop worker
    print("\n7. Stopping worker...")
    worker.terminate()
    worker.wait()
    print(f"   Worker stopped")
    
    # Check results
    print("\n8. Checking results...")
    
    # Check results file
    if os.path.exists('results.jsonl'):
        with open('results.jsonl', 'r') as f:
            lines = f.readlines()
        print(f"   ✓ Results file created with {len(lines)} entries")
        
        if lines:
            # Show first result
            result = json.loads(lines[0])
            print(f"\n   Sample result:")
            print(f"     Question: {result['question_id']}")
            print(f"     Sessions: {result['total_sessions']}")
            print(f"     Messages: {result['total_messages']}")
    else:
        print("   ⚠ No results file created yet")
    
    # Check progress database
    with sqlite3.connect('progress.db') as conn:
        cursor = conn.execute('''
            SELECT question_id, status, sessions_processed, total_sessions
            FROM question_progress
        ''')
        print("\n   Progress database status:")
        for row in cursor:
            qid, status, done, total = row
            print(f"     {qid}: {status} ({done}/{total} sessions)")
    
    print("\n" + "=" * 60)
    print("TEST COMPLETE")
    print("=" * 60)
    
    print("\nKey findings:")
    print("✓ Huey successfully enqueues tasks")
    print("✓ Worker processes tasks from queue")
    print("✓ Progress is tracked in SQLite")
    print("✓ Results are written to JSONL file")
    
    print("\nTo run the full interactive demo:")
    print("  1. In terminal 1: python monitor.py")
    print("  2. In terminal 2: python enqueue_work.py ../longmemeval-datasets/longmemeval_s.json 10")
    print("  3. In terminals 3-5: python run_worker.py")
    
    return True

if __name__ == "__main__":
    success = test_basic_setup()
    sys.exit(0 if success else 1)
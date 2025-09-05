#!/usr/bin/env python
"""
All-in-one demo script to showcase Huey's capabilities.
Run this to see Huey in action with the LongMemEval dataset.
"""

import os
import sys
import json
import time
import sqlite3
import subprocess
from datetime import datetime

def setup_environment():
    """Install requirements and clean up old data."""
    print("🔧 Setting up environment...")
    
    # Install huey if not present
    try:
        import huey
    except ImportError:
        print("Installing huey...")
        subprocess.run([sys.executable, "-m", "pip", "install", "huey"], check=True)
    
    # Clean up old data for fresh demo
    for db_file in ['huey_tasks.db', 'progress.db', 'results.jsonl']:
        if os.path.exists(db_file):
            os.remove(db_file)
            print(f"  Cleaned up {db_file}")
    
    print("✅ Environment ready\n")

def check_dataset():
    """Check if dataset exists and return path."""
    datasets = [
        '../longmemeval-datasets/longmemeval_s.json',
        '../longmemeval-datasets/longmemeval_m.json',
        '../longmemeval-datasets/longmemeval_5s.json',
        '../longmemeval-datasets/longmemeval_1s.json',
    ]
    
    for path in datasets:
        if os.path.exists(path):
            # Check size
            try:
                with open(path, 'r') as f:
                    data = json.load(f)
                print(f"📊 Found dataset: {path}")
                print(f"   Contains {len(data)} questions")
                
                # Check if it has the expected structure
                if data and 'haystack_sessions' in data[0]:
                    sessions_count = len(data[0].get('haystack_sessions', []))
                    print(f"   First question has {sessions_count} sessions")
                return path
            except Exception as e:
                print(f"   Warning: Could not read {path}: {e}")
                continue
    
    print("❌ No dataset found. Please ensure longmemeval-datasets folder exists.")
    print("   Expected locations:")
    for path in datasets:
        print(f"     {path}")
    sys.exit(1)

def run_demo():
    """Run the complete demo."""
    print("=" * 70)
    print("  HUEY TASK QUEUE DEMO - LongMemEval Question Processing".center(70))
    print("=" * 70)
    print()
    
    # Setup
    setup_environment()
    dataset_path = check_dataset()
    
    # Import after setup
    from tasks import init_progress_db
    
    print("\n📝 DEMO OVERVIEW:")
    print("  1. Enqueue questions from LongMemEval dataset")
    print("  2. Run 3 worker processes in parallel")
    print("  3. Each worker processes questions independently")
    print("  4. Track progress at session level")
    print("  5. Show how Huey prevents task collision")
    print()
    
    input("Press Enter to start the demo...")
    
    # Step 1: Initialize and enqueue work
    print("\n" + "=" * 70)
    print("STEP 1: ENQUEUE TASKS")
    print("=" * 70)
    
    # Limit to first 10 questions for demo
    print(f"\n📥 Enqueueing first 10 questions from dataset...")
    subprocess.run([sys.executable, "enqueue_work.py", dataset_path, "10"], check=True)
    
    # Step 2: Start workers
    print("\n" + "=" * 70)
    print("STEP 2: START WORKERS")
    print("=" * 70)
    print("\n🚀 Starting 3 worker processes...")
    
    # Start workers as background processes
    workers = []
    for i in range(3):
        worker = subprocess.Popen(
            [sys.executable, "run_worker.py"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        workers.append(worker)
        print(f"  Started worker {i+1} (PID: {worker.pid})")
    
    print("\n⏳ Processing questions (this will take about 30 seconds)...")
    print("   Workers are processing in parallel - no task collisions!")
    
    # Monitor progress
    print("\n" + "=" * 70)
    print("STEP 3: MONITOR PROGRESS")
    print("=" * 70)
    
    for i in range(15):  # Monitor for 30 seconds
        time.sleep(2)
        
        # Get progress stats
        with sqlite3.connect('progress.db') as conn:
            cursor = conn.execute('''
                SELECT 
                    status,
                    COUNT(*) as count
                FROM question_progress 
                GROUP BY status
            ''')
            
            stats = {row[0]: row[1] for row in cursor}
        
        # Display inline progress
        pending = stats.get('pending', 0)
        processing = stats.get('processing', 0) 
        completed = stats.get('completed', 0)
        failed = stats.get('failed', 0)
        
        print(f"\r  Progress: ✅ {completed} completed | 🔄 {processing} processing | ⏳ {pending} pending | ❌ {failed} failed", end="", flush=True)
        
        # Check if all done
        if completed >= 10:
            print("\n\n✅ All questions processed!")
            break
    
    # Stop workers
    print("\n\n🛑 Stopping workers...")
    for worker in workers:
        worker.terminate()
        worker.wait()
    
    # Step 4: Show results
    print("\n" + "=" * 70)
    print("STEP 4: RESULTS")
    print("=" * 70)
    
    # Show some results
    if os.path.exists('results.jsonl'):
        print("\n📊 Sample Results (first 3 questions):")
        with open('results.jsonl', 'r') as f:
            for i, line in enumerate(f):
                if i >= 3:
                    break
                result = json.loads(line)
                print(f"\n  Question: {result['question_id']}")
                print(f"    Type: {result['question_type']}")
                print(f"    Sessions: {result['total_sessions']}")
                print(f"    Messages: {result['total_messages']}")
                print(f"    Avg messages/session: {result['average_messages_per_session']:.2f}")
    
    # Show which workers processed what
    print("\n👥 Worker Distribution:")
    with sqlite3.connect('progress.db') as conn:
        cursor = conn.execute('''
            SELECT 
                worker_pid,
                COUNT(*) as questions_processed
            FROM question_progress 
            WHERE worker_pid IS NOT NULL
            GROUP BY worker_pid
        ''')
        
        for row in cursor:
            print(f"  Worker PID {row[0]}: {row[1]} questions")
    
    print("\n" + "=" * 70)
    print("DEMO COMPLETE!")
    print("=" * 70)
    print("\n📚 Key Takeaways:")
    print("  ✅ Huey automatically distributes tasks among workers")
    print("  ✅ No task collision - each question processed once")
    print("  ✅ Progress tracked at session level in our SQLite DB")
    print("  ✅ Tasks persist in Huey's SQLite queue (survive crashes)")
    print("  ✅ Can combine Huey's queue with custom progress tracking")
    
    print("\n💡 Try running 'python monitor.py' while workers are running")
    print("   to see real-time progress monitoring!")

if __name__ == "__main__":
    run_demo()
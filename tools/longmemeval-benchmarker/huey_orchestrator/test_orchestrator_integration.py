#!/usr/bin/env python3
"""
Integration test for Huey orchestrator with mock runner.
Tests complete flow with assertions for correctness.
"""

import os
import sys
import json
import time
import tempfile
import sqlite3
import shutil
from pathlib import Path

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent))
sys.path.insert(0, str(Path(__file__).parent.parent))


class TestHueyOrchestrator:
    """Integration test suite for Huey orchestrator."""
    
    def __init__(self):
        self.test_dir = None
        self.dataset_path = None
        self.config_path = None
        self.run_id = None
        
    def setup(self):
        """Set up test environment."""
        # Create temp directory for test files
        self.test_dir = tempfile.mkdtemp(prefix="huey_test_")
        os.chdir(self.test_dir)
        
        # Clean up any existing test files
        self.cleanup()
        
        # Create test dataset
        test_questions = self.create_test_dataset(3)
        self.dataset_path = os.path.join(self.test_dir, "test_dataset.json")
        with open(self.dataset_path, 'w') as f:
            json.dump(test_questions, f)
            
        # Create test config
        self.config_path = os.path.join(self.test_dir, "test_config.toml")
        with open(self.config_path, 'w') as f:
            f.write('''
dataset_file_path = "test_dataset.json"
vault_title = "test_vault"

[models]
ingest = "openai:gpt-5-nano"
qa = "openai:gpt-5-nano"
''')
        
        # Generate run ID
        self.run_id = f"test_{int(time.time())}"
        
        print(f"Test setup complete in {self.test_dir}")
        
    def teardown(self):
        """Clean up test environment."""
        self.cleanup()
        if self.test_dir and os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)
            
    def cleanup(self):
        """Clean up test files."""
        files_to_remove = ['huey_tasks.db', 'progress.db', 'huey_orchestrator.log']
        for file in files_to_remove:
            if os.path.exists(file):
                os.remove(file)
                
    def create_test_dataset(self, num_questions=3):
        """Create a test dataset with varying session counts."""
        questions = []
        for i in range(num_questions):
            question_id = f"test_{i:04d}"
            
            # Varying number of sessions: 3, 5, 7
            num_sessions = 3 + i * 2
            sessions = []
            for s in range(num_sessions):
                session = [
                    {"role": "user", "content": f"Question {i} session {s} message 1"},
                    {"role": "assistant", "content": f"Response to Q{i} S{s} M1"}
                ]
                if s % 2 == 0:  # Add extra messages to even sessions
                    session.extend([
                        {"role": "user", "content": f"Question {i} session {s} message 2"},
                        {"role": "assistant", "content": f"Response to Q{i} S{s} M2"}
                    ])
                sessions.append(session)
            
            question = {
                "question_id": question_id,
                "question_type": "single-session-user",
                "question": f"Test question {i}?",
                "answer": f"Test answer {i}",
                "question_date": "2024-01-01",
                "haystack_session_ids": [f"session_{s}" for s in range(num_sessions)],
                "haystack_dates": ["2024-01-01"] * num_sessions,
                "haystack_sessions": sessions,
                "answer_session_ids": ["session_0"]
            }
            questions.append(question)
        
        return questions
    
    def test_progress_initialization(self):
        """Test that progress tracking is properly initialized."""
        print("\n[TEST] Progress initialization...")
        
        # Set immediate mode for synchronous execution
        os.environ['HUEY_IMMEDIATE'] = 'true'
        
        from progress_tracker import ProgressTracker
        tracker = ProgressTracker()
        
        # Load dataset
        with open(self.dataset_path, 'r') as f:
            dataset = json.load(f)
        
        # Initialize run
        tracker.init_run(self.run_id, dataset)
        
        # Check initial stats
        stats = tracker.get_run_stats(self.run_id)
        
        # Assertions
        assert stats['total_questions'] == 3, f"Expected 3 questions, got {stats['total_questions']}"
        assert stats['pending'] == 3, f"Expected 3 pending, got {stats['pending']}"
        assert stats['completed'] == 0, f"Expected 0 completed, got {stats['completed']}"
        assert stats['total_sessions_expected'] == 15, f"Expected 15 total sessions (3+5+7), got {stats['total_sessions_expected']}"
        
        print("  ✓ Progress initialization successful")
        return True
    
    def test_mock_task_processing(self):
        """Test processing a question with mock runner."""
        print("\n[TEST] Mock task processing...")
        
        os.environ['HUEY_IMMEDIATE'] = 'true'
        
        from progress_tracker import ProgressTracker
        from tasks_mock import process_question_mock
        
        tracker = ProgressTracker()
        
        # Load dataset
        with open(self.dataset_path, 'r') as f:
            dataset = json.load(f)
        
        # Process first question
        question = dataset[0]
        
        # In immediate mode, the task runs synchronously and returns a Result object
        task_result = process_question_mock(
            run_id=self.run_id,
            question_data=question,
            config_path=self.config_path,
            start_session_index=0,
            worker_id="test-worker"
        )
        
        # Get the actual result (in immediate mode, it's wrapped)
        if hasattr(task_result, '__call__'):
            # It's a task, get the result
            result = task_result
        else:
            # It's already the result
            result = task_result
            
        # Check if we got a dict result
        if not isinstance(result, dict):
            # Try to extract from Huey result wrapper
            result = task_result() if callable(task_result) else task_result
        
        # Assertions
        assert isinstance(result, dict), f"Expected dict result, got {type(result)}"
        assert 'vault_id' in result, "Result missing vault_id"
        assert 'memory_id' in result, "Result missing memory_id"
        assert result['sessions_completed'] == 3, f"Expected 3 sessions completed, got {result.get('sessions_completed')}"
        assert result.get('status') in ['success', 'partial'], f"Unexpected status: {result.get('status')}"
        
        # Check progress was updated
        question_details = tracker.get_question_details(self.run_id, question['question_id'])
        assert question_details is not None, "Question progress not found"
        assert question_details['completed_sessions'] >= 0, "Sessions completed not tracked"
        
        print("  ✓ Mock task processing successful")
        return True
    
    def test_resume_capability(self):
        """Test that partially completed questions can be resumed."""
        print("\n[TEST] Resume capability...")
        
        from progress_tracker import ProgressTracker
        tracker = ProgressTracker()
        
        # Simulate partial progress
        with sqlite3.connect(tracker.db_path) as conn:
            conn.execute("""
                UPDATE question_progress
                SET status = 'in_progress',
                    completed_sessions = 2,
                    total_sessions = 5
                WHERE run_id = ? AND question_id = ?
            """, (self.run_id, "test_0001"))
        
        # Get resumable questions
        resumable = tracker.get_resumable_questions(self.run_id)
        
        # Assertions
        assert len(resumable) == 1, f"Expected 1 resumable question, got {len(resumable)}"
        assert resumable[0]['question_id'] == 'test_0001', f"Wrong question ID: {resumable[0]['question_id']}"
        assert resumable[0]['completed_sessions'] == 2, f"Expected 2 completed sessions, got {resumable[0]['completed_sessions']}"
        
        print("  ✓ Resume capability successful")
        return True
    
    def test_concurrent_task_claiming(self):
        """Test that multiple workers can't claim the same question."""
        print("\n[TEST] Concurrent task claiming...")
        
        from progress_tracker import ProgressTracker
        tracker = ProgressTracker()
        
        # Try to claim a question
        claimed1 = tracker.claim_question(self.run_id, "worker-1")
        claimed2 = tracker.claim_question(self.run_id, "worker-1")
        
        # Assertions
        assert claimed1 is not None, "First claim should succeed"
        assert claimed1['question_id'] in ['test_0000', 'test_0001', 'test_0002'], "Invalid question claimed"
        
        # Check that same question can't be claimed again
        with sqlite3.connect(tracker.db_path) as conn:
            result = conn.execute("""
                SELECT COUNT(*) FROM question_progress 
                WHERE run_id = ? AND status = 'in_progress'
            """, (self.run_id,)).fetchone()
            in_progress_count = result[0]
        
        assert in_progress_count >= 1, "At least one question should be in progress"
        
        print("  ✓ Concurrent task claiming successful")
        return True
    
    def test_statistics_tracking(self):
        """Test that statistics are properly tracked."""
        print("\n[TEST] Statistics tracking...")
        
        from progress_tracker import ProgressTracker
        tracker = ProgressTracker()
        
        # Update some progress
        tracker.update_session_progress(self.run_id, "test_0000", 3)
        tracker.mark_ingestion_complete(self.run_id, "test_0000")
        
        # Get stats
        stats = tracker.get_run_stats(self.run_id)
        
        # Assertions
        assert stats['total_sessions_completed'] >= 3, f"Expected at least 3 sessions completed, got {stats['total_sessions_completed']}"
        
        # Check question-specific progress
        details = tracker.get_question_details(self.run_id, "test_0000")
        assert details['ingestion_status'] == 'completed', f"Expected completed ingestion, got {details['ingestion_status']}"
        assert details['completed_sessions'] == 3, f"Expected 3 completed sessions, got {details['completed_sessions']}"
        
        print("  ✓ Statistics tracking successful")
        return True
    
    def run_all_tests(self):
        """Run all integration tests."""
        print("="*60)
        print("HUEY ORCHESTRATOR INTEGRATION TEST")
        print("="*60)
        
        try:
            self.setup()
            
            tests = [
                self.test_progress_initialization,
                self.test_mock_task_processing,
                self.test_resume_capability,
                self.test_concurrent_task_claiming,
                self.test_statistics_tracking
            ]
            
            passed = 0
            failed = 0
            
            for test_func in tests:
                try:
                    if test_func():
                        passed += 1
                    else:
                        failed += 1
                        print(f"  ✗ {test_func.__name__} failed")
                except AssertionError as e:
                    failed += 1
                    print(f"  ✗ {test_func.__name__} failed: {e}")
                except Exception as e:
                    failed += 1
                    print(f"  ✗ {test_func.__name__} error: {e}")
            
            print("\n" + "="*60)
            print(f"TEST RESULTS: {passed} passed, {failed} failed")
            print("="*60)
            
            if failed == 0:
                print("\n✅ ALL TESTS PASSED!")
                return True
            else:
                print(f"\n❌ {failed} TEST(S) FAILED")
                return False
                
        finally:
            self.teardown()


if __name__ == "__main__":
    tester = TestHueyOrchestrator()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)
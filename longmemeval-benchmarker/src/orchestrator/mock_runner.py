#!/usr/bin/env python3
"""
Mock single_question_runner for testing the Huey orchestrator.
Simulates the behavior of the real runner without actually calling Mycelian.
"""

import json
import time
import random
import argparse
import sys
from pathlib import Path
from datetime import datetime
import uuid


def mock_single_question_runner(
    question_json_path: str,
    config_path: str,
    memory_title: str,
    start_session_index: int = 0,
    output_dir: str = None
):
    """
    Mock implementation of single_question_runner.

    Args:
        question_json_path: Path to JSON file with question data
        config_path: Path to config TOML (ignored in mock)
        memory_title: Title for the memory (e.g., "run_123_abc123")
        start_session_index: Session index to start from
        output_dir: Directory for outputs

    Returns:
        JSON string with results
    """
    # Load question data
    with open(question_json_path, 'r') as f:
        question_data = json.load(f)

    question_id = question_data['question_id']
    sessions = question_data.get('haystack_sessions', [])
    total_sessions = len(sessions)

    print(f"[MOCK] Processing question {question_id}")
    print(f"[MOCK] Memory title: {memory_title}")
    print(f"[MOCK] Starting from session {start_session_index}/{total_sessions}")

    # Simulate vault/memory creation (only if starting from beginning)
    if start_session_index == 0:
        vault_id = f"vault-{uuid.uuid4()}"
        memory_id = f"memory-{uuid.uuid4()}"
        print(f"[MOCK] Created vault: {vault_id}")
        print(f"[MOCK] Created memory: {memory_id}")
    else:
        # Simulate retrieving existing IDs
        vault_id = f"vault-existing-{question_id[:8]}"
        memory_id = f"memory-existing-{question_id[:8]}"
        print(f"[MOCK] Using existing vault: {vault_id}")
        print(f"[MOCK] Using existing memory: {memory_id}")

    # Process sessions
    sessions_completed = 0
    messages_processed = 0

    for idx in range(start_session_index, total_sessions):
        session = sessions[idx]

        # Simulate processing time (faster than real)
        process_time = random.uniform(0.1, 0.3)
        time.sleep(process_time)

        # Count messages in session
        session_messages = len(session) if isinstance(session, list) else 0
        messages_processed += session_messages
        sessions_completed += 1

        # Log progress every 10 sessions
        if (idx + 1) % 10 == 0 or (idx + 1) == total_sessions:
            print(f"[MOCK] Progress: {idx + 1}/{total_sessions} sessions")

        # Simulate occasional failure (5% chance)
        if random.random() < 0.05 and idx > start_session_index:
            print(f"[MOCK] Simulated failure at session {idx}")
            # Return partial progress
            result = {
                "status": "partial",
                "question_id": question_id,
                "vault_id": vault_id,
                "memory_id": memory_id,
                "memory_title": memory_title,
                "sessions_completed": sessions_completed - 1,  # Last one failed
                "sessions_total": total_sessions,
                "messages_processed": messages_processed,
                "error": f"Simulated failure at session {idx}"
            }
            return json.dumps(result)

    print(f"[MOCK] Completed all {sessions_completed} sessions")

    # Return success result
    result = {
        "status": "success",
        "question_id": question_id,
        "vault_id": vault_id,
        "memory_id": memory_id,
        "memory_title": memory_title,
        "sessions_completed": sessions_completed,
        "sessions_total": total_sessions,
        "messages_processed": messages_processed,
        "processing_time": random.uniform(1, 5)
    }

    return json.dumps(result)


def mock_qa_runner(
    question_json_path: str,
    vault_id: str,
    memory_id: str,
    config_path: str,
    output_dir: str = None
):
    """
    Mock implementation of QA runner.

    Args:
        question_json_path: Path to JSON file with question data
        vault_id: Mycelian vault ID
        memory_id: Mycelian memory ID
        config_path: Path to config TOML
        output_dir: Directory for outputs

    Returns:
        JSON string with QA results
    """
    # Load question data
    with open(question_json_path, 'r') as f:
        question_data = json.load(f)

    question_id = question_data['question_id']
    expected_answer = question_data.get('answer', '')

    print(f"[MOCK QA] Running QA for question {question_id}")
    print(f"[MOCK QA] Using memory: {memory_id} in vault: {vault_id}")

    # Simulate QA processing
    time.sleep(random.uniform(0.5, 1.5))

    # Generate mock answer (sometimes correct, sometimes not)
    is_correct = random.random() > 0.3  # 70% accuracy
    if is_correct:
        predicted_answer = expected_answer
    else:
        predicted_answer = f"Wrong answer for {question_id}"

    print(f"[MOCK QA] Generated answer: {predicted_answer[:50]}...")

    # Create hypothesis
    hypothesis = {
        "question_id": question_id,
        "predicted_answer": predicted_answer,
        "expected_answer": expected_answer,
        "is_correct": is_correct,
        "confidence": random.uniform(0.5, 1.0),
        "vault_id": vault_id,
        "memory_id": memory_id,
        "timestamp": datetime.now().isoformat()
    }

    # Write hypothesis to file if output_dir provided
    if output_dir:
        output_path = Path(output_dir)
        output_path.mkdir(parents=True, exist_ok=True)
        hypothesis_file = output_path / f"hypothesis_{question_id}.json"
        with open(hypothesis_file, 'w') as f:
            json.dump(hypothesis, f, indent=2)
        print(f"[MOCK QA] Wrote hypothesis to {hypothesis_file}")

    return json.dumps({
        "status": "success",
        "question_id": question_id,
        "is_correct": is_correct,
        "hypothesis": hypothesis
    })


def main():
    """CLI interface for the mock runner."""
    parser = argparse.ArgumentParser(description='Mock runner for testing')
    parser.add_argument('--mode', choices=['ingest', 'qa'], required=True,
                       help='Mode to run in')
    parser.add_argument('--question-json', required=True,
                       help='Path to question JSON file')
    parser.add_argument('--config', required=True,
                       help='Path to config TOML file')
    parser.add_argument('--memory-title',
                       help='Memory title (for ingest mode)')
    parser.add_argument('--start-session', type=int, default=0,
                       help='Session index to start from (for ingest mode)')
    parser.add_argument('--vault-id',
                       help='Vault ID (for QA mode)')
    parser.add_argument('--memory-id',
                       help='Memory ID (for QA mode)')
    parser.add_argument('--output-dir',
                       help='Output directory')

    args = parser.parse_args()

    if args.mode == 'ingest':
        if not args.memory_title:
            print("Error: --memory-title required for ingest mode")
            sys.exit(1)

        result = mock_single_question_runner(
            question_json_path=args.question_json,
            config_path=args.config,
            memory_title=args.memory_title,
            start_session_index=args.start_session,
            output_dir=args.output_dir
        )
        print(result)

    elif args.mode == 'qa':
        if not args.vault_id or not args.memory_id:
            print("Error: --vault-id and --memory-id required for QA mode")
            sys.exit(1)

        result = mock_qa_runner(
            question_json_path=args.question_json,
            vault_id=args.vault_id,
            memory_id=args.memory_id,
            config_path=args.config,
            output_dir=args.output_dir
        )
        print(result)


if __name__ == '__main__':
    main()

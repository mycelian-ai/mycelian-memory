#!/usr/bin/env python3
"""
CLI wrapper for SingleQuestionRunner that supports resume and JSON output.
This wraps the existing SingleQuestionRunner to add orchestrator features.
"""

import argparse
import json
import sys
import logging
import os
from pathlib import Path
from typing import Dict, Any, Optional

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from single_question_runner import SingleQuestionRunner
from memory_manager import MemoryManager
from mycelian_memory_agent.mcp_utils import create_mcp_client

try:
    import tomllib
except ImportError:
    import tomli as tomllib  # Python < 3.11


def setup_logging(verbose: bool = False):
    """Set up logging configuration."""
    level = logging.INFO if verbose else logging.WARNING
    logging.basicConfig(
        level=level,
        format='%(asctime)s [%(name)s:%(levelname)s] %(message)s'
    )
    return logging.getLogger('runner_cli')


def load_question_data(question_json: str) -> Dict[str, Any]:
    """Load question data from JSON file."""
    with open(question_json, 'r') as f:
        return json.load(f)


def load_config(config_path: str) -> Any:
    """Load configuration from TOML file."""
    with open(config_path, 'rb') as f:
        config_dict = tomllib.load(f)
    
    # Use same config structure as benchmarker
    from benchmarker import _SimpleConfig
    return _SimpleConfig(config_dict)


def run_ingestion(
    question_data: Dict[str, Any],
    config: Any,
    memory_title: str,
    start_session_index: int = 0,
    vault_id: Optional[str] = None,
    run_id: Optional[str] = None,
    output_format: str = "json",
    verbose: bool = False
) -> Dict[str, Any]:
    """
    Run ingestion for a question, supporting resume from specific session.
    
    Returns:
        Dict with vault_id, memory_id, sessions_completed, etc.
    """
    logger = setup_logging(verbose)
    question_id = question_data.get('question_id', 'unknown')
    
    # Get sessions to process
    all_sessions = question_data.get('haystack_sessions', [])
    total_sessions = len(all_sessions)
    
    logger.info(f"Processing question {question_id}: sessions {start_session_index}/{total_sessions}")
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    # Get or create vault
    memory_mgr = MemoryManager(mcp_client, debug=False)
    if not vault_id:
        vault_id = memory_mgr.ensure_vault(config.vault_title, config.vault_id)
    
    # Create or get memory
    if not run_id:
        run_id = f"cli_run_{int(os.getpid())}"
    
    memory_id = memory_mgr.ensure_memory(vault_id, memory_title, memory_type="NOTES")
    
    logger.info(f"Using vault_id={vault_id}, memory_id={memory_id}")
    
    # If starting from middle, we need to slice sessions
    if start_session_index > 0:
        # Create modified question data with only remaining sessions
        modified_question = question_data.copy()
        modified_question['haystack_sessions'] = all_sessions[start_session_index:]
        
        # Map session IDs correctly
        if 'haystack_session_ids' in modified_question:
            all_session_ids = modified_question['haystack_session_ids']
            modified_question['haystack_session_ids'] = all_session_ids[start_session_index:]
        
        if 'haystack_dates' in modified_question:
            all_dates = modified_question['haystack_dates']
            modified_question['haystack_dates'] = all_dates[start_session_index:]
    else:
        modified_question = question_data
    
    # Initialize runner
    runner = SingleQuestionRunner(config, mcp_client=mcp_client, mode="ingestion")
    
    # Create a StringIO for log output
    from io import StringIO
    log_buffer = StringIO()
    
    try:
        # Run ingestion only
        result = runner.run_question(
            modified_question, 
            vault_id=vault_id, 
            run_id=run_id, 
            log=log_buffer,
            memory_id=memory_id
        )
        
        # Calculate actual sessions processed
        sessions_completed = start_session_index + len(modified_question.get('haystack_sessions', []))
        
        # Count messages processed
        messages_processed = sum(
            len(session) if isinstance(session, list) else 0
            for session in modified_question.get('haystack_sessions', [])
        )
        
        # Build result
        output = {
            "status": "success",
            "question_id": question_id,
            "vault_id": vault_id,
            "memory_id": memory_id,
            "memory_title": memory_title,
            "sessions_completed": sessions_completed,
            "sessions_total": total_sessions,
            "messages_processed": messages_processed
        }
        
        if sessions_completed < total_sessions:
            output["status"] = "partial"
            output["next_session_index"] = sessions_completed
            
        return output
        
    except Exception as e:
        logger.error(f"Error processing question: {e}")
        
        # Try to return partial progress
        sessions_done = start_session_index  # At minimum, we started from here
        
        return {
            "status": "failed",
            "question_id": question_id,
            "vault_id": vault_id,
            "memory_id": memory_id,
            "memory_title": memory_title,
            "sessions_completed": sessions_done,
            "sessions_total": total_sessions,
            "messages_processed": 0,
            "error": str(e)
        }


def run_qa(
    question_data: Dict[str, Any],
    config: Any,
    vault_id: str,
    memory_id: str,
    run_id: Optional[str] = None,
    output_format: str = "json",
    verbose: bool = False
) -> Dict[str, Any]:
    """
    Run QA phase for a question with existing memory.
    
    Returns:
        Dict with hypothesis and results.
    """
    logger = setup_logging(verbose)
    question_id = question_data.get('question_id', 'unknown')
    
    logger.info(f"Running QA for question {question_id} with memory {memory_id}")
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    if not run_id:
        run_id = f"cli_qa_{int(os.getpid())}"
    
    # Initialize runner for QA only
    runner = SingleQuestionRunner(config, mcp_client=mcp_client, mode="qa")
    
    # Create log buffer
    from io import StringIO
    log_buffer = StringIO()
    
    try:
        # Run QA only
        result = runner.run_question(
            question_data,
            vault_id=vault_id,
            run_id=run_id,
            log=log_buffer,
            memory_id=memory_id
        )
        
        return {
            "status": "success",
            "question_id": question_id,
            "hypothesis": result.get("hypothesis", ""),
            "is_correct": None  # Would need expected answer to evaluate
        }
        
    except Exception as e:
        logger.error(f"QA failed: {e}")
        return {
            "status": "failed",
            "question_id": question_id,
            "hypothesis": "",
            "error": str(e)
        }


def main():
    """CLI entry point."""
    parser = argparse.ArgumentParser(
        description='Run single question with resume support and JSON output'
    )
    parser.add_argument('--mode', choices=['ingest', 'qa', 'all'], required=True,
                       help='Execution mode')
    parser.add_argument('--question-json', required=True,
                       help='Path to question JSON file')
    parser.add_argument('--config', required=True,
                       help='Path to config TOML file')
    parser.add_argument('--memory-title', 
                       help='Memory title (default: derived from question_id)')
    parser.add_argument('--start-session', type=int, default=0,
                       help='Session index to start from (for resume)')
    parser.add_argument('--vault-id',
                       help='Vault ID (for QA mode or explicit vault)')
    parser.add_argument('--memory-id',
                       help='Memory ID (required for QA mode)')
    parser.add_argument('--run-id',
                       help='Run ID for tracking')
    parser.add_argument('--output-format', choices=['json', 'text'], default='json',
                       help='Output format')
    parser.add_argument('-v', '--verbose', action='store_true',
                       help='Verbose output')
    
    args = parser.parse_args()
    
    # Load data
    question_data = load_question_data(args.question_json)
    config = load_config(args.config)
    
    # Derive memory title if not provided
    if not args.memory_title:
        question_id = question_data.get('question_id', 'unknown')
        run_id = args.run_id or f"cli_{int(os.getpid())}"
        args.memory_title = f"{run_id}_{question_id}"
    
    # Execute based on mode
    if args.mode == 'ingest':
        result = run_ingestion(
            question_data=question_data,
            config=config,
            memory_title=args.memory_title,
            start_session_index=args.start_session,
            vault_id=args.vault_id,
            run_id=args.run_id,
            output_format=args.output_format,
            verbose=args.verbose
        )
    
    elif args.mode == 'qa':
        if not args.vault_id or not args.memory_id:
            print("Error: --vault-id and --memory-id required for QA mode", file=sys.stderr)
            sys.exit(1)
        
        result = run_qa(
            question_data=question_data,
            config=config,
            vault_id=args.vault_id,
            memory_id=args.memory_id,
            run_id=args.run_id,
            output_format=args.output_format,
            verbose=args.verbose
        )
    
    else:  # all
        # Run ingestion first
        ingestion_result = run_ingestion(
            question_data=question_data,
            config=config,
            memory_title=args.memory_title,
            start_session_index=args.start_session,
            vault_id=args.vault_id,
            run_id=args.run_id,
            output_format=args.output_format,
            verbose=args.verbose
        )
        
        if ingestion_result['status'] == 'success':
            # Then run QA
            qa_result = run_qa(
                question_data=question_data,
                config=config,
                vault_id=ingestion_result['vault_id'],
                memory_id=ingestion_result['memory_id'],
                run_id=args.run_id,
                output_format=args.output_format,
                verbose=args.verbose
            )
            
            # Merge results
            result = {**ingestion_result, **qa_result}
        else:
            result = ingestion_result
    
    # Output result
    if args.output_format == 'json':
        print(json.dumps(result))
    else:
        print(f"Status: {result.get('status', 'unknown')}")
        print(f"Question: {result.get('question_id', 'unknown')}")
        if 'vault_id' in result:
            print(f"Vault: {result['vault_id']}")
        if 'memory_id' in result:
            print(f"Memory: {result['memory_id']}")
        if 'sessions_completed' in result:
            print(f"Sessions: {result['sessions_completed']}/{result.get('sessions_total', '?')}")
        if 'hypothesis' in result:
            print(f"Hypothesis: {result['hypothesis'][:100]}...")
        if 'error' in result:
            print(f"Error: {result['error']}")
    
    return 0 if result.get('status') == 'success' else 1


if __name__ == '__main__':
    sys.exit(main())
#!/usr/bin/env python3
"""Debug script for testing QA phase directly on an existing memory.

Usage:
    python debug_qa.py --memory-id <memory-id> --vault-id <vault-id> [--question <question>]
    
Example:
    python debug_qa.py \
        --memory-id 74401f84-05f9-45df-8cd4-ff4d3ae14af7 \
        --vault-id e04ca555-b87e-490a-9807-5c7577c4e226 \
        --question "What is the name and color of Sarah's dog?"
"""

import argparse
import json
import logging
import sys
from typing import Dict, Any

from src.mycelian_memory_agent import create_mcp_client
from src.memory_manager import MemoryManager


def search_and_qa(memory_id: str, vault_id: str, question: str, model: str = "openai:gpt-5-2025-08-07", 
                  use_two_pass: bool = False) -> Dict[str, Any]:
    """Run search and QA on an existing memory."""
    
    # Set up logging
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(name)s] %(levelname)s: %(message)s"
    )
    logger = logging.getLogger("debug_qa")
    
    # Create MCP client
    logger.info("Creating MCP client...")
    mcp_client = create_mcp_client()
    
    # Create memory manager
    mm = MemoryManager(mcp_client, debug=True)
    
    # Check if two-pass search is requested
    if use_two_pass:
        logger.info(f"Using TWO-PASS search for memory {memory_id} with query: {question}")
        from src.single_question_runner import _two_pass_search
        search_result = _two_pass_search(mm, memory_id, question, model, logger)
    else:
        # Search memories
        logger.info(f"Searching memory {memory_id} with query: {question}")
        # Use new ke/kc parameters for better context retrieval
        search_result = mm.search_memories(memory_id, query=question, top_ke=5, top_kc=3)
    
    # Log search results in detail
    logger.info("=" * 60)
    logger.info("SEARCH RESULTS:")
    logger.info("=" * 60)
    
    if isinstance(search_result, dict):
        # Check for entries
        entries = search_result.get("entries", [])
        logger.info(f"Found {len(entries)} entries")
        for i, entry in enumerate(entries[:5], 1):  # Show first 5
            if isinstance(entry, dict):
                summary = entry.get("summary", "(no summary)")
                logger.info(f"  Entry {i}: {summary[:100]}...")
        
        # Check for latest context
        latest_ctx = search_result.get("latestContext") or search_result.get("latest_context")
        if latest_ctx:
            logger.info(f"\nLatest Context (first 500 chars):")
            logger.info(f"  {latest_ctx[:500]}...")
        else:
            logger.info("\nNo latest context found")
        
        # Check for best context (deprecated)
        best_ctx = search_result.get("bestContext") or search_result.get("best_context")
        if best_ctx:
            logger.info(f"\nBest Context (deprecated, first 500 chars):")
            logger.info(f"  {best_ctx[:500]}...")
        else:
            logger.info("\nNo best context found (deprecated field)")
        
        # Check for context shards (new)
        contexts = search_result.get("contexts", [])
        logger.info(f"\nContext Shards: Found {len(contexts)} shards")
        for i, ctx in enumerate(contexts[:3], 1):  # Show up to 3
            if isinstance(ctx, dict):
                score = ctx.get("score", 0)
                timestamp = ctx.get("timestamp", "")
                content = ctx.get("context", "")
                logger.info(f"  Shard {i} (score: {score:.3f}, timestamp: {timestamp}):")
                logger.info(f"    {content[:300]}...")
    else:
        logger.warning(f"Unexpected search result type: {type(search_result)}")
        logger.info(f"Search result: {search_result}")
    
    # Build context for QA
    logger.info("\n" + "=" * 60)
    logger.info("BUILDING QA CONTEXT:")
    logger.info("=" * 60)
    
    from src.single_question_runner import _build_qa_context
    context = _build_qa_context(search_result, 10)
    
    if context:
        logger.info(f"Built context of {len(context)} characters")
        logger.info(f"Context preview (first 1000 chars):")
        logger.info(f"  {context[:1000]}...")
    else:
        logger.warning("No context built from search results")
    
    # Run QA
    logger.info("\n" + "=" * 60)
    logger.info("RUNNING QA:")
    logger.info("=" * 60)
    
    from src.single_question_runner import _run_qa
    logger.info(f"Running QA with model: {model}")
    logger.info(f"Question: {question}")
    
    answer = _run_qa(model, question, context)
    
    logger.info(f"\nAnswer: {answer}")
    
    return {
        "memory_id": memory_id,
        "vault_id": vault_id,
        "question": question,
        "context_length": len(context) if context else 0,
        "entries_found": len(search_result.get("entries", [])) if isinstance(search_result, dict) else 0,
        "has_latest_context": bool((search_result.get("latestContext") or search_result.get("latest_context")) if isinstance(search_result, dict) else False),
        "has_best_context": bool((search_result.get("bestContext") or search_result.get("best_context")) if isinstance(search_result, dict) else False),
        "answer": answer
    }


def main():
    parser = argparse.ArgumentParser(description="Debug QA phase on existing memory")
    parser.add_argument("--memory-id", required=True, help="Memory ID to query")
    parser.add_argument("--vault-id", required=True, help="Vault ID containing the memory")
    parser.add_argument("--question", default="What is the name and color of Sarah's dog?",
                       help="Question to ask (default: Sarah's dog question)")
    parser.add_argument("--model", default="openai:gpt-5-2025-08-07",
                       help="Model to use for QA (default: openai:gpt-5-2025-08-07)")
    parser.add_argument("--two-pass", action="store_true", help="Use two-pass search algorithm")
    parser.add_argument("--json", action="store_true", help="Output result as JSON")
    
    args = parser.parse_args()
    
    try:
        result = search_and_qa(
            memory_id=args.memory_id,
            vault_id=args.vault_id,
            question=args.question,
            model=args.model,
            use_two_pass=args.two_pass
        )
        
        if args.json:
            print("\n" + json.dumps(result, indent=2))
        else:
            print("\n" + "=" * 60)
            print("SUMMARY:")
            print("=" * 60)
            print(f"Memory ID: {result['memory_id']}")
            print(f"Vault ID: {result['vault_id']}")
            print(f"Question: {result['question']}")
            print(f"Context Length: {result['context_length']} chars")
            print(f"Entries Found: {result['entries_found']}")
            print(f"Has Latest Context: {result['has_latest_context']}")
            print(f"Has Best Context: {result['has_best_context']}")
            print(f"\nAnswer: {result['answer']}")
        
        return 0
        
    except Exception as e:
        logging.exception(f"Error during QA debug: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
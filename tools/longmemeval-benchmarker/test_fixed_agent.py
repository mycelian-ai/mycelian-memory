#!/usr/bin/env python3
"""Test script to verify the agent doesn't get stuck in infinite loops."""

import sys
import os
import asyncio
import logging
from typing import Any

# Add src to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from src.agent.mycelian_memory_agent import build_agent

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


async def test_agent():
    """Test that the agent handles messages without infinite loops."""
    
    # Test configuration
    vault_id = "test-vault-123"
    memory_id = "test-memory-456"
    model_id = "gpt-4o-mini-2024-07-18"
    
    logger.info("Creating agent...")
    try:
        agent = build_agent(
            model_id=model_id,
            vault_id=vault_id,
            memory_id=memory_id,
            debug=True
        )
        logger.info("Agent created successfully")
    except Exception as e:
        logger.error(f"Failed to create agent: {e}")
        return False
    
    # Test 1: SESSION_START should trigger get_context and list_entries
    logger.info("\n=== Test 1: SESSION_START ===")
    try:
        thread_id = "test-thread-1"
        result = agent.process_message(
            message_type="system",
            content="SESSION_START",
            thread_id=thread_id
        )
        logger.info(f"SESSION_START processed without infinite loop")
    except Exception as e:
        logger.error(f"Test 1 failed: {e}")
        return False
    
    # Test 2: User message should trigger add_entry
    logger.info("\n=== Test 2: User Message ===")
    try:
        result = agent.process_message(
            message_type="conversation",
            content="Hello, I want to plan a trip to Japan.",
            thread_id=thread_id,
            role="user",
            msg_idx=1
        )
        logger.info(f"User message processed without infinite loop")
    except Exception as e:
        logger.error(f"Test 2 failed: {e}")
        return False
    
    # Test 3: Assistant message should trigger add_entry
    logger.info("\n=== Test 3: Assistant Message ===")
    try:
        result = agent.process_message(
            message_type="conversation",
            content="I'd be happy to help you plan your trip to Japan! Japan offers a wonderful mix of ancient traditions and modern technology.",
            thread_id=thread_id,
            role="assistant",
            msg_idx=2
        )
        logger.info(f"Assistant message processed without infinite loop")
    except Exception as e:
        logger.error(f"Test 3 failed: {e}")
        return False
    
    # Test 4: Check thread state
    logger.info("\n=== Test 4: Check Thread State ===")
    try:
        state = agent._agent.get_state({"configurable": {"thread_id": thread_id}})
        messages = state.values.get("messages", [])
        logger.info(f"Thread has {len(messages)} messages")
        
        # Count message types
        from langchain_core.messages import SystemMessage, ChatMessage, AIMessage, ToolMessage
        sys_count = sum(1 for m in messages if isinstance(m, SystemMessage))
        chat_count = sum(1 for m in messages if isinstance(m, ChatMessage))
        ai_count = sum(1 for m in messages if isinstance(m, AIMessage))
        tool_count = sum(1 for m in messages if isinstance(m, ToolMessage))
        
        logger.info(f"Message types: System={sys_count}, Chat={chat_count}, AI={ai_count}, Tool={tool_count}")
        
        # Check for duplicate add_entry calls
        add_entry_count = 0
        for msg in messages:
            if isinstance(msg, AIMessage) and hasattr(msg, 'tool_calls'):
                for tc in msg.tool_calls:
                    if tc.get("name") == "add_entry":
                        add_entry_count += 1
        
        logger.info(f"add_entry calls: {add_entry_count}")
        
        # We should have exactly 2 add_entry calls (one for user, one for assistant)
        if add_entry_count != 2:
            logger.error(f"Expected 2 add_entry calls, got {add_entry_count}")
            return False
            
    except Exception as e:
        logger.error(f"Test 4 failed: {e}")
        return False
    
    logger.info("\n=== All tests passed! ===")
    return True


def main():
    """Main function to run the test."""
    try:
        # Get or create event loop
        try:
            loop = asyncio.get_running_loop()
            # If there's already a running loop, run in it
            success = loop.run_until_complete(test_agent())
        except RuntimeError:
            # No running loop, create one
            success = asyncio.run(test_agent())
        
        if success:
            print("\n✅ SUCCESS: Agent handles messages without infinite loops!")
            sys.exit(0)
        else:
            print("\n❌ FAILURE: Agent still has issues")
            sys.exit(1)
    except Exception as e:
        logger.error(f"Test failed with error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
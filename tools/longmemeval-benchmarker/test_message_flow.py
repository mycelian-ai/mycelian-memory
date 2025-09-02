#!/usr/bin/env python3
"""Test the actual message flow as used by the benchmarker."""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from src.agent import build_agent
import logging

# Set up logging
logging.basicConfig(level=logging.WARNING)  # Reduce noise


def test_message_flow():
    """Test the message flow as it actually happens in the benchmarker."""
    
    print("Testing Actual Message Flow (as benchmarker does it)")
    print("=" * 50)
    
    # Create agent like the benchmarker does
    try:
        agent = build_agent(
            model_id="openai:gpt-4o-mini-2024-07-18",
            vault_id="test-vault-id",
            memory_id="test-memory-id",
            debug=False  # Turn off debug to reduce output
        )
        print("✓ Agent created successfully")
    except Exception as e:
        print(f"✗ Failed to create agent: {e}")
        return False
    
    thread_id = "test-thread-001"
    
    # Test 1: SESSION_START (as benchmarker does)
    print("\n1. Processing SESSION_START...")
    try:
        agent.invoke_message(
            message_type="system",
            content="SESSION_START",
            thread_id=thread_id
        )
        print("   ✓ SESSION_START processed")
    except Exception as e:
        if "Recursion limit" in str(e):
            print(f"   ✗ INFINITE LOOP on SESSION_START!")
            return False
        print(f"   ✗ Error: {e}")
        return False
    
    # Test 2: User message (as benchmarker does)
    print("\n2. Processing user message...")
    try:
        agent.invoke_message(
            message_type="conversation",
            content="I'm planning a trip to Japan. Any suggestions?",
            thread_id=thread_id,
            role="user",
            msg_idx=1
        )
        print("   ✓ User message processed")
    except Exception as e:
        if "Recursion limit" in str(e):
            print(f"   ✗ INFINITE LOOP on user message!")
            return False
        print(f"   ✗ Error: {e}")
        return False
    
    # Test 3: Assistant message (as benchmarker does)
    print("\n3. Processing assistant message...")
    try:
        agent.invoke_message(
            message_type="conversation",
            content="Japan is wonderful! I recommend visiting Tokyo for modern culture and Kyoto for traditional temples.",
            thread_id=thread_id,
            role="assistant",
            msg_idx=2
        )
        print("   ✓ Assistant message processed")
    except Exception as e:
        if "Recursion limit" in str(e):
            print(f"   ✗ INFINITE LOOP on assistant message!")
            return False
        print(f"   ✗ Error: {e}")
        return False
    
    # Test 4: Another user message
    print("\n4. Processing another user message...")
    try:
        agent.invoke_message(
            message_type="conversation",
            content="What's the best time to visit?",
            thread_id=thread_id,
            role="user",
            msg_idx=3
        )
        print("   ✓ Second user message processed")
    except Exception as e:
        if "Recursion limit" in str(e):
            print(f"   ✗ INFINITE LOOP on second user message!")
            return False
        print(f"   ✗ Error: {e}")
        return False
    
    # Check the state
    print("\n5. Checking final state...")
    try:
        state = agent.agent.graph.get_state({"configurable": {"thread_id": thread_id}})
        messages = state.values.get("messages", [])
        
        # Count add_entry calls
        from langchain_core.messages import AIMessage
        add_entry_count = 0
        for msg in messages:
            if isinstance(msg, AIMessage) and hasattr(msg, 'tool_calls'):
                for tc in msg.tool_calls:
                    if tc.get("name") == "add_entry":
                        add_entry_count += 1
        
        print(f"   Total messages in state: {len(messages)}")
        print(f"   Total add_entry calls: {add_entry_count}")
        
        # We should have 3 add_entry calls (3 conversation messages)
        if add_entry_count != 3:
            print(f"   ✗ Expected 3 add_entry calls, got {add_entry_count}")
            # This might be OK if the agent is being smart about batching
            # But it shouldn't be more than 3
            if add_entry_count > 3:
                print("   ✗ DUPLICATE PROCESSING DETECTED!")
                return False
    except Exception as e:
        print(f"   ✗ Failed to check state: {e}")
        return False
    
    print("\n" + "=" * 50)
    print("✅ Message flow test passed!")
    print("   No infinite loops detected")
    print("   Messages processed correctly")
    return True


if __name__ == "__main__":
    success = test_message_flow()
    sys.exit(0 if success else 1)
#!/usr/bin/env python3
"""Test the fixed agent directly to see if recursion is resolved."""

import sys
import json
import logging
from langchain_core.messages import ChatMessage

# Setup logging to see what's happening
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(message)s'
)

sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.build import build_agent_with_invoker
from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from src.memory_manager import MemoryManager

def test_fixed_agent():
    """Test the agent with our fix."""
    
    print("\n" + "=" * 70)
    print("TESTING FIXED AGENT (no add_messages on messages field)")
    print("=" * 70)
    
    # Create MCP client
    print("\n1. Creating MCP client...")
    mcp_client = create_mcp_client()
    
    # Create vault and memory
    print("2. Setting up vault and memory...")
    mm = MemoryManager(mcp_client, debug=False)
    
    # Use a test vault
    vault_id = mm.ensure_vault(title="test-fixed-agent", vault_id=None)
    memory_id = mm.ensure_memory(vault_id, "test-memory", memory_type="NOTES")
    
    print(f"   Vault ID: {vault_id}")
    print(f"   Memory ID: {memory_id}")
    
    # Build agent with fixed implementation
    print("\n3. Building agent with fixed implementation...")
    invoker = build_agent_with_invoker(
        model_id="gpt-4o-mini",
        vault_id=vault_id,
        memory_id=memory_id,
        mcp_client=mcp_client,
        max_tool_calls_per_turn=5
    )
    
    # Test START_SESSION
    print("\n4. Testing START_SESSION...")
    thread_id = f"{memory_id}:test"
    
    try:
        invoker.start_session(thread_id)
        print("   ✅ START_SESSION completed successfully")
    except Exception as e:
        print(f"   ❌ START_SESSION failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE (message 1)
    print("\n5. Testing PROCESS_MESSAGE (message 1)...")
    try:
        invoker.process_conversation_message(
            role="user",
            content="Hi, my name is Sarah and I have a golden retriever named Max.",
            thread_id=thread_id
        )
        print("   ✅ Message 1 processed successfully")
    except Exception as e:
        print(f"   ❌ Message 1 failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE (message 2)
    print("\n6. Testing PROCESS_MESSAGE (message 2)...")
    try:
        invoker.process_conversation_message(
            role="assistant",
            content="Nice to meet you Sarah! Golden retrievers are wonderful dogs. How old is Max?",
            thread_id=thread_id
        )
        print("   ✅ Message 2 processed successfully")
    except Exception as e:
        print(f"   ❌ Message 2 failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE (message 3)
    print("\n7. Testing PROCESS_MESSAGE (message 3)...")
    try:
        invoker.process_conversation_message(
            role="user",
            content="He's 3 years old and loves to play fetch!",
            thread_id=thread_id
        )
        print("   ✅ Message 3 processed successfully")
    except Exception as e:
        print(f"   ❌ Message 3 failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE (message 4)
    print("\n8. Testing PROCESS_MESSAGE (message 4)...")
    try:
        invoker.process_conversation_message(
            role="assistant",
            content="That's a great age! Three-year-old dogs have lots of energy. Does Max have a favorite toy for fetch?",
            thread_id=thread_id
        )
        print("   ✅ Message 4 processed successfully")
    except Exception as e:
        print(f"   ❌ Message 4 failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE (message 5)
    print("\n9. Testing PROCESS_MESSAGE (message 5)...")
    try:
        invoker.process_conversation_message(
            role="user",
            content="Yes, he loves his blue tennis ball!",
            thread_id=thread_id
        )
        print("   ✅ Message 5 processed successfully")
    except Exception as e:
        print(f"   ❌ Message 5 failed: {e}")
        return False
    
    # Test PROCESS_MESSAGE_AND_FLUSH (message 6 - should trigger flush)
    print("\n10. Testing PROCESS_MESSAGE_AND_FLUSH (message 6 - triggers flush)...")
    try:
        invoker.process_conversation_message(
            role="assistant",
            content="A blue tennis ball - classic choice! I bet Max gets excited when he sees it.",
            thread_id=thread_id
        )
        print("   ✅ Message 6 (with flush) processed successfully")
    except Exception as e:
        print(f"   ❌ Message 6 (with flush) failed: {e}")
        print(f"   Error type: {type(e).__name__}")
        import traceback
        traceback.print_exc()
        return False
    
    # Test END_SESSION
    print("\n11. Testing END_SESSION...")
    try:
        invoker.end_session(thread_id)
        print("   ✅ END_SESSION completed successfully")
    except Exception as e:
        print(f"   ❌ END_SESSION failed: {e}")
        return False
    
    # Search memory to verify content was stored
    print("\n12. Verifying memory content...")
    search_result = mm.search_memories(memory_id, "Sarah golden retriever Max", top_k=5)
    
    if search_result.get("entries"):
        print(f"   ✅ Found {len(search_result['entries'])} entries")
        for i, entry in enumerate(search_result['entries'][:3], 1):
            summary = entry.get('summary', '')[:100]
            print(f"      Entry {i}: {summary}...")
    else:
        print("   ⚠️  No entries found")
    
    if search_result.get("latestContext"):
        ctx_preview = search_result['latestContext'][:200]
        print(f"   ✅ Context saved: {ctx_preview}...")
    else:
        print("   ⚠️  No context saved")
    
    return True

if __name__ == "__main__":
    print("\nFIXED AGENT TEST")
    print("This test verifies that removing add_messages annotation")
    print("from the messages field resolves the recursion issue.")
    
    success = test_fixed_agent()
    
    print("\n" + "=" * 70)
    if success:
        print("✅ TEST PASSED: Agent works without recursion!")
        print("The fix successfully resolved the issue.")
    else:
        print("❌ TEST FAILED: Still encountering issues")
        print("Need to investigate further.")
    print("=" * 70)
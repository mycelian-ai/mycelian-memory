#!/usr/bin/env python3
"""Test script to verify refactored agent and runner work correctly."""

import sys
import os
sys.path.insert(0, os.path.dirname(__file__))

from src.mycelian_memory_agent import build_agent
from src.memory_manager import MemoryManager

def test_agent_creation():
    """Test that agent can be created."""
    print("Testing agent creation...")
    agent = build_agent("gpt-4o-mini", max_tool_calls_per_turn=5, debug=True)
    print("✓ Agent created successfully")
    
    # Test that agent has only the essential methods
    required_methods = ['invoke_message', 'bind_memory']
    for method in required_methods:
        assert hasattr(agent, method), f"Missing method: {method}"
        print(f"✓ Method '{method}' exists")
    
    # Test that vault/memory methods and flush were removed
    removed_methods = ['ensure_vault', 'ensure_memory', 'search_memories', 'run_session', 'flush_context']
    for method in removed_methods:
        assert not hasattr(agent, method), f"Method '{method}' should have been removed"
        print(f"✓ Method '{method}' correctly removed")
    
    agent.close()
    print("✓ Agent closed successfully")

def test_memory_manager():
    """Test that MemoryManager handles vault/memory operations."""
    print("\nTesting MemoryManager...")
    # We can't fully test without MCP client, but we can check the class exists
    assert MemoryManager is not None
    print("✓ MemoryManager class exists")
    
    # Check required methods
    required_methods = ['ensure_vault', 'ensure_memory', 'search_memories']
    for method in required_methods:
        assert hasattr(MemoryManager, method), f"Missing method: {method}"
        print(f"✓ MemoryManager.{method} exists")

def test_runner_imports():
    """Test that runner can be imported."""
    print("\nTesting runner imports...")
    from src.runner import _validate_and_normalize_message, _with_control_messages
    print("✓ Runner helper functions imported successfully")
    
    # Test message validation
    msg = {"role": "user", "content": "Hello"}
    normalized = _validate_and_normalize_message(msg)
    assert normalized["type"] == "conversation"
    assert normalized["role"] == "user"
    assert normalized["content"] == "Hello"
    print("✓ Message validation works")
    
    # Test control message insertion
    msgs = [{"role": "user", "content": "Hi"}, {"role": "assistant", "content": "Hello"}]
    with_controls = _with_control_messages(msgs, every=6)
    assert with_controls[0]["type"] == "system"  # SESSION_START
    assert with_controls[-1]["type"] == "system"  # SESSION_END
    print(f"✓ Control messages inserted (total: {len(with_controls)} messages)")

if __name__ == "__main__":
    try:
        test_agent_creation()
        test_memory_manager()
        test_runner_imports()
        print("\n✅ All tests passed!")
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
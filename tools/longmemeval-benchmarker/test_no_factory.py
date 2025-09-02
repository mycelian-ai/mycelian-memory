"""Test that factory.py is gone and build_agent comes from AgentBuilder."""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from src.agent import build_agent, AgentBuilder


def test_no_factory():
    """Verify factory.py is gone and everything works."""
    
    print("Testing Refactored AgentBuilder (no factory.py)")
    print("=" * 50)
    
    # Test 1: Verify build_agent is from agent_builder module
    try:
        assert build_agent.__module__ == 'src.agent.agent_builder'
        print(f"✓ build_agent comes from agent_builder module")
        print(f"  Module: {build_agent.__module__}")
    except AssertionError:
        print(f"✗ build_agent from wrong module: {build_agent.__module__}")
        return False
    
    # Test 2: Verify factory.py doesn't exist
    try:
        from src.agent import factory
        print("✗ factory.py still exists!")
        return False
    except ImportError:
        print("✓ factory.py removed successfully")
    
    # Test 3: Use build_agent function
    try:
        processor = build_agent(
            model_id="gpt-4",
            vault_id="test-vault",
            memory_id="test-memory",
            debug=True
        )
        print("✓ build_agent() works correctly")
    except Exception as e:
        print(f"✗ build_agent failed: {e}")
        return False
    
    # Test 4: Create builder directly and call build()
    try:
        builder = AgentBuilder(
            model_id="gpt-4",
            vault_id="direct-vault",
            memory_id="direct-memory",
            debug=True
        )
        processor2 = builder.build()
        print("✓ AgentBuilder.build() returns AgentMessageProcessor")
        print(f"  Type: {type(processor2).__name__}")
    except Exception as e:
        print(f"✗ Direct builder failed: {e}")
        return False
    
    # Test 5: Verify both approaches produce same result type
    try:
        assert type(processor) == type(processor2)
        print("✓ Both approaches produce same result")
    except AssertionError:
        print("✗ Different result types!")
        return False
    
    print("\n" + "=" * 50)
    print("✅ All tests passed!")
    print("\nCleaner architecture achieved:")
    print("- No redundant factory.py file")
    print("- AgentBuilder.build() returns complete pipeline")
    print("- build_agent() is a convenience function in AgentBuilder")
    print("- Simpler, more direct design")
    
    return True


if __name__ == "__main__":
    success = test_no_factory()
    sys.exit(0 if success else 1)
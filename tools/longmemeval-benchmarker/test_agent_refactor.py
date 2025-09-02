"""Test the refactored Agent and AgentBuilder implementation."""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from src.agent.agent_builder import AgentBuilder
from src.agent.agent import Agent


def test_agent_refactor():
    """Test that AgentBuilder and Agent work correctly."""
    
    print("Testing Agent Refactor Implementation...")
    print("=" * 50)
    
    # Test 1: Create AgentBuilder
    try:
        builder = AgentBuilder(
            model_id="gpt-4",
            vault_id="test-vault-123",
            memory_id="test-memory-456",
            max_tool_calls_per_turn=5,
            debug=True
        )
        print("✓ AgentBuilder created successfully")
        print(f"  - Config: vault={builder.config['vault_id']}, memory={builder.config['memory_id']}")
    except Exception as e:
        print(f"✗ Failed to create AgentBuilder: {e}")
        return False
    
    # Test 2: Build Agent
    try:
        agent = builder.build()
        print("✓ Agent built successfully")
    except Exception as e:
        print(f"✗ Failed to build Agent: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    # Test 3: Verify Agent type and structure
    try:
        assert isinstance(agent, Agent), f"Wrong type: {type(agent)}"
        print("✓ Agent is correct type")
        
        # Check agent has required attributes
        assert hasattr(agent, 'llm'), "Missing LLM"
        assert hasattr(agent, 'tools'), "Missing tools"
        assert hasattr(agent, 'system_prompt'), "Missing system prompt"
        assert hasattr(agent, 'compiled_graph'), "Missing compiled graph"
        assert hasattr(agent, 'helpers'), "Missing helpers"
        print("✓ Agent has all required attributes")
        
        # Check helpers were created
        assert 'tool_logger' in agent.helpers, "Missing tool_logger"
        assert 'state_debugger' in agent.helpers, "Missing state_debugger"
        assert 'message_logger' in agent.helpers, "Missing message_logger"
        assert 'mcp_client' in agent.helpers, "Missing mcp_client"
        print("✓ All helper objects present")
        
    except AssertionError as e:
        print(f"✗ Agent validation failed: {e}")
        return False
    
    # Test 4: Verify graph compilation
    try:
        assert agent.compiled_graph is not None, "Graph not compiled"
        assert agent.graph is agent.compiled_graph, "Graph property mismatch"
        print("✓ Graph compiled successfully")
    except AssertionError as e:
        print(f"✗ Graph compilation issue: {e}")
        return False
    
    # Test 5: Verify we're reusing existing helper classes
    try:
        from src.agent.tool_logger import ToolLogger
        from src.agent.state_debugger import StateDebugger
        from src.agent.message_logger import MessageLogger
        from src.agent.prompt_builder import PromptBuilder
        
        assert isinstance(agent.helpers['tool_logger'], ToolLogger)
        assert isinstance(agent.helpers['state_debugger'], StateDebugger)
        assert isinstance(agent.helpers['message_logger'], MessageLogger)
        print("✓ Correctly reusing existing helper classes")
    except (ImportError, AssertionError) as e:
        print(f"✗ Helper class issue: {e}")
        return False
    
    # Test 6: Test cleanup
    try:
        agent.close()
        print("✓ Agent cleanup successful")
    except Exception as e:
        print(f"⚠ Cleanup warning: {e}")
    
    print("\n" + "=" * 50)
    print("✅ All tests passed!")
    print("\nSummary:")
    print("- AgentBuilder orchestrates infrastructure setup")
    print("- Agent owns graph topology (behavior definition)")
    print("- Successfully reusing existing helper classes")
    print("- Graph compilation working correctly")
    
    return True


if __name__ == "__main__":
    success = test_agent_refactor()
    sys.exit(0 if success else 1)
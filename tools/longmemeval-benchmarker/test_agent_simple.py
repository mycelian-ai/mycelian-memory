#!/usr/bin/env python3
"""Simple test of the agent without full benchmarker."""

import asyncio
from langchain.chat_models import init_chat_model
from src.mycelian_memory_agent.agent import MycelianMemoryAgent
from src.mycelian_memory_agent.control_state import ControlState
from langchain_core.messages import ChatMessage

# Simple mock tools for testing
def create_mock_tools():
    from langchain_core.tools import tool
    
    @tool
    def get_context(vault_id: str, memory_id: str) -> str:
        """Get context from memory."""
        return "Mock context content"
    
    @tool
    def list_entries(vault_id: str, memory_id: str) -> list:
        """List entries in memory."""
        return []
    
    @tool
    def add_entry(vault_id: str, memory_id: str, raw_entry: str, summary: str) -> dict:
        """Add an entry to memory."""
        return {"entry_id": "mock-entry-id"}
    
    @tool
    def await_consistency(memory_id: str) -> dict:
        """Await consistency."""
        return {"status": "consistent"}
    
    @tool
    def put_context(vault_id: str, memory_id: str, content: str) -> dict:
        """Update context."""
        return {"status": "updated"}
    
    return [get_context, list_entries, add_entry, await_consistency, put_context]

def main():
    # Initialize with mock tools
    llm = init_chat_model("openai:gpt-4o-mini")
    tools = create_mock_tools()
    prompts = {}
    
    agent = MycelianMemoryAgent(
        llm=llm,
        tools=tools,
        prompts=prompts,
        vault_id="test-vault",
        memory_id="test-memory"
    )
    
    # Test START_SESSION
    print("Testing START_SESSION...")
    try:
        result = agent.invoke(
            control=ControlState.START_SESSION,
            thread_id="test-thread-1"
        )
        print("✓ START_SESSION completed")
    except Exception as e:
        print(f"✗ START_SESSION failed: {e}")
        import traceback
        traceback.print_exc()
    
    # Test PROCESS_MESSAGE
    print("\nTesting PROCESS_MESSAGE...")
    try:
        msg = ChatMessage(role="user", content="Hello, this is a test message")
        result = agent.invoke(
            control=ControlState.PROCESS_MESSAGE,
            thread_id="test-thread-1",
            to_process=msg
        )
        print("✓ PROCESS_MESSAGE completed")
    except Exception as e:
        print(f"✗ PROCESS_MESSAGE failed: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()
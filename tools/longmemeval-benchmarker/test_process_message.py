#!/usr/bin/env python3
"""Test PROCESS_MESSAGE flow."""

from langchain.chat_models import init_chat_model
from src.mycelian_memory_agent.agent import MycelianMemoryAgent
from src.mycelian_memory_agent.control_state import ControlState
from langchain_core.messages import ChatMessage
from langchain_core.tools import tool

# Simple mock tools for testing
def create_mock_tools():
    @tool
    def get_context(vault_id: str, memory_id: str) -> str:
        """Get context from memory."""
        print(f"  [TOOL EXECUTED] get_context({vault_id}, {memory_id})")
        return "Mock context content"
    
    @tool
    def list_entries(vault_id: str, memory_id: str, limit: int = 10) -> list:
        """List entries in memory."""
        print(f"  [TOOL EXECUTED] list_entries({vault_id}, {memory_id}, limit={limit})")
        return ["entry1", "entry2"]
    
    @tool
    def add_entry(vault_id: str, memory_id: str, raw_entry: str, summary: str) -> dict:
        """Add an entry to memory."""
        print(f"  [TOOL EXECUTED] add_entry(...)")
        return {"entry_id": "mock-entry-id"}
    
    @tool
    def await_consistency(memory_id: str) -> dict:
        """Await consistency."""
        print(f"  [TOOL EXECUTED] await_consistency({memory_id})")
        return {"status": "consistent"}
    
    @tool
    def put_context(vault_id: str, memory_id: str, content: str) -> dict:
        """Update context."""
        print(f"  [TOOL EXECUTED] put_context(...)")
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
    
    thread_id = "test-thread-1"
    
    # First, run START_SESSION
    print("1. Running START_SESSION...")
    config = {"configurable": {"thread_id": thread_id}}
    initial_state = {
        "control": ControlState.START_SESSION,
        "to_process": [],
        "tool_history": [],
        "messages": []
    }
    result = agent.graph.invoke(initial_state, config)
    print("✓ START_SESSION completed\n")
    
    # Now test PROCESS_MESSAGE
    print("2. Testing PROCESS_MESSAGE...")
    msg = ChatMessage(role="user", content="Hello, this is a test message")
    
    config = {"configurable": {"thread_id": thread_id}, "recursion_limit": 10}
    initial_state = {
        "control": ControlState.PROCESS_MESSAGE,
        "to_process": [msg],
        "conversation_history": [msg],
        "tool_history": [],
        "messages": []
    }
    
    print("\nStreaming graph execution for PROCESS_MESSAGE:")
    try:
        for i, chunk in enumerate(agent.graph.stream(initial_state, config)):
            print(f"\nIteration {i}:")
            for node_name, node_output in chunk.items():
                print(f"  Node: {node_name}")
                if "messages" in node_output:
                    for msg in node_output["messages"]:
                        msg_type = type(msg).__name__
                        if hasattr(msg, 'tool_calls') and msg.tool_calls:
                            print(f"    -> {msg_type} with tool_calls: {[tc.get('name', tc) for tc in msg.tool_calls]}")
                        elif hasattr(msg, 'name'):
                            print(f"    -> {msg_type} from tool: {msg.name}")
                        else:
                            content = str(msg.content) if hasattr(msg, 'content') else str(msg)
                            print(f"    -> {msg_type}: {content[:100]}")
                if "tool_history" in node_output:
                    print(f"    -> tool_history updated: {len(node_output['tool_history'])} items")
                    
            if i > 10:  # Safety limit
                print("\nStopping - too many iterations")
                break
                
    except Exception as e:
        print(f"\n✗ Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()
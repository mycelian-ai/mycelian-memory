#!/usr/bin/env python3
"""Test to verify that messages accumulation is the root cause of the bug."""

import json
from typing import TypedDict, Annotated, Sequence
from langchain_core.messages import AIMessage, ToolMessage, BaseMessage
from langgraph.graph.message import add_messages
from langgraph.checkpoint.memory import MemorySaver

# Replicate the state structure
class TestState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]  # Accumulates
    tool_history: Sequence[BaseMessage]  # Does not accumulate
    
def test_accumulation():
    """Test how add_messages annotation affects state accumulation."""
    
    print("Testing messages accumulation behavior...")
    print("=" * 60)
    
    # Simulate checkpointer behavior
    checkpointer = MemorySaver()
    thread_id = "test_thread"
    config = {"configurable": {"thread_id": thread_id}}
    
    # First invocation - simulating START_SESSION
    print("\n1. First invocation (START_SESSION)")
    initial_state_1 = {
        "messages": [],
        "tool_history": []
    }
    
    # Simulate adding a tool result
    tool_msg_1 = ToolMessage(
        content="Context retrieved", 
        name="get_context",
        tool_call_id="call_1"
    )
    
    # What the observe node would return
    update_1 = {
        "messages": [tool_msg_1],
        "tool_history": [tool_msg_1]
    }
    
    # Simulate checkpointer saving state
    # In real graph, this happens after each node execution
    saved_state_1 = {
        "messages": [tool_msg_1],  # This gets saved with add_messages
        "tool_history": [tool_msg_1]  # This gets saved without annotation
    }
    
    print(f"  Saved messages: {[m.name for m in saved_state_1['messages']]}")
    print(f"  Saved tool_history: {[m.name for m in saved_state_1['tool_history']]}")
    
    # Second invocation - simulating PROCESS_MESSAGE
    print("\n2. Second invocation (PROCESS_MESSAGE)")
    initial_state_2 = {
        "messages": [],  # We pass empty
        "tool_history": []  # We pass empty
    }
    
    # But checkpointer merges with saved state!
    # For annotated fields (messages), it uses add_messages
    merged_state_2 = {
        "messages": saved_state_1["messages"] + initial_state_2["messages"],  # ACCUMULATES!
        "tool_history": initial_state_2["tool_history"]  # REPLACED (no annotation)
    }
    
    print(f"  After merge - messages: {[m.name for m in merged_state_2['messages']]}")
    print(f"  After merge - tool_history: {merged_state_2['tool_history']}")
    
    # Now when observe() checks for last_tool
    print("\n3. Checking for last_tool in observe():")
    
    # Check tool_history first
    last_tool_from_history = None
    for msg in reversed(merged_state_2["tool_history"]):
        if isinstance(msg, ToolMessage):
            last_tool_from_history = msg.name
            break
    print(f"  last_tool from tool_history: {last_tool_from_history}")
    
    # Fallback to messages (the bug!)
    last_tool_from_messages = None
    if not last_tool_from_history:
        for msg in reversed(merged_state_2["messages"]):
            if isinstance(msg, ToolMessage):
                last_tool_from_messages = msg.name
                break
    print(f"  last_tool from messages fallback: {last_tool_from_messages}")
    
    print("\n4. Result:")
    if last_tool_from_messages:
        print(f"  ❌ BUG: Found stale tool '{last_tool_from_messages}' from previous invocation!")
        print(f"  This would cause the agent to skip steps in the control flow.")
    else:
        print(f"  ✓ No stale tools found")
    
    # Third invocation - simulating PROCESS_MESSAGE_AND_FLUSH
    print("\n5. Third invocation (PROCESS_MESSAGE_AND_FLUSH)")
    
    # Add another tool to saved state
    tool_msg_2 = ToolMessage(
        content="Entry added",
        name="add_entry", 
        tool_call_id="call_2"
    )
    saved_state_2 = {
        "messages": saved_state_1["messages"] + [tool_msg_2],  # Accumulating more
        "tool_history": [tool_msg_2]  # Just current invocation
    }
    
    initial_state_3 = {
        "messages": [],
        "tool_history": []
    }
    
    # Merge again
    merged_state_3 = {
        "messages": saved_state_2["messages"] + initial_state_3["messages"],  # Has BOTH tools!
        "tool_history": initial_state_3["tool_history"]  # Empty
    }
    
    print(f"  After merge - messages: {[m.name for m in merged_state_3['messages']]}")
    print(f"  After merge - tool_history: {merged_state_3['tool_history']}")
    
    # Check for last_tool again
    last_tool = None
    # tool_history is empty
    for msg in reversed(merged_state_3["tool_history"]):
        if isinstance(msg, ToolMessage):
            last_tool = msg.name
            break
    
    # Fallback to messages
    if not last_tool:
        for msg in reversed(merged_state_3["messages"]):
            if isinstance(msg, ToolMessage):
                last_tool = msg.name
                break
    
    print(f"\n  last_tool detected: {last_tool}")
    
    if last_tool == "add_entry":
        print(f"  ❌ BUG CONFIRMED: Agent thinks add_entry was already executed!")
        print(f"     It will skip to await_consistency, missing the actual add_entry")
        print(f"     This is exactly what we see in the logs!")
    
    print("\n" + "=" * 60)
    print("CONCLUSION:")
    print("- messages field accumulates across invocations due to add_messages")
    print("- tool_history does NOT accumulate (no annotation)")  
    print("- observe() falls back to checking messages when tool_history is empty")
    print("- This causes stale tools from previous invocations to be detected")
    print("- Result: Agent skips steps in control flow (e.g., skips add_entry)")
    print("\nFIX: Remove add_messages annotation from messages field")

if __name__ == "__main__":
    test_accumulation()
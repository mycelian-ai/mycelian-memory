#!/usr/bin/env python3
"""Debug test to understand the agent's behavior."""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from langchain_core.messages import SystemMessage, ChatMessage
from langgraph.graph import StateGraph, MessagesState, END, START
from langgraph.checkpoint.memory import InMemorySaver
from langchain.chat_models import init_chat_model
import json

def debug_agent():
    """Debug the agent's behavior step by step."""
    
    print("Debug Agent Behavior")
    print("=" * 50)
    
    # Create minimal graph like the agent does
    graph = StateGraph(MessagesState)
    llm = init_chat_model("openai:gpt-4o-mini")
    
    # Simple LLM node that just echoes for control messages
    def llm_node(state):
        messages = state["messages"]
        print(f"\nLLM Node called with {len(messages)} messages")
        
        # Print each message
        for i, msg in enumerate(messages):
            msg_type = type(msg).__name__
            if hasattr(msg, 'message_type'):
                msg_type += f"(type={msg.message_type})"
            content = str(msg.content)[:50] + "..." if len(str(msg.content)) > 50 else str(msg.content)
            print(f"  [{i}] {msg_type}: {content}")
        
        # Check the last message
        if messages:
            last_msg = messages[-1]
            print(f"\nProcessing last message: {type(last_msg).__name__}")
            
            # For control messages, don't call tools
            if isinstance(last_msg, SystemMessage) and getattr(last_msg, 'message_type', None) == 'control':
                print("  -> Control message detected, returning empty response")
                return {"messages": []}
            
            # For conversation messages, pretend to call add_entry
            if isinstance(last_msg, ChatMessage):
                print("  -> ChatMessage detected, would call add_entry")
                return {"messages": []}
        
        return {"messages": []}
    
    # Simple routing - never continue to tools for testing
    def should_continue(state):
        print("Should continue called - returning END")
        return END
    
    graph.add_node("llm", llm_node)
    graph.add_edge(START, "llm")
    graph.add_conditional_edges("llm", should_continue, {END: END})
    
    checkpointer = InMemorySaver()
    compiled = graph.compile(checkpointer=checkpointer)
    
    thread_id = "test-thread"
    config = {"configurable": {"thread_id": thread_id}}
    
    # Test 1: SESSION_START
    print("\n1. Testing SESSION_START...")
    msg = SystemMessage(content="SESSION_START", message_type="control")
    try:
        result = compiled.invoke({"messages": [msg]}, config)
        print("  ✓ SESSION_START completed")
    except Exception as e:
        print(f"  ✗ Error: {e}")
        return
    
    # Test 2: User message
    print("\n2. Testing user message...")
    msg = ChatMessage(role="user", content="Hello", idx=1)
    try:
        result = compiled.invoke({"messages": [msg]}, config)
        print("  ✓ User message completed")
    except Exception as e:
        print(f"  ✗ Error: {e}")
        return
    
    # Check accumulated state
    print("\n3. Checking accumulated state...")
    state = compiled.get_state(config)
    print(f"  State has {len(state.values['messages'])} messages")
    
    print("\n✓ All tests passed - no infinite loop!")

if __name__ == "__main__":
    debug_agent()
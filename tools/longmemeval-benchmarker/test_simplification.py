#!/usr/bin/env python3
"""Test potential simplifications while keeping async execution."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from langchain_core.messages import AIMessage, ToolMessage, BaseMessage
from langgraph.prebuilt import ToolNode
from typing import TypedDict, Annotated, Sequence
from langgraph.graph import StateGraph, END
from langgraph.graph.message import add_messages
from langchain.chat_models import init_chat_model

# Define minimal state
class SimpleState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]

async def test_simplified_agent():
    """Test a simplified agent design."""
    print("=" * 60)
    print("Testing Simplified Agent Design")
    print("=" * 60)
    
    # 1. Get MCP tools
    mcp_client = create_mcp_client()
    tools = await mcp_client.get_tools()
    print(f"Got {len(tools)} MCP tools")
    
    # 2. Create LLM with tools bound
    llm = init_chat_model("openai:gpt-4o-mini")
    llm_with_tools = llm.bind_tools(tools)
    
    # 3. Build simplified graph
    workflow = StateGraph(SimpleState)
    
    # Simple agent function - just calls LLM
    async def call_llm(state):
        response = await llm_with_tools.ainvoke(state["messages"])
        return {"messages": [response]}
    
    # Add nodes
    workflow.add_node("agent", call_llm)
    workflow.add_node("tools", ToolNode(tools))
    
    # Simple routing - if tool calls, go to tools, else end
    def should_continue(state):
        messages = state["messages"]
        last_message = messages[-1]
        if hasattr(last_message, "tool_calls") and last_message.tool_calls:
            return "tools"
        return "end"
    
    # Set up edges
    workflow.set_entry_point("agent")
    workflow.add_conditional_edges("agent", should_continue, {"tools": "tools", "end": END})
    workflow.add_edge("tools", "agent")  # After tools, go back to agent
    
    # Compile
    graph = workflow.compile()
    
    # Test 1: Simple tool calling
    print("\n1. Test simple tool call (list_vaults):")
    result = await graph.ainvoke({
        "messages": [AIMessage(content="List all vaults")]
    })
    
    print(f"   Messages in result: {len(result['messages'])}")
    for msg in result["messages"]:
        if isinstance(msg, AIMessage) and msg.tool_calls:
            print(f"   AI called tools: {[tc['name'] for tc in msg.tool_calls]}")
        elif isinstance(msg, ToolMessage):
            print(f"   Tool result preview: {msg.content[:100]}...")
        elif isinstance(msg, AIMessage):
            print(f"   AI response: {msg.content[:200]}...")
    
    # Test 2: Direct tool invocation (bypassing LLM)
    print("\n2. Test direct tool invocation:")
    result = await graph.ainvoke({
        "messages": [
            AIMessage(
                content="Getting context",
                tool_calls=[{
                    "id": "test_1",
                    "name": "get_context",
                    "args": {"vault_id": "test", "memory_id": "test"}
                }]
            )
        ]
    }, {"recursion_limit": 2})  # Limit recursion for testing
    
    print(f"   Messages: {len(result['messages'])}")
    for msg in result["messages"]:
        if isinstance(msg, ToolMessage):
            if "Error" in msg.content:
                print(f"   Tool error: {msg.content[:100]}")
            else:
                print(f"   Tool success: {msg.content[:100]}")

async def test_invoker_simplification():
    """Test if we can simplify the invoker pattern."""
    print("\n" + "=" * 60)
    print("Testing Invoker Simplification")
    print("=" * 60)
    
    # Instead of complex invoker, just use the graph directly
    mcp_client = create_mcp_client()
    tools = await mcp_client.get_tools()
    
    llm = init_chat_model("openai:gpt-4o-mini")
    llm_with_tools = llm.bind_tools(tools)
    
    # Minimal graph
    workflow = StateGraph(SimpleState)
    
    async def agent_node(state):
        # Get last message to understand context
        messages = state["messages"]
        # Call LLM
        response = await llm_with_tools.ainvoke(messages)
        return {"messages": [response]}
    
    workflow.add_node("agent", agent_node)
    workflow.add_node("tools", ToolNode(tools))
    
    def router(state):
        last_msg = state["messages"][-1]
        if hasattr(last_msg, "tool_calls") and last_msg.tool_calls:
            return "tools"
        return "end"
    
    workflow.set_entry_point("agent")
    workflow.add_conditional_edges("agent", router, {"tools": "tools", "end": END})
    workflow.add_edge("tools", "agent")
    
    graph = workflow.compile()
    
    # Simple usage - no invoker needed
    print("\n1. Direct graph usage (no invoker):")
    
    # Start session equivalent - just pass empty messages
    state = {"messages": []}
    
    # Process message equivalent - just add to messages
    user_msg = AIMessage(content="Hello, list my vaults")
    state = await graph.ainvoke({"messages": state["messages"] + [user_msg]})
    
    print(f"   Final state has {len(state['messages'])} messages")
    
    # The state itself maintains conversation history
    print(f"   Conversation can continue by passing state back")

if __name__ == "__main__":
    asyncio.run(test_simplified_agent())
    asyncio.run(test_invoker_simplification())
#!/usr/bin/env python3
"""Test if ToolNode can handle async tools in sync graph execution."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from langchain_core.messages import AIMessage, ToolMessage
from langchain_core.tools import StructuredTool
from langgraph.prebuilt import ToolNode
from typing import TypedDict, Annotated, Sequence
from langgraph.graph import StateGraph, END
from langgraph.graph.message import add_messages
from langchain_core.messages import BaseMessage

# Create an async-only tool (simulating MCP tools)
async def async_tool_func(text: str) -> str:
    """An async-only tool function."""
    print(f"Async tool called with: {text}")
    await asyncio.sleep(0.1)  # Simulate async work
    return f"Processed: {text}"

# Create StructuredTool with only coroutine (no sync func)
async_tool = StructuredTool.from_function(
    coroutine=async_tool_func,  # Only async, no func
    name="process_text",
    description="Process text asynchronously"
)

# Define state for graph
class GraphState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]

def test_sync_execution_with_async_tools():
    """Test synchronous graph execution with async tools."""
    print("=" * 60)
    print("Testing SYNC graph.invoke() with ASYNC-only tools")
    print("=" * 60)
    
    # Create ToolNode with async tool
    tool_node = ToolNode([async_tool])
    
    # Build a simple graph
    workflow = StateGraph(GraphState)
    
    # Add nodes
    workflow.add_node("tools", tool_node)
    
    # Set entry point and edges
    workflow.set_entry_point("tools")
    workflow.add_edge("tools", END)
    
    # Compile graph
    graph = workflow.compile()
    
    # Create initial state with tool call
    initial_state = {
        "messages": [
            AIMessage(
                content="Processing text",
                tool_calls=[{
                    "id": "call_1",
                    "name": "process_text",
                    "args": {"text": "Hello from sync graph!"}
                }]
            )
        ]
    }
    
    print("\n1. Testing graph.invoke() (SYNC) with async tool:")
    try:
        # Try synchronous execution
        result = graph.invoke(initial_state)
        print("✓ Success with sync invoke!")
        
        # Check result
        for msg in result["messages"]:
            if isinstance(msg, ToolMessage):
                print(f"   Tool result: {msg.content}")
                
    except NotImplementedError as e:
        print(f"✗ NotImplementedError: {e}")
    except Exception as e:
        print(f"✗ Error: {e}")
        import traceback
        traceback.print_exc()
    
    print("\n2. Testing graph.ainvoke() (ASYNC) with async tool:")
    async def test_async():
        try:
            result = await graph.ainvoke(initial_state)
            print("✓ Success with async invoke!")
            
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"   Tool result: {msg.content}")
        except Exception as e:
            print(f"✗ Error: {e}")
    
    asyncio.run(test_async())
    
    # Test with MCP tools directly
    print("\n3. Testing with actual MCP tools:")
    from src.mycelian_memory_agent.mcp_utils import create_mcp_client
    
    async def test_mcp_sync():
        mcp_client = create_mcp_client()
        tools = await mcp_client.get_tools()
        
        # Find list_vaults tool
        list_vaults = None
        for tool in tools:
            if tool.name == "list_vaults":
                list_vaults = tool
                break
        
        if not list_vaults:
            print("list_vaults tool not found")
            return
            
        print(f"Found MCP tool: {list_vaults.name}")
        print(f"Has func: {hasattr(list_vaults, 'func') and list_vaults.func is not None}")
        print(f"Has coroutine: {hasattr(list_vaults, 'coroutine') and list_vaults.coroutine is not None}")
        
        # Create ToolNode with MCP tool
        tool_node = ToolNode([list_vaults])
        
        # Build graph
        workflow = StateGraph(GraphState)
        workflow.add_node("tools", tool_node)
        workflow.set_entry_point("tools")
        workflow.add_edge("tools", END)
        graph = workflow.compile()
        
        # Test state
        test_state = {
            "messages": [
                AIMessage(
                    content="Listing vaults",
                    tool_calls=[{
                        "id": "call_2",
                        "name": "list_vaults",
                        "args": {}
                    }]
                )
            ]
        }
        
        print("\n   Testing graph.invoke() with MCP tool:")
        try:
            result = graph.invoke(test_state)
            print("   ✓ Sync invoke worked with MCP tool!")
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"      Found vaults: {msg.content[:100]}...")
        except NotImplementedError as e:
            print(f"   ✗ NotImplementedError: {e}")
        except Exception as e:
            print(f"   ✗ Error: {e}")
    
    asyncio.run(test_mcp_sync())

if __name__ == "__main__":
    test_sync_execution_with_async_tools()
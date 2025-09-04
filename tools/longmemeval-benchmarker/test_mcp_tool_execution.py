#!/usr/bin/env python3
"""Test MCP tool execution with LangGraph ToolNode."""

import asyncio
import json
from typing import Any, Dict
from langchain_core.messages import AIMessage, ToolMessage
from langchain_core.tools import StructuredTool
from langgraph.prebuilt import ToolNode

# First, let's test with a simple sync tool
def sync_add(a: int, b: int) -> int:
    """Add two numbers."""
    print(f"SYNC: Adding {a} + {b}")
    return a + b

# Then an async tool
async def async_add(a: int, b: int) -> int:
    """Add two numbers asynchronously."""
    print(f"ASYNC: Adding {a} + {b}")
    await asyncio.sleep(0.1)  # Simulate async work
    return a + b

# Create a wrapper that makes async tools sync-compatible
def create_sync_wrapper(async_tool):
    """Create a sync-compatible wrapper for an async tool."""
    
    def sync_invoke(**kwargs) -> Any:
        """Synchronous wrapper function."""
        print(f"Wrapper invoking async tool: {async_tool.name} with args: {kwargs}")
        return asyncio.run(async_tool.ainvoke(kwargs))
    
    # Create a new StructuredTool with the sync function
    return StructuredTool.from_function(
        func=sync_invoke,
        name=async_tool.name,
        description=async_tool.description
    )

def test_tool_execution():
    print("=" * 60)
    print("Testing Tool Execution with ToolNode")
    print("=" * 60)
    
    # Create tools
    sync_tool = StructuredTool.from_function(
        func=sync_add,
        name="sync_add",
        description="Add two numbers synchronously"
    )
    
    # Create async tool (this will fail with ToolNode)
    async_tool_raw = StructuredTool.from_function(
        coroutine=async_add,
        name="async_add",
        description="Add two numbers asynchronously"
    )
    
    # Test 1: Sync tool with ToolNode
    print("\n1. Testing SYNC tool with ToolNode:")
    tool_node = ToolNode([sync_tool])
    
    # Create an AIMessage with tool call
    ai_msg = AIMessage(
        content="Let me add those numbers",
        tool_calls=[{
            "id": "call_1",
            "name": "sync_add",
            "args": {"a": 5, "b": 3}
        }]
    )
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success! Result: {result}")
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"  Tool result: {msg.content}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    
    # Test 2: Raw async tool with ToolNode (will fail)
    print("\n2. Testing RAW ASYNC tool with ToolNode:")
    tool_node = ToolNode([async_tool_raw])
    
    ai_msg = AIMessage(
        content="Let me add those numbers",
        tool_calls=[{
            "id": "call_2",
            "name": "async_add",
            "args": {"a": 7, "b": 4}
        }]
    )
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success! Result: {result}")
    except NotImplementedError as e:
        print(f"✗ Expected failure: {e}")
    except Exception as e:
        print(f"✗ Unexpected error: {e}")
    
    # Test 3: Wrapped async tool with ToolNode
    print("\n3. Testing WRAPPED ASYNC tool with ToolNode:")
    wrapped_tool = create_sync_wrapper(async_tool_raw)
    tool_node = ToolNode([wrapped_tool])
    
    ai_msg = AIMessage(
        content="Let me add those numbers",
        tool_calls=[{
            "id": "call_3",
            "name": "async_add",
            "args": {"a": 10, "b": 5}
        }]
    )
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success! Result: {result}")
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"  Tool result: {msg.content}")
    except Exception as e:
        print(f"✗ Failed: {e}")

def test_mcp_tool_simulation():
    """Test with a simulated MCP tool structure."""
    print("\n" + "=" * 60)
    print("Testing Simulated MCP Tool")
    print("=" * 60)
    
    # Simulate an MCP tool that only has ainvoke (like the real ones)
    class MockMCPTool:
        def __init__(self):
            self.name = "add_entry"
            self.description = "Add an entry to memory"
            self.args_schema = None  # MCP tools might not have this
            
        async def ainvoke(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
            """Async-only invoke like MCP tools have."""
            print(f"MCP Tool: Adding entry with data: {input_data}")
            await asyncio.sleep(0.1)
            return {"status": "success", "entry_id": "test-123"}
        
        # Note: No invoke method!
    
    # Test without wrapper (will fail)
    print("\n1. Raw MCP tool (will fail):")
    mcp_tool = MockMCPTool()
    tool_node = ToolNode([mcp_tool])
    
    ai_msg = AIMessage(
        content="Adding entry",
        tool_calls=[{
            "id": "call_4",
            "name": "add_entry",
            "args": {"content": "test entry"}
        }]
    )
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success: {result}")
    except AttributeError as e:
        print(f"✗ Expected failure (no invoke method): {e}")
    except Exception as e:
        print(f"✗ Error: {e}")
    
    # Test with wrapper - need to convert to StructuredTool first
    print("\n2. Wrapped MCP tool:")
    
    # First convert the MCP tool to a StructuredTool
    async def mcp_tool_func(**kwargs):
        """Wrapper for MCP tool."""
        return await mcp_tool.ainvoke(kwargs)
    
    # Create async StructuredTool
    async_structured = StructuredTool.from_function(
        coroutine=mcp_tool_func,
        name=mcp_tool.name,
        description=mcp_tool.description
    )
    
    # Then wrap it for sync
    wrapped_mcp = create_sync_wrapper(async_structured)
    tool_node = ToolNode([wrapped_mcp])
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success: {result}")
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"  Tool result: {msg.content}")
    except Exception as e:
        print(f"✗ Failed: {e}")

if __name__ == "__main__":
    test_tool_execution()
    test_mcp_tool_simulation()
    
    print("\n" + "=" * 60)
    print("CONCLUSION:")
    print("- Sync tools work directly with ToolNode")
    print("- Async tools need a wrapper to add invoke() method")
    print("- MCP tools are async-only and need wrapping")
    print("=" * 60)
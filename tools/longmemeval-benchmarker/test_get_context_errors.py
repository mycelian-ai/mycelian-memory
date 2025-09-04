#!/usr/bin/env python3
"""Test get_context tool error handling when required arguments are missing."""

import asyncio
import json
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from langchain_core.messages import AIMessage, ToolMessage
from langgraph.prebuilt import ToolNode

async def test_get_context_errors():
    """Test get_context tool error handling with missing arguments."""
    print("=" * 60)
    print("Testing get_context Error Handling")
    print("=" * 60)
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    # Get tools
    tools = await mcp_client.get_tools()
    print(f"\nFound {len(tools)} MCP tools")
    
    # Find get_context tool which requires vault_id and memory_id
    get_context_tool = None
    for tool in tools:
        if tool.name == "get_context":
            get_context_tool = tool
            break
    
    if not get_context_tool:
        print("ERROR: get_context tool not found!")
        return
    
    print(f"\nFound tool: {get_context_tool.name}")
    print(f"Description: {get_context_tool.description}")
    
    # Create ToolNode with the tool
    tool_node = ToolNode([get_context_tool])
    
    # Test 1: Call with no arguments at all
    print("\n1. Testing with no arguments:")
    ai_msg = AIMessage(
        content="Getting context with no args",
        tool_calls=[{
            "id": "call_no_args",
            "name": "get_context",
            "args": {}
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (no args):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content[:500]}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")
    
    # Test 2: Call with only vault_id (missing memory_id)
    print("\n2. Testing with missing memory_id:")
    ai_msg = AIMessage(
        content="Getting context with only vault_id",
        tool_calls=[{
            "id": "call_missing_memory",
            "name": "get_context",
            "args": {
                "vault_id": "test-vault-id"
                # Missing: memory_id
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (missing memory_id):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content[:500]}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")
    
    # Test 3: Call with only memory_id (missing vault_id)
    print("\n3. Testing with missing vault_id:")
    ai_msg = AIMessage(
        content="Getting context with only memory_id",
        tool_calls=[{
            "id": "call_missing_vault",
            "name": "get_context",
            "args": {
                "memory_id": "test-memory-id"
                # Missing: vault_id
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (missing vault_id):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content[:500]}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")
    
    # Test 4: Call with invalid IDs (non-existent)
    print("\n4. Testing with invalid IDs:")
    ai_msg = AIMessage(
        content="Getting context with invalid IDs",
        tool_calls=[{
            "id": "call_invalid_ids",
            "name": "get_context",
            "args": {
                "vault_id": "non-existent-vault",
                "memory_id": "non-existent-memory"
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (invalid IDs):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content[:500]}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")

if __name__ == "__main__":
    asyncio.run(test_get_context_errors())
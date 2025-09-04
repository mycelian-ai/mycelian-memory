#!/usr/bin/env python3
"""Test MCP tool error handling when required arguments are missing."""

import asyncio
import json
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from langchain_core.messages import AIMessage, ToolMessage
from langgraph.prebuilt import ToolNode

async def test_mcp_error_handling():
    """Test MCP tools error handling with missing arguments."""
    print("=" * 60)
    print("Testing MCP Tool Error Handling")
    print("=" * 60)
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    # Get tools
    tools = await mcp_client.get_tools()
    print(f"\nFound {len(tools)} MCP tools")
    
    # Find add_entry tool which requires vault_id, memory_id, raw_entry, summary
    add_entry_tool = None
    for tool in tools:
        if tool.name == "add_entry":
            add_entry_tool = tool
            break
    
    if not add_entry_tool:
        print("ERROR: add_entry tool not found!")
        return
    
    print(f"\nFound tool: {add_entry_tool.name}")
    print(f"Description: {add_entry_tool.description}")
    
    # Create ToolNode with the tool
    tool_node = ToolNode([add_entry_tool])
    
    # Test 1: Call with missing required arguments (missing memory_id, raw_entry, summary)
    print("\n1. Testing with missing required arguments:")
    ai_msg = AIMessage(
        content="Adding entry with missing args",
        tool_calls=[{
            "id": "call_missing_args",
            "name": "add_entry",
            "args": {
                "vault_id": "test-vault-id"
                # Missing: memory_id, raw_entry, summary
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (missing args):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")
    
    # Test 2: Call with invalid argument types
    print("\n2. Testing with invalid argument types:")
    ai_msg = AIMessage(
        content="Adding entry with invalid types",
        tool_calls=[{
            "id": "call_invalid_types",
            "name": "add_entry",
            "args": {
                "vault_id": 12345,  # Should be string
                "memory_id": True,  # Should be string
                "raw_entry": None,  # Should be string
                "summary": []  # Should be string
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (invalid types):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content}")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")
    
    # Test 3: Call with empty required fields
    print("\n3. Testing with empty required fields:")
    ai_msg = AIMessage(
        content="Adding entry with empty fields",
        tool_calls=[{
            "id": "call_empty_fields",
            "name": "add_entry",
            "args": {
                "vault_id": "",
                "memory_id": "",
                "raw_entry": "",
                "summary": ""
            }
        }]
    )
    
    try:
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print("Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response (empty fields):")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    print(f"  Content: {msg.content[:200]}...")
                    print(f"  Is Error: {'error' in msg.content.lower()}")
    except Exception as e:
        print(f"✗ Exception raised: {e}")

if __name__ == "__main__":
    asyncio.run(test_mcp_error_handling())
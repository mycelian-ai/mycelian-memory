#!/usr/bin/env python3
"""Test MCP tools with async execution - the natural way."""

import asyncio
import json
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from langchain_core.messages import AIMessage, ToolMessage
from langgraph.prebuilt import ToolNode

async def test_mcp_async_execution():
    """Test MCP tools with async execution."""
    print("=" * 60)
    print("Testing MCP Tools with Async Execution")
    print("=" * 60)
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    # Get tools
    tools = await mcp_client.get_tools()
    print(f"\nFound {len(tools)} MCP tools")
    
    # Find list_vaults tool
    list_vaults_tool = None
    for tool in tools:
        if tool.name == "list_vaults":
            list_vaults_tool = tool
            break
    
    if not list_vaults_tool:
        print("ERROR: list_vaults tool not found!")
        return
    
    print(f"\nFound tool: {list_vaults_tool.name}")
    print(f"Description: {list_vaults_tool.description}")
    
    # Create ToolNode with the async MCP tool directly (no wrapper)
    tool_node = ToolNode([list_vaults_tool])
    
    # Create an AIMessage requesting list_vaults
    ai_msg = AIMessage(
        content="Let me list the vaults",
        tool_calls=[{
            "id": "call_test_1",
            "name": "list_vaults",
            "args": {}
        }]
    )
    
    print("\nCalling ToolNode with ainvoke (async)...")
    try:
        # Use ainvoke for async execution - this should work with async tools
        result = await tool_node.ainvoke({"messages": [ai_msg]})
        print(f"✓ Success! Got result from async ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response:")
                    print(f"  Tool Call ID: {msg.tool_call_id}")
                    
                    # Parse and display the vaults
                    try:
                        data = json.loads(msg.content)
                        if isinstance(data, list):
                            print(f"\n  Found {len(data)} vaults:")
                            for vault in data[:5]:  # Show first 5
                                if isinstance(vault, dict):
                                    vault_id = vault.get('vaultId', 'unknown')
                                    title = vault.get('title', 'unknown')
                                    print(f"    - {title} (ID: {vault_id})")
                        else:
                            print(f"  Content: {msg.content}")
                    except json.JSONDecodeError:
                        print(f"  Raw content: {msg.content[:500]}")
    except Exception as e:
        print(f"✗ Failed: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_mcp_async_execution())
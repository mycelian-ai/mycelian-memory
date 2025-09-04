#!/usr/bin/env python3
"""Test that wrapped MCP tools actually work with real vault operations."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from src.mycelian_memory_agent.build import create_sync_wrapper
from langchain_core.messages import AIMessage, ToolMessage
from langgraph.prebuilt import ToolNode

async def test_wrapped_mcp_tools():
    """Test MCP tools with sync wrapper."""
    print("=" * 60)
    print("Testing Wrapped MCP Tools with Real Operations")
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
    
    print(f"\nFound list_vaults tool: {list_vaults_tool.name}")
    print(f"Description: {list_vaults_tool.description}")
    
    # Test 1: Try calling it directly (should fail)
    print("\n1. Testing direct invocation (expected to fail):")
    try:
        result = list_vaults_tool.invoke({})
        print(f"Unexpected success: {result}")
    except NotImplementedError as e:
        print(f"✓ Expected failure: {e}")
    
    # Test 2: Wrap it and try again
    print("\n2. Testing with sync wrapper:")
    wrapped_tool = create_sync_wrapper(list_vaults_tool)
    
    # Create ToolNode with wrapped tool
    tool_node = ToolNode([wrapped_tool])
    
    # Create an AIMessage requesting list_vaults
    ai_msg = AIMessage(
        content="Let me list the vaults",
        tool_calls=[{
            "id": "call_test_1",
            "name": "list_vaults",
            "args": {}
        }]
    )
    
    try:
        result = tool_node.invoke({"messages": [ai_msg]})
        print(f"✓ Success! Got result from ToolNode")
        
        if "messages" in result:
            for msg in result["messages"]:
                if isinstance(msg, ToolMessage):
                    print(f"\nTool Response:")
                    print(f"  Content: {msg.content}")
                    
                    # Try to parse the content
                    import json
                    try:
                        data = json.loads(msg.content)
                        if isinstance(data, list):
                            print(f"\nFound {len(data)} vaults:")
                            for vault in data[:3]:  # Show first 3
                                if isinstance(vault, dict):
                                    vault_id = vault.get('vaultId', 'unknown')
                                    title = vault.get('title', 'unknown')
                                    print(f"  - {title} (ID: {vault_id})")
                    except:
                        print(f"  Raw content: {msg.content[:200]}")
    except Exception as e:
        print(f"✗ Failed: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_wrapped_mcp_tools())
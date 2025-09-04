#!/usr/bin/env python3
"""Debug the build issue with async tools."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client

def test_build():
    """Test the build process."""
    print("Creating MCP client...")
    mcp_client = create_mcp_client()
    
    print(f"MCP client type: {type(mcp_client)}")
    print(f"get_tools attr: {hasattr(mcp_client, 'get_tools')}")
    
    # Check what get_tools returns
    result = mcp_client.get_tools()
    print(f"get_tools() returns: {type(result)}")
    print(f"Is coroutine: {asyncio.iscoroutine(result)}")
    
    # Try to await it
    async def _get_tools():
        return await mcp_client.get_tools()
    
    try:
        tools = asyncio.run(_get_tools())
        print(f"Successfully got {len(tools)} tools")
        if tools:
            print(f"First tool type: {type(tools[0])}")
    except Exception as e:
        print(f"Error getting tools: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    test_build()
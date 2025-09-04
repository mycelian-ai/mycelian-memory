#!/usr/bin/env python3
"""Examine the structure of MCP tools to understand how to make them work with ToolNode."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client

async def examine_mcp_tools():
    """Examine what MCP tools look like."""
    print("=" * 60)
    print("Examining MCP Tool Structure")
    print("=" * 60)
    
    # Create MCP client
    mcp_client = create_mcp_client()
    
    # Get tools
    tools = await mcp_client.get_tools()
    
    print(f"\nNumber of tools: {len(tools)}")
    
    # Examine first tool
    if tools:
        tool = tools[0]
        print(f"\nFirst tool: {tool.name if hasattr(tool, 'name') else 'NO NAME'}")
        print(f"Type: {type(tool)}")
        print(f"Module: {tool.__class__.__module__}")
        print(f"Class: {tool.__class__.__name__}")
        
        # Check what attributes it has
        print("\nAttributes:")
        for attr in dir(tool):
            if not attr.startswith('_'):
                try:
                    value = getattr(tool, attr)
                    if not callable(value):
                        print(f"  {attr}: {value!r}")
                    else:
                        print(f"  {attr}: <method>")
                except:
                    print(f"  {attr}: <error accessing>")
        
        # Check if it's a LangChain tool
        from langchain_core.tools import BaseTool, StructuredTool
        print(f"\nIs BaseTool? {isinstance(tool, BaseTool)}")
        print(f"Is StructuredTool? {isinstance(tool, StructuredTool)}")
        
        # Try to invoke it
        print("\n" + "=" * 40)
        print("Testing invocation:")
        
        # Check for invoke method
        if hasattr(tool, 'invoke'):
            print("Has invoke() method")
            try:
                # Try sync invoke (will likely fail)
                result = tool.invoke({"vault_id": "test", "memory_id": "test"})
                print(f"Sync invoke worked: {result}")
            except NotImplementedError as e:
                print(f"Sync invoke not implemented: {e}")
            except Exception as e:
                print(f"Sync invoke error: {e}")
        else:
            print("No invoke() method")
        
        # Check for ainvoke method
        if hasattr(tool, 'ainvoke'):
            print("Has ainvoke() method")
            try:
                # Try async invoke
                result = await tool.ainvoke({"vault_id": "test", "memory_id": "test"})
                print(f"Async invoke worked: {result}")
            except Exception as e:
                print(f"Async invoke error: {e}")
        else:
            print("No ainvoke() method")

if __name__ == "__main__":
    asyncio.run(examine_mcp_tools())
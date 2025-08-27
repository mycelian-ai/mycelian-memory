"""Debug single Memory Agent turn to see recursion pattern."""

import asyncio
import os
from src.agents.memory_agent import MycelianMemoryAgent

async def debug_single_turn():
    """Debug exactly what happens in a single Memory Agent turn."""
    
    # Enable LangChain debugging
    os.environ["LANGCHAIN_TRACING_V2"] = "true"
    os.environ["LANGCHAIN_ENDPOINT"] = "http://localhost:8000"  # Won't work but enables debug
    
    print("🔍 DEBUGGING SINGLE MEMORY AGENT TURN")
    print("=" * 60)
    
    # Use existing vault for consistency
    vault_id = "f9671811-669d-46d1-8ecc-9cdacb8fe191"  # tiny-test-vault
    
    # Create Memory Agent
    memory_agent = MycelianMemoryAgent(
        mcp_url="http://localhost:11546/mcp",
        model="anthropic.claude-3-haiku-20240307-v1:0",
        transport="streamable_http",
        aws_region="us-west-2"
    )
    
    await memory_agent.initialize()
    
    # Create test memory
    from langchain_mcp_adapters.client import MultiServerMCPClient
    import json
    
    mcp_client = MultiServerMCPClient({
        "mycelian": {
            "url": "http://localhost:11546/mcp",
            "transport": "streamable_http"
        }
    })
    
    tools = await mcp_client.get_tools()
    create_memory_tool = None
    for tool in tools:
        if tool.name == "create_memory_in_vault":
            create_memory_tool = tool
            break
    
    result = await create_memory_tool.ainvoke({
        "vault_id": vault_id,
        "title": "debug-single-turn",
        "memory_type": "CHAT",
        "description": "Debug single turn to find recursion issue"
    })
    
    if isinstance(result, str):
        result_dict = json.loads(result)
    else:
        result_dict = result
        
    memory_id = result_dict.get("memoryId")
    print(f"✅ Created debug memory: {memory_id}")
    
    try:
        print("\n🔄 CALLING observe_turn - WATCH FOR RECURSION")
        print("-" * 60)
        
        # Single turn that should be simple
        await memory_agent.observe_turn(
            role="user",
            content="Hello world, this is a test.",
            memory_id=memory_id,
            vault_id=vault_id
        )
        
        print("✅ observe_turn completed successfully!")
        
    except Exception as e:
        print(f"❌ observe_turn failed: {e}")
        
        if "recursion" in str(e).lower():
            print("\n🔍 RECURSION ISSUE CONFIRMED")
            print("The LangChain ReAct agent is calling tools in an endless loop")
            print("This is NOT the Memory Agent calling itself")
            print("This is the ReAct agent's internal tool-calling loop")
    
    finally:
        await memory_agent.close()

if __name__ == "__main__":
    asyncio.run(debug_single_turn())
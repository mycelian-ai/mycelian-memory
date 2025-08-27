"""Unit test for Memory Agent - test single message storage."""

import asyncio
from src.agents.memory_agent import MycelianMemoryAgent

async def test_memory_agent_single_turn():
    """Test if Memory Agent can store a single conversation turn."""
    
    # Use the existing vault and create a new memory for testing
    vault_id = "83039116-4348-48d7-b35e-f8ff3d4df7b8"  # longmemeval-test vault
    
    # Initialize Memory Agent
    memory_agent = MycelianMemoryAgent(
        mcp_url="http://localhost:11546/mcp",
        model="anthropic.claude-3-haiku-20240307-v1:0",
        transport="streamable_http",
        aws_region="us-west-2"
    )
    
    await memory_agent.initialize()
    
    # Create a test memory
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
    
    # Create test memory
    result = await create_memory_tool.ainvoke({
        "vault_id": vault_id,
        "title": "unit-test-memory",
        "memory_type": "CHAT",
        "description": "Unit test for Memory Agent single turn"
    })
    
    if isinstance(result, str):
        result_dict = json.loads(result)
    else:
        result_dict = result
        
    memory_id = result_dict.get("memoryId")
    print(f"✅ Created test memory: {memory_id}")
    
    try:
        # Initialize session
        print("🔄 Initializing memory session...")
        await memory_agent.initialize_session(
            conversation_id="unit_test_001",
            memory_id=memory_id,
            vault_id=vault_id
        )
        
        # Observe a single turn
        print("🔄 Observing single turn...")
        await memory_agent.observe_turn(
            role="user",
            content="Hello, this is a test message to check if memory agent stores data properly.",
            memory_id=memory_id,
            vault_id=vault_id
        )
        
        # Finalize session
        print("🔄 Finalizing session...")
        await memory_agent.finalize_session(
            conversation_id="unit_test_001", 
            memory_id=memory_id,
            vault_id=vault_id
        )
        
        print("✅ Memory Agent processing complete")
        
        # Now check if the data was stored
        print("\n📋 Checking stored data...")
        
        list_entries_tool = None
        get_context_tool = None
        
        for tool in tools:
            if tool.name == "list_entries":
                list_entries_tool = tool
            elif tool.name == "get_context":
                get_context_tool = tool
        
        # Check entries
        entries_result = await list_entries_tool.ainvoke({
            "vault_id": vault_id,
            "memory_id": memory_id,
            "limit": "5"
        })
        
        if isinstance(entries_result, str):
            entries_data = json.loads(entries_result)
        else:
            entries_data = entries_result
            
        entries = entries_data.get("entries", [])
        print(f"📊 Found {len(entries)} entries")
        
        for i, entry in enumerate(entries):
            print(f"\n--- Entry {i+1} ---")
            print(f"Summary: {entry.get('summary', 'No summary')}")
            print(f"Content: {entry.get('rawEntry', 'No content')[:200]}...")
        
        # Check context
        context_result = await get_context_tool.ainvoke({
            "vault_id": vault_id,
            "memory_id": memory_id
        })
        
        print(f"\n📄 Context length: {len(context_result)} chars")
        print(f"Context preview: {context_result[:300]}...")
        
        if len(entries) > 0:
            print("\n✅ SUCCESS: Memory Agent stored data properly!")
        else:
            print("\n❌ FAILURE: No entries found - Memory Agent not storing data")
            
    finally:
        await memory_agent.close()

if __name__ == "__main__":
    asyncio.run(test_memory_agent_single_turn())
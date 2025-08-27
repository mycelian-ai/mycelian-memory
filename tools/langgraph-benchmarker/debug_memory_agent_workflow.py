"""Debug Memory Agent parameters in workflow vs unit test context."""

import asyncio
import json
from src.workflows.end_to_end import create_longmemeval_workflow
from src.config import load_config_from_toml
from src.utils.dataset_loader import load_longmemeval_dataset

async def debug_workflow_memory_agent():
    """Compare Memory Agent parameters between workflow and unit test."""
    
    print("🔍 DEBUGGING MEMORY AGENT IN WORKFLOW CONTEXT")
    print("=" * 60)
    
    # Load config and dataset
    config = load_config_from_toml('test_tiny.toml')
    dataset = load_longmemeval_dataset(
        repo_path=config.longmemeval_repo_path,
        dataset_variant=config.dataset_variant,
        sample_size=1
    )
    
    conversation = dataset[0]
    print(f"📋 Processing conversation: {conversation['question_id']}")
    print(f"📋 Haystack sessions: {len(conversation['haystack_sessions'])}")
    print(f"📋 Total turns: {sum(len(session) for session in conversation['haystack_sessions'])}")
    
    # Create workflow
    workflow = await create_longmemeval_workflow(config)
    
    try:
        print("\n🔄 RUNNING WORKFLOW - WATCH FOR MEMORY AGENT CALLS")
        print("-" * 60)
        
        # Prepare initial state
        from src.types import LongMemEvalState
        import uuid
        run_id = str(uuid.uuid4())[:8]
        
        initial_state = LongMemEvalState(
            conversation_id=conversation["question_id"],
            haystack_sessions=conversation["haystack_sessions"],
            question=conversation["question"],
            expected_answer=conversation["answer"],
            memory_id="",  # Will be set by setup node
            vault_id="",   # Will be set by setup node
            run_id=run_id,
            ingestion_complete=False,
            qa_response="",
            evaluation_score=0.0,
            benchmark_results={}
        )
        
        # Execute workflow
        final_state = await workflow.workflow.ainvoke(initial_state)
        
        print(f"\n✅ WORKFLOW COMPLETED")
        print(f"📊 Memory ID used: {final_state.get('memory_id', 'UNKNOWN')}")
        print(f"📊 Vault ID used: {final_state.get('vault_id', 'UNKNOWN')}")
        print(f"📊 Ingestion complete: {final_state.get('ingestion_complete', False)}")
        
        # Check what was actually stored
        memory_id = final_state.get('memory_id')
        vault_id = final_state.get('vault_id')
        
        if memory_id and vault_id:
            print(f"\n🔍 CHECKING STORED DATA")
            print(f"Vault ID: {vault_id}")
            print(f"Memory ID: {memory_id}")
            
            # Use MCP tools to check entries
            from langchain_mcp_adapters.client import MultiServerMCPClient
            
            mcp_client = MultiServerMCPClient({
                "mycelian": {
                    "url": "http://localhost:11546/mcp", 
                    "transport": "streamable_http"
                }
            })
            
            tools = await mcp_client.get_tools()
            list_entries_tool = None
            get_context_tool = None
            
            for tool in tools:
                if tool.name == "list_entries":
                    list_entries_tool = tool
                elif tool.name == "get_context":
                    get_context_tool = tool
            
            if list_entries_tool:
                entries_result = await list_entries_tool.ainvoke({
                    "vault_id": vault_id,
                    "memory_id": memory_id,
                    "limit": "10"
                })
                
                if isinstance(entries_result, str):
                    entries_data = json.loads(entries_result)
                else:
                    entries_data = entries_result
                    
                entries = entries_data.get("entries", [])
                print(f"📊 Workflow stored {len(entries)} entries")
                
                for i, entry in enumerate(entries):
                    print(f"  Entry {i+1}: {entry.get('summary', 'No summary')}")
            
            if get_context_tool:
                context_result = await get_context_tool.ainvoke({
                    "vault_id": vault_id,
                    "memory_id": memory_id
                })
                
                context_length = len(context_result) if context_result else 0
                print(f"📊 Context length: {context_length} chars")
                if context_length > 100:
                    print(f"📄 Context preview: {context_result[:200]}...")
        
        print("\n🔍 COMPARISON WITH UNIT TEST:")
        print("- Unit test Memory Agent: ✅ Stores 1 entry + context")
        print(f"- Workflow Memory Agent: {'✅' if len(entries) > 0 else '❌'} Stores {len(entries)} entries")
        
        if len(entries) == 0:
            print("\n❌ ISSUE CONFIRMED: Memory Agent not persisting data in workflow")
            print("🔧 Need to investigate:")
            print("  1. Exact vault_id/memory_id parameters passed")
            print("  2. LangChain agent tool call success/failure") 
            print("  3. Message format differences")
        else:
            print("\n✅ SUCCESS: Memory Agent working in workflow!")
            
    finally:
        await workflow.close()

if __name__ == "__main__":
    asyncio.run(debug_workflow_memory_agent())
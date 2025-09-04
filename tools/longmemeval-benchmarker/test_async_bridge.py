#!/usr/bin/env python3
"""Test if we can use sync functions with async graph at one bridge point."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from langchain_core.messages import AIMessage, ToolMessage, BaseMessage
from langgraph.prebuilt import ToolNode
from typing import TypedDict, Annotated, Sequence
from langgraph.graph import StateGraph, END
from langgraph.graph.message import add_messages
from langchain.chat_models import init_chat_model

class SimpleState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]

class SyncAgent:
    """A synchronous agent class that internally uses async graph."""
    
    def __init__(self):
        # Build everything synchronously
        print("Building agent synchronously...")
        
        # Get tools - needs one async bridge
        self.tools = self._get_tools_sync()
        print(f"Got {len(self.tools)} tools")
        
        # Create LLM synchronously
        self.llm = init_chat_model("openai:gpt-4o-mini")
        self.llm_with_tools = self.llm.bind_tools(self.tools)
        
        # Build graph synchronously
        self.graph = self._build_graph_sync()
        print("Graph built successfully")
    
    def _get_tools_sync(self):
        """Bridge to get async tools synchronously."""
        async def _get():
            mcp_client = create_mcp_client()
            return await mcp_client.get_tools()
        return asyncio.run(_get())
    
    def _build_graph_sync(self):
        """Build graph synchronously."""
        workflow = StateGraph(SimpleState)
        
        # Define async node function (required by LangGraph)
        async def agent_node(state):
            response = await self.llm_with_tools.ainvoke(state["messages"])
            return {"messages": [response]}
        
        # Add nodes
        workflow.add_node("agent", agent_node)
        workflow.add_node("tools", ToolNode(self.tools))
        
        # Routing
        def router(state):
            last_msg = state["messages"][-1]
            if hasattr(last_msg, "tool_calls") and last_msg.tool_calls:
                return "tools"
            return "end"
        
        workflow.set_entry_point("agent")
        workflow.add_conditional_edges("agent", router, {"tools": "tools", "end": END})
        workflow.add_edge("tools", "agent")
        
        return workflow.compile()
    
    def process_message(self, content: str, thread_id: str = "default"):
        """Process a message synchronously - single async bridge here."""
        # Build state synchronously
        state = {"messages": [AIMessage(content=content)]}
        
        # Single async bridge point for graph execution
        async def _run():
            return await self.graph.ainvoke(state, {"recursion_limit": 10})
        
        # Run and return synchronously
        return asyncio.run(_run())
    
    def process_message_with_context(self, messages: list):
        """Process with message history - synchronous interface."""
        async def _run():
            return await self.graph.ainvoke({"messages": messages}, {"recursion_limit": 10})
        return asyncio.run(_run())

class CompletelySync:
    """Test if we can avoid async completely by using sync graph."""
    
    def __init__(self):
        print("\nTrying completely synchronous approach...")
        
        # Get tools with async bridge
        async def _get():
            mcp_client = create_mcp_client()
            return await mcp_client.get_tools()
        self.tools = asyncio.run(_get())
        print(f"Got {len(self.tools)} tools")
        
        # Build sync graph
        workflow = StateGraph(SimpleState)
        
        # Try sync node function
        def agent_node_sync(state):
            # This won't work with async tools, but let's try
            response = self.llm.invoke(state["messages"])  # Sync invoke
            return {"messages": [response]}
        
        llm = init_chat_model("openai:gpt-4o-mini")
        self.llm = llm.bind_tools(self.tools)
        
        workflow.add_node("agent", agent_node_sync)
        workflow.add_node("tools", ToolNode(self.tools))
        
        def router(state):
            last_msg = state["messages"][-1]
            if hasattr(last_msg, "tool_calls") and last_msg.tool_calls:
                return "tools"
            return "end"
        
        workflow.set_entry_point("agent")
        workflow.add_conditional_edges("agent", router, {"tools": "tools", "end": END})
        workflow.add_edge("tools", "agent")
        
        self.graph = workflow.compile()
    
    def process_message(self, content: str):
        """Try completely sync execution."""
        state = {"messages": [AIMessage(content=content)]}
        # Try sync invoke
        return self.graph.invoke(state, {"recursion_limit": 10})

def test_sync_interfaces():
    """Test different levels of sync/async abstraction."""
    
    print("=" * 60)
    print("Test 1: Sync Interface with Async Bridge")
    print("=" * 60)
    
    # Create agent with sync interface
    agent = SyncAgent()
    
    # Use it synchronously - no async/await needed here!
    print("\nProcessing message (sync interface):")
    result = agent.process_message("List all vaults")
    
    print(f"Got {len(result['messages'])} messages")
    for msg in result["messages"][-2:]:  # Last 2 messages
        if isinstance(msg, ToolMessage):
            print(f"  Tool result: {msg.content[:100]}...")
        elif isinstance(msg, AIMessage):
            print(f"  AI response: {msg.content[:100]}...")
    
    print("\n" + "=" * 60)
    print("Test 2: Completely Sync (Will Fail with Async Tools)")
    print("=" * 60)
    
    try:
        sync_agent = CompletelySync()
        result = sync_agent.process_message("List vaults")
        print(f"Unexpected success: {result}")
    except Exception as e:
        print(f"Expected failure: {type(e).__name__}")
        if "StructuredTool does not support sync invocation" in str(e):
            print("✓ Confirmed: Can't use sync graph.invoke() with async MCP tools")

def test_minimal_async():
    """Test the absolute minimum async needed."""
    print("\n" + "=" * 60)
    print("Test 3: Minimal Async Pattern")
    print("=" * 60)
    
    # Everything sync except one run function
    def run_agent(message: str) -> dict:
        """Completely sync function that internally bridges to async."""
        
        # Build everything in sync context
        async def _build_and_run():
            # Get tools
            mcp_client = create_mcp_client()
            tools = await mcp_client.get_tools()
            
            # Build graph
            llm = init_chat_model("openai:gpt-4o-mini").bind_tools(tools)
            
            workflow = StateGraph(SimpleState)
            
            async def agent(state):
                return {"messages": [await llm.ainvoke(state["messages"])]}
            
            workflow.add_node("agent", agent)
            workflow.add_node("tools", ToolNode(tools))
            
            def route(state):
                last = state["messages"][-1]
                return "tools" if hasattr(last, "tool_calls") and last.tool_calls else "end"
            
            workflow.set_entry_point("agent")
            workflow.add_conditional_edges("agent", route, {"tools": "tools", "end": END})
            workflow.add_edge("tools", "agent")
            
            graph = workflow.compile()
            
            # Run
            return await graph.ainvoke({"messages": [AIMessage(content=message)]})
        
        # Single bridge point
        return asyncio.run(_build_and_run())
    
    # Usage is completely synchronous!
    print("Calling run_agent synchronously...")
    result = run_agent("List vaults")
    print(f"Got {len(result['messages'])} messages")
    print("✓ Works with single asyncio.run() bridge!")

if __name__ == "__main__":
    test_sync_interfaces()
    test_minimal_async()
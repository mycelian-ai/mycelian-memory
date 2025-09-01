#!/usr/bin/env python3
"""Quick test to verify agent can make tool calls."""

import os
import sys
import asyncio

# Set env vars
os.environ["OPENAI_API_KEY"] = os.environ.get("OPENAI_API_KEY", "")
os.environ["MYCELIAN_MCP_URL"] = "http://localhost:11546/mcp"

from src.mycelian_memory_agent import MycelianMemoryAgent
from src.memory_manager import MemoryManager

def test_agent():
    print("Creating agent...")
    agent = MycelianMemoryAgent(
        model_id="openai:gpt-5-nano-2025-08-07",
        max_tool_calls_per_turn=5,
        debug=True
    )
    
    print("\nCreating memory manager...")
    memory_mgr = MemoryManager(agent._mcp, debug=True)
    
    print("\nEnsuring vault...")
    vault_id = memory_mgr.ensure_vault("test-vault")
    print(f"Vault ID: {vault_id}")
    
    print("\nEnsuring memory...")
    memory_id = memory_mgr.ensure_memory(vault_id, "test-memory", "NOTES")
    print(f"Memory ID: {memory_id}")
    
    print("\nBinding memory to agent...")
    agent.bind_memory(vault_id, memory_id)
    
    print("\nSending SESSION_START control message...")
    result = agent.invoke_message(
        type="system",
        content="SESSION_START",
    )
    
    print(f"\nAgent response: {result}")
    
    print("\nSending a test message...")
    result = agent.invoke_message(
        type="conversation",
        content="Hello, this is a test message.",
        role="user",
        msg_idx=1,
    )
    
    print(f"\nAgent response: {result}")
    
    agent.close()
    print("\nTest complete!")

if __name__ == "__main__":
    test_agent()
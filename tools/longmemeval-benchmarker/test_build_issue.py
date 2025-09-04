#!/usr/bin/env python3
"""Reproduce the exact build issue."""

import asyncio
import sys
sys.path.append("/Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker")

from src.mycelian_memory_agent.build import build_agent_with_invoker

def test():
    try:
        invoker = build_agent_with_invoker(
            model_id="gpt-4o-mini",
            vault_id="test-vault",
            memory_id="test-memory"
        )
        print(f"Success! Got invoker: {type(invoker)}")
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    test()
#!/usr/bin/env python3
"""Test individual components of the LangGraph benchmarker."""

import asyncio
import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from src.config import load_config_from_toml, setup_logging
from src.utils.dataset_loader import load_longmemeval_dataset
from src.types import BenchmarkConfig

async def test_config_loading():
    """Test configuration loading from TOML file."""
    print("🧪 Testing configuration loading...")
    
    try:
        config = load_config_from_toml("test.toml")
        print(f"✅ Configuration loaded successfully")
        print(f"   Dataset variant: {config.dataset_variant}")
        print(f"   Vault name: {config.vault_name}")
        print(f"   MCP URL: {config.mcp_url}")
        print(f"   Debug logging: {config.enable_debug_logging}")
        return config
    except Exception as e:
        print(f"❌ Configuration loading failed: {e}")
        return None

async def test_dataset_loading(config: BenchmarkConfig):
    """Test dataset loading with the test sample."""
    print("\n🧪 Testing dataset loading...")
    
    try:
        dataset = load_longmemeval_dataset(
            repo_path=config.longmemeval_repo_path,
            dataset_variant=config.dataset_variant,
            validate=True
        )
        
        print(f"✅ Dataset loaded successfully")
        print(f"   Questions count: {len(dataset)}")
        
        # Show first question
        if dataset:
            q = dataset[0]
            print(f"   First question ID: {q['question_id']}")
            print(f"   Question type: {q['question_type']}")
            print(f"   Question: {q['question']}")
            print(f"   Answer: {q['answer']}")
            print(f"   Haystack sessions: {len(q.get('haystack_sessions', []))}")
        
        return dataset
    except Exception as e:
        print(f"❌ Dataset loading failed: {e}")
        return None

async def test_mcp_connection(config: BenchmarkConfig):
    """Test MCP connection to Mycelian Memory server."""
    print("\n🧪 Testing MCP connection...")
    
    try:
        from src.utils.mcp_client import MycelianMCPClient
        
        client = MycelianMCPClient(
            mcp_url=config.mcp_url,
            transport=config.mcp_transport
        )
        
        # Try to initialize and test
        await client.initialize()
        print("✅ MCP client initialized successfully")
        
        # Test basic operations
        try:
            tools = await client.get_tools()
            print(f"   Available tools: {len(tools)}")
        except Exception as e:
            print(f"   Warning: Could not get tools: {e}")
        
        await client.close()
        return True
    except Exception as e:
        print(f"❌ MCP connection failed: {e}")
        print("   Make sure Mycelian MCP server is running on localhost:11546")
        return False

async def main():
    """Run all component tests."""
    print("🚀 Starting LangGraph Benchmarker Component Tests\n")
    
    # Test 1: Configuration loading
    config = await test_config_loading()
    if not config:
        return
    
    # Setup logging
    setup_logging(config)
    
    # Test 2: Dataset loading
    dataset = await test_dataset_loading(config)
    if not dataset:
        return
    
    # Test 3: MCP connection
    mcp_ok = await test_mcp_connection(config)
    
    print(f"\n📊 Test Results Summary:")
    print(f"   Configuration: ✅")
    print(f"   Dataset Loading: ✅")
    print(f"   MCP Connection: {'✅' if mcp_ok else '❌'}")
    
    if mcp_ok:
        print(f"\n🎉 All components ready for testing!")
        print(f"Next step: Run individual workflow components")
    else:
        print(f"\n⚠️  MCP server not available - start it first:")
        print(f"   cd /path/to/mycelian-memory && make start-dev-mycelian-server")

if __name__ == "__main__":
    asyncio.run(main())
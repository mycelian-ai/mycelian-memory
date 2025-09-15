"""MCP client utilities for creating and managing MCP connections."""

import os
from typing import Optional

from langchain_mcp_adapters.client import MultiServerMCPClient  # type: ignore


def create_mcp_client(
    server_url: Optional[str] = None,
    server_name: str = "mycelian-memory-streamable"
) -> MultiServerMCPClient:
    """Create a configured MCP client.

    This is a utility function to create MCP clients that can be shared
    between multiple components (AgentBuilder, MemoryManager, etc.)

    Args:
        server_url: Optional MCP server URL (defaults to env var or localhost)
        server_name: Name of the MCP server (default: mycelian-memory-streamable)

    Returns:
        Configured MultiServerMCPClient instance
    """
    DEFAULT_MCP_URL = "http://localhost:11546/mcp"
    url = server_url or os.environ.get("MYCELIAN_MCP_URL", DEFAULT_MCP_URL)

    return MultiServerMCPClient({
        server_name: {
            "url": url,
            "transport": "streamable_http"
        }
    })

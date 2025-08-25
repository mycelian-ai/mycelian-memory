"""Utility to load the MCP tool JSON-Schema via the mycelianCli from live MCP server.

Requires mycelianCli binary and mycelian-mcp-server to be built.
"""
from __future__ import annotations

import json
import subprocess
from typing import List, Dict
from pathlib import Path
import logging

logger = logging.getLogger(__name__)


def _find_mycelian_cli_binary() -> Path | None:
    """Find the mycelianCli binary."""
    repo_root = Path(__file__).resolve().parents[2]
    
    # Try deterministic location first, then fallbacks
    candidates = [
        repo_root / "bin/mycelianCli",                # Deterministic build path (preferred)
        repo_root / "bin/synapse",                    # Legacy name (deprecated)
        repo_root / "clients/go/bin/synapse",         # Legacy location (deprecated)
        repo_root / "tools/mycelianCli/mycelianCli",  # Direct build in tools
        Path("mycelianCli"),                          # In PATH
    ]
    
    for candidate in candidates:
        try:
            # Test if binary exists and is executable
            result = subprocess.run([str(candidate), "--help"], 
                                  capture_output=True, timeout=5)
            if result.returncode == 0:
                return candidate
        except (subprocess.TimeoutExpired, FileNotFoundError, OSError):
            continue
    return None


def load_tools_schema() -> List[Dict]:
    """Return the canonical MCP tools JSON-Schema from live MCP server.

    Requires:
    1. mycelianCli binary (make build-mycelian-cli)
    2. mycelian-mcp-server binary (make build-mcp-server)
    
    Or build both with: make build
    """
    mycelian_cli_binary = _find_mycelian_cli_binary()
    if not mycelian_cli_binary:
        raise RuntimeError(
            "mycelianCli binary not found. Build it with: "
            "make build-mycelian-cli"
        )
    
    # Call mycelianCli get-tools-schema (always live now)
    result = subprocess.run([
        str(mycelian_cli_binary), 
        "get-tools-schema"
    ], capture_output=True, text=True, timeout=15)
    
    if result.returncode != 0:
        raise RuntimeError(f"mycelianCli failed: {result.stderr}")
    
    # Parse JSON response
    raw = json.loads(result.stdout)
    logger.info("Loaded tools schema from mycelianCli (live MCP server)")

    # Convert MCP-style "inputSchema" to Anthropic-required "input_schema"
    # and remove fields that Anthropic doesn't accept
    converted: List[Dict] = []
    for t in raw:
        t = dict(t)  # shallow copy
        
        # Convert inputSchema to input_schema
        if "inputSchema" in t and "input_schema" not in t:
            t["input_schema"] = t.pop("inputSchema")
        elif "parameters" in t and "input_schema" not in t:
            # Fallback for OpenAI-style format
            t["input_schema"] = t.pop("parameters")
        
        # Remove fields that Anthropic API doesn't accept
        anthropic_incompatible_fields = ["annotations"]
        for field in anthropic_incompatible_fields:
            if field in t:
                t.pop(field)
        
        converted.append(t)
    return converted

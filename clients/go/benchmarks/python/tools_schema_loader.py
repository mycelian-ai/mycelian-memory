"""Utility to load the MCP tool JSON-Schema via the `synapse` CLI.

Falls back to a bundled copy so unit tests do not require the CLI binary.
"""
from __future__ import annotations

import json
import subprocess
import importlib.resources as r
from typing import List, Dict
from pathlib import Path

_SOURCE_PATH = "client/schema/tools.schema.json"


def _load_fallback_schema() -> List[Dict]:
    """Load schema directly from repo path for unit tests (no CLI)."""
    repo_root = Path(__file__).resolve().parents[2]
    p = repo_root / _SOURCE_PATH
    with open(p, "r", encoding="utf-8") as f:
        return json.load(f)


def load_tools_schema() -> List[Dict]:
    """Return the canonical MCP tools JSON-Schema.

    Order of resolution:
    1. `synapse get-tools-schema` (preferred – ensures latest version).
    2. Bundled copy `_tools_schema.json` for offline use & CI.
    """
    try:
        out = subprocess.check_output([
            "synapse",
            "get-tools-schema",
        ], text=True)
    except Exception:
        raw = _load_fallback_schema()
    else:
        raw = json.loads(out)

    # Convert OpenAI-style "parameters" to Anthropic-required "input_schema"
    converted: List[Dict] = []
    for t in raw:
        if "parameters" in t and "input_schema" not in t:
            t = dict(t)  # shallow copy
            t["input_schema"] = t.pop("parameters")
        converted.append(t)
    return converted

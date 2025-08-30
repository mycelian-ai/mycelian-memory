from __future__ import annotations

from typing import Dict
import json as _json


# Control command constants
SYSTEM_START: str = "SESSION_START"
FLUSH_CONTEXT: str = "FLUSH_CONTEXT"
SESSION_END: str = "SESSION_END"


def parse_message(content_str: str) -> Dict[str, str]:
    """Parse a JSON-string message into a minimal schema dict.

    Returns a dict with keys: type, role, content (missing keys default to empty strings).
    """
    try:
        obj = _json.loads(content_str)
        if isinstance(obj, dict):
            return {
                "type": str(obj.get("type", "")),
                "role": str(obj.get("role", "")),
                "content": str(obj.get("content", "")),
            }
    except Exception:
        pass
    return {"type": "", "role": "", "content": content_str}


def is_system(msg: Dict[str, str]) -> bool:
    return msg.get("type") == "system"


def is_conversation(msg: Dict[str, str]) -> bool:
    return msg.get("type") == "conversation"


def to_llm_payload(content_str: str) -> Dict[str, object]:
    """Wrap the JSON-string message as a simple chat payload."""
    return {"messages": [{"role": "user", "content": content_str}]}



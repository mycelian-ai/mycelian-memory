from typing import Any, Dict, List

# Placeholder for actual LangGraph imports and MCP tool registration

def build_agent(model_id: str) -> Any:
    """Return a callable that accepts a list of messages and performs no-ops.

    Real implementation will create a prebuilt LangGraph agent with:
      - dynamic system prompt
      - MCP tools (add_entry, put_context, get_context, list_entries, await_consistency, search_memories)
    """
    def run_session(messages: List[Dict[str, str]]) -> None:
        # TODO: integrate LangGraph create_react_agent and stream messages turn-by-turn
        _ = (model_id, messages)
        return None
    return run_session

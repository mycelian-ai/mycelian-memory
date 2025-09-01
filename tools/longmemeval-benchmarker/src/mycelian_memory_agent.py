from typing import Any, Dict, List, Optional, TextIO
import os
import asyncio
import logging
import json
from tenacious_agent_invoker import invoke_with_backoff


class MycelianMemoryAgent:
    """LangGraph prebuilt ReAct agent wired to MCP tools over HTTP.

    Provides a simple `run_session(messages)` interface compatible with the
    runner. Tools are bound to a specific (vault_id, memory_id) via closures.
    """

    def __init__(self, model_id: str, max_tool_calls_per_turn: int = 5, server_url: Optional[str] = None, debug: bool = False):
        from langchain.chat_models import init_chat_model  # type: ignore
        from langchain_mcp_adapters.client import MultiServerMCPClient  # type: ignore

        self._model_id = model_id
        self._max_tools = max_tool_calls_per_turn
        self._server_url = server_url or os.environ.get("MYCELIAN_MCP_URL", "http://localhost:11546/mcp")
        self._debug = bool(debug)
        self._log_stream: Optional[TextIO] = None
        self._current_msg_ctx: Dict[str, Any] = {}

        # MCP client over HTTP
        self._server_name = "mycelian-memory-streamable"
        self._mcp = MultiServerMCPClient({
            self._server_name: {"url": self._server_url, "transport": "streamable_http"}
        })

        # Memory binding (set per question)
        self._vault_id: Optional[str] = None
        self._memory_id: Optional[str] = None
        
        # Read local rules file
        local_rules_path = os.path.join(os.path.dirname(__file__), "context_summary_rules.md")
        try:
            with open(local_rules_path, "r", encoding="utf-8") as f:
                rules = f.read()
        except Exception:
            rules = ""

        async def _get_prompts() -> Dict[str, Any]:
            try:
                # MCP tool: get_default_prompts(memory_type) - only for templates
                res = await self._mcp.call_tool(self._server_name, "get_default_prompts", {"memory_type": "chat"})  # type: ignore[attr-defined]
                return res or {}
            except Exception:
                return {}

        prompts = asyncio.run(_get_prompts())
        templates = prompts.get("templates") or {}
        entry_capture_prompt = templates.get("entry_capture_prompt") or ""
        summary_prompt = templates.get("summary_prompt") or ""
        context_prompt = templates.get("context_prompt") or ""

        # Compose one static system message: prefix + rules + entry + summary + context
        prefix = (
            "You are the Mycelian Memory Agent. You OBSERVE a conversation between a USER and an AI ASSISTANT. "
            "You do not role-play either participant. Your task is to capture durable memory using MCP tools."
            "\n\nMESSAGE HANDLING:\n"
            "- You receive individual messages (user or assistant turns) with msg_idx to determine when to flush context\n"
            "- System messages (type=system) contain session control commands\n"
            "- Conversation messages (type=conversation) are the actual dialogue to persist\n"
            "\nCRITICAL MEMORY RULES:\n"
            "1. For each conversation message you receive:\n"
            "   - Call add_entry EXACTLY ONCE to persist it\n"
            "   - NEVER call add_entry multiple times for the same message\n"
            "   - Each message gets ONE entry, no exceptions\n"
            "\n2. For system control messages, interpret commands but NEVER persist:\n"
            "   • SESSION_START → Call get_context, then list_entries(limit=10) if resuming\n"
            "   • FLUSH_CONTEXT → Call await_consistency, then put_context immediately\n"
            "   • SESSION_END → Call await_consistency, then put_context to finalize\n"
            "\n3. Use msg_idx to decide when to flush context (every ~6 messages)\n"
            "\nREMEMBER: One message = One add_entry call. No duplicates!\n"
        )
        self._system_prompt: str = (
            (prefix or "").strip()
            + "\n\n"
            + str(rules).strip()
            + "\n\n"
            + str(entry_capture_prompt).strip()
            + "\n\n"
            + str(summary_prompt).strip()
            + "\n\n"
            + str(context_prompt).strip()
        ).strip()

        # Load MCP tools dynamically once
        async def _load_tools():
            return await self._mcp.get_tools()  # type: ignore[attr-defined]

        self._agent_tools = asyncio.run(_load_tools())

        # Defer agent construction until memory is bound
        self._llm = init_chat_model(self._model_id)
        self._agent = None
        # Agent is stateless - thread_id passed per invoke

    def set_log_stream(self, stream: Optional[TextIO]) -> None:
        self._log_stream = stream

    def close(self) -> None:
        try:
            close_fn = getattr(self._mcp, "close", None)
            if callable(close_fn):
                close_fn()
        except Exception:
            # Best-effort shutdown for local runs
            pass

    def _dump_agent_state(self, thread_id: str) -> None:
        """Log a brief snapshot of the agent's checkpointed state."""
        if not self._debug or self._agent is None:
            return
        try:
            cfg = {"configurable": {"thread_id": thread_id}}
            get_state = getattr(self._agent, "get_state", None)
            if callable(get_state):
                state = self._agent.get_state(cfg)
                msg_count = len(state.get("messages", [])) if isinstance(state, dict) else None
                if msg_count:
                    logging.getLogger("lme.agent").info(
                        "STATE thread_id=%s messages=%d", thread_id, msg_count
                    )
        except Exception:
            pass  # Silent failure for state dump

    def bind_memory(self, vault_id: str, memory_id: str) -> None:
        from langgraph.prebuilt import create_react_agent  # type: ignore
        from langgraph.checkpoint.memory import InMemorySaver  # type: ignore
        self._vault_id = vault_id
        self._memory_id = memory_id
        # Supply IDs via prompt guidance only
        prompt_with_ids = (
            self._system_prompt
            + "\n\n"
            + f"For all MCP tool calls, use vault_id or vaultId = '{vault_id}'."
            + "\n"
            + f"If a tool requires a memory identifier (memory_id or memoryId), use '{memory_id}'."
            + "\n\n"
            + "IMPORTANT: Memory has been pre-initialized with current context and recent entries. "
            + "Use your working context for decision making. Follow the tool usage rules strictly."
        )

        # Use a checkpointer so we only send the latest message; state persists by thread_id
        self._checkpointer = InMemorySaver()
        # Wrap tools to log generic tool calls (name + args) to lme.tool using StructuredTool
        from langchain_core.tools import StructuredTool  # type: ignore

        def _wrap_tool(t: Any) -> Any:
            name = getattr(t, "name", "tool")
            description = getattr(t, "description", "")
            args_schema = getattr(t, "args_schema", None)

            def _log(args: Dict[str, Any]) -> None:
                try:
                    args_str = json.dumps(args, ensure_ascii=False)
                except Exception:
                    args_str = str(args)
                if len(args_str) > 1500:
                    args_str = args_str[:1497] + "..."
                ctx = self._current_msg_ctx or {}
                logging.getLogger("lme.tool").info(
                    "TOOL_CALL tool=%s thread_id=%s role=%s msg_idx=%s args=%s",
                    name,
                    str(ctx.get("thread_id") or ""),
                    str(ctx.get("role") or ""),
                    str(ctx.get("msg_idx") or ""),
                    args_str,
                )

            async def _acor(input: Dict[str, Any] | None = None, **kwargs):  # type: ignore[override]
                args: Dict[str, Any] = dict(input or {})
                args.update(kwargs or {})
                try:
                    _log(args)
                except Exception:
                    pass
                if hasattr(t, "ainvoke"):
                    return await t.ainvoke(args)  # type: ignore[attr-defined]
                return t.invoke(args)

            def _func(input: Dict[str, Any] | None = None, **kwargs):  # type: ignore[override]
                args: Dict[str, Any] = dict(input or {})
                args.update(kwargs or {})
                try:
                    _log(args)
                except Exception:
                    pass
                if hasattr(t, "invoke"):
                    return t.invoke(args)
                return asyncio.run(t.ainvoke(args))  # type: ignore[attr-defined]

            try:
                if args_schema is not None:
                    return StructuredTool(
                        name=name,
                        description=description,
                        args_schema=args_schema,
                        func=_func,
                        coroutine=_acor,
                    )
            except Exception:
                pass
            return t

        wrapped_tools = [_wrap_tool(t) for t in self._agent_tools]
        self._agent = create_react_agent(
            model=self._llm,
            tools=wrapped_tools,
            prompt=prompt_with_ids,
            checkpointer=self._checkpointer,
        )

        # Agent ready for invocation

    def invoke_message(self, type: str, content: str, thread_id: str, role: str = None, msg_idx: int = None) -> Any:
        """Process a single message through the agent per message_spec.md.
        
        Args:
            type: 'conversation' or 'system'
            content: The message content (non-empty string)
            thread_id: Session thread identifier for checkpointer
            role: For conversation messages, 'user' or 'assistant' (required)
            msg_idx: For conversation messages, 1-based message index (required)
            
        Returns:
            Agent response from LangGraph
        """
        if self._agent is None:
            raise RuntimeError("Agent not initialized. Call bind_memory first.")
        
        # Build message per spec
        if type == "system":
            if role is not None or msg_idx is not None:
                raise ValueError("System messages must not have role or msg_idx")
            msg = {"type": "system", "content": content}
            payload = {"messages": [{"role": "system", "content": content}]}
        elif type == "conversation":
            if role is None or msg_idx is None:
                raise ValueError("Conversation messages require role and msg_idx")
            if role not in ("user", "assistant"):
                raise ValueError(f"Invalid role: {role}")
            if not isinstance(msg_idx, int) or msg_idx < 1:
                raise ValueError(f"msg_idx must be positive integer, got: {msg_idx}")
            msg = {"type": "conversation", "role": role, "content": content, "msg_idx": msg_idx}
            payload = {"messages": [{"role": role, "content": content}]}
        else:
            raise ValueError(f"Invalid type: {type}")
        
        if not isinstance(content, str) or not content.strip():
            raise ValueError("content must be non-empty string")
        
        config = {"configurable": {"thread_id": thread_id}}

        # Record current message context (not used for logging now)
        self._current_msg_ctx = {"thread_id": thread_id, "type": type, "role": role, "msg_idx": msg_idx, "memory_id": self._memory_id}
        
        # Minimal agent logging to question log via lme.agent
        agent_log = logging.getLogger("lme.agent")
        try:
            if type == "system":
                prev = (content or "").strip()
                if len(prev) > 200:
                    prev = prev[:197] + "..."
                agent_log.info("SYS thread_id=%s content=%s", thread_id, prev)
            else:
                txt = (content or "").strip()
                prev = txt[:197] + "..." if len(txt) > 200 else txt
                agent_log.info(
                    "MSG thread_id=%s msg_idx=%d role=%s chars=%d preview=%s",
                    thread_id,
                    int(msg_idx or 0),
                    str(role or ""),
                    len(content or ""),
                    prev,
                )
        except Exception:
            pass

        # Invoke agent with retry logic; surface retry notes in debug mode
        result = invoke_with_backoff(
            lambda: self._agent.invoke(payload, config),
            debug=self._debug,
            log=lambda m: logging.getLogger("lme.agent").info(str(m)),
        )

        # Result summary (messages count) for quick visibility
        try:
            cfg = {"configurable": {"thread_id": thread_id}}
            get_state = getattr(self._agent, "get_state", None)
            if callable(get_state):
                state = self._agent.get_state(cfg)
                msg_count = len(state.get("messages", [])) if isinstance(state, dict) else None
                if msg_count is not None:
                    agent_log.info("RESULT thread_id=%s messages=%s", thread_id, msg_count)
        except Exception:
            pass
        
        return result


 


def build_agent(model_id: str, max_tool_calls_per_turn: int = 5, provider_type: str | None = None, debug: bool = False) -> MycelianMemoryAgent:
    # OpenAI-only simplification
    resolved_model = model_id if str(model_id).startswith("openai:") else f"openai:{model_id}"
    return MycelianMemoryAgent(model_id=resolved_model, max_tool_calls_per_turn=max_tool_calls_per_turn, debug=debug)

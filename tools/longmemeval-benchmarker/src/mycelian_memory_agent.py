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
            agent_log = logging.getLogger("lme.agent")
            if callable(get_state):
                state = self._agent.get_state(cfg)

                def _extract_messages(obj: Any):
                    try:
                        if isinstance(obj, dict):
                            if isinstance(obj.get("messages"), list):
                                return obj.get("messages")
                            if isinstance(obj.get("values"), dict):
                                return _extract_messages(obj.get("values"))
                            if isinstance(obj.get("state"), dict):
                                return _extract_messages(obj.get("state"))
                        for attr in ("messages", "values", "state"):
                            if hasattr(obj, attr):
                                val = getattr(obj, attr)
                                if isinstance(val, list):
                                    return val
                                if isinstance(val, dict):
                                    return _extract_messages(val)
                    except Exception:
                        return None
                    return None

                msgs = _extract_messages(state)
                if isinstance(msgs, list):
                    agent_log.info("STATE thread_id=%s messages=%d", thread_id, len(msgs))
                    # Emit one line per message for audit in debug mode
                    def _to_preview(content: Any, limit: int = 200) -> str:
                        try:
                            if isinstance(content, list):
                                # Join text parts if present
                                parts: list[str] = []
                                for it in content:
                                    if isinstance(it, dict):
                                        txt = it.get("text") or it.get("content") or ""
                                        if isinstance(txt, str):
                                            parts.append(txt)
                                    elif isinstance(it, str):
                                        parts.append(it)
                                s = "\n".join(p for p in parts if p)
                            elif isinstance(content, dict):
                                s = json.dumps(content, ensure_ascii=False)
                            else:
                                s = str(content)
                        except Exception:
                            s = str(content)
                        s = (s or "").strip()
                        if len(s) > limit:
                            return s[: limit - 3] + "..."
                        return s

                    def _as_mapping(m: Any) -> Dict[str, Any]:
                        if isinstance(m, dict):
                            return m
                        for attr in ("to_dict", "dict"):
                            try:
                                fn = getattr(m, attr, None)
                                if callable(fn):
                                    d = fn()
                                    if isinstance(d, dict):
                                        return d
                            except Exception:
                                pass
                        out: Dict[str, Any] = {}
                        for k in ("type", "role", "content", "name", "additional_kwargs", "tool_calls", "tool"):
                            if hasattr(m, k):
                                try:
                                    out[k] = getattr(m, k)
                                except Exception:
                                    pass
                        if not out:
                            out["repr"] = str(m)
                        return out

                    for idx, m in enumerate(msgs, start=1):
                        dm = _as_mapping(m)
                        mtype = dm.get("type") or type(m).__name__
                        role = dm.get("role") or ("tool" if dm.get("tool") else "")
                        # Tool calls summary (for AI messages)
                        tc_names: list[str] = []
                        try:
                            tcs = None
                            if isinstance(dm.get("tool_calls"), list):
                                tcs = dm.get("tool_calls")
                            elif isinstance(dm.get("additional_kwargs"), dict):
                                ak = dm.get("additional_kwargs")
                                if isinstance(ak.get("tool_calls"), list):
                                    tcs = ak.get("tool_calls")
                            if isinstance(tcs, list):
                                for tc in tcs:
                                    name = None
                                    if isinstance(tc, dict):
                                        if isinstance(tc.get("function"), dict):
                                            name = tc.get("function", {}).get("name")
                                        name = name or tc.get("name") or tc.get("type")
                                    else:
                                        # Fallback to string form
                                        name = str(tc)
                                    if name:
                                        tc_names.append(str(name))
                        except Exception:
                            pass

                        preview = _to_preview(dm.get("content"))
                        if tc_names:
                            agent_log.info(
                                "STATE_MSG idx=%d type=%s role=%s tool_calls=%d[%s] content=%s",
                                idx,
                                str(mtype),
                                str(role or ""),
                                len(tc_names),
                                ",".join(tc_names)[:120],
                                preview,
                            )
                        else:
                            agent_log.info(
                                "STATE_MSG idx=%d type=%s role=%s content=%s",
                                idx,
                                str(mtype),
                                str(role or ""),
                                preview,
                            )
                else:
                    stype = type(state).__name__
                    if isinstance(state, dict):
                        keys = list(state.keys())
                        agent_log.info(
                            "STATE thread_id=%s state_type=%s keys=%s",
                            thread_id,
                            stype,
                            ",".join(keys)[:120],
                        )
                    else:
                        agent_log.info("STATE thread_id=%s state_type=%s", thread_id, stype)
        except Exception:
            pass  # Silent failure for state dump

    def bind_memory(self, vault_id: str, memory_id: str) -> None:
        from langgraph.prebuilt import ToolNode  # type: ignore
        from langgraph.checkpoint.memory import InMemorySaver  # type: ignore
        from langgraph.graph import StateGraph, MessagesState, START, END  # type: ignore
        from langchain_core.messages import SystemMessage, ChatMessage, AIMessage, ToolMessage  # type: ignore
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
            + "\n\n"
            + "ENFORCEMENT (concise):\n"
            + "IDENTITY: You are 'mycelian'. Use role='mycelian' only for self-reference; never persist role='mycelian'.\n"
            + "OUTPUT: Emit ONLY MCP tool calls; never write assistant chat text.\n"
            + "CONVERSATION: One add_entry per (thread_id,msg_idx).\n"
            + "  Build raw_entry following the entry_capture_prompt; build summary following the summary_prompt.\n"
            + "  Tags MUST be {role} only. DO NOT include msg_idx in tags.\n"
            + "SYSTEM: SESSION_START → get_context once then list_entries(limit=10). FLUSH_CONTEXT/SESSION_END → await_consistency then put_context. Do not persist system messages.\n"
            + "NO_CONTEXT_ON_CONVO: Never call get_context or list_entries during conversation messages.\n"
            + "FLUSH_MOD6: Use msg_idx ONLY to decide flushing. If msg_idx % 6 == 0 → await_consistency then put_context.\n"
            + "NO_DUPES: If a tool errors, do NOT reissue add_entry for the same (thread_id,msg_idx)."
        )

        # Use a checkpointer so we only send the latest message; state persists by thread_id
        self._checkpointer = InMemorySaver()

        # In-place tool logging wrappers to avoid altering tool types
        def _attach_logging_inplace(t: Any) -> None:
            if getattr(t, "_lme_wrapped", False):
                return
            name = getattr(t, "name", "tool")
            orig_invoke = getattr(t, "invoke", None)
            orig_ainvoke = getattr(t, "ainvoke", None)

            def _log_args(args: Dict[str, Any]) -> None:
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

            if callable(orig_invoke):
                def _wrapped_invoke(input: Any = None, **kwargs):  # type: ignore[override]
                    args: Dict[str, Any] = {}
                    if isinstance(input, dict):
                        args.update(input)
                    elif input is not None:
                        args["_input"] = input
                    if kwargs:
                        args.update(kwargs)
                    try:
                        _log_args(args)
                    except Exception:
                        pass
                    return orig_invoke(input, **kwargs)
                try:
                    import types
                    t.invoke = types.MethodType(_wrapped_invoke, t)  # type: ignore[attr-defined]
                except Exception:
                    t.invoke = _wrapped_invoke  # type: ignore[attr-defined]

            if callable(orig_ainvoke):
                async def _wrapped_ainvoke(input: Any = None, **kwargs):  # type: ignore[override]
                    args: Dict[str, Any] = {}
                    if isinstance(input, dict):
                        args.update(input)
                    elif input is not None:
                        args["_input"] = input
                    if kwargs:
                        args.update(kwargs)
                    try:
                        _log_args(args)
                    except Exception:
                        pass
                    return await orig_ainvoke(input, **kwargs)
                try:
                    import types
                    t.ainvoke = types.MethodType(_wrapped_ainvoke, t)  # type: ignore[attr-defined]
                except Exception:
                    t.ainvoke = _wrapped_ainvoke  # type: ignore[attr-defined]

            try:
                setattr(t, "_lme_wrapped", True)
            except Exception:
                pass

        for _t in self._agent_tools:
            try:
                _attach_logging_inplace(_t)
            except Exception:
                pass

        # Build custom tool-only graph: model sees only system+chat; returns tool_calls only
        llm_with_tools = self._llm.bind_tools(self._agent_tools)

        def _curate_for_model(msgs: list[Any]) -> list[Any]:
            curated: list[Any] = []
            for m in msgs:
                # Exclude historical SystemMessage entries; we always prepend the latest system prompt.
                # Include:
                # - ChatMessage (prior conversation turns)
                # - ToolMessage (tool results for progress)
                # - AIMessage only when it contains tool_calls (provider pairing requirement)
                if isinstance(m, (ChatMessage, ToolMessage)):
                    curated.append(m)
                else:
                    try:
                        if isinstance(m, AIMessage) and getattr(m, "tool_calls", None):
                            curated.append(m)
                    except Exception:
                        pass
            return curated

        def llm_call(state: MessagesState):
            model_input = [SystemMessage(content=prompt_with_ids)]
            curated = _curate_for_model(state["messages"])
            # Include the latest SystemMessage (e.g., SESSION_START) if present, while
            # still excluding historical system messages from prior turns.
            try:
                last_msg = state["messages"][-1]
                if isinstance(last_msg, SystemMessage):
                    curated.append(last_msg)
            except Exception:
                pass
            model_input += curated
            # Debug log: input sizes
            try:
                logging.getLogger("lme.agent").info(
                    "LLM_CALL curated_messages=%d", len(model_input)
                )
            except Exception:
                pass
            resp = llm_with_tools.invoke(model_input)
            # Debug log: tool_calls count
            try:
                tc = getattr(resp, "tool_calls", []) or []
                logging.getLogger("lme.agent").info(
                    "LLM_CALL tool_calls=%d", len(tc)
                )
                # Log planned tool calls with args for visibility
                for tcall in tc:
                    try:
                        name = tcall.get("name") if isinstance(tcall, dict) else getattr(tcall, "name", "")
                        args = tcall.get("args") if isinstance(tcall, dict) else getattr(tcall, "args", {})
                        try:
                            args_str = json.dumps(args, ensure_ascii=False)
                        except Exception:
                            args_str = str(args)
                        if len(args_str) > 1500:
                            args_str = args_str[:1497] + "..."
                        logging.getLogger("lme.tool").info(
                            "TOOL_PLAN tool=%s args=%s",
                            str(name or ""),
                            args_str,
                        )
                    except Exception:
                        pass
            except Exception:
                pass
            return {
                "messages": [
                    AIMessage(
                        content="",
                        tool_calls=getattr(resp, "tool_calls", []),
                        additional_kwargs=getattr(resp, "additional_kwargs", {}),
                    )
                ]
            }

        def should_continue(state: MessagesState):
            messages = state["messages"]
            last = messages[-1]
            return "tools" if getattr(last, "tool_calls", None) else END

        builder = StateGraph(MessagesState)
        builder.add_node("llm_call", llm_call)
        builder.add_node("tools", ToolNode(self._agent_tools))
        builder.add_edge(START, "llm_call")
        builder.add_conditional_edges(
            "llm_call",
            should_continue,
            {"tools": "tools", END: END},
        )
        builder.add_edge("tools", "llm_call")

        self._agent = builder.compile(checkpointer=self._checkpointer)

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
        
        # Build message per spec (LangChain Message objects for custom graph)
        if type == "system":
            if role is not None or msg_idx is not None:
                raise ValueError("System messages must not have role or msg_idx")
            from langchain_core.messages import SystemMessage as _SysMsg  # type: ignore
            lc_msg = _SysMsg(content=content)
        elif type == "conversation":
            if role is None or msg_idx is None:
                raise ValueError("Conversation messages require role and msg_idx")
            if role not in ("user", "assistant"):
                raise ValueError(f"Invalid role: {role}")
            if not isinstance(msg_idx, int) or msg_idx < 1:
                raise ValueError(f"msg_idx must be positive integer, got: {msg_idx}")
            from langchain_core.messages import ChatMessage as _ChatMsg  # type: ignore
            lc_msg = _ChatMsg(role=role, content=content)
        else:
            raise ValueError(f"Invalid type: {type}")
        
        if not isinstance(content, str) or not content.strip():
            raise ValueError("content must be non-empty string")
        
        config = {"configurable": {"thread_id": thread_id, "recursion_limit": 12}}

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

        # Invoke agent asynchronously so tools run via coroutine paths
        payload = {"messages": [lc_msg]}
        def _invoke_async():
            async def _run():
                return await asyncio.wait_for(self._agent.ainvoke(payload, config), timeout=60.0)
            return asyncio.run(_run())
        try:
            result = invoke_with_backoff(
                _invoke_async,
                debug=self._debug,
                log=lambda m: logging.getLogger("lme.agent").info(str(m)),
            )
        except Exception as e:
            logging.getLogger("lme.agent").info("AINVOKE_ERROR %s", str(e)[:200])
            raise

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
        
        # Dump full agent state snapshot after each message in debug mode
        try:
            self._dump_agent_state(thread_id)
        except Exception:
            pass

        return result


 


def build_agent(model_id: str, max_tool_calls_per_turn: int = 5, provider_type: str | None = None, debug: bool = False) -> MycelianMemoryAgent:
    # OpenAI-only simplification
    resolved_model = model_id if str(model_id).startswith("openai:") else f"openai:{model_id}"
    return MycelianMemoryAgent(model_id=resolved_model, max_tool_calls_per_turn=max_tool_calls_per_turn, debug=debug)

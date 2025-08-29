from typing import Any, Dict, List, Tuple, Optional
import os
import asyncio
from langchain_core.tools import StructuredTool  # type: ignore


class MycelianMemoryAgent:
    """LangGraph prebuilt ReAct agent wired to MCP tools over HTTP.

    Provides a simple `run_session(messages)` interface compatible with the
    runner. Tools are bound to a specific (vault_id, memory_id) via closures.
    """

    def __init__(self, model_id: str, max_tool_calls_per_turn: int = 5, server_url: Optional[str] = None):
        from langchain.chat_models import init_chat_model  # type: ignore
        from langchain_mcp_adapters.client import MultiServerMCPClient  # type: ignore

        self._model_id = model_id
        self._max_tools = max_tool_calls_per_turn
        self._server_url = server_url or os.environ.get("MYCELIAN_MCP_URL", "http://localhost:11546/mcp")
        self._debug = bool(os.environ.get("LME_DEBUG"))

        # MCP client over HTTP
        self._server_name = "mycelian-memory-streamable"
        self._mcp = MultiServerMCPClient({
            self._server_name: {"url": self._server_url, "transport": "streamable_http"}
        })

        # Memory binding (set per question)
        self._vault_id: Optional[str] = None
        self._memory_id: Optional[str] = None

        # Load default prompts (system guidance + templates) from MCP
        import asyncio as _asyncio

        async def _get_prompts() -> Dict[str, Any]:
            try:
                # MCP tool: get_default_prompts(memory_type)
                res = await self._mcp.call_tool(self._server_name, "get_default_prompts", {"memory_type": "chat"})  # type: ignore[attr-defined]
                return res or {}
            except Exception:
                return {}

        prompts = _asyncio.run(_get_prompts())
        rules = prompts.get("context_summary_rules") or ""
        templates = prompts.get("templates") or {}
        entry_capture_prompt = templates.get("entry_capture_prompt") or ""
        summary_prompt = templates.get("summary_prompt") or ""
        context_prompt = templates.get("context_prompt") or ""

        # Compose one static system message: prefix + rules + entry + summary + context
        prefix = (
            "You are the Mycelian Memory Agent. You OBSERVE a conversation between a USER and an AI ASSISTANT. "
            "You do not role-play either participant. Your sole task is to capture durable memory for this conversation "
            "in Mycelian using the MCP tools."
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
        import asyncio as _asyncio_tools

        async def _load_tools():
            return await self._mcp.get_tools()  # type: ignore[attr-defined]

        self._base_tools = _asyncio_tools.run(_load_tools())
        # Build logging-wrapped tools for observability (no behavior change)
        self._tools_with_logging = [self._wrap_tool_with_logging(t) for t in self._base_tools]

        # Defer agent construction until memory is bound
        self._llm = init_chat_model(self._model_id)
        self._agent = None

    def close(self) -> None:
        try:
            close_fn = getattr(self._mcp, "close", None)
            if callable(close_fn):
                close_fn()
        except Exception:
            # Best-effort shutdown for local runs
            pass

    def _tool_by_name(self, name: str) -> Any:
        for t in self._base_tools:
            if getattr(t, "name", None) == name:
                return t
        raise RuntimeError(f"MCP tool not found: {name}")

    def _call_tool(self, name: str, arguments: Dict[str, Any]) -> Any:
        t = self._tool_by_name(name)
        if self._debug:
            prev = str(arguments)
            if len(prev) > 200:
                prev = prev[:200] + "…"
            print(f"[agent][mcp] call {name} args={prev}")
        if hasattr(t, "ainvoke"):
            result = asyncio.run(t.ainvoke(arguments))  # type: ignore[attr-defined]
        elif hasattr(t, "invoke"):
            result = t.invoke(arguments)
        else:
            raise RuntimeError(f"MCP tool is not invocable: {name}")
        # Normalize: adapter often returns JSON text; parse when possible
        if isinstance(result, (dict, list)):
            if self._debug:
                rp = str(result)
                if len(rp) > 200:
                    rp = rp[:200] + "…"
                print(f"[agent][mcp] {name} -> {rp}")
            return result
        try:
            import json as _json
            parsed = _json.loads(result) if isinstance(result, str) else result
            if self._debug:
                rp = str(parsed)
                if len(rp) > 200:
                    rp = rp[:200] + "…"
                print(f"[agent][mcp] {name} -> {rp}")
            return parsed
        except Exception:
            if self._debug:
                rp = str(result)
                if len(rp) > 200:
                    rp = rp[:200] + "…"
                print(f"[agent][mcp] {name} -> {rp}")
            return result

    def ensure_vault(self, title: Optional[str], vault_id: Optional[str]) -> str:
        if vault_id:
            return vault_id
        if not title:
            raise ValueError("either vault_id or vault_title is required")
        out = self._call_tool("list_vaults", {})
        vaults_iter = []
        if isinstance(out, list):
            vaults_iter = out
        elif isinstance(out, dict):
            vaults_iter = out.get("vaults", []) or out.get("items", [])
        target = (title or "").strip().lower()
        for v in vaults_iter:
            vt = v.get("title") if isinstance(v, dict) else None
            if isinstance(vt, str) and vt.strip().lower() == target:
                vid = None
                if isinstance(v, dict):
                    vid = v.get("id") or v.get("vault_id") or v.get("vaultId")
                if vid:
                    return vid
        try:
            created = self._call_tool("create_vault", {"title": title, "description": "longmemeval run"})
        except Exception as e:
            # Fallback for duplicate constraint or server returns 409/500 for existing title
            msg = str(e)
            if ("duplicate" in msg.lower()) or ("23505" in msg) or ("already exists" in msg.lower()):
                out2 = self._call_tool("list_vaults", {})
                vaults_iter2 = []
                if isinstance(out2, list):
                    vaults_iter2 = out2
                elif isinstance(out2, dict):
                    vaults_iter2 = out2.get("vaults", []) or out2.get("items", [])
                for v in vaults_iter2:
                    vt = v.get("title") if isinstance(v, dict) else None
                    if isinstance(vt, str) and vt.strip().lower() == target:
                        vid2 = None
                        if isinstance(v, dict):
                            vid2 = v.get("id") or v.get("vault_id") or v.get("vaultId")
                        if vid2:
                            return vid2
            raise
        vid = None
        if isinstance(created, dict):
            vid = created.get("id") or created.get("vault_id") or created.get("vaultId")
        elif isinstance(created, str):
            vid = created
        if not vid:
            raise RuntimeError("create_vault did not return an id")
        return vid

    def ensure_memory(self, vault_id: str, title: str, memory_type: str = "NOTES") -> str:
        try:
            created = self._call_tool(
                "create_memory_in_vault",
                {"vault_id": vault_id, "title": title, "memory_type": memory_type, "description": ""},
            )
        except Exception:
            created = {}
        mid = None
        if isinstance(created, dict):
            mid = created.get("id") or created.get("memory_id") or created.get("memoryId")
        elif isinstance(created, str):
            mid = created
        if mid:
            return mid
        listed = self._call_tool("list_memories", {"vault_id": vault_id})
        memories = []
        if isinstance(listed, list):
            memories = listed
        elif isinstance(listed, dict):
            memories = listed.get("memories", []) or listed.get("items", [])
        for m in memories:
            mt = m.get("title") if isinstance(m, dict) else None
            if mt == title:
                mmid = None
                if isinstance(m, dict):
                    mmid = m.get("id") or m.get("memory_id") or m.get("memoryId")
                if mmid:
                    return mmid
        raise RuntimeError("create_memory_in_vault did not return an id and it was not found in list_memories")

    def bind_memory(self, vault_id: str, memory_id: str) -> None:
        from langgraph.prebuilt import create_react_agent  # type: ignore
        self._vault_id = vault_id
        self._memory_id = memory_id
        # Keep tools unchanged (behavior), but use logging-wrapped versions for observability.
        # Supply IDs via prompt guidance so the LLM includes them when required
        prompt_with_ids = (
            self._system_prompt
            + "\n\n"
            + f"For all MCP tool calls, use vault_id or vaultId = '{vault_id}'."
            + "\n"
            + f"If a tool requires a memory identifier (memory_id or memoryId), use '{memory_id}'."
            + "\n\n"
            + "SESSION BOOTSTRAP – Before the first model response of each new session, you MUST strictly follow this sequence:"
            + "\n1) Call get_context() immediately."
            + "\n2) If it returns the exact default placeholder string:"
            + "\n   'This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available.'"
            + "\n   then immediately call put_context({}) to initialize an empty context."
            + "\n3) Call list_entries(limit=10) and merge any missing facts into your working context BEFORE replying to the user."

        )

        self._agent = create_react_agent(
            model=self._llm,
            tools=self._tools_with_logging,
            prompt=prompt_with_ids,
        )

    def run_session(self, messages: List[Dict[str, str]]) -> Tuple[int, int]:
        # Maintain rolling transcript externally per session; pass full history each turn
        turns = 0
        tool_calls = 0
        history: List[Dict[str, str]] = []
        for m in messages:
            history.append(m)
            role = m.get("role")
            content = m.get("content", "")
            print(f"[agent][turn] {turns+1} role={role} len(content)={len(content)}")
            _ = self._agent.invoke({"messages": history})
            print(f"[agent][turn] {turns+1} -> completed")
            turns += 1
        return turns, tool_calls

    def _wrap_tool_with_logging(self, tool: Any) -> Any:
        name = getattr(tool, "name", "tool")
        description = getattr(tool, "description", "")
        args_schema = getattr(tool, "args_schema", None)

        # If we cannot read args_schema, fall back to the original tool (no wrapping)
        if args_schema is None:
            return tool

        async def _acoroutine(**kwargs: Any) -> Any:
            prev = str(kwargs)
            if len(prev) > 200:
                prev = prev[:200] + "…"
            if self._debug:
                print(f"[agent][tool] {name} args={prev}")
            try:
                if hasattr(tool, "ainvoke"):
                    res = await tool.ainvoke(kwargs)  # type: ignore[attr-defined]
                else:
                    # run sync in thread via run_in_executor not needed here; call directly
                    res = tool.invoke(kwargs)
                if self._debug:
                    rp = str(res)
                    if len(rp) > 200:
                        rp = rp[:200] + "…"
                    print(f"[agent][tool] {name} -> SUCCESS: {rp}")
                return res
            except Exception as e:
                if self._debug:
                    print(f"[agent][tool] {name} -> ERROR: {e}")
                raise

        def _func(**kwargs: Any) -> Any:
            prev = str(kwargs)
            if len(prev) > 200:
                prev = prev[:200] + "…"
            if self._debug:
                print(f"[agent][tool] {name} (sync) args={prev}")
            try:
                # MCP tools are async-only, so run them in an event loop
                if hasattr(tool, "ainvoke"):
                    res = asyncio.run(tool.ainvoke(kwargs))  # type: ignore[attr-defined]
                else:
                    res = tool.invoke(kwargs)
                if self._debug:
                    rp = str(res)
                    if len(rp) > 200:
                        rp = rp[:200] + "…"
                    print(f"[agent][tool] {name} (sync) -> SUCCESS: {rp}")
                return res
            except Exception as e:
                if self._debug:
                    print(f"[agent][tool] {name} (sync) -> ERROR: {e}")
                raise

        # Construct a new StructuredTool with identical schema and name/description
        try:
            return StructuredTool.from_function(
                name=name,
                description=description,
                args_schema=args_schema,
                func=_func,
                coroutine=_acoroutine,
            )
        except Exception:
            # If wrapping fails, use the original tool
            return tool


class _BoundArgsTool:
    """Wrap a LangChain Tool to bind default arguments (e.g., vault_id, memory_id).

    - Merges provided input with default_args on each call; user input wins on conflict.
    - Exposes name/description/args so the agent can plan normally.
    - Supports both async and sync invocation paths used by adapters.
    """

    def __init__(self, base_tool: Any, default_args: Dict[str, Any]) -> None:
        self._base = base_tool
        self._defaults = dict(default_args)
        self.name = getattr(base_tool, "name", "tool")
        self.description = getattr(base_tool, "description", "")
        self.args = getattr(base_tool, "args", {}) or {}

    def _merge(self, input: Dict[str, Any]) -> Dict[str, Any]:
        merged = dict(self._defaults)
        merged.update(input or {})
        return merged

    async def ainvoke(self, input: Dict[str, Any]) -> Any:  # type: ignore[override]
        args = self._merge(input)
        if hasattr(self._base, "ainvoke"):
            return await self._base.ainvoke(args)  # type: ignore[attr-defined]
        if hasattr(self._base, "invoke"):
            # Run sync path in a thread if needed by caller; here we call directly
            return self._base.invoke(args)
        raise RuntimeError("Wrapped tool is not invocable")

    def invoke(self, input: Dict[str, Any]) -> Any:  # type: ignore[override]
        args = self._merge(input)
        if hasattr(self._base, "invoke"):
            return self._base.invoke(args)
        # Fallback: run async path in a new loop
        return asyncio.run(self._base.ainvoke(args))  # type: ignore[attr-defined]


def build_agent(model_id: str, max_tool_calls_per_turn: int = 5, provider_type: str | None = None) -> MycelianMemoryAgent:
    resolved_model = model_id
    if provider_type and provider_type.lower() == "bedrock" and not str(model_id).startswith("bedrock:"):
        resolved_model = f"bedrock:{model_id}"
    return MycelianMemoryAgent(model_id=resolved_model, max_tool_calls_per_turn=max_tool_calls_per_turn)



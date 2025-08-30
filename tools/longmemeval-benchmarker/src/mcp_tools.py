from __future__ import annotations

from typing import Any, Callable, List


def wrap_tools_with_logging(
    base_tools: List[Any],
    log: Callable[[str], None],
    debug: bool,
    await_consistency_fn: Callable[[str], None],
    get_memory_id_fn: Callable[[], str | None],
) -> List[Any]:
    """Wrap LangChain tools to add logging and durability behavior.

    - Logs args and results when debug is True
    - Before put_context, calls await_consistency(memory_id)
    - Returns StructuredTool wrappers with same name/description/args
    """
    try:
        # Local import to avoid global linter warnings when package is absent
        from langchain_core.tools import StructuredTool  # type: ignore
    except Exception:  # pragma: no cover
        StructuredTool = None  # type: ignore

    wrapped: List[Any] = []

    for tool in base_tools:
        name = getattr(tool, "name", "tool")
        description = getattr(tool, "description", "")
        args_schema = getattr(tool, "args_schema", None)

        if args_schema is None or StructuredTool is None:
            # Fallback: no wrapping possible
            wrapped.append(tool)
            continue

        async def _acoroutine(**kwargs: Any) -> Any:
            prev = str(kwargs)
            if len(prev) > 200:
                prev = prev[:200] + "…"
            if debug:
                log(f"[agent][tool] {name} args={prev}")
            try:
                if name == "put_context":
                    mid = get_memory_id_fn() or ""
                    if mid:
                        try:
                            await_consistency_fn(mid)
                        except Exception:
                            pass
                res = await tool.ainvoke(kwargs) if hasattr(tool, "ainvoke") else tool.invoke(kwargs)  # type: ignore[attr-defined]
                if debug:
                    rp = str(res)
                    if len(rp) > 200:
                        rp = rp[:200] + "…"
                    log(f"[agent][tool] {name} -> SUCCESS: {rp}")
                return res
            except Exception as e:  # pragma: no cover
                if debug:
                    log(f"[agent][tool] {name} -> ERROR: {e}")
                raise

        def _func(**kwargs: Any) -> Any:
            prev = str(kwargs)
            if len(prev) > 200:
                prev = prev[:200] + "…"
            if debug:
                log(f"[agent][tool] {name} (sync) args={prev}")
            try:
                if name == "put_context":
                    mid = get_memory_id_fn() or ""
                    if mid:
                        try:
                            await_consistency_fn(mid)
                        except Exception:
                            pass
                res = tool.invoke(kwargs) if hasattr(tool, "invoke") else tool.ainvoke(kwargs)  # type: ignore[attr-defined]
                if debug:
                    rp = str(res)
                    if len(rp) > 200:
                        rp = rp[:200] + "…"
                    log(f"[agent][tool] {name} (sync) -> SUCCESS: {rp}")
                return res
            except Exception as e:  # pragma: no cover
                if debug:
                    log(f"[agent][tool] {name} (sync) -> ERROR: {e}")
                raise

        try:
            wrapped.append(
                StructuredTool.from_function(  # type: ignore
                    name=name,
                    description=description,
                    args_schema=args_schema,
                    func=_func,
                    coroutine=_acoroutine,
                )
            )
        except Exception:
            wrapped.append(tool)

    return wrapped



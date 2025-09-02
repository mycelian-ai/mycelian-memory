"""ToolLogger class for handling tool call logging."""

import json
import logging
import types
from typing import Any, Callable, Dict, Optional


class ToolLogger:
    """Handles logging for tool invocations and LLM responses."""
    
    # Constants
    LOG_TRUNCATION_LIMIT = 1500
    LOG_TRUNCATION_SUFFIX_LEN = 3  # Length of "..."
    WRAPPED_TOOL_ATTR = "_lme_wrapped"  # Attribute to mark wrapped tools
    
    def __init__(self, debug: bool = False):
        """Initialize ToolLogger with debug mode setting."""
        self.debug = debug
        self.current_msg_ctx: Dict[str, Any] = {}
    
    def set_logging_context(self, context: Dict[str, Any]) -> None:
        """Set the current message context for tool call logging."""
        self.current_msg_ctx = context
    
    def _format_args(self, args: Any) -> str:
        """Format arguments for logging, with truncation if needed."""
        try:
            args_str = json.dumps(args, ensure_ascii=False)
        except (TypeError, ValueError):
            args_str = str(args)
        
        if len(args_str) > self.LOG_TRUNCATION_LIMIT:
            args_str = args_str[:self.LOG_TRUNCATION_LIMIT - self.LOG_TRUNCATION_SUFFIX_LEN] + "..."
        return args_str
    
    def _extract_args(self, input_arg: Any, kwargs: Dict[str, Any]) -> Dict[str, Any]:
        """Extract arguments from various input formats."""
        args: Dict[str, Any] = {}
        if isinstance(input_arg, dict):
            args.update(input_arg)
        elif input_arg is not None:
            args["_input"] = input_arg
        if kwargs:
            args.update(kwargs)
        return args
    
    def _create_wrapper(self, original_func: Callable, log_func: Callable, is_async: bool = False) -> Callable:
        """Create a wrapper function for logging tool calls."""
        if is_async:
            async def wrapped_async(input_arg: Any = None, **kwargs) -> Any:
                args = self._extract_args(input_arg, kwargs)
                try:
                    log_func(args)
                except (AttributeError, RuntimeError):
                    pass  # Logging failed, continue execution
                return await original_func(input_arg, **kwargs)
            return wrapped_async
        else:
            def wrapped_sync(input_arg: Any = None, **kwargs) -> Any:
                args = self._extract_args(input_arg, kwargs)
                try:
                    log_func(args)
                except (AttributeError, RuntimeError):
                    pass  # Logging failed, continue execution
                return original_func(input_arg, **kwargs)
            return wrapped_sync
    
    def _attach_method(self, tool: Any, method_name: str, wrapped_func: Callable) -> None:
        """Attach a wrapped method to a tool object."""
        try:
            setattr(tool, method_name, types.MethodType(wrapped_func, tool))
        except TypeError:
            # Fallback if MethodType binding fails
            setattr(tool, method_name, wrapped_func)
    
    def attach_logging_to_tool(self, tool: Any) -> None:
        """Attach logging wrapper to a tool for debugging."""
        if not self.debug:
            return
            
        if getattr(tool, self.WRAPPED_TOOL_ATTR, False):
            return
            
        name = getattr(tool, "name", "tool")
        orig_invoke = getattr(tool, "invoke", None)
        orig_ainvoke = getattr(tool, "ainvoke", None)

        def _log_args(args: Dict[str, Any]) -> None:
            args_str = self._format_args(args)
            context = self.current_msg_ctx or {}
            logging.getLogger("lme.tool").info(
                "TOOL_CALL tool=%s thread_id=%s role=%s msg_idx=%s args=%s",
                name,
                str(context.get("thread_id") or ""),
                str(context.get("role") or ""),
                str(context.get("msg_idx") or ""),
                args_str,
            )

        if callable(orig_invoke):
            wrapped_invoke = self._create_wrapper(orig_invoke, _log_args, is_async=False)
            self._attach_method(tool, 'invoke', wrapped_invoke)

        if callable(orig_ainvoke):
            wrapped_ainvoke = self._create_wrapper(orig_ainvoke, _log_args, is_async=True)
            self._attach_method(tool, 'ainvoke', wrapped_ainvoke)

        try:
            setattr(tool, self.WRAPPED_TOOL_ATTR, True)
        except AttributeError:
            pass  # Tool doesn't support attribute assignment
    
    def log_tool_calls(self, response: Any) -> None:
        """Log planned tool calls from LLM response."""
        if not self.debug:
            return
            
        try:
            tool_calls = getattr(response, "tool_calls", []) or []
            logging.getLogger("lme.agent").info(
                "LLM_CALL tool_calls=%d", len(tool_calls)
            )
            
            for tool_call in tool_calls:
                try:
                    name = tool_call.get("name") if isinstance(tool_call, dict) else getattr(tool_call, "name", "")
                    args = tool_call.get("args") if isinstance(tool_call, dict) else getattr(tool_call, "args", {})
                    args_str = self._format_args(args)
                    
                    logging.getLogger("lme.tool").info(
                        "TOOL_PLAN tool=%s args=%s",
                        str(name or ""),
                        args_str,
                    )
                except (AttributeError, KeyError):
                    pass  # Tool call format not recognized
        except Exception:
            pass
    
    def log_llm_call(self, num_messages: int) -> None:
        """Log LLM invocation with message count."""
        if not self.debug:
            return
        
        try:
            logging.getLogger("lme.agent").info(
                "LLM_CALL curated_messages=%d", num_messages
            )
        except Exception:
            pass
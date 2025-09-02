"""StateDebugger class for agent state debugging and logging."""

from typing import Any, Dict, List, Optional
import json
import logging


class StateDebugger:
    """Handles agent state debugging and logging for development."""
    
    # Constants
    LOG_TRUNCATION_LIMIT = 1500
    LOG_TRUNCATION_SUFFIX_LEN = 3
    TOOL_NAME_LOG_LIMIT = 120
    DEFAULT_PREVIEW_LIMIT = 200
    
    def __init__(self, debug: bool = False):
        """Initialize StateDebugger with debug mode setting."""
        self.debug = debug
        self.logger = logging.getLogger("lme.agent")
    
    def dump_state(self, agent: Any, thread_id: str) -> None:
        """Log a brief snapshot of the agent's checkpointed state.
        
        Args:
            agent: The LangGraph agent instance
            thread_id: Thread identifier for the current session
        """
        if not self.debug or agent is None:
            return
            
        try:
            config = {"configurable": {"thread_id": thread_id}}
            get_state = getattr(agent, "get_state", None)
            
            if not callable(get_state):
                return
                
            state = agent.get_state(config)
            messages = self._extract_messages(state)
            
            if isinstance(messages, list):
                self._log_state_summary(thread_id, messages)
                self._log_message_details(messages)
            else:
                state_type = type(state).__name__
                self.logger.info("STATE thread_id=%s type=%s", thread_id, state_type)
                
        except (AttributeError, RuntimeError, TypeError):
            pass  # Silent failure for state dump
    
    def _extract_messages(self, obj: Any) -> Optional[List]:
        """Recursively extract messages from various state object formats.
        
        Args:
            obj: State object from agent.get_state()
            
        Returns:
            List of messages if found, None otherwise
        """
        try:
            if isinstance(obj, dict):
                # Direct messages key
                if isinstance(obj.get("messages"), list):
                    return obj.get("messages")
                # Nested in values
                if isinstance(obj.get("values"), dict):
                    return self._extract_messages(obj.get("values"))
                # Nested in state
                if isinstance(obj.get("state"), dict):
                    return self._extract_messages(obj.get("state"))
            
            # Check object attributes
            for attr in ("messages", "values", "state"):
                if hasattr(obj, attr):
                    val = getattr(obj, attr)
                    if isinstance(val, list):
                        return val
                    if isinstance(val, dict):
                        return self._extract_messages(val)
                        
        except (AttributeError, TypeError, KeyError):
            pass
            
        return None
    
    def _log_state_summary(self, thread_id: str, messages: List) -> None:
        """Log a summary of the agent state.
        
        Args:
            thread_id: Thread identifier
            messages: List of messages in the state
        """
        self.logger.info("STATE thread_id=%s messages=%d", thread_id, len(messages))
    
    def _log_message_details(self, messages: List) -> None:
        """Log details for each message in the state.
        
        Args:
            messages: List of messages to log
        """
        for idx, message in enumerate(messages, start=1):
            message_dict = self._message_to_dict(message)
            message_type = message_dict.get("type") or type(message).__name__
            role = message_dict.get("role") or ("tool" if message_dict.get("tool") else "")
            
            # Extract tool calls if present
            tool_call_names = self._extract_tool_calls(message_dict)
            
            # Format content preview
            content_preview = self._format_preview(message_dict.get("content"))
            
            # Log the message
            if tool_call_names:
                self.logger.info(
                    "STATE_MSG idx=%d type=%s role=%s tool_calls=%d[%s] content=%s",
                    idx,
                    str(message_type),
                    str(role or ""),
                    len(tool_call_names),
                    ",".join(tool_call_names)[:self.TOOL_NAME_LOG_LIMIT],
                    content_preview,
                )
            else:
                self.logger.info(
                    "STATE_MSG idx=%d type=%s role=%s content=%s",
                    idx,
                    str(message_type),
                    str(role or ""),
                    content_preview,
                )
    
    def _message_to_dict(self, message: Any) -> Dict[str, Any]:
        """Convert a message object to a dictionary representation.
        
        Args:
            message: Message object (could be dict or object with attributes)
            
        Returns:
            Dictionary representation of the message
        """
        if isinstance(message, dict):
            return message
            
        # Try to convert using object methods
        for attr in ("to_dict", "dict"):
            try:
                func = getattr(message, attr, None)
                if callable(func):
                    result_dict = func()
                    if isinstance(result_dict, dict):
                        return result_dict
            except (AttributeError, TypeError):
                pass
        
        # Fallback: extract known attributes
        output: Dict[str, Any] = {}
        for key in ("type", "role", "content", "name", "additional_kwargs", "tool_calls", "tool"):
            if hasattr(message, key):
                try:
                    output[key] = getattr(message, key)
                except AttributeError:
                    pass
                    
        if not output:
            output["repr"] = str(message)
            
        return output
    
    def _extract_tool_calls(self, message_dict: Dict[str, Any]) -> List[str]:
        """Extract tool call names from a message dictionary.
        
        Args:
            message_dict: Dictionary representation of a message
            
        Returns:
            List of tool call names
        """
        tool_call_names: List[str] = []
        
        try:
            tool_calls = None
            
            # Direct tool_calls
            if isinstance(message_dict.get("tool_calls"), list):
                tool_calls = message_dict.get("tool_calls")
            # Nested in additional_kwargs
            elif isinstance(message_dict.get("additional_kwargs"), dict):
                additional_kwargs = message_dict.get("additional_kwargs")
                if isinstance(additional_kwargs.get("tool_calls"), list):
                    tool_calls = additional_kwargs.get("tool_calls")
            
            if isinstance(tool_calls, list):
                for tool_call in tool_calls:
                    name = None
                    if isinstance(tool_call, dict):
                        # Check for function.name pattern
                        if isinstance(tool_call.get("function"), dict):
                            name = tool_call.get("function", {}).get("name")
                        # Direct name or type
                        name = name or tool_call.get("name") or tool_call.get("type")
                    else:
                        # Fallback to string representation
                        name = str(tool_call)
                    
                    if name:
                        tool_call_names.append(str(name))
                        
        except (AttributeError, KeyError, TypeError):
            pass
            
        return tool_call_names
    
    def _format_preview(self, content: Any, limit: Optional[int] = None) -> str:
        """Format content for preview with truncation if needed.
        
        Args:
            content: Content to format (could be string, list, dict, etc.)
            limit: Maximum length for preview (default: DEFAULT_PREVIEW_LIMIT)
            
        Returns:
            Formatted preview string
        """
        if limit is None:
            limit = self.DEFAULT_PREVIEW_LIMIT
            
        try:
            if isinstance(content, list):
                # Join text parts if present
                parts: List[str] = []
                for item in content:
                    if isinstance(item, dict):
                        text = item.get("text") or item.get("content") or ""
                        if isinstance(text, str):
                            parts.append(text)
                    elif isinstance(item, str):
                        parts.append(item)
                formatted = "\n".join(p for p in parts if p)
            elif isinstance(content, dict):
                formatted = json.dumps(content, ensure_ascii=False)
            else:
                formatted = str(content)
        except (TypeError, ValueError):
            formatted = str(content)
        
        formatted = (formatted or "").strip()
        
        if len(formatted) > limit:
            return formatted[:limit - self.LOG_TRUNCATION_SUFFIX_LEN] + "..."
            
        return formatted

"""AgentMessageProcessor - Handles runtime message processing for the Agent."""

import asyncio
import logging
from typing import Any, Dict, Optional

from langchain_core.messages import ChatMessage as _ChatMsg  # type: ignore
from langchain_core.messages import SystemMessage as _SysMsg  # type: ignore

from src.tenacious_agent_invoker import invoke_with_backoff

from .agent import Agent


class AgentMessageProcessor:
    """Handles runtime message processing for the Agent.
    
    This class is responsible for:
    - Message validation and creation
    - Managing execution context
    - Invoking the Agent's compiled graph
    - Handling results and logging
    - Session/thread management
    
    All logic extracted from MycelianMemoryAgent to focus solely on runtime.
    """
    
    # Constants (from MycelianMemoryAgent)
    RECURSION_LIMIT = 12
    AGENT_TIMEOUT = 180.0
    
    def __init__(self, agent: Agent, config: Dict[str, Any], helpers: Dict[str, Any]):
        """Initialize with Agent and helper objects.
        
        Args:
            agent: The compiled Agent with graph
            config: Configuration from builder (vault_id, memory_id, debug, etc.)
            helpers: Dictionary of helper objects (loggers, debuggers)
        """
        self.agent = agent
        self.config = config
        self.helpers = helpers
        self._current_msg_ctx: Dict[str, Any] = {}
        
        # Extract helpers for easier access
        self._tool_logger = helpers.get('tool_logger')
        self._state_debugger = helpers.get('state_debugger')
        self._message_logger = helpers.get('message_logger')
    
    def invoke_message(
        self,
        message_type: str,
        content: str,
        thread_id: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> Any:
        """Process a single message through the agent per message_spec.md.
        
        This is the main entry point that orchestrates the full message pipeline.
        Extracted directly from MycelianMemoryAgent.invoke_message()
        
        Args:
            message_type: 'conversation' or 'system'
            content: The message content (non-empty string)
            thread_id: Session thread identifier for checkpointer
            role: For conversation messages, 'user' or 'assistant' (required)
            msg_idx: For conversation messages, 1-based message index (required)
            
        Returns:
            Agent response from LangGraph
        """
        # 1. Validate message
        self._validate_message(message_type, content, role, msg_idx)
        
        # 2. Create LangChain message with metadata
        lc_msg = self._create_langchain_message(message_type, content, role, msg_idx)
        
        # 3. Set logging context
        self._set_logging_context(thread_id, message_type, role, msg_idx)
        
        # 4. Log the incoming message
        self._log_message(message_type, content, thread_id, role, msg_idx)
        
        # 5. Invoke the agent
        result = self._invoke_agent(lc_msg, thread_id)
        
        # 6. Log the result
        self._log_result(thread_id)
        
        # 7. Dump debug state if enabled
        if self.config.get('debug'):
            self._dump_agent_state(thread_id)
        
        return result
    
    def _validate_message(
        self,
        message_type: str,
        content: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> None:
        """Validate message parameters according to message_spec.md.
        
        Extracted from MycelianMemoryAgent._validate_message()
        """
        if self.agent.graph is None:
            raise RuntimeError("Agent not initialized.")
        
        if not isinstance(content, str) or not content.strip():
            raise ValueError("content must be non-empty string")
        
        if message_type == "system":
            if role is not None or msg_idx is not None:
                raise ValueError("System messages must not have role or msg_idx")
        elif message_type == "conversation":
            if role is None or msg_idx is None:
                raise ValueError("Conversation messages require role and msg_idx")
            if role not in ("user", "assistant"):
                raise ValueError(f"Invalid role: {role}")
            if not isinstance(msg_idx, int) or msg_idx < 1:
                raise ValueError(f"msg_idx must be positive integer, got: {msg_idx}")
        else:
            raise ValueError(f"Invalid message_type: {message_type}")
    
    def _create_langchain_message(
        self,
        message_type: str,
        content: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> Any:
        """Create appropriate LangChain message object based on message type.
        
        Extracted from MycelianMemoryAgent._create_langchain_message()
        """
        if message_type == "system":
            # Control messages trigger actions
            return _SysMsg(
                content=content,
                message_type="control"  # Mark as control command
            )
        elif message_type == "conversation":
            # Conversation messages with index for tracking
            return _ChatMsg(
                role=role,
                content=content,
                idx=msg_idx  # Add msg_idx as metadata field
            )
        else:
            raise ValueError(f"Invalid message_type: {message_type}")
    
    def _set_logging_context(
        self,
        thread_id: str,
        message_type: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> None:
        """Set the current message context for debug logging and tool call tracking.
        
        Extracted from MycelianMemoryAgent._set_logging_context()
        """
        context = {
            "thread_id": thread_id,
            "type": message_type,
            "role": role,
            "msg_idx": msg_idx,
            "memory_id": self.config.get('memory_id')
        }
        self._current_msg_ctx = context
        
        # Set context in tool logger if available
        if self._tool_logger:
            self._tool_logger.set_logging_context(context)
    
    def _log_message(
        self,
        message_type: str,
        content: str,
        thread_id: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> None:
        """Delegate message logging to MessageLogger.
        
        Extracted from MycelianMemoryAgent._log_message()
        """
        if self._message_logger:
            self._message_logger.log_message(message_type, content, thread_id, role, msg_idx)
    
    def _invoke_agent(self, lc_msg: Any, thread_id: str) -> Any:
        """Invoke the agent with the message and handle retries.
        
        Extracted from MycelianMemoryAgent._invoke_agent()
        """
        config = {
            "configurable": {
                "thread_id": thread_id,
                "recursion_limit": self.RECURSION_LIMIT
            }
        }
        payload = {"messages": [lc_msg]}
        
        def _invoke_async():
            async def _run():
                return await asyncio.wait_for(
                    self.agent.graph.ainvoke(payload, config),
                    timeout=self.AGENT_TIMEOUT
                )
            return asyncio.run(_run())
        
        try:
            return invoke_with_backoff(
                _invoke_async,
                debug=self.config.get('debug', False),
                log=lambda m: logging.getLogger("lme.agent").info(str(m)),
            )
        except Exception as e:
            if self._message_logger:
                self._message_logger.log_error(str(e))
            raise
    
    def _log_result(self, thread_id: str) -> None:
        """Log the result summary after agent invocation.
        
        Extracted from MycelianMemoryAgent._log_result()
        """
        try:
            cfg = {"configurable": {"thread_id": thread_id}}
            get_state = getattr(self.agent.graph, "get_state", None)
            if callable(get_state):
                state = self.agent.graph.get_state(cfg)
                msg_count = len(state.get("messages", [])) if isinstance(state, dict) else None
                if self._message_logger:
                    self._message_logger.log_result(thread_id, msg_count)
        except (AttributeError, RuntimeError, TypeError):
            pass  # Silent failure for logging
    
    def _dump_agent_state(self, thread_id: str) -> None:
        """Delegate state dumping to StateDebugger.
        
        Extracted from MycelianMemoryAgent._dump_agent_state()
        """
        if self._state_debugger:
            self._state_debugger.dump_state(self.agent.graph, thread_id)
    
    def close(self) -> None:
        """Clean up resources.
        
        Delegates to Agent for cleanup.
        """
        if self.agent:
            self.agent.close()
    
    @property
    def mcp_client(self) -> Any:
        """Access the MCP client for administrative operations.
        
        Returns:
            The MCP client instance
        """
        return self.helpers.get('mcp_client')
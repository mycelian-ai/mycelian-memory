from typing import Any, Dict, List, Optional, TextIO
import os
import asyncio
import logging
import json
import types
from src.tenacious_agent_invoker import invoke_with_backoff
from langchain.chat_models import init_chat_model  # type: ignore
from langchain_mcp_adapters.client import MultiServerMCPClient  # type: ignore
from langgraph.prebuilt import ToolNode  # type: ignore
from langgraph.checkpoint.memory import InMemorySaver  # type: ignore
from langgraph.graph import StateGraph, MessagesState, START, END  # type: ignore
from langchain_core.messages import SystemMessage, ChatMessage, AIMessage, ToolMessage  # type: ignore
from langchain_core.messages import SystemMessage as _SysMsg  # type: ignore
from langchain_core.messages import ChatMessage as _ChatMsg  # type: ignore

# Import new specialized classes
from src.prompt_builder import PromptBuilder
from src.tool_logger import ToolLogger
from src.graph_builder import GraphBuilder
from src.state_debugger import StateDebugger


class MycelianMemoryAgent:
    """LangGraph prebuilt ReAct agent wired to MCP tools over HTTP.

    Provides a simple `run_session(messages)` interface compatible with the
    runner. Tools are bound to a specific (vault_id, memory_id) via closures.
    """
    
    # Constants
    FLUSH_INTERVAL = 6
    ENTRIES_LIMIT = 10
    LOG_TRUNCATION_LIMIT = 1500
    LOG_TRUNCATION_SUFFIX_LEN = 3  # Length of "..."
    TOOL_NAME_LOG_LIMIT = 120
    MESSAGE_LOG_LIMIT = 200
    MESSAGE_PREVIEW_LIMIT = 197
    DEFAULT_MCP_URL = "http://localhost:11546/mcp"
    SERVER_NAME = "mycelian-memory-streamable"
    MEMORY_TYPE = "chat"  # Default memory type for prompts
    RECURSION_LIMIT = 12  # Maximum recursion depth for agent
    RULES_FILE = "context_summary_rules.md"
    WRAPPED_TOOL_ATTR = "_lme_wrapped"  # Attribute to mark wrapped tools
    AGENT_TIMEOUT = 180.0  # Timeout for agent invocation in seconds

    def __init__(self, model_id: str, vault_id: str, memory_id: str, max_tool_calls_per_turn: int = 5, server_url: Optional[str] = None, debug: bool = False):
        self._initialize_config(model_id, vault_id, memory_id, max_tool_calls_per_turn, server_url, debug)
        self._setup_mcp_client()
        self._setup_components()
        self._setup_tools()
        self._build_agent_graph()
    
    def _initialize_config(self, model_id: str, vault_id: str, memory_id: str, max_tool_calls_per_turn: int, server_url: Optional[str], debug: bool) -> None:
        """Initialize basic configuration and instance variables."""
        self._model_id = model_id
        self._max_tools = max_tool_calls_per_turn
        self._server_url = server_url or os.environ.get("MYCELIAN_MCP_URL", self.DEFAULT_MCP_URL)
        self._debug = bool(debug)
        self._log_stream: Optional[TextIO] = None
        self._current_msg_ctx: Dict[str, Any] = {}
        self._vault_id = str(vault_id)
        self._memory_id = str(memory_id)
        self._agent = None
        self._checkpointer = None
    
    def _setup_mcp_client(self) -> None:
        """Initialize the MCP client for communication."""
        self._server_name = self.SERVER_NAME
        self._mcp = MultiServerMCPClient({
            self._server_name: {"url": self._server_url, "transport": "streamable_http"}
        })
    
    def _setup_components(self) -> None:
        """Initialize specialized components for prompt building and logging."""
        # Initialize prompt builder
        self._prompt_builder = PromptBuilder(
            vault_id=self._vault_id,
            memory_id=self._memory_id,
            mcp_client=self._mcp,
            server_name=self._server_name
        )
        self._system_prompt = self._prompt_builder.build_system_prompt()
        
        # Initialize tool logger
        self._tool_logger = ToolLogger(debug=self._debug)
        
        # Initialize state debugger
        self._state_debugger = StateDebugger(debug=self._debug)


    
    def _setup_tools(self) -> None:
        """Load MCP tools and initialize LLM."""
        # Load MCP tools dynamically
        async def _load_tools():
            return await self._mcp.get_tools()  # type: ignore[attr-defined]

        self._agent_tools = asyncio.run(_load_tools())
        
        # Initialize LLM
        self._llm = init_chat_model(self._model_id)
        
        # Setup checkpointer
        self._checkpointer = InMemorySaver()
        
        # Apply logging wrappers if debug is enabled
        if self._debug:
            for tool in self._agent_tools:
                try:
                    self._tool_logger.attach_logging_to_tool(tool)
                except Exception:
                    pass

    
    def _build_agent_graph(self) -> None:
        """Build the LangGraph agent with tools and prompts."""
        # Initialize graph builder
        graph_builder = GraphBuilder(
            llm=self._llm,
            tools=self._agent_tools,
            system_prompt=self._system_prompt,
            tool_logger=self._tool_logger
        )
        
        # Build and compile the agent
        self._agent = graph_builder.build_agent()
        self._checkpointer = graph_builder.checkpointer

    def set_log_stream(self, stream: Optional[TextIO]) -> None:
        self._log_stream = stream

    def close(self) -> None:
        try:
            close_fn = getattr(self._mcp, "close", None)
            if callable(close_fn):
                close_fn()
        except (AttributeError, RuntimeError):
            # Best-effort shutdown for local runs
            pass

    def _dump_agent_state(self, thread_id: str) -> None:
        """Delegate state dumping to StateDebugger."""
        self._state_debugger.dump_state(self._agent, thread_id)

    
    # Removed bind_memory in favor of constructor-provided IDs

    def _validate_message(self, type: str, content: str, role: str = None, msg_idx: int = None) -> None:
        """Validate message parameters according to message_spec.md."""
        if self._agent is None:
            raise RuntimeError("Agent not initialized.")
        
        if not isinstance(content, str) or not content.strip():
            raise ValueError("content must be non-empty string")
        
        if type == "system":
            if role is not None or msg_idx is not None:
                raise ValueError("System messages must not have role or msg_idx")
        elif type == "conversation":
            if role is None or msg_idx is None:
                raise ValueError("Conversation messages require role and msg_idx")
            if role not in ("user", "assistant"):
                raise ValueError(f"Invalid role: {role}")
            if not isinstance(msg_idx, int) or msg_idx < 1:
                raise ValueError(f"msg_idx must be positive integer, got: {msg_idx}")
        else:
            raise ValueError(f"Invalid type: {type}")
    
    def _create_langchain_message(self, type: str, content: str, role: str = None) -> Any:
        """Create appropriate LangChain message object based on type."""
        if type == "system":
            return _SysMsg(content=content)
        elif type == "conversation":
            return _ChatMsg(role=role, content=content)
        else:
            raise ValueError(f"Invalid type: {type}")
    
    def _update_context(self, thread_id: str, type: str, role: str = None, msg_idx: int = None) -> None:
        """Update the current message context for logging."""
        context = {
            "thread_id": thread_id,
            "type": type,
            "role": role,
            "msg_idx": msg_idx,
            "memory_id": self._memory_id
        }
        self._current_msg_ctx = context
        self._tool_logger.set_context(context)
    
    def _log_message(self, type: str, content: str, thread_id: str, role: str = None, msg_idx: int = None) -> None:
        """Log the incoming message for debugging."""
        agent_log = logging.getLogger("lme.agent")
        try:
            if type == "system":
                prev = (content or "").strip()
                if len(prev) > self.MESSAGE_LOG_LIMIT:
                    prev = prev[:self.MESSAGE_PREVIEW_LIMIT] + "..."
                agent_log.info("SYS thread_id=%s content=%s", thread_id, prev)
            else:
                txt = (content or "").strip()
                prev = txt[:self.MESSAGE_PREVIEW_LIMIT] + "..." if len(txt) > self.MESSAGE_LOG_LIMIT else txt
                agent_log.info(
                    "MSG thread_id=%s msg_idx=%d role=%s chars=%d preview=%s",
                    thread_id,
                    int(msg_idx or 0),
                    str(role or ""),
                    len(content or ""),
                    prev,
                )
        except (AttributeError, RuntimeError, TypeError):
            pass
    
    def _invoke_agent(self, lc_msg: Any, thread_id: str) -> Any:
        """Invoke the agent with the message and handle retries."""
        config = {"configurable": {"thread_id": thread_id, "recursion_limit": self.RECURSION_LIMIT}}
        payload = {"messages": [lc_msg]}
        
        def _invoke_async():
            async def _run():
                return await asyncio.wait_for(self._agent.ainvoke(payload, config), timeout=self.AGENT_TIMEOUT)
            return asyncio.run(_run())
        
        try:
            return invoke_with_backoff(
                _invoke_async,
                debug=self._debug,
                log=lambda m: logging.getLogger("lme.agent").info(str(m)),
            )
        except Exception as e:
            logging.getLogger("lme.agent").info("AINVOKE_ERROR %s", str(e)[:self.MESSAGE_LOG_LIMIT])
            raise
    
    def _log_result(self, thread_id: str) -> None:
        """Log the result summary after agent invocation."""
        agent_log = logging.getLogger("lme.agent")
        try:
            cfg = {"configurable": {"thread_id": thread_id}}
            get_state = getattr(self._agent, "get_state", None)
            if callable(get_state):
                state = self._agent.get_state(cfg)
                msg_count = len(state.get("messages", [])) if isinstance(state, dict) else None
                if msg_count is not None:
                    agent_log.info("RESULT thread_id=%s messages=%s", thread_id, msg_count)
        except (AttributeError, RuntimeError, TypeError):
            pass
    
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
        # Validate input parameters
        self._validate_message(type, content, role, msg_idx)
        
        # Create the appropriate message object
        lc_msg = self._create_langchain_message(type, content, role)
        
        # Update context for logging
        self._update_context(thread_id, type, role, msg_idx)
        
        # Log the incoming message
        self._log_message(type, content, thread_id, role, msg_idx)
        
        # Invoke the agent
        result = self._invoke_agent(lc_msg, thread_id)
        
        # Log the result
        self._log_result(thread_id)
        
        # Dump full agent state snapshot after each message in debug mode
        if self._debug:
            self._dump_agent_state(thread_id)

        return result


 


def build_agent(model_id: str, vault_id: str, memory_id: str, max_tool_calls_per_turn: int = 5, provider_type: str | None = None, debug: bool = False) -> MycelianMemoryAgent:
    # OpenAI-only simplification
    resolved_model = model_id if str(model_id).startswith("openai:") else f"openai:{model_id}"
    return MycelianMemoryAgent(model_id=resolved_model, vault_id=vault_id, memory_id=memory_id, max_tool_calls_per_turn=max_tool_calls_per_turn, debug=debug)

"""Agent - Defines agent behavior through LangGraph topology."""

from typing import Any, Dict, List, Optional

from langchain_core.messages import AIMessage, ChatMessage, SystemMessage, ToolMessage  # type: ignore
from langgraph.graph import END, START, MessagesState, StateGraph  # type: ignore
from langgraph.prebuilt import ToolNode  # type: ignore


class Agent:
    """Defines the agent's behavior through graph topology.
    
    This class is responsible for:
    - Defining the graph structure (nodes, edges, conditions)
    - Compiling the graph into an executable agent
    - Exposing the compiled graph for execution
    
    The graph topology IS the agent's logic - it defines what the agent does.
    Extracted and refactored from GraphBuilder to own its behavior.
    """
    
    # Constants
    RECURSION_LIMIT = 12
    
    def __init__(
        self,
        llm: Any,
        tools: list,
        system_prompt: str,
        checkpointer: Any,
        config: Dict[str, Any],
        helpers: Optional[Dict[str, Any]] = None
    ):
        """Initialize Agent with dependencies from builder.
        
        Args:
            llm: Language model instance
            tools: List of available tools
            system_prompt: System prompt for the agent
            checkpointer: Checkpointer for conversation state
            config: Configuration dictionary from builder
            helpers: Helper objects for logging/debugging
        """
        self.llm = llm
        self.tools = tools
        self.system_prompt = system_prompt
        self.checkpointer = checkpointer
        self.config = config
        self.helpers = helpers or {}
        
        # Build and compile the graph
        self._build_graph()
        self.compiled_graph = self._compile()
    
    def _build_graph(self) -> None:
        """Define the graph topology - nodes, edges, and conditions.
        
        This method defines the agent's behavior by constructing the graph.
        Extracted from GraphBuilder.build_agent()
        """
        # Create the graph with message state
        self.graph_builder = StateGraph(MessagesState)
        
        # Bind tools to LLM
        self.llm_with_tools = self.llm.bind_tools(self.tools)
        
        # Add nodes
        self.graph_builder.add_node("llm_call", self._llm_node)
        self.graph_builder.add_node("tools", ToolNode(self.tools))
        
        # Add edges
        self.graph_builder.add_edge(START, "llm_call")
        self.graph_builder.add_conditional_edges(
            "llm_call",
            self._should_continue,
            {"tools": "tools", END: END}
        )
        self.graph_builder.add_edge("tools", "llm_call")
    
    def _llm_node(self, state: MessagesState) -> Dict[str, List[AIMessage]]:
        """LLM node that processes messages and decides on actions.
        
        Following the official LangGraph pattern - simpler is better.
        
        Args:
            state: Current conversation state
            
        Returns:
            Updated state with LLM response
        """
        # Build input with system prompt + all messages
        model_input = [SystemMessage(content=self.system_prompt)]
        
        # Add conversation history (curated for model)
        curated = self._curate_for_model(state["messages"])
        model_input += curated
        
        # Log LLM call if debugging
        tool_logger = self.helpers.get("tool_logger")
        if self.config.get('debug') and tool_logger:
            tool_logger.log_llm_call(len(model_input))
        
        # Invoke LLM
        resp = self.llm_with_tools.invoke(model_input)
        
        # Log tool calls if debugging
        if self.config.get('debug') and tool_logger:
            tool_logger.log_tool_calls(resp)
        
        # Return the AI's response (with or without tool calls)
        return {
            "messages": [
                AIMessage(
                    content=getattr(resp, "content", ""),
                    tool_calls=getattr(resp, "tool_calls", []),
                    additional_kwargs=getattr(resp, "additional_kwargs", {}),
                )
            ]
        }
    
    
    def _should_continue(self, state: MessagesState) -> str:
        """Determine whether to continue with tool execution or end.
        
        Extracted from GraphBuilder.should_continue()
        
        Args:
            state: Current conversation state
            
        Returns:
            "tools" if tools should be called, END otherwise
        """
        messages = state["messages"]
        if not messages:
            return END
        
        last_message = messages[-1]
        
        # Check if the last message has tool calls
        if getattr(last_message, "tool_calls", None):
            return "tools"
        
        return END
    
    def _curate_for_model(self, messages: List[Any]) -> List[Any]:
        """Filter messages for model input, keeping only relevant message types.
        
        This helps the LLM understand the conversation flow and what's been processed.

        Args:
            messages: List of messages to filter

        Returns:
            Curated list of messages for the model
        """
        curated: List[Any] = []
        
        for message in messages:
            # Keep system control messages (SESSION_START, etc) 
            if isinstance(message, SystemMessage):
                # Only keep control messages, not prompts
                if getattr(message, 'message_type', None) == 'control':
                    curated.append(message)
            
            # Keep conversation messages
            elif isinstance(message, ChatMessage):
                curated.append(message)
            
            # Keep AI messages with tool calls (shows what's been done)
            elif isinstance(message, AIMessage):
                if getattr(message, "tool_calls", None):
                    curated.append(message)
            
            # Keep tool results
            elif isinstance(message, ToolMessage):
                curated.append(message)
        
        return curated
    
    def _compile(self) -> Any:
        """Compile the graph into an executable agent.
        
        Returns:
            Compiled LangGraph agent
        """
        return self.graph_builder.compile(checkpointer=self.checkpointer)
    
    @property
    def graph(self) -> Any:
        """Access the compiled graph for execution.
        
        Returns:
            The compiled LangGraph agent
        """
        return self.compiled_graph
    
    def close(self) -> None:
        """Clean up resources if needed.
        
        Closes MCP client if available in helpers.
        """
        mcp_client = self.helpers.get('mcp_client')
        if mcp_client:
            try:
                close_fn = getattr(mcp_client, "close", None)
                if callable(close_fn):
                    close_fn()
            except (AttributeError, RuntimeError):
                pass  # Best effort cleanup
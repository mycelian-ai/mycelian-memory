"""MycelianMemoryAgent - Clean implementation following the control-based protocol."""

import asyncio
import json
import logging
import uuid
from datetime import datetime
from typing import TypedDict, Annotated, Sequence, Optional, Union, Dict, Any, List
from typing_extensions import TypedDict

from langchain_core.messages import BaseMessage, AIMessage, ToolMessage, ChatMessage
from langgraph.graph import StateGraph, END
from langgraph.graph.message import add_messages
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver

from .control_state import ControlState
from src.async_utils import run as run_async

# Setup logger for audit trail
# Using "lme.agent" to integrate with benchmarker's logging system
logger = logging.getLogger("lme.agent")
DEFAULT_AGENT_LOGGER = "lme.agent"

# Define allowed tools for each control state and last tool combination
ALLOWED_TOOLS = {
    ControlState.START_SESSION: {
        None: ["get_context"],              # No tool executed yet -> get_context
        "get_context": ["list_entries"],    # After get_context -> list_entries
        "list_entries": []                  # After list_entries -> done
    },
    ControlState.PROCESS_MESSAGE: {
        None: ["add_entry"],                # No tool executed yet -> add_entry (LLM call)
        "add_entry": []                     # After add_entry -> done
    },
    ControlState.FLUSH: {
        None: ["await_consistency"],        # No tool executed yet -> await_consistency
        "await_consistency": ["put_context"], # After await_consistency -> put_context (LLM call)
        "put_context": []                   # After put_context -> done
    },
    ControlState.END_SESSION: {
        None: ["await_consistency"],        # No tool executed yet -> await_consistency
        "await_consistency": ["put_context"],  # After await_consistency -> put_context (LLM call)
        "put_context": []                   # After put_context -> done
    }
}


class AgentState(TypedDict):
    """State structure for the agent."""
    conversation_history: Annotated[Sequence[ChatMessage], add_messages]  # Accumulates across invocations
    to_process: Sequence[ChatMessage]  # Current ChatMessage to process (replaced each invocation)
    tool_history: Sequence[Union[AIMessage, ToolMessage]]  # Tool flow for current invocation (not checkpointed)
    control: ControlState  # Control state driving execution
    messages: Sequence[BaseMessage]  # Tool I/O buffer for ToolNode (per-invocation only, no accumulation)


class MycelianMemoryAgent:
    """Agent that observes conversations and manages memory through MCP tools.

    Uses checkpointer for state persistence across invocations.
    Each thread_id maintains its own conversation state.
    """

    def __init__(self, llm, tools: list, prompts: Dict[str, str],
                 vault_id: str, memory_id: str, context_only: bool = True,
                 logger: Optional[logging.Logger] = None):
        """Initialize the agent.

        Args:
            llm: Language model with tool calling capability
            tools: List of MCP tools
            prompts: Dictionary of prompt templates from MCP
            vault_id: Vault identifier for memory storage
            memory_id: Memory identifier for this conversation
            context_only: If True, skip add_entry and flush operations (default: True)
        """
        self.llm = llm
        self.tools = tools
        self.prompts = prompts
        self.vault_id = vault_id
        self.memory_id = memory_id
        self.context_only = context_only
        self.logger = logger or logging.getLogger(f"{DEFAULT_AGENT_LOGGER}.{memory_id}")
        try:
            self.logger.info(json.dumps({
                "event": "agent_init",
                "timestamp": datetime.utcnow().isoformat(),
                "vault_id": vault_id,
                "memory_id": memory_id,
                "context_only": context_only,
                "tools_available": [getattr(t, 'name', str(type(t).__name__)) for t in tools]
            }))
        except (TypeError, AttributeError):
            # Handle mock objects in tests
            pass

        # Bind tools to LLM for tool calling
        self.llm_with_tools = llm.bind_tools(tools)

        # Create tool node for executing tool calls
        self.tool_node = ToolNode(tools)

        # Create checkpointer for state persistence
        self.checkpointer = MemorySaver()

        # Build the graph
        self.graph = self._build_graph()

    def _build_graph(self):
        """Build the state graph with observe and tools nodes."""
        workflow = StateGraph(AgentState)

        # Add nodes
        workflow.add_node("observe", self.observe)
        workflow.add_node("tools", self.tool_node)

        # Set entry point
        workflow.set_entry_point("observe")

        # Add conditional edge from observe
        workflow.add_conditional_edges(
            "observe",
            self.should_execute_tools,
            {
                "execute": "tools",
                "end": END
            }
        )

        # Tools go back to observe to check for next action
        workflow.add_edge("tools", "observe")

        # Compile with checkpointer
        return workflow.compile(checkpointer=self.checkpointer)

    def _get_tool_by_name(self, name: str):
        """Get a tool by its name from the tools list."""
        for tool in self.tools:
            if hasattr(tool, 'name') and tool.name == name:
                return tool
        raise ValueError(f"Tool '{name}' not found in tools list")

    def _invoke_llm_with_retry(self, messages):
        """Invoke LLM with automatic retry on transient errors.

        LangChain's built-in retry handles rate limits and transient failures.
        """
        # Generate unique invocation ID for tracking
        invocation_id = str(uuid.uuid4())[:8]

        # LangChain handles retry internally with max_retries parameter
        response = self.llm_with_tools.invoke(messages)

        # Log tool calls with invocation ID
        self._log_llm_tool_calls(response, invocation_id)
        return response

    def _filter_tool_calls(self, response: AIMessage, control: ControlState, last_tool: Optional[str]) -> AIMessage:
        """Filter tool calls to only allowed ones for current state.

        Logs compliance violations when unexpected tools are filtered.
        """
        if not response.tool_calls:
            return response

        # Get allowed tools for current state and last tool
        allowed = ALLOWED_TOOLS.get(control, {}).get(last_tool, [])

        # Filter tool calls
        original_calls = response.tool_calls[:]
        filtered_calls = []
        removed_tools = []

        for call in original_calls:
            # Handle both dict and object forms of tool_calls
            tool_name = call.get('name') if isinstance(call, dict) else getattr(call, 'name', None)
            if tool_name in allowed:
                filtered_calls.append(call)
            else:
                removed_tools.append(tool_name)

        # Log compliance violation if tools were filtered
        if removed_tools:
            logger.warning(json.dumps({
                "event": "compliance_violation",
                "timestamp": datetime.utcnow().isoformat(),
                "violation_type": "unexpected_tool_calls",
                "control": control.value,
                "last_tool": last_tool,
                "allowed_tools": allowed,
                "removed_tools": removed_tools,
                "kept_tools": [call.get('name') if isinstance(call, dict) else getattr(call, 'name', None)
                               for call in filtered_calls]
            }))

        # Update response with filtered calls
        response.tool_calls = filtered_calls
        return response

    def _check_put_context_called(self, tool_history: List[BaseMessage]) -> bool:
        """Check if put_context was already called in recent tool history.

        Args:
            tool_history: The tool history to check

        Returns:
            True if put_context was found, False otherwise
        """
        for msg in reversed(tool_history):
            if isinstance(msg, AIMessage) and hasattr(msg, 'tool_calls'):
                for tool_call in (msg.tool_calls or []):
                    tool_name = tool_call.get('name') if isinstance(tool_call, dict) else getattr(tool_call, 'name', None)
                    if tool_name == 'put_context':
                        return True
            # Stop at the first await_consistency ToolMessage (our checkpoint)
            if isinstance(msg, ToolMessage) and msg.name == 'await_consistency':
                break
        return False

    def observe(self, state: AgentState) -> Dict[str, Any]:
        """Main observation node that determines next action based on control state.

        This method implements the control-based routing logic for different
        operation sequences.
        """
        tool_history = list(state.get("tool_history", []))
        control = state.get("control")
        conversation_history = state.get("conversation_history", [])
        to_process = state.get("to_process", [])
        messages = state.get("messages", [])

        # Copy any new ToolMessages from messages to tool_history
        # ToolNode adds results to messages, we need them in tool_history for tracking
        for msg in messages:
            if isinstance(msg, ToolMessage) and msg not in tool_history:
                tool_history.append(msg)

        self.logger.info(json.dumps({
                "event": "observe_start",
                "timestamp": datetime.utcnow().isoformat(),
                "control": control.value if control else None,
                "tool_history_count": len(tool_history),
                "conversation_count": len(conversation_history),
                "to_process_count": len(to_process)
            }))

        # Determine last tool executed by checking tool_history ONLY
        # tool_history accumulates within this invocation and resets between invocations
        # We do NOT check messages to avoid cross-invocation pollution
        last_tool = None

        if tool_history:
            for msg in reversed(tool_history):
                if isinstance(msg, ToolMessage):
                    last_tool = msg.name
                    # Rely on MCP/ToolNode to raise on actual tool failures; do not
                    # attempt to infer errors by scanning payload text.
                    break

        # START_SESSION: Create tool calls for ToolNode to execute
        if control == ControlState.START_SESSION:
            if last_tool is None:
                # First tool: await_consistency to ensure previous writes are complete
                args = {"memory_id": self.memory_id}

                self.logger.info(json.dumps({
                        "event": "creating_tool_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "tool": "await_consistency",
                        "args": args
                    }))

                # Create AIMessage with tool_calls for ToolNode to process
                tool_call = {
                    "id": "await_consistency_call_start",
                    "name": "await_consistency",
                    "args": args
                }

                ai_msg = AIMessage(
                    content="Ensuring previous session's writes are complete.",
                    tool_calls=[tool_call]
                )

                # Return with updated tool_history (includes copied ToolMessages)
                return {"tool_history": tool_history + [ai_msg], "messages": [ai_msg]}

            elif last_tool == "await_consistency":
                # Second tool: get_context
                args = {
                    "vault_id": self.vault_id,
                    "memory_id": self.memory_id
                }

                self.logger.info(json.dumps({
                        "event": "creating_tool_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "tool": "get_context",
                        "args": args
                    }))

                # Create AIMessage with tool_calls for ToolNode to process
                tool_call = {
                    "id": "get_context_call_1",
                    "name": "get_context",
                    "args": args
                }

                ai_msg = AIMessage(
                    content="Retrieving stored context from previous sessions.",
                    tool_calls=[tool_call]
                )

                # Return with updated tool_history (includes copied ToolMessages)
                return {"tool_history": tool_history + [ai_msg], "messages": [ai_msg]}

            elif last_tool == "get_context":
                # Third tool: list_entries
                args = {
                    "vault_id": self.vault_id,
                    "memory_id": self.memory_id,
                    "limit": 10
                }

                self.logger.info(json.dumps({
                        "event": "creating_tool_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "tool": "list_entries",
                        "args": args
                    }))

                # Create AIMessage with tool_calls for ToolNode to process
                tool_call = {
                    "id": "list_entries_call_1",
                    "name": "list_entries",
                    "args": args
                }

                ai_msg = AIMessage(
                    content="Fetching the 10 most recent entries.",
                    tool_calls=[tool_call]
                )

                # Return with updated tool_history (includes copied ToolMessages)
                return {"tool_history": tool_history + [ai_msg], "messages": [ai_msg]}

            elif last_tool == "list_entries":
                # All three tools completed, extract results and finish
                # Find the context and entries from tool_history
                context_text = ""
                entries_text = ""

                for msg in tool_history:
                    if isinstance(msg, ToolMessage):
                        if msg.name == "get_context":
                            context_text = msg.content
                            # Log the retrieved context content
                            self.logger.info(json.dumps({
                                "event": "get_context_retrieved",
                                "timestamp": datetime.utcnow().isoformat(),
                                "memory_id": self.memory_id,
                                "context_length": len(context_text),
                                "context_preview": context_text[:500] if context_text else "[empty]"
                            }))
                        elif msg.name == "list_entries":
                            entries_text = msg.content

                # Add retrieved context (prefixed with [previous_context]) and entries to conversation_history
                context_msg = ChatMessage(
                    role="system",
                    content=f"[previous_context]\n{context_text}"
                )
                entries_msg = ChatMessage(
                    role="system",
                    content=f"Recent entries:\n{entries_text}"
                )

                # Mark complete and update conversation history
                return {
                    # FIX: Append to existing conversation_history instead of replacing it
                    "conversation_history": conversation_history + [context_msg, entries_msg],
                    # Return with updated tool_history
                    "tool_history": tool_history + [AIMessage(content="Session started.")]
                }

        # PROCESS_MESSAGE sequence: add_entry only (or just accumulate if context_only)
        elif control == ControlState.PROCESS_MESSAGE:
            # If context_only mode, just add to conversation history without calling add_entry
            if self.context_only:
                if not to_process:
                    raise ValueError("No message to process in PROCESS_MESSAGE state")

                self.logger.info(json.dumps({
                    "event": "context_only_accumulate",
                    "timestamp": datetime.utcnow().isoformat(),
                    "message_role": to_process[0].role if to_process else None,
                    "message_preview": to_process[0].content[:200] if to_process and to_process[0].content else None
                }))

                # Just add to conversation history and return
                # FIX: Append to existing conversation_history instead of replacing it
                return {
                    "conversation_history": conversation_history + to_process,
                    "tool_history": tool_history + [AIMessage(content="Message accumulated (context-only).")]
                }

            # Normal mode: process with add_entry
            # Only relevant tool for this state is add_entry
            if last_tool not in [None, "add_entry"]:
                # Ignore tools from other control states, treat as starting fresh
                last_tool = None

            if last_tool is None:
                # Only tool: add_entry (needs LLM for summary)
                if not to_process:
                    raise ValueError("No message to process in PROCESS_MESSAGE state")

                self.logger.info(json.dumps({
                        "event": "llm_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "add_entry",
                        "message_role": to_process[0].role,
                        "message_preview": to_process[0].content[:200] if to_process[0].content else None
                    }))

                prompt = build_add_entry_prompt(
                    conversation_history, to_process[0], self.prompts,
                    self.vault_id, self.memory_id
                )
                llm_messages = [
                    {"role": "system", "content": prompt},
                    {"role": "user", "content": "Execute the required operation."}
                ]
                # Log the full message array being sent to the model (add_entry)
                try:
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "add_entry",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages),
                        "messages": llm_messages
                    }))
                except (TypeError, ValueError):
                    # Fallback: omit messages if they are not JSON serializable in unexpected cases
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "add_entry",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages)
                    }))
                response = self._invoke_llm_with_retry(llm_messages)
                # Filter tool calls to only allowed ones
                response = self._filter_tool_calls(response, control, last_tool)
                # Also add to messages for ToolNode
                # Manually accumulate tool_history
                current_history = list(state.get("tool_history", []))
                return {"tool_history": current_history + [response], "messages": [response]}

            elif last_tool == "add_entry":
                # Complete
                # Complete - return with updated tool_history
                return {"tool_history": tool_history + [AIMessage(content="Message processed.")]}

        # FLUSH sequence: await_consistency → put_context (skip if context_only)
        elif control == ControlState.FLUSH:
            # Skip flush entirely in context_only mode
            if self.context_only:
                self.logger.info(json.dumps({
                    "event": "flush_skipped",
                    "timestamp": datetime.utcnow().isoformat(),
                    "reason": "context_only_mode"
                }))
                return {"tool_history": tool_history + [AIMessage(content="Flush skipped (context-only).")]}

            # Check if put_context was already called in this flush
            # This handles the case where LLM returns multiple tools including put_context
            if self._check_put_context_called(tool_history):
                return {"tool_history": tool_history + [AIMessage(content="Flushed to context.")]}

            # Only relevant tools for this state
            if last_tool not in [None, "await_consistency", "put_context"]:
                # Ignore tools from other control states, treat as starting fresh
                last_tool = None

            if last_tool is None:
                # First tool: await_consistency
                args = {"memory_id": self.memory_id}

                self.logger.info(json.dumps({
                        "event": "creating_tool_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "tool": "await_consistency",
                        "args": args
                    }))

                # Create AIMessage with tool_calls for ToolNode to process
                tool_call = {
                    "id": "await_consistency_call_1",
                    "name": "await_consistency",
                    "args": args
                }

                ai_msg = AIMessage(
                    content="Ensuring all entries are persisted before updating context.",
                    tool_calls=[tool_call]
                )

                # Return with updated tool_history
                return {"tool_history": tool_history + [ai_msg], "messages": [ai_msg]}

            elif last_tool == "await_consistency":
                # Second tool: put_context (needs LLM for synthesis)
                self.logger.info(json.dumps({
                        "event": "llm_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context",
                        "conversation_count": len(conversation_history)
                    }))

                prompt = build_put_context_prompt(
                    conversation_history, self.prompts,
                    self.vault_id, self.memory_id, self.logger
                )
                llm_messages = [
                    {"role": "system", "content": prompt},
                    {"role": "user", "content": "Execute the required operation."}
                ]
                # Log the full message array being sent to the model (put_context during FLUSH)
                try:
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages),
                        "messages": llm_messages
                    }))
                except (TypeError, ValueError):
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages)
                    }))
                response = self._invoke_llm_with_retry(llm_messages)
                # Filter tool calls to only allowed ones
                response = self._filter_tool_calls(response, control, last_tool)
                # Also add to messages for ToolNode
                # Manually accumulate tool_history
                current_history = list(state.get("tool_history", []))
                return {"tool_history": current_history + [response], "messages": [response]}

            # This case is now handled at the beginning of FLUSH section
            # elif last_tool == "put_context":
            #     return {"tool_history": tool_history + [AIMessage(content="Flushed to context.")]}

        # END_SESSION sequence: await_consistency → put_context
        elif control == ControlState.END_SESSION:
            # Check if put_context was already called in this end session
            # This handles the case where LLM returns multiple tools including put_context
            if self._check_put_context_called(tool_history):
                return {"tool_history": tool_history + [AIMessage(content="Session ended.")]}

            # Only relevant tools for this state
            if last_tool not in [None, "await_consistency", "put_context"]:
                # Ignore tools from other control states, treat as starting fresh
                last_tool = None

            if last_tool is None:
                # First tool: await_consistency
                args = {"memory_id": self.memory_id}

                self.logger.info(json.dumps({
                        "event": "creating_tool_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "tool": "await_consistency",
                        "args": args
                    }))

                # Create AIMessage with tool_calls for ToolNode to process
                tool_call = {
                    "id": "await_consistency_call_2",
                    "name": "await_consistency",
                    "args": args
                }

                ai_msg = AIMessage(
                    content="Ensuring all entries are persisted before ending session.",
                    tool_calls=[tool_call]
                )

                # Return with updated tool_history (includes copied ToolMessages)
                return {"tool_history": tool_history + [ai_msg], "messages": [ai_msg]}

            elif last_tool == "await_consistency":
                # Second tool: put_context (needs LLM for synthesis)
                self.logger.info(json.dumps({
                        "event": "llm_call",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context_final",
                        "conversation_count": len(conversation_history)
                    }))

                prompt = build_put_context_prompt(
                    conversation_history, self.prompts,
                    self.vault_id, self.memory_id, self.logger
                )
                llm_messages = [
                    {"role": "system", "content": prompt},
                    {"role": "user", "content": "Execute the required operation."}
                ]
                # Log the full message array being sent to the model (final put_context)
                try:
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context_final",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages),
                        "messages": llm_messages
                    }))
                except (TypeError, ValueError):
                    self.logger.info(json.dumps({
                        "event": "llm_input_messages_full",
                        "timestamp": datetime.utcnow().isoformat(),
                        "purpose": "put_context_final",
                        "memory_id": self.memory_id,
                        "vault_id": self.vault_id,
                        "messages_count": len(llm_messages)
                    }))
                response = self._invoke_llm_with_retry(llm_messages)
                # Filter tool calls to only allowed ones
                response = self._filter_tool_calls(response, control, last_tool)
                # Also add to messages for ToolNode
                # Manually accumulate tool_history
                current_history = list(state.get("tool_history", []))
                return {"tool_history": current_history + [response], "messages": [response]}

            # This case is now handled at the beginning of END_SESSION section
            # elif last_tool == "put_context":
            #     return {"tool_history": tool_history + [AIMessage(content="Session ended.")]}

        # Should not reach here
        raise ValueError(f"Unexpected state: control={control}, last_tool={last_tool}")

    def should_execute_tools(self, state: AgentState) -> str:
        """Determine whether to execute tools or end.

        Returns:
            "execute" if there are tool calls to make
            "end" if the operation is complete
        """
        # Check messages field for tool calls (this is what ToolNode expects)
        messages = state.get("messages", [])
        if messages:
            last_msg = messages[-1]
            if isinstance(last_msg, AIMessage) and hasattr(last_msg, 'tool_calls') and last_msg.tool_calls:
                return "execute"

        # Also check tool_history for completion
        tool_history = state.get("tool_history", [])

        # If we have no tool history, we should continue
        if not tool_history:
            return "execute"

        # Get the last message in tool_history
        last_message = tool_history[-1] if tool_history else None

        # If the last message is an AIMessage with tool calls, execute them
        if isinstance(last_message, AIMessage) and hasattr(last_message, 'tool_calls') and last_message.tool_calls:
            return "execute"

        # If the last message is an AIMessage with content (completion message), end
        if isinstance(last_message, AIMessage) and last_message.content:
            return "end"

        # If the last message is a ToolMessage, continue processing
        if isinstance(last_message, ToolMessage):
            return "execute"

        # Default to ending
        return "end"

    def _log_llm_tool_calls(self, response, invocation_id: str) -> None:
        """Helper to log LLM tool calls from response.

        Args:
            response: The LLM response with tool calls
            invocation_id: Unique ID for this LLM invocation
        """
        if hasattr(response, 'tool_calls') and response.tool_calls:
            for idx, call in enumerate(response.tool_calls):
                tool_name = call.get('name', 'unknown') if isinstance(call, dict) else getattr(call, 'name', 'unknown')
                args = call.get('args', {}) if isinstance(call, dict) else getattr(call, 'args', {})

                # Enhanced logging for put_context to see what content is being stored
                if tool_name == 'put_context' and 'content' in args:
                    self.logger.info(json.dumps({
                        "event": "put_context_content",
                        "timestamp": datetime.utcnow().isoformat(),
                        "invocation_id": invocation_id,
                        "memory_id": args.get('memory_id', ''),
                        "content_length": len(args['content']),
                        "content_preview": args['content'][:1000] if args['content'] else "[empty]"
                    }))

                self.logger.info(json.dumps({
                    "event": "llm_tool_call",
                    "timestamp": datetime.utcnow().isoformat(),
                    "invocation_id": invocation_id,
                    "tool_index": idx,
                    "tool_count": len(response.tool_calls),
                    "tool": tool_name,
                    "args": args
                }))

    def invoke(self, control: ControlState, thread_id: str,
               to_process: Optional[ChatMessage] = None) -> Any:
        """Execute based on control state.

        Uses checkpointer for state persistence across invocations.
        Thread_id identifies the conversation thread for the checkpointer.

        Args:
            control: The control state determining which operation to perform
            thread_id: Unique identifier for this conversation thread
            to_process: Optional message to process (for PROCESS_MESSAGE operations)

        Returns:
            The result of the graph execution
        """
        self.logger.info(json.dumps({
                "event": "agent_invoke",
                "timestamp": datetime.utcnow().isoformat(),
                "control": control.value,
                "thread_id": thread_id,
                "has_message": bool(to_process),
                "message_role": to_process.role if to_process else None,
                "message_preview": to_process.content[:200] if to_process and to_process.content else None
            }))
        # Configuration for checkpointer
        config = {"configurable": {"thread_id": thread_id}}

        # Build initial state for this invocation
        initial_state = {
            "control": control,
            "to_process": [to_process] if to_process else [],
            "tool_history": [],  # Starts fresh each invocation
            "messages": []  # For ToolNode compatibility
        }

        # If processing a message, also add it to conversation_history
        # The checkpointer will accumulate it with previous messages
        if to_process:
            initial_state["conversation_history"] = [to_process]
        else:
            initial_state["conversation_history"] = []

        # Invoke the graph with the initial state and config
        # Single async bridge point
        async def _run_graph():
            return await self.graph.ainvoke(initial_state, config)

        result = run_async(_run_graph())

        self.logger.info(json.dumps({
                "event": "agent_complete",
                "timestamp": datetime.utcnow().isoformat(),
                "control": control.value,
                "thread_id": thread_id
            }))

        return result


# Prompt building functions

AGENT_PREFIX = """You are Mycelian's Memory Agent. Your job is to observe a conversation between a user and an AI Assistant and accurately store, retrieve and manage memories."""


def format_messages(messages: Sequence[ChatMessage]) -> str:
    """Format a sequence of ChatMessages for display in prompts."""
    formatted = []
    for msg in messages:
        formatted.append(f"Role: {msg.role}\nContent: {msg.content}")
    return "\n\n".join(formatted)


def build_add_entry_prompt(conversation_history: Sequence[ChatMessage],
                          to_process: ChatMessage,
                          prompts: Dict[str, str],
                          vault_id: str,
                          memory_id: str) -> str:
    """Build prompt for add_entry tool call.

    Args:
        conversation_history: Full conversation including retrieved context
        to_process: Current message to process
        prompts: Dictionary containing MCP prompt templates
        vault_id: The vault ID to use
        memory_id: The memory ID to use

    Returns:
        Formatted prompt for LLM to generate add_entry tool call
    """
    # Validate we have context (should always be present after START_SESSION)
    if not conversation_history:
        # Allow empty history for first message (edge case)
        context = "No previous context available."
    else:
        # Format previous conversation for context (all except current)
        context = format_messages(conversation_history)

    # Validate current message
    if not to_process:
        raise ValueError("No message to process")

    # Get prompts from MCP
    entry_capture_prompt = prompts.get("entry_capture_prompt", "")

    # Load our enhanced summary prompt that uses conversation context
    import os
    enhanced_prompt_path = os.path.join(
        os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
        "prompts", "chat", "summary_prompt.md"
    )

    if os.path.exists(enhanced_prompt_path):
        with open(enhanced_prompt_path, 'r') as f:
            summary_prompt = f.read()
    else:
        # Fallback to MCP prompt if enhanced version not found
        summary_prompt = prompts.get("summary_prompt", "")

    prompt = f"""{AGENT_PREFIX}

Current Operation: PROCESS_MESSAGE
Vault ID: {vault_id}
Memory ID: {memory_id}

Previous conversation context (including retrieved context from previous sessions):
{context}

Current message to process:
Role: {to_process.role}
Content: {to_process.content}

INSTRUCTION: Call the add_entry tool for this single message following the rules below.
Use vault_id="{vault_id}" and memory_id="{memory_id}" when calling the tool.
USE THE CONVERSATION CONTEXT to resolve all pronouns and references when creating the summary.

---
ENTRY CAPTURE RULES:
{entry_capture_prompt}

---
SUMMARY GENERATION RULES:
{summary_prompt}"""

    return prompt


def build_structured_conversation(messages: Sequence[ChatMessage],
                                 prompts: Dict[str, str]) -> list:
    """Separate messages into typed sections for clear context synthesis.

    Args:
        messages: Conversation messages potentially containing previous context
        prompts: Dictionary containing MCP prompt templates

    Returns:
        List of typed sections with clear boundaries
    """
    sections = []

    # Always add system prompt section first
    context_prompt = prompts.get("context_prompt", "")
    sections.append({
        "type": "system_prompt",
        "content": f"{AGENT_PREFIX}\n\n{context_prompt}" if context_prompt else AGENT_PREFIX
    })

    # Separate previous context from current session messages
    previous_messages = []
    current_messages = []

    for msg in messages:
        # Check if message starts with [previous_context] tag (only for system messages)
        # This prevents false positives from content that mentions the tag
        if msg.role == "system" and msg.content.startswith("[previous_context]"):
            previous_messages.append(msg)
        else:
            # Messages without the tag are current session
            # Only include user and assistant messages in current session
            if msg.role in ["user", "assistant"]:
                current_messages.append({
                    "role": msg.role,
                    "content": msg.content
                })
            elif msg.role == "system":
                # Other system messages go to previous section
                # (like "Recent entries" messages)
                previous_messages.append(msg)

    # Add previous context section if we have any
    if previous_messages:
        sections.append({
            "type": "previous_context",
            "content": format_messages(previous_messages)
        })

    # Add current session section if we have any
    if current_messages:
        sections.append({
            "type": "current_session_messages",
            "content": current_messages
        })

    return sections


def format_structured_prompt(sections: list) -> str:
    """Format typed sections into final prompt with clear markers.

    Args:
        sections: List of typed conversation sections

    Returns:
        Formatted prompt string with clear section boundaries
    """
    parts = []

    for section in sections:
        if section["type"] == "system_prompt":
            parts.append(f"=== SYSTEM INSTRUCTIONS ===\n{section['content']}")

        elif section["type"] == "previous_context":
            parts.append(f"=== PREVIOUS CONTEXT ===\n{section['content']}")

        elif section["type"] == "current_session_messages":
            parts.append("=== CURRENT SESSION ===")
            # Format each message in the current session
            for msg in section["content"]:
                parts.append(f"Role: {msg['role']}\nContent: {msg['content']}")

    return "\n\n".join(parts)


def build_put_context_prompt(conversation_history: Sequence[ChatMessage],
                            prompts: Dict[str, str],
                            vault_id: str,
                            memory_id: str,
                            logger: Optional[logging.Logger] = None) -> str:
    """Build prompt for put_context tool call.

    Args:
        conversation_history: Full conversation to synthesize
        prompts: Dictionary containing MCP prompt templates

    Returns:
        Formatted prompt for LLM to generate put_context tool call
    """
    # Validate we have conversation history
    if not conversation_history:
        raise ValueError("No conversation history to synthesize")

    # Build structured conversation sections
    sections = build_structured_conversation(conversation_history, prompts)

    # Format into final prompt with clear markers
    structured_prompt = format_structured_prompt(sections)

    # Log the conversation being sent to LLM for context synthesis
    (logger or logging.getLogger(f"{DEFAULT_AGENT_LOGGER}.{memory_id}")).info(json.dumps({
        "event": "put_context_llm_input",
        "timestamp": datetime.utcnow().isoformat(),
        "memory_id": memory_id,
        "conversation_count": len(conversation_history),
        "structured_sections": len(sections),
        "section_types": [s["type"] for s in sections]
    }))

    # Build the final prompt with structured conversation
    prompt = f"""{structured_prompt}

=== OPERATION DETAILS ===
Current Operation: CONTEXT_SYNTHESIS
Vault ID: {vault_id}
Memory ID: {memory_id}

=== CRITICAL INSTRUCTIONS ===
You MUST call ONLY the put_context tool - no other tools.
- Call put_context with vault_id="{vault_id}" and memory_id="{memory_id}"
- Do NOT call add_entry, await_consistency, or any other tools
- Return ONLY a single put_context tool call

=== OUTPUT RULES ===
STRICT OUTPUT RULES FOR put_context.content:
- Return ONLY the context body. Do NOT include any headings, titles, or prefaces such as "Context synthesized...", "Summary:", or similar.
- Do NOT include meta commentary or labels. Start directly with the synthesized context content.
- Use clear paragraphs or bullet points as needed, but avoid a leading label line.
- IMPORTANT: Always preserve ALL facts from the Facts section of PREVIOUS CONTEXT, even when topics differ"""

    return prompt

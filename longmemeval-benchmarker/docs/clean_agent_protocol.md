# Clean Agent Protocol

## Core Message Specification

The MycelianMemoryAgent protocol uses an Invoker pattern to encapsulate message building and control flow.

### Agent API (Simple and Clean)

```python
class MycelianMemoryAgent:
    def invoke(self, control: ControlState, thread_id: str,
               to_process: Optional[ChatMessage] = None) -> None:
        """Execute based on control state.

        Uses checkpointer for state persistence across invocations.
        Thread_id identifies the conversation thread for the checkpointer.
        """
        # Checkpointer maintains conversation_history across invocations
        # Each thread_id has its own persistent state
```

### Invoker API (Handles Complexity)

```python
class MycelianAgentInvoker:
    """Encapsulates message building and control determination."""

    def __init__(self, agent: MycelianMemoryAgent):
        self.agent = agent
        self.msg_count = 0

    def start_session(self, thread_id: str) -> None:
        """Start a new session."""
        self.msg_count = 0
        self.agent.invoke(control=ControlState.START_SESSION, thread_id=thread_id)

    def process_conversation_message(self, role: str, content: str, thread_id: str) -> None:
        """Process a conversation message, handling flush automatically."""
        self.msg_count += 1

        # Build message internally
        message = ChatMessage(role=role, content=content)

        # Determine control internally using enum
        if self.msg_count % 6 == 0:
            control = ControlState.PROCESS_MESSAGE_AND_FLUSH
        else:
            control = ControlState.PROCESS_MESSAGE

        self.agent.invoke(control=control, thread_id=thread_id, to_process=message)

    def end_session(self, thread_id: str) -> None:
        """End the session."""
        self.agent.invoke(control=ControlState.END_SESSION, thread_id=thread_id)
```

### Usage in Runner (Dead Simple)

```python
# In SingleQuestionRunner
invoker = MycelianAgentInvoker(agent)

# Start session
invoker.start_session(thread_id)

# Process messages - invoker handles everything
for m in session.get("messages", []):
    invoker.process_conversation_message(
        role=m["role"],
        content=m["content"],
        thread_id=thread_id
    )

# End session
invoker.end_session(thread_id)
```

### Benefits

- **Runner stays simple** - Just passes raw data
- **Invoker encapsulates complexity** - Message building, counting, control logic
- **Agent stays focused** - Just executes based on control
- **Clean separation** - Each layer has one job

## Agent State Design

The state structure handles context accumulation and processing.

### State Definition

```python
from enum import Enum
from typing import Sequence, Union
from typing_extensions import TypedDict, Annotated
from langgraph.graph.message import add_messages
from langchain_core.messages import ChatMessage, AIMessage, ToolMessage

class ControlState(Enum):
    """Control states for agent execution."""
    START_SESSION = "start_session"
    PROCESS_MESSAGE = "process_message"
    PROCESS_MESSAGE_AND_FLUSH = "process_message_and_flush"
    END_SESSION = "end_session"

class AgentState(TypedDict):
    """State structure for the agent."""
    conversation_history: Annotated[Sequence[ChatMessage], add_messages]  # Accumulates across invocations
    to_process: Sequence[ChatMessage]  # Current ChatMessage to process (replaced each invocation)
    tool_history: Sequence[Union[AIMessage, ToolMessage]]  # Tool flow for current invocation only
    control: ControlState  # Control state driving execution
```

### What Goes in Each Field

- **conversation_history**: Only `ChatMessage` instances (role="user", "assistant", etc.)
- **to_process**: Single `ChatMessage` being processed
- **tool_history**:
  - `AIMessage` with tool_calls
  - `ToolMessage` with results
- **control**: ControlState enum value (START_SESSION, PROCESS_MESSAGE, etc.)

### Annotation Behavior

- Fields WITH `add_messages` annotation accumulate (append new messages)
- Fields WITHOUT annotation get replaced (overwrite with new value)
- So `to_process` gets replaced each invocation while others accumulate

### Example State Flow

```python
# When processing a message
State: {
    conversation_history: [ChatMessage(role="user", ...), ChatMessage(role="assistant", ...)],
    to_process: [ChatMessage(role="user", content="Hello")],  # Current one
    tool_history: [],  # Empty initially
    control: ControlState.PROCESS_MESSAGE
}

# After tool execution
State: {
    conversation_history: [...same ChatMessages...],
    to_process: [...same current ChatMessage...],
    tool_history: [
        AIMessage(tool_calls=[{"name": "add_entry", ...}]),
        ToolMessage(name="add_entry", content="Entry added")
    ],
    control: ControlState.PROCESS_MESSAGE
}
```

## Agent's Tool Orchestration Protocol

The agent orchestrates tool calls through LLM-generated summaries and context management.

### How Tool Arguments Are Generated

The LLM is responsible for generating all tool arguments, including summaries:

1. **add_entry** - LLM reads a single message (either user OR assistant) and generates:
   - `summary`: Summary of the message content
   - `raw_entry`: The actual message text
   - `tags`: Optional metadata about the message

2. **put_context** - LLM reads entire conversation history and generates:
   - `content`: Synthesized summary of the entire session

### Context Integration Flow

Retrieved context from tools is integrated into `conversation_history`:

1. **After get_context**:
   ```python
   # Tool returns previous session's context
   context_text = tool_result.content

   # Add to conversation_history as system message
   context_msg = ChatMessage(
       role="system",
       content=f"Previous session context:\n{context_text}"
   )
   return {"conversation_history": [context_msg]}
   ```

2. **After list_entries**:
   ```python
   # Tool returns recent entries
   entries_text = tool_result.content

   # Add to conversation_history as system message
   entries_msg = ChatMessage(
       role="system",
       content=f"Recent entries:\n{entries_text}"
   )
   return {"conversation_history": [entries_msg]}
   ```

### Prompt Construction for add_entry

When processing messages, the LLM receives both context and current message:

```python
prompt = f"""Previous conversation context:
{format_messages(conversation_history[:-1])}  # All previous messages including retrieved context

Current message to process:
{format_messages(to_process)}  # Just the single current message (user OR assistant)

INSTRUCTION: Call add_entry tool to record this message.
Generate a summary that captures the key information from this message.
Consider the previous context when creating the summary."""
```

### Tool Execution Example

For a PROCESS_MESSAGE control state with a user message:

1. **Observe node** builds prompt with conversation history and current single message
2. **LLM** generates tool call:
   ```python
   AIMessage(
       tool_calls=[{
           "name": "add_entry",
           "args": {
               "summary": "User struggling with cold start problem in recommendation systems",
               "raw_entry": "I'm still having issues with the cold start problem we discussed",
               "tags": {"role": "user", "topic": "recommendation_systems"}
           }
       }]
   )
   ```
3. **Tools node** executes the tool with LLM-generated arguments
4. **Tool** returns confirmation as ToolMessage
5. **Observe node** detects completion and exits

### State Updates During Execution

- `conversation_history`: Accumulates all messages + retrieved context
- `to_process`: Contains only current message being processed
- `tool_history`: Tracks AIMessage → ToolMessage flow
- `control`: Remains constant during a single invocation

This separation ensures the LLM always has full context while clearly identifying what needs processing.

## State Persistence with Checkpointer

The agent uses LangGraph's checkpointer for state persistence across invocations.

### How It Works

```python
class MycelianMemoryAgent:
    def __init__(self, ...):
        self.checkpointer = MemorySaver()  # Create checkpointer
        self.graph = self._build_graph()

    def _build_graph(self):
        workflow = StateGraph(AgentState)
        # ... add nodes and edges ...
        return workflow.compile(checkpointer=self.checkpointer)

    def invoke(self, control: ControlState, thread_id: str, to_process: Optional[ChatMessage] = None):
        config = {"configurable": {"thread_id": thread_id}}

        # Initial state for this invocation
        state = {
            "control": control,
            "to_process": [to_process] if to_process else [],
            "conversation_history": [to_process] if to_process else [],  # Also add to history
            "tool_history": []  # Starts fresh each invocation
        }

        return self.graph.invoke(state, config)
```

### State Accumulation Behavior

With checkpointer and `add_messages` annotation:

1. **conversation_history** (has annotation) → Accumulates across invocations
   - Each message gets appended
   - Retrieved context from START_SESSION gets added
   - Full conversation available for END_SESSION

2. **to_process** (no annotation) → Replaced each invocation
   - Only contains current message being processed
   - Cleared for next invocation

3. **tool_history** (no annotation) → Replaced each invocation
   - Only tracks tool calls for current invocation
   - Starts fresh to avoid confusion between invocations

### Benefits

- **No manual tracking**: Invoker doesn't need to maintain conversation_history
- **Thread isolation**: Each thread_id has independent state
- **Automatic persistence**: State survives across invocations
- **Simple API**: Invoker just passes control and current message

## Conditional Tool Calling Logic

The observe node uses control state and last tool executed to determine next action:

```python
def observe(self, state):
    tool_history = state.get("tool_history", [])
    control = state.get("control")
    conversation_history = state.get("conversation_history", [])  # Has accumulated messages from checkpointer
    to_process = state.get("to_process", [])  # Just current message

    # Determine last tool executed
    last_tool = None
    if tool_history and isinstance(tool_history[-1], ToolMessage):
        last_tool = tool_history[-1].name

    # START_SESSION: Direct tool calls (no LLM needed)
    if control == ControlState.START_SESSION:
        # Direct execution without LLM - these are deterministic calls
        context_result = tools["get_context"](
            vault_id=vault_id,
            memory_id=memory_id
        )

        entries_result = tools["list_entries"](
            vault_id=vault_id,
            memory_id=memory_id,
            limit=10
        )

        # Add results to conversation_history
        context_msg = ChatMessage(role="system", content=f"Previous context:\n{context_result}")
        entries_msg = ChatMessage(role="system", content=f"Recent entries:\n{entries_result}")

        return {
            "conversation_history": [context_msg, entries_msg],
            "tool_history": []  # No tool history needed for direct calls
        }

    # PROCESS_MESSAGE sequence: add_entry only
    elif control == ControlState.PROCESS_MESSAGE:
        if last_tool is None:
            # Only tool: add_entry
            prompt = build_add_entry_prompt(conversation_history, to_process)
        elif last_tool == "add_entry":
            # Complete
            return {"tool_history": [AIMessage(content="Message processed.")]}

    # PROCESS_MESSAGE_AND_FLUSH sequence: add_entry → await_consistency → put_context
    elif control == ControlState.PROCESS_MESSAGE_AND_FLUSH:
        if last_tool is None:
            # First tool: add_entry (needs LLM for summary)
            prompt = build_add_entry_prompt(conversation_history, to_process)
        elif last_tool == "add_entry":
            # Second tool: await_consistency (direct call - no LLM needed)
            tools["await_consistency"](memory_id=memory_id)
            # Then need to call put_context
            prompt = build_put_context_prompt(conversation_history)
        elif last_tool == "put_context":
            # Complete
            return {"tool_history": [AIMessage(content="Flushed to context.")]}

    # END_SESSION sequence: await_consistency → put_context
    elif control == ControlState.END_SESSION:
        if last_tool is None:
            # First tool: await_consistency (direct call - no LLM needed)
            tools["await_consistency"](memory_id=memory_id)
            # Then need to call put_context
            prompt = build_put_context_prompt(conversation_history)
        elif last_tool == "put_context":
            # Complete
            return {"tool_history": [AIMessage(content="Session ended.")]}

    # Call LLM with the generated prompt
    response = llm_with_tools.invoke([
        {"role": "system", "content": prompt},
        {"role": "user", "content": "Execute the required operation."}
    ])

    # Special handling for context updates
    if control == ControlState.START_SESSION and last_tool == "get_context":
        context_text = tool_history[-1].content
        context_msg = ChatMessage(role="system", content=f"Previous context:\n{context_text}")
        return {
            "conversation_history": [context_msg],
            "tool_history": [response]
        }

    return {"tool_history": [response]}
```

### Key Points

1. **Progress Tracking**: Uses `last_tool` to know where we are in each sequence
2. **Context Integration**: Adds retrieved context/entries to `conversation_history` at the right points
3. **Completion Detection**: Each sequence knows when it's done and returns appropriate completion message
4. **Prompt Building**: Calls helper functions (`build_add_entry_prompt`, `build_put_context_prompt`) for complex prompts
5. **State Updates**: Returns updates to both `conversation_history` and `messages` as needed

## Prompt Design

The agent uses structured prompts for each operation, incorporating prompts from MCP and context from the conversation.

### Common Agent Prefix

All prompts start with this prefix to establish the agent's role:

```python
AGENT_PREFIX = """You are Mycelian's Memory Agent. Your job is to observe a conversation between a user and an AI Assistant and accurately store, retrieve and manage memories."""
```

### PROCESS_MESSAGE Prompt (add_entry)

This prompt handles capturing a single message to memory:

```python
def build_add_entry_prompt(vault_id, memory_id, conversation_history, to_process, prompts):
    # Validate we have context (should always be present after START_SESSION)
    if not conversation_history:
        raise ValueError("No conversation history found - START_SESSION may have failed")

    # Validate current message
    if not to_process:
        raise ValueError("No message to process")

    # Get prompts from MCP
    entry_capture_prompt = prompts.get("entry_capture_prompt")
    summary_prompt = prompts.get("summary_prompt")

    # Format previous conversation for context
    context = format_messages(conversation_history[:-1])  # All except current

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

---
ENTRY CAPTURE RULES:
{entry_capture_prompt}

---
SUMMARY GENERATION RULES:
{summary_prompt}
"""
    return prompt
```

### PUT_CONTEXT Prompt (context synthesis)

This prompt handles synthesizing the entire conversation into a structured context document:

```python
def build_put_context_prompt(vault_id, memory_id, conversation_history, prompts):
    # Validate we have conversation history
    if not conversation_history:
        raise ValueError("No conversation history to synthesize")

    # Get the context prompt from MCP
    context_prompt = prompts.get("context_prompt")

    # Format all conversation messages
    full_conversation = format_messages(conversation_history)

    prompt = f"""{AGENT_PREFIX}

Current Operation: CONTEXT_SYNTHESIS
Vault ID: {vault_id}
Memory ID: {memory_id}

Full conversation to synthesize into context:
{full_conversation}

INSTRUCTION: Call the put_context tool to save a synthesized context document following the rules below.

---
CONTEXT MAINTENANCE RULES:
{context_prompt}
"""
    return prompt
```

### Key Prompt Components

1. **Agent Prefix**: Establishes the agent's role and purpose
2. **Operation Context**: Identifies current operation (PROCESS_MESSAGE, START_SESSION, etc.)
3. **Memory IDs**: Vault and Memory IDs for tool calls
4. **Conversation Context**: Previous messages including retrieved context from START_SESSION
5. **Current Data**: The specific data to process (message, context to save, etc.)
6. **Instructions**: Clear, specific instructions for the tool call
7. **Attached Rules**: Relevant prompts from MCP (entry_capture_prompt, summary_prompt, context_prompt)

### Tool Call Strategy

The agent uses a hybrid approach for tool calls:

**Direct Tool Calls (No LLM or Prompts)**:
- `get_context` - Simple retrieval with fixed parameters
- `list_entries` - Simple retrieval with fixed parameters
- `await_consistency` - Simple synchronization call

**LLM-Driven Tool Calls (Require Prompts)**:
- `add_entry` - Requires LLM to generate summaries (uses `build_add_entry_prompt`)
- `put_context` - Requires LLM to synthesize context (uses `build_put_context_prompt`)

This optimization reduces latency and cost for deterministic operations while leveraging the LLM's capabilities for content generation.

### Prompt Usage by Operation

- **START_SESSION**: No prompts (direct tool calls)
- **PROCESS_MESSAGE**: Uses `build_add_entry_prompt`
- **PROCESS_MESSAGE_AND_FLUSH**: Uses `build_add_entry_prompt` then `build_put_context_prompt`
- **END_SESSION**: Uses `build_put_context_prompt`

### Prompts Source

The prompts are retrieved from MCP and include:
- `client/prompts/default/chat/entry_capture_prompt.md` - Rules for capturing entries
- `client/prompts/default/chat/summary_prompt.md` - Rules for generating summaries
- `client/prompts/default/chat/context_prompt.md` - Rules for maintaining context

These prompts are not duplicated but attached to each operation's prompt as needed.

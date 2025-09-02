# Chapter 4: The Observer Pattern - Building an Agent That Watches and Remembers

With conversation memory solved, I faced a new challenge: building an agent that could observe conversations and extract meaningful memories. Not participate, but watch, analyze, and remember. This journey led me through several failed attempts before finding an elegant solution.

## The Vision

The goal was clear: create an observer agent that processes conversation turns and records structured memories using tools. Like a note-taking assistant sitting in on a meeting, it would capture the essence of each exchange without participating.

The twist? The observer would role-play as if it were the assistant in the conversation, creating memories from a first-person perspective. "I told them about the weather" rather than "The assistant told them about the weather."

## The Initial Approach: Simple Role-Playing

My first attempt was naive optimism. Just give the observer the conversation and ask it to record memories:

```python
def observe(self, state):
    conversation = state["messages"]
    
    prompt = """You are an AI assistant reviewing a conversation you just had.
    Record important memories about what you learned."""
    
    response = llm_with_tools.invoke(prompt + conversation)
    return {"messages": [response]}
```

This worked... sort of. The observer would analyze the conversation and call the `add_entry` tool. But with checkpointing enabled, problems emerged quickly.

## The Accumulation Problem

Here's what happened with checkpointing:

**Turn 1**: Observer sees [H1, A1] → Records 1 entry
**Turn 2**: Observer sees [H1, A1, H2, A2] → Records 2 entries (for both exchanges)
**Turn 3**: Observer sees [H1, A1, H2, A2, H3, A3] → Records 3 entries

The observer kept re-recording the same exchanges. By turn 4, I had massive duplication. The checkpointer was doing its job - accumulating conversation history - but the observer couldn't tell what it had already processed.

## Failed Solution 1: Tracking Processed Entries

My first fix attempt added state tracking:

```python
class ObserverState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], operator.add]
    entries_recorded: int  # Track how many we've processed
```

The idea: tell the observer "You've already recorded N entries, now just process entry N+1." But this felt like fighting the framework. We were building stateful tracking on top of stateful checkpointing. It was complexity on complexity.

## Failed Solution 2: Processing Only Latest

Next, I tried processing only the newest messages:

```python
def observe(self, state):
    all_messages = state["messages"]
    # Get only the last two messages (latest exchange)
    latest_exchange = all_messages[-2:]
    
    # Process just these
    record_memory(latest_exchange)
```

This eliminated duplicates but lost critical context. The observer analyzing turn 3 couldn't reference information from turns 1 and 2. Memories became shallow and disconnected: "User mentioned something" instead of "User expanded on their earlier point about vegetarian preferences."

## The Revelation: Context vs Processing

The breakthrough came when I realized we needed TWO different types of information:
1. **Context**: The full conversation history (for understanding)
2. **To Process**: Just the latest exchange (for recording)

But how to maintain both with checkpointing's automatic accumulation?

## Failed Solution 3: Multiple Checkpointers

I briefly considered using two checkpointers - one for context, one for processing. This was architectural madness. The complexity would have been overwhelming, and it violated the principle of simplicity.

## The Elegant Solution: Dual-Field State

The solution was beautiful in its simplicity. Use different merge strategies for different fields:

```python
class ContextObserverState(TypedDict):
    context: Annotated[Sequence[BaseMessage], operator.add]  # Accumulates
    to_process: Sequence[BaseMessage]  # No annotation = replaced
    messages: Annotated[Sequence[BaseMessage], operator.add]  # For tools
```

With this structure:
- `context` accumulates all messages (full history)
- `to_process` gets replaced each turn (only latest)
- `messages` handles tool execution flow

## The Implementation Reality

When invoking the graph:

```python
input_state = {
    "context": [human_msg, ai_msg],      # Gets added to history
    "to_process": [human_msg, ai_msg],   # Replaces previous
}
```

The observer now had everything it needed:
- Full conversation context for rich understanding
- Clear indication of what's new to process
- No duplication, no confusion

## The Tool Integration

The observer used a specialized tool for recording:

```python
@tool
def add_entry(user_summary: str, assistant_summary: str, context_notes: str = ""):
    """Record a contextual summary of a user-assistant exchange"""
    entry = {
        "user": user_summary,
        "assistant": assistant_summary,
        "context": context_notes,
        "timestamp": datetime.now()
    }
    database.append(entry)
    return "Entry recorded"
```

Each turn produced exactly one entry with summaries of both the user's message and assistant's response, enriched by context from the full conversation.

## The Prompt Engineering

The prompt became crucial for quality memories:

```python
prompt = f"""You are an AI assistant analyzing a conversation exchange.

Context: {summary_of_previous_exchanges}

Current exchange to summarize:
User: {new_human_message}
Assistant: {new_ai_message}

Create a summary that:
1. Captures what the user communicated
2. Captures how you responded
3. Notes important context from earlier

Use the add_entry tool to record this summary."""
```

The observer could now create entries like: "User asked about dietary restrictions in Japan, building on their earlier mention of being vegetarian."

## The Working System

The final system produced beautiful results:

**Turn 1**: "User introduced themselves as Sarah, planning Japan trip"
**Turn 2**: "Sarah expressed vegetarian concerns for Japan travel (context: first trip)"
**Turn 3**: "Sarah interested in temple photography during cherry blossom season"

Each entry stood alone but was enriched by context. No duplicates, no lost information.

## The Lessons Learned

This journey taught me several crucial lessons:

1. **Don't Fight the Framework**: Working with LangGraph's patterns is better than against them
2. **Separation of Concerns**: Different data needs different handling strategies
3. **Context Matters**: Full history enables rich understanding
4. **Simplicity Wins**: The dual-field approach was simpler than all the complex alternatives

## The Philosophical Insight

The observer pattern revealed something profound about memory and context. Memory isn't just recording events - it's understanding how events relate to each other. The dual-field state pattern elegantly captured this distinction between seeing everything and processing something specific.

The failed attempts weren't wasted effort. Each failure taught me more about LangGraph's architecture and pushed me toward the elegant solution. Sometimes you need to explore the complex paths to appreciate the simple one.

**Word count: 1,051**
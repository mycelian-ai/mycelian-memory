# Chapter 2: Building a Simple Agent - Tools, Decisions, and the Execution Dance

Armed with my newfound understanding of state, I set out to build a simple agent that could actually do things. Not just chat, but use tools, make decisions, and solve problems. What followed was a journey through the fascinating choreography of agent-tool interactions in LangGraph.

## The Tool Revolution

My first surprise: tools in LangGraph aren't complicated. They're just Python functions with a decorator:

```python
@tool
def calculator(expression: str) -> float:
    """Evaluate a mathematical expression"""
    return eval(expression, {"__builtins__": {}})
```

That `@tool` decorator transforms a simple function into something an LLM can understand and invoke. But how does the LLM know about it? This question led me down a rabbit hole of discovery.

## The Binding Ceremony

The magic happens through "binding" - connecting tools to the LLM:

```python
llm = ChatOpenAI(model="gpt-4o-mini")
llm_with_tools = llm.bind_tools([calculator, get_time, get_fact])
```

When you bind tools, LangGraph converts your Python functions into JSON schemas and sends them to the LLM alongside your messages. The LLM sees something like:

```json
{
  "name": "calculator",
  "description": "Evaluate a mathematical expression",
  "parameters": {
    "expression": {"type": "string"}
  }
}
```

The LLM reads this and thinks, "Ah, when someone asks about math, I can call this calculator function with an expression string." It's surprisingly elegant.

## The Graph Architecture

Building the agent graph felt like assembling a Rube Goldberg machine that actually works:

```python
def build_graph():
    workflow = StateGraph(AgentState)
    
    workflow.add_node("agent", call_agent)
    workflow.add_node("tools", ToolNode(tools))
    
    workflow.add_conditional_edges(
        "agent",
        should_continue,
        {"continue": "tools", "end": END}
    )
    
    workflow.add_edge("tools", "agent")
    
    return workflow.compile()
```

The flow is beautifully simple: agent decides → router checks → tools execute (if needed) → back to agent → end. But understanding this flow took me through several misconceptions.

## The Double Agent Pattern

Here's what blew my mind: for each tool use, the agent node executes TWICE. First to decide to use a tool, then again to process the result. It's like asking a colleague who has a calculator:

1. You: "What's 25 times 4?"
2. Colleague: *reaches for calculator* (first agent call - decides to use tool)
3. Calculator: "100" (tool execution)
4. Colleague: "It's 100!" (second agent call - formulates response)

This pattern ensures the agent can properly contextualize tool results before responding to the user.

## The Execution Trace

To really understand what was happening, I added logging everywhere:

```python
def call_agent(state):
    print(f"Agent received {len(state['messages'])} messages")
    response = llm_with_tools.invoke(state["messages"])
    
    if response.tool_calls:
        print(f"Agent calling tools: {[tc['name'] for tc in response.tool_calls]}")
    else:
        print(f"Agent final response: {response.content[:50]}...")
    
    return {"messages": [response]}
```

Running "What's 15 + 27?" produced this enlightening trace:

```
Agent received 1 messages
Agent calling tools: ['calculator']
Tool 'calculator' returned: 42
Agent received 3 messages
Agent final response: 15 + 27 equals 42
```

Those 3 messages in the second call are: HumanMessage (question), AIMessage (with tool_calls), and ToolMessage (result). The agent sees the complete interaction history and formulates the final response.

## The Tool Call Format

Understanding how tool calls work was crucial. When the LLM decides to use a tool, it doesn't just name it - it provides structured data:

```python
tool_call = {
    "id": "call_123",
    "name": "calculator",
    "args": {"expression": "15 + 27"}
}
```

The ToolNode automatically executes this, calling `calculator.invoke({"expression": "15 + 27"})`. The result becomes a ToolMessage, completing the cycle.

## The SimpleAgent Implementation

Putting it all together, my SimpleAgent looked like this:

```python
class SimpleAgent:
    def __init__(self):
        self.llm = ChatOpenAI(model="gpt-4o-mini")
        self.llm_with_tools = self.llm.bind_tools(tools)
        self.tool_node = ToolNode(tools)
        self.graph = self._build_graph()
    
    def run(self, user_input: str):
        initial_state = {"messages": [HumanMessage(content=user_input)]}
        result = self.graph.invoke(initial_state)
        return result["messages"][-1].content
```

Clean, simple, stateless. Each `run()` call is independent, starting fresh with only the user's input.

## The Stateless Reality

This is where I hit my first major limitation. The agent worked beautifully for single interactions:

- "What's 25 * 4?" → "100"
- "What time is it?" → "It's 2:30 PM"
- "Tell me a fact" → "Honey never spoils"

But ask follow-up questions:

- "What's 25 * 4?" → "100"
- "What did I just ask?" → "I don't have any context about previous questions"

The agent had amnesia. Every `run()` call created a fresh state with no memory of previous interactions. This was by design - the graph is stateless for good architectural reasons - but it meant no conversation continuity.

## The Tool Selection Intelligence

What amazed me was how intelligently the LLM selected tools. Given multiple tools, it consistently chose correctly:

- Math question → calculator
- Time question → get_current_time
- Fact request → get_random_fact
- Casual chat → no tool

The LLM's training includes understanding function descriptions and matching them to user intent. It's not programmed rules but learned behavior.

## The Error Handling Gap

One challenge I discovered: error handling in tools needs careful thought. If a tool fails, the agent might get confused. I learned to make tools return error messages as strings rather than throwing exceptions:

```python
@tool
def calculator(expression: str) -> str:
    try:
        return str(eval(expression, {"__builtins__": {}}))
    except:
        return "Error: Invalid expression"
```

This way, the agent can gracefully handle and explain errors to users.

## The Architectural Insights

Building SimpleAgent taught me several architectural principles:

1. **Separation of Concerns**: The LLM decides, tools execute, graph orchestrates
2. **Stateless by Default**: Each execution is isolated and predictable
3. **Tool Abstraction**: Tools are just functions; the framework handles the complexity
4. **Message Accumulation**: The state pattern naturally preserves interaction history

## The Next Challenge

SimpleAgent could answer questions and use tools, but it couldn't remember anything between conversations. The stateless design that made it robust also made it forgetful. This limitation led me to the next challenge: adding memory.

The journey from understanding state to building a functional agent was enlightening. The elegant separation between decision-making (LLM), execution (tools), and orchestration (graph) makes LangGraph powerful yet approachable. But the stateless nature, while architecturally sound, presents a challenge for conversational continuity.

**Word count: 1,089**
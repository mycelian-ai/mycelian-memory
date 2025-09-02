def visualize_flow():
    print("""
    AGENT WORKFLOW VISUALIZATION
    ============================
    
    1. USER INPUT: "What is 15 + 27?"
       ↓
    2. AGENT NODE (First Call)
       - Receives: [HumanMessage("What is 15 + 27?")]
       - LLM decides: "I need to use the calculator tool"
       - Returns: AIMessage with tool_calls=[{name: "calculator", args: {expression: "15+27"}}]
       ↓
    3. ROUTER (Conditional Edge)
       - Checks last message for tool_calls
       - Decision: "continue" (has tool calls) → go to tools node
       ↓
    4. TOOL NODE
       - Executes calculator("15+27")
       - Returns: ToolMessage(content="42.0")
       - State now has: [HumanMessage, AIMessage(with tool call), ToolMessage]
       ↓
    5. AGENT NODE (Second Call)
       - Receives all 3 messages
       - LLM sees the tool result
       - Returns: AIMessage("15 + 27 equals 42")
       ↓
    6. ROUTER (Conditional Edge)
       - Checks last message for tool_calls
       - Decision: "end" (no tool calls) → finish
       ↓
    7. FINAL OUTPUT: "15 + 27 equals 42"
    
    
    KEY CONCEPTS:
    =============
    
    • bind_tools(): Gives the LLM awareness of available tools
    • tool_calls: LLM's way of saying "I want to use this tool"
    • ToolNode: Automatically executes tools and returns results
    • Conditional edges: Dynamic routing based on state
    • Message accumulation: Each message is added to conversation history
    """)


if __name__ == "__main__":
    visualize_flow()
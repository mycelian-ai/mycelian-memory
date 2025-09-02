import os
from agent import SimpleAgent


def visualize_simple_agent():
    agent = SimpleAgent()
    
    try:
        from IPython.display import Image, display
        img = Image(agent.graph.get_graph().draw_mermaid_png())
        display(img)
        print("Graph displayed in notebook")
    except:
        print("Generating Mermaid diagram...")
        mermaid_code = agent.graph.get_graph().draw_mermaid()
        print("\n" + "="*50)
        print("LANGGRAPH STRUCTURE (Mermaid Format)")
        print("="*50)
        print(mermaid_code)
        
        with open("agent_graph.mmd", "w") as f:
            f.write(mermaid_code)
        print("\nSaved to agent_graph.mmd")
        print("View at: https://mermaid.live/")
    
    print("\n" + "="*50)
    print("GRAPH STRUCTURE (Text)")
    print("="*50)
    graph = agent.graph.get_graph()
    
    print("\nNodes:")
    for node in graph.nodes:
        print(f"  • {node}")
    
    print("\nEdges:")
    for edge in graph.edges:
        print(f"  • {edge[0]} → {edge[1]}")
    
    print("\n" + "="*50)
    print("VISUAL ASCII REPRESENTATION")
    print("="*50)
    print("""
         ┌─────────┐
         │  START  │
         └────┬────┘
              ↓
         ┌─────────┐
         │  agent  │ ←─────┐
         └────┬────┘       │
              ↓            │
        ┌──────────┐       │
        │ router?  │       │
        └─┬─────┬──┘       │
          ↓     ↓          │
    [tool_calls] [no tools]│
          ↓     ↓          │
     ┌─────────┐│          │
     │  tools  ││          │
     └────┬────┘│          │
          └─────┘          │
              └────────────┘
                    ↓
              ┌─────────┐
              │   END   │
              └─────────┘
    
    FLOW EXPLANATION:
    1. START → agent: Begin with user message
    2. agent: LLM decides (use tool or respond)
    3. Router checks: Has tool_calls?
       - YES → go to tools node
       - NO → go to END
    4. tools → agent: Tool result goes back to agent
    5. agent: Process result and respond
    6. Router checks again → usually END
    """)


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        os.environ["OPENAI_API_KEY"] = "dummy-key-for-visualization"
    
    visualize_simple_agent()
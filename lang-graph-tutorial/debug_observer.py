import os
from langchain_core.messages import HumanMessage, AIMessage
from observer_agent import ObserverAgent, memories_db


def debug_observer():
    memories_db.clear()
    
    observer = ObserverAgent()
    
    # Simulate what the observer sees on turn 2
    state = {
        "messages": [
            HumanMessage("Hi, I'm Alex"),
            AIMessage("Hello Alex! Nice to meet you."),
            HumanMessage("I like pizza"),
            AIMessage("Pizza is great! What's your favorite topping?")
        ]
    }
    
    print("State has 2 exchanges (4 messages total)")
    print("Expecting 2 add_entry calls...")
    print()
    
    # Call observe directly
    result = observer.observe(state)
    
    print(f"Observer made {len(result['messages'][0].tool_calls) if result['messages'][0].tool_calls else 0} tool calls")
    
    if result['messages'][0].tool_calls:
        for i, tc in enumerate(result['messages'][0].tool_calls, 1):
            print(f"\nTool call {i}:")
            print(f"  Tool: {tc['name']}")
            print(f"  Args: {tc['args']}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Set OPENAI_API_KEY")
    else:
        debug_observer()
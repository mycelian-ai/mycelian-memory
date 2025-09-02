import json
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage
import os


@tool
def multiply(a: int, b: int) -> int:
    """Multiply two numbers together"""
    return a * b


@tool
def add(x: float, y: float) -> float:
    """Add two numbers together"""
    return x + y


def show_tool_schema():
    print("TOOL SCHEMAS (What LLM sees)")
    print("="*50)
    
    for t in [multiply, add]:
        print(f"\nTool: {t.name}")
        print(f"Description: {t.description}")
        print(f"Schema: {json.dumps(t.args_schema.model_json_schema(), indent=2)}")


def show_bind_tools_effect():
    print("\n" + "="*50)
    print("WHAT bind_tools() DOES")
    print("="*50)
    
    llm = ChatOpenAI(model="gpt-4o-mini", api_key=os.getenv("OPENAI_API_KEY"))
    
    print("\n1. Without tools:")
    response1 = llm.invoke([HumanMessage("What is 5 times 3?")])
    print(f"   Response: {response1.content}")
    print(f"   Tool calls: {response1.tool_calls}")
    
    print("\n2. With tools bound:")
    llm_with_tools = llm.bind_tools([multiply, add])
    response2 = llm_with_tools.invoke([HumanMessage("What is 5 times 3?")])
    print(f"   Response: {response2.content}")
    print(f"   Tool calls: {response2.tool_calls}")


def show_tool_conversion():
    print("\n" + "="*50)
    print("TOOL INFORMATION SENT TO LLM")
    print("="*50)
    
    tool_info = {
        "type": "function",
        "function": {
            "name": "multiply",
            "description": "Multiply two numbers together",
            "parameters": {
                "type": "object",
                "properties": {
                    "a": {
                        "type": "integer",
                        "description": "First number"
                    },
                    "b": {
                        "type": "integer", 
                        "description": "Second number"
                    }
                },
                "required": ["a", "b"]
            }
        }
    }
    
    print("\nThis is what gets sent to the LLM API:")
    print(json.dumps(tool_info, indent=2))
    
    print("\nThe LLM uses this information to:")
    print("1. Understand what tools are available")
    print("2. Know what each tool does (description)")
    print("3. Know what arguments each tool needs")
    print("4. Understand the type of each argument")


def demonstrate_llm_reasoning():
    print("\n" + "="*50)
    print("LLM's DECISION PROCESS")
    print("="*50)
    
    print("""
    User: "What is 15 multiplied by 4?"
    
    LLM's internal reasoning:
    1. User wants multiplication
    2. I have a 'multiply' tool available
    3. It needs two arguments: 'a' (integer) and 'b' (integer)
    4. From the query: a=15, b=4
    5. Generate tool call:
       {
         "name": "multiply",
         "args": {"a": 15, "b": 4}
       }
    """)
    
    print("\n" + "-"*30)
    
    print("""
    User: "Add 3.5 and 2.7"
    
    LLM's internal reasoning:
    1. User wants addition
    2. I have an 'add' tool available
    3. It needs two arguments: 'x' (float) and 'y' (float)
    4. From the query: x=3.5, y=2.7
    5. Generate tool call:
       {
         "name": "add",
         "args": {"x": 3.5, "y": 2.7}
       }
    """)


def show_complete_flow():
    print("\n" + "="*50)
    print("COMPLETE FLOW WITH REAL LLM")
    print("="*50)
    
    if not os.getenv("OPENAI_API_KEY"):
        print("Set OPENAI_API_KEY to see real example")
        return
    
    llm = ChatOpenAI(model="gpt-4o-mini", temperature=0)
    llm_with_tools = llm.bind_tools([multiply, add])
    
    test_messages = [
        "Calculate 7 times 8",
        "What's 12.5 plus 7.3?",
        "I need to multiply 6 by 9 and then add 10 to the result"
    ]
    
    for msg in test_messages:
        print(f"\nUser: {msg}")
        response = llm_with_tools.invoke([HumanMessage(msg)])
        
        if response.tool_calls:
            for tc in response.tool_calls:
                print(f"  LLM decides to call: {tc['name']}")
                print(f"  With arguments: {tc['args']}")
                result = None
                if tc['name'] == 'multiply':
                    result = multiply.invoke(tc['args'])
                elif tc['name'] == 'add':
                    result = add.invoke(tc['args'])
                print(f"  Result: {result}")
        else:
            print(f"  Direct response: {response.content}")


if __name__ == "__main__":
    show_tool_schema()
    show_tool_conversion()
    demonstrate_llm_reasoning()
    
    if os.getenv("OPENAI_API_KEY"):
        show_bind_tools_effect()
        show_complete_flow()
    else:
        print("\n" + "="*50)
        print("Set OPENAI_API_KEY to see live examples")
import json
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage
from langchain_core.utils.function_calling import convert_to_openai_function
import os


@tool
def calculate(expression: str) -> float:
    """Evaluate a mathematical expression like '2+3*4' or '10/2'"""
    try:
        return eval(expression, {"__builtins__": {}})
    except:
        return "Error evaluating expression"


def show_what_llm_receives():
    print("HOW LLM KNOWS ABOUT TOOLS")
    print("="*50)
    
    print("\n1. Your Python function with @tool decorator:")
    print("""
    @tool
    def calculate(expression: str) -> float:
        '''Evaluate a mathematical expression'''
        return eval(expression)
    """)
    
    print("\n2. Gets converted to OpenAI function format:")
    openai_function = convert_to_openai_function(calculate)
    print(json.dumps(openai_function, indent=2))
    
    print("\n3. This JSON is sent to the LLM with your message")
    print("   The LLM sees:")
    print("   - Function name: 'calculate'")
    print("   - What it does: 'Evaluate a mathematical expression'")
    print("   - Required parameters: 'expression' (string type)")


def show_llm_decision_process():
    print("\n" + "="*50)
    print("LLM's DECISION MAKING PROCESS")
    print("="*50)
    
    print("""
    STEP 1: User asks "What is 25 + 17?"
    
    STEP 2: LLM analyzes the request:
        - Intent: Calculate a sum
        - Available tools: [calculate]
        - Tool matches intent: YES
    
    STEP 3: LLM determines arguments:
        - Tool needs 'expression' parameter
        - Extract from user query: "25 + 17"
        
    STEP 4: LLM generates tool call:
        {
            "name": "calculate",
            "args": {"expression": "25 + 17"}
        }
    """)


def demonstrate_with_real_llm():
    print("\n" + "="*50)
    print("REAL EXAMPLE WITH LLM")
    print("="*50)
    
    if not os.getenv("OPENAI_API_KEY"):
        print("Set OPENAI_API_KEY to see live demo")
        return
    
    llm = ChatOpenAI(model="gpt-4o-mini", temperature=0)
    
    print("\nWithout tools (LLM calculates in its head):")
    response = llm.invoke([HumanMessage("What is 25 + 17?")])
    print(f"Response: {response.content}")
    print(f"Tool calls: {response.tool_calls}")
    
    print("\n" + "-"*30)
    
    print("\nWith tools bound (LLM uses the tool):")
    llm_with_tools = llm.bind_tools([calculate])
    response = llm_with_tools.invoke([HumanMessage("What is 25 + 17?")])
    print(f"Response: {response.content}")
    print(f"Tool calls: {json.dumps(response.tool_calls, indent=2)}")
    
    if response.tool_calls:
        tool_call = response.tool_calls[0]
        result = calculate.invoke(tool_call["args"])
        print(f"\nTool execution result: {result}")


def show_multiple_tools_example():
    print("\n" + "="*50)
    print("MULTIPLE TOOLS - HOW LLM CHOOSES")
    print("="*50)
    
    @tool
    def get_weather(city: str) -> str:
        """Get the current weather for a city"""
        return f"Sunny, 72°F in {city}"
    
    @tool
    def get_time() -> str:
        """Get the current time"""
        from datetime import datetime
        return datetime.now().strftime("%H:%M:%S")
    
    tools = [calculate, get_weather, get_time]
    
    print("\nAvailable tools:")
    for t in tools:
        print(f"  - {t.name}: {t.description}")
    
    print("\nLLM decision examples:")
    print("""
    User: "What's the weather in Paris?"
    → LLM chooses: get_weather(city="Paris")
    
    User: "What time is it?"
    → LLM chooses: get_time()
    
    User: "Calculate 15% of 200"
    → LLM chooses: calculate(expression="200 * 0.15")
    
    User: "Tell me a joke"
    → LLM chooses: No tool (responds directly)
    """)


if __name__ == "__main__":
    show_what_llm_receives()
    show_llm_decision_process()
    demonstrate_with_real_llm()
    show_multiple_tools_example()
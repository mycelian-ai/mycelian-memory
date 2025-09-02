from langchain_core.tools import tool


@tool
def multiply(a: int, b: int) -> int:
    """Multiply two numbers"""
    return a * b


def regular_multiply(a: int, b: int) -> int:
    """Regular Python function"""
    return a * b


def explain_tool_decorator():
    print("THE @tool DECORATOR TRANSFORMATION")
    print("="*50)
    
    print("\n1. Regular function:")
    print(f"   Type: {type(regular_multiply)}")
    print(f"   Call: regular_multiply(3, 4) = {regular_multiply(3, 4)}")
    
    print("\n2. @tool decorated function:")
    print(f"   Type: {type(multiply)}")
    print(f"   Class: {multiply.__class__.__name__}")
    
    print("\n3. Tool has special attributes:")
    print(f"   .name: {multiply.name}")
    print(f"   .description: {multiply.description}")
    print(f"   .args_schema: {multiply.args_schema}")
    
    print("\n4. Tool has multiple ways to call it:")
    
    print("\n   a) Using .invoke() method (preferred):")
    result3 = multiply.invoke({"a": 3, "b": 4})
    print(f"      multiply.invoke(" + str({"a": 3, "b": 4}) + f") = {result3}")
    
    print("\n   b) Using .run() method (legacy):")
    result2 = multiply.run({"a": 3, "b": 4})
    print(f"      multiply.run(" + str({"a": 3, "b": 4}) + f") = {result2}")
    
    print("\n5. Why use .invoke()?")
    print("   - Consistent interface across all LangChain components")
    print("   - Handles input validation")
    print("   - Can be used in chains and workflows")
    print("   - Supports async operations")


def show_invoke_details():
    print("\n" + "="*50)
    print("WHAT HAPPENS INSIDE .invoke()")
    print("="*50)
    
    args = {"a": 5, "b": 7}
    
    print(f"\nInput: {args}")
    print("\nSteps inside .invoke():")
    print("1. Validates input against args_schema")
    print("2. Extracts arguments from dictionary")
    print("3. Calls the underlying function")
    print("4. Returns the result")
    
    result = multiply.invoke(args)
    print(f"\nResult: {result}")
    
    print("\nNote: The underlying function is called with unpacked args")


def show_tool_call_format():
    print("\n" + "="*50)
    print("TOOL CALL FORMAT FROM LLM")
    print("="*50)
    
    tool_call = {
        "id": "call_abc123",
        "name": "multiply",
        "args": {"a": 10, "b": 20}
    }
    
    print(f"\nLLM returns tool_call: {tool_call}")
    print(f"\nExtract args: {tool_call['args']}")
    print(f"\nInvoke tool: multiply.invoke({tool_call['args']})")
    print(f"\nResult: {multiply.invoke(tool_call['args'])}")


if __name__ == "__main__":
    explain_tool_decorator()
    show_invoke_details()
    show_tool_call_format()
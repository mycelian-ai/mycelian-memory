from langchain_core.tools import tool
from langchain_core.messages import ToolMessage, AIMessage
import json


@tool
def multiply(a: int, b: int) -> int:
    """Multiply two numbers"""
    return a * b


def demonstrate_tool_structure():
    print("Tool Properties:")
    print(f"  Name: {multiply.name}")
    print(f"  Description: {multiply.description}")
    print(f"  Args schema: {multiply.args_schema.schema()}")
    print()
    
    
def demonstrate_tool_calling():
    print("Tool Calling Flow:")
    print()
    
    ai_message_with_tool_call = AIMessage(
        content="I'll multiply 5 and 7 for you.",
        tool_calls=[{
            "id": "call_123",
            "name": "multiply",
            "args": {"a": 5, "b": 7}
        }]
    )
    
    print("1. AI decides to use a tool:")
    print(f"   Content: {ai_message_with_tool_call.content}")
    print(f"   Tool call: {ai_message_with_tool_call.tool_calls[0]}")
    print()
    
    tool_call = ai_message_with_tool_call.tool_calls[0]
    result = multiply.invoke(tool_call["args"])
    
    print("2. Tool executes:")
    print(f"   multiply({tool_call['args']['a']}, {tool_call['args']['b']}) = {result}")
    print()
    
    tool_message = ToolMessage(
        content=str(result),
        tool_call_id=tool_call["id"]
    )
    
    print("3. Tool result returned as ToolMessage:")
    print(f"   Content: {tool_message.content}")
    print(f"   Tool call ID: {tool_message.tool_call_id}")
    print()
    
    print("4. AI processes result and responds to user")


if __name__ == "__main__":
    demonstrate_tool_structure()
    demonstrate_tool_calling()
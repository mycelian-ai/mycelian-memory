from typing import TypedDict, Annotated, Sequence, List
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage
import operator


class SimpleState(TypedDict):
    counter: int
    names: List[str]


class AnnotatedState(TypedDict):
    counter: Annotated[int, operator.add]
    names: Annotated[List[str], operator.add]


class MessageState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], operator.add]


def demonstrate_simple_state():
    state = {"counter": 0, "names": []}
    
    print("Without Annotation - Values get REPLACED:")
    state["counter"] = 5
    state["names"] = ["Alice"]
    print(f"State: {state}")
    
    state["counter"] = 3  
    state["names"] = ["Bob"]
    print(f"State after update: {state}")
    print()


def demonstrate_annotated_behavior():
    print("With Annotation - Values get ACCUMULATED:")
    print("Initial: counter=5, names=['Alice']")
    print("Update:  counter=3, names=['Bob']")
    print("Result:  counter=8, names=['Alice', 'Bob']")
    print()


def demonstrate_message_state():
    print("Message State Example:")
    
    messages = [
        HumanMessage(content="Hello"),
        AIMessage(content="Hi there!"),
        HumanMessage(content="How are you?"),
        AIMessage(content="I'm doing well!")
    ]
    
    print("Conversation flow:")
    for msg in messages:
        role = "Human" if isinstance(msg, HumanMessage) else "AI"
        print(f"  {role}: {msg.content}")
    
    print("\nWith operator.add annotation:")
    print("  - New messages are APPENDED to the list")
    print("  - Conversation history is preserved")
    print("  - Each node sees the full context")


if __name__ == "__main__":
    print("=" * 50)
    print("UNDERSTANDING LANGGRAPH STATE")
    print("=" * 50)
    print()
    
    demonstrate_simple_state()
    demonstrate_annotated_behavior()
    demonstrate_message_state()
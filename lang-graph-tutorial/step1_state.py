from typing import TypedDict, Annotated, Sequence
from langchain_core.messages import BaseMessage
import operator


class AgentState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], operator.add]


if __name__ == "__main__":
    print("Agent State defined")
    print(f"Fields: {AgentState.__annotations__}")
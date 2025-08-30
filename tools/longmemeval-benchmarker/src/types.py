from typing import TypedDict, Literal


class ChatMessage(TypedDict):
    role: Literal["user", "assistant"]
    content: str




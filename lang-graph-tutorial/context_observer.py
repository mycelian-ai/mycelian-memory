import os
from typing import TypedDict, Annotated, Sequence, List
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, ToolMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver
import operator
from datetime import datetime


class ContextObserverState(TypedDict):
    context: Annotated[Sequence[BaseMessage], operator.add]  # Accumulates all messages
    to_process: Sequence[BaseMessage]  # Replaced each time - only latest exchange
    messages: Annotated[Sequence[BaseMessage], operator.add]  # For tool execution flow


entries_db = []


@tool
def add_entry(user_summary: str, assistant_summary: str, context_notes: str = "") -> str:
    """Record a contextual summary of a user-assistant exchange"""
    entry = {
        "timestamp": datetime.now().isoformat(),
        "user": user_summary,
        "assistant": assistant_summary,
        "context": context_notes,
        "entry_number": len(entries_db) + 1
    }
    entries_db.append(entry)
    return f"Entry #{entry['entry_number']} recorded"


class ContextObserver:
    def __init__(self):
        self.llm = ChatOpenAI(
            model="gpt-4o-mini",
            temperature=0,
            api_key=os.getenv("OPENAI_API_KEY")
        )
        self.tools = [add_entry]
        self.llm_with_tools = self.llm.bind_tools(self.tools)
        self.tool_node = ToolNode(self.tools)
        self.memory = MemorySaver()
        self.graph = self._build_graph()
    
    def _build_graph(self):
        workflow = StateGraph(ContextObserverState)
        
        workflow.add_node("analyze", self.analyze_exchange)
        workflow.add_node("record", self.tool_node)
        
        workflow.set_entry_point("analyze")
        
        workflow.add_conditional_edges(
            "analyze",
            self.should_record,
            {
                "record": "record",
                "skip": END
            }
        )
        
        workflow.add_edge("record", END)
        
        return workflow.compile(checkpointer=self.memory)
    
    def analyze_exchange(self, state: ContextObserverState):
        context_messages = state.get("context", [])
        new_messages = state.get("to_process", [])
        
        if not new_messages or len(new_messages) < 2:
            return {"messages": [AIMessage(content="No complete exchange to process")]}
        
        # Extract the new exchange
        new_human = new_messages[0] if isinstance(new_messages[0], HumanMessage) else None
        new_ai = new_messages[1] if len(new_messages) > 1 and isinstance(new_messages[1], AIMessage) else None
        
        if not new_human or not new_ai:
            return {"messages": [AIMessage(content="Invalid exchange format")]}
        
        # Build context summary from previous messages
        context_summary = ""
        if context_messages:
            # Find previous exchanges (excluding the current one)
            previous_count = (len(context_messages) - len(new_messages)) // 2
            if previous_count > 0:
                context_summary = f"This is exchange #{previous_count + 1} in an ongoing conversation. "
                
                # Add relevant context
                for msg in context_messages[:-len(new_messages)]:
                    if isinstance(msg, HumanMessage) and any(keyword in msg.content.lower() for keyword in ["name", "i'm", "i am"]):
                        context_summary += f"The user has previously mentioned: {msg.content[:100]}... "
                        break
        
        prompt = f"""You are an AI assistant analyzing a conversation exchange.
        
Context: {context_summary if context_summary else "This is the first exchange."}

Current exchange to summarize:
User: {new_human.content}
Assistant: {new_ai.content}

Create a summary that:
1. Captures what the user communicated (facts, questions, preferences)
2. Captures how the assistant responded (information provided, questions asked)
3. Notes any important context that enriches understanding of this exchange

Use the add_entry tool to record this summary. Include context notes if the previous conversation provides important background."""
        
        response = self.llm_with_tools.invoke([
            {"role": "system", "content": prompt},
            {"role": "user", "content": "Create a contextual summary of this exchange."}
        ])
        
        return {"messages": [response]}
    
    def should_record(self, state: ContextObserverState):
        messages = state.get("messages", [])
        if messages and hasattr(messages[-1], 'tool_calls') and messages[-1].tool_calls:
            return "record"
        return "skip"
    
    def process_turn(self, human_msg: str, ai_msg: str, thread_id: str):
        config = {"configurable": {"thread_id": thread_id}}
        
        input_state = {
            "context": [
                HumanMessage(content=human_msg),
                AIMessage(content=ai_msg)
            ],
            "to_process": [
                HumanMessage(content=human_msg),
                AIMessage(content=ai_msg)
            ]
        }
        
        result = self.graph.invoke(input_state, config)
        return result


def test_context_observer():
    print("="*60)
    print("CONTEXT-AWARE OBSERVER TEST")
    print("="*60)
    
    entries_db.clear()
    observer = ContextObserver()
    thread_id = "test-context-001"
    
    conversations = [
        ("Hi, I'm Alice and I work as a data scientist", 
         "Hello Alice! It's great to meet a data scientist. What kind of projects do you work on?"),
        
        ("I mainly work on recommendation systems for e-commerce",
         "That's fascinating! Recommendation systems are crucial for e-commerce. Do you use collaborative filtering or content-based approaches?"),
        
        ("We use a hybrid approach, combining both methods",
         "Smart choice! Hybrid approaches often give the best results by leveraging the strengths of both methods."),
        
        ("I'm particularly interested in handling cold start problems",
         "Cold start is indeed challenging! How do you currently handle new users or items with no interaction history?")
    ]
    
    for i, (human, ai) in enumerate(conversations, 1):
        print(f"\n--- Turn {i} ---")
        print(f"Human: {human[:50]}...")
        print(f"AI: {ai[:50]}...")
        
        observer.process_turn(human, ai, thread_id)
        
        if entries_db:
            latest = entries_db[-1]
            print(f"\nEntry #{latest['entry_number']}:")
            print(f"  User: {latest['user']}")
            print(f"  Assistant: {latest['assistant']}")
            if latest.get('context'):
                print(f"  Context: {latest['context']}")
    
    print("\n" + "="*60)
    print("ALL ENTRIES WITH CONTEXT")
    print("="*60)
    
    for entry in entries_db:
        print(f"\nEntry #{entry['entry_number']}:")
        print(f"  User: {entry['user']}")
        print(f"  Assistant: {entry['assistant']}")
        if entry.get('context'):
            print(f"  Context: {entry['context']}")
    
    print(f"\nTotal entries: {len(entries_db)}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY")
    else:
        test_context_observer()
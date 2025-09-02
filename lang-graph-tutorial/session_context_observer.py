import os
from typing import TypedDict, Annotated, Sequence, Optional, Dict
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, ToolMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver
import operator
from datetime import datetime


class SessionObserverState(TypedDict):
    context: Annotated[Sequence[BaseMessage], operator.add]  # Accumulates all messages
    to_process: Sequence[BaseMessage]  # Replaced each turn - only latest exchange
    messages: Annotated[Sequence[BaseMessage], operator.add]  # For tool execution flow
    control: Optional[str]  # "get_context", "put_context", or None for normal processing
    thread_id: Optional[str]  # Thread identifier for context storage


# In-memory storage for demo (replace with database in production)
context_store: Dict[str, str] = {}
entries_db = []


@tool
def get_context(memory_id: str) -> str:
    """Retrieve accumulated context from previous sessions"""
    stored_context = context_store.get(memory_id, "No previous context available.")
    return stored_context


@tool
def put_context(memory_id: str, context_summary: str) -> str:
    """Save synthesized context for future sessions"""
    context_store[memory_id] = context_summary
    timestamp = datetime.now().isoformat()
    return f"Context saved at {timestamp}"


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


class SessionContextObserver:
    def __init__(self):
        self.llm = ChatOpenAI(
            model="gpt-4o-mini",
            temperature=0,
            api_key=os.getenv("OPENAI_API_KEY")
        )
        self.tools = [get_context, put_context, add_entry]
        self.llm_with_tools = self.llm.bind_tools(self.tools)
        self.tool_node = ToolNode(self.tools)
        self.memory = MemorySaver()
        self.graph = self._build_graph()
    
    def _build_graph(self):
        workflow = StateGraph(SessionObserverState)
        
        workflow.add_node("observe", self.observe)
        workflow.add_node("tools", self.tool_node)
        
        workflow.set_entry_point("observe")
        
        workflow.add_conditional_edges(
            "observe",
            self.should_execute_tools,
            {
                "execute": "tools",
                "end": END
            }
        )
        
        workflow.add_edge("tools", END)
        
        return workflow.compile(checkpointer=self.memory)
    
    def observe(self, state: SessionObserverState):
        messages = state.get("messages", [])
        control = state.get("control", None)
        
        # Check if we're coming back from tool execution
        # Only return early if we just executed the tool for the current control action
        if messages and isinstance(messages[-1], ToolMessage):
            tool_msg = messages[-1]
            # Only return if this is the tool we were expecting for the current control
            if control == "get_context" and tool_msg.name == "get_context":
                return {"messages": [AIMessage(content="Context loaded.")]}
            elif control == "put_context" and tool_msg.name == "put_context":
                return {"messages": [AIMessage(content="Context saved.")]}
            elif control is None and tool_msg.name == "add_entry":
                return {"messages": [AIMessage(content="Entry recorded.")]}
        
        # Get thread_id from state or use default
        thread_id = state.get("thread_id", "default-memory")
        
        # Build focused prompts based on control
        if control == "get_context":
            prompt = f"""You are an AI assistant starting a new session.

INSTRUCTION: Call the get_context tool with memory_id="{thread_id}" to retrieve your stored knowledge from previous sessions."""
            
        elif control == "put_context":
            conversation_messages = state.get("context", [])
            
            if not conversation_messages:
                # If no context from state, use a dummy message
                conversation_messages = [
                    HumanMessage(content="User had a conversation"),
                    AIMessage(content="Assistant responded")
                ]
            
            prompt = f"""You are an AI assistant ending a session.

Full conversation to synthesize:
{self._format_messages(conversation_messages)}

INSTRUCTION: Call the put_context tool with memory_id="{thread_id}" and a context_summary that includes who the user is, topics discussed, and key facts."""
            
        else:  # Normal processing
            to_process = state.get("to_process", [])
            context_messages = state.get("context", [])
            
            # Calculate which exchange number this is
            exchange_num = (len(context_messages) // 2)
            
            prompt = f"""You are an AI assistant analyzing conversation exchanges.

This is exchange #{exchange_num} in the current session.

Current exchange to process:
{self._format_messages(to_process) if to_process else "No new exchange to process."}

INSTRUCTION: Call the add_entry tool to record a summary of this exchange.
Focus on capturing:
- What the user communicated (questions, information, preferences)
- How the assistant responded (advice, information, actions)
- Any important context that enriches understanding"""
        
        response = self.llm_with_tools.invoke([
            {"role": "system", "content": prompt},
            {"role": "user", "content": "Execute the required operation."}
        ])
        
        return {"messages": [response]}
    
    def should_execute_tools(self, state: SessionObserverState):
        messages = state.get("messages", [])
        if messages and hasattr(messages[-1], 'tool_calls') and messages[-1].tool_calls:
            return "execute"
        return "end"
    
    def _format_messages(self, messages):
        formatted = []
        for msg in messages:
            if isinstance(msg, HumanMessage):
                formatted.append(f"User: {msg.content}")
            elif isinstance(msg, AIMessage) and msg.content:
                formatted.append(f"Assistant: {msg.content}")
        return "\n".join(formatted)
    
    # High-level API methods
    def start_session(self, thread_id: str):
        """Load context at the beginning of a session"""
        config = {"configurable": {"thread_id": thread_id}}
        
        input_state = {
            "control": "get_context",
            "context": [],
            "to_process": [],
            "messages": [],
            "thread_id": thread_id
        }
        result = self.graph.invoke(input_state, config)
        
        # Extract context if retrieved  
        for msg in result.get("messages", []):
            if isinstance(msg, ToolMessage):
                print(f"Retrieved context: {msg.content[:100]}...")
                break
        
        return result
    
    def process_turn(self, human_msg: str, ai_msg: str, thread_id: str):
        """Process a normal conversation turn"""
        config = {"configurable": {"thread_id": thread_id}}
        input_state = {
            "control": None,  # Normal processing
            "context": [
                HumanMessage(content=human_msg),
                AIMessage(content=ai_msg)
            ],
            "to_process": [
                HumanMessage(content=human_msg),
                AIMessage(content=ai_msg)
            ],
            "thread_id": thread_id
        }
        result = self.graph.invoke(input_state, config)
        return result
    
    def end_session(self, thread_id: str):
        """Save context at the end of a session"""
        config = {"configurable": {"thread_id": thread_id}}
        
        # Get the current state to access accumulated context
        try:
            state = self.graph.get_state(config)
            context_messages = state.values.get("context", []) if state.values else []
        except:
            context_messages = []
        
        input_state = {
            "control": "put_context",
            "context": context_messages,  # Pass accumulated context
            "to_process": [],
            "messages": [],
            "thread_id": thread_id
        }
        result = self.graph.invoke(input_state, config)
        return result


def test_session_flow():
    print("="*60)
    print("SESSION-BASED CONTEXT OBSERVER TEST")
    print("="*60)
    
    observer = SessionContextObserver()
    thread_id = "user-alice-001"
    memory_id = thread_id  # Using same ID for simplicity
    
    # Simulate first session with 4 exchanges
    print("\n=== SESSION 1 (4 exchanges) ===")
    print("Starting session...")
    observer.start_session(thread_id)
    
    print("\nProcessing conversation turns...")
    
    # Turn 1
    observer.process_turn(
        "Hi, I'm Alice and I work as a data scientist",
        "Hello Alice! It's great to meet a data scientist. What kind of projects do you work on?",
        thread_id
    )
    print(f"After turn 1: {len(entries_db)} entries recorded")
    
    # Turn 2
    observer.process_turn(
        "I mainly work on recommendation systems for e-commerce",
        "That's fascinating! Recommendation systems are crucial for e-commerce. Do you use collaborative filtering or content-based approaches?",
        thread_id
    )
    print(f"After turn 2: {len(entries_db)} entries recorded")
    
    # Turn 3
    observer.process_turn(
        "We use a hybrid approach, but cold start is a big challenge",
        "Cold start is indeed challenging. How do you handle new users with no interaction history?",
        thread_id
    )
    print(f"After turn 3: {len(entries_db)} entries recorded")
    
    # Turn 4
    observer.process_turn(
        "We use demographic data and popular items initially",
        "That's a practical approach. Using demographics for initial segmentation can provide reasonable starting recommendations.",
        thread_id
    )
    print(f"After turn 4: {len(entries_db)} entries recorded")
    
    print("\nEnding session 1...")
    observer.end_session(thread_id)
    
    stored_context = context_store.get(memory_id, 'None')
    print(f"\n=== STORED CONTEXT FROM SESSION 1 ===")
    print(stored_context if stored_context != 'None' else "No context stored")
    print("="*40)
    
    # Clear entries for second session
    entries_db.clear()
    
    # Create a fresh observer for session 2 to avoid message accumulation
    observer2 = SessionContextObserver()
    
    # Simulate second session with 4 exchanges
    print("\n=== SESSION 2 (4 exchanges, later) ===")
    print("Starting new session...")
    result = observer2.start_session(thread_id)
    
    # Check what context was retrieved
    retrieved_context = context_store.get(memory_id, "No context")
    print(f"\n=== RETRIEVED CONTEXT AT SESSION 2 START ===")
    print(retrieved_context if retrieved_context != "No context" else "No previous context")
    print("="*40)
    
    print("\nProcessing new conversation turns...")
    
    # Turn 1
    observer2.process_turn(
        "I'm still struggling with the cold start problem we discussed",
        "I understand you're still working on the cold start problem. As a data scientist working on recommendation systems, this is indeed a key challenge.",
        thread_id
    )
    print(f"After turn 1: {len(entries_db)} entries recorded")
    
    # Turn 2
    observer2.process_turn(
        "Have you worked with any solutions for this?",
        "Yes, besides demographic segmentation and popular items, you might consider using transfer learning from similar domains or implementing a questionnaire for new users.",
        thread_id
    )
    print(f"After turn 2: {len(entries_db)} entries recorded")
    
    # Turn 3
    observer2.process_turn(
        "Transfer learning sounds interesting. How would that work?",
        "You could leverage user behavior patterns from related e-commerce categories or even from other recommendation systems in your company to bootstrap predictions for new users.",
        thread_id
    )
    print(f"After turn 3: {len(entries_db)} entries recorded")
    
    # Turn 4
    observer2.process_turn(
        "That makes sense. I'll explore that approach.",
        "Great! Transfer learning can significantly reduce the cold start period. Feel free to ask if you need help with implementation details.",
        thread_id
    )
    print(f"After turn 4: {len(entries_db)} entries recorded")
    
    print("\nEnding session 2...")
    observer2.end_session(thread_id)
    
    updated_context = context_store.get(memory_id, 'None')
    print(f"\n=== UPDATED CONTEXT AFTER SESSION 2 ===")
    print(updated_context if updated_context != 'None' else "No context stored")
    print("="*40)
    
    print("\n" + "="*60)
    print("ALL ENTRIES RECORDED IN SESSION 2")
    print("="*60)
    
    for entry in entries_db:
        print(f"\nEntry #{entry['entry_number']}:")
        print(f"  User: {entry['user']}")
        print(f"  Assistant: {entry['assistant']}")
        if entry.get('context'):
            print(f"  Context: {entry['context']}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY")
    else:
        test_session_flow()
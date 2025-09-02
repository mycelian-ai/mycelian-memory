import os
from typing import TypedDict, Annotated, Sequence, List
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, ToolMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver
import operator
import json
from datetime import datetime


class ObserverState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], operator.add]
    entries_recorded: int  # Track how many entries we've already recorded


memories_db = []


@tool
def add_entry(user_message_summary: str, assistant_message_summary: str) -> str:
    """Record a summary of the user's message and the assistant's response"""
    entry = {
        "timestamp": datetime.now().isoformat(),
        "user": user_message_summary,
        "assistant": assistant_message_summary,
        "type": "conversation_entry"
    }
    memories_db.append(entry)
    return f"Entry recorded - User: {user_message_summary[:50]}... | Assistant: {assistant_message_summary[:50]}..."


class ObserverAgent:
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
        workflow = StateGraph(ObserverState)
        
        workflow.add_node("observer", self.observe)
        workflow.add_node("tools", self.tool_node)
        
        workflow.set_entry_point("observer")
        
        workflow.add_conditional_edges(
            "observer",
            self.should_continue,
            {
                "continue": "tools",
                "end": END
            }
        )
        
        workflow.add_edge("tools", "observer")
        
        return workflow.compile(checkpointer=self.memory)
    
    def observe(self, state: ObserverState):
        messages = state["messages"]
        
        # Check if we're coming back from tool execution
        if any(isinstance(msg, ToolMessage) for msg in messages):
            # Tools have executed, we're done
            return {"messages": [AIMessage(content="Memories recorded successfully.")]}
        
        # First time through - analyze the conversation
        conversation_messages = [msg for msg in messages if isinstance(msg, (HumanMessage, AIMessage)) and not hasattr(msg, 'tool_calls')]
        
        prompt = f"""You are an AI assistant reviewing a conversation you just had with a user.
Your job is to create a summary of each exchange using the add_entry tool.

For each user-assistant exchange in the conversation:
1. Summarize what the user said (key points, questions, information shared)
2. Summarize what you (the assistant) responded (advice given, information provided, questions asked)

Focus on capturing the essence of each message - what was communicated and why it matters.
Use the add_entry tool to record these summaries.

Here's the conversation:"""
        
        llm_messages = [
            {"role": "system", "content": prompt}
        ]
        
        for msg in conversation_messages:
            if isinstance(msg, HumanMessage):
                llm_messages.append({"role": "user", "content": msg.content})
            elif isinstance(msg, AIMessage):
                llm_messages.append({"role": "assistant", "content": msg.content})
        
        llm_messages.append({
            "role": "user", 
            "content": "Please create a summary entry for EACH exchange in this conversation. If there are 4 exchanges, call add_entry 4 times."
        })
        
        response = self.llm_with_tools.invoke(llm_messages)
        return {"messages": [response]}
    
    def should_continue(self, state: ObserverState):
        last_message = state["messages"][-1]
        if hasattr(last_message, 'tool_calls') and last_message.tool_calls:
            return "continue"
        return "end"
    
    def process_turn(self, human_message: str, ai_message: str, thread_id: str):
        config = {"configurable": {"thread_id": thread_id}}
        
        turn = {
            "messages": [
                HumanMessage(content=human_message),
                AIMessage(content=ai_message)
            ]
        }
        
        result = self.graph.invoke(turn, config)
        return result


def run_observer_demo():
    print("="*60)
    print("OBSERVER AGENT DEMO")
    print("="*60)
    
    sample_conversation = [
        {
            "human": "Hi! I'm Sarah and I'm planning a trip to Japan next month.",
            "assistant": "Hello Sarah! How exciting that you're planning a trip to Japan next month! Is this your first time visiting?"
        },
        {
            "human": "Yes, first time! I'm a bit nervous about the language barrier. I'm vegetarian too, which I heard can be challenging there.",
            "assistant": "Those are valid concerns! For the language barrier, I'd recommend downloading translation apps and learning a few basic phrases. As for vegetarian dining, while it can be challenging, there are many Buddhist temples that serve shojin ryori (vegetarian temple food), and cities like Tokyo have increasing vegetarian options."
        },
        {
            "human": "That's helpful! I'm particularly interested in visiting temples and gardens. I love photography and want to capture the cherry blossoms if possible.",
            "assistant": "Perfect timing for cherry blossoms! Late March to early April is peak season. For temples and gardens, I'd highly recommend Kyoto - places like Kinkaku-ji, Fushimi Inari, and the Arashiyama Bamboo Grove are stunning for photography. The Philosopher's Path during cherry blossom season would be ideal for you."
        },
        {
            "human": "Should I book accommodations in advance? I prefer quiet places over party hostels.",
            "assistant": "Absolutely book in advance, especially during cherry blossom season! Since you prefer quiet places, consider staying in a traditional ryokan (Japanese inn) for part of your trip, or look for business hotels which tend to be quieter than hostels. Areas like Asakusa in Tokyo or the Higashiyama district in Kyoto offer peaceful accommodations with easy access to temples."
        }
    ]
    
    observer = ObserverAgent()
    thread_id = "sarah-conversation-001"
    
    for i, turn in enumerate(sample_conversation, 1):
        print(f"\n--- Processing Turn {i} ---")
        print(f"Human: {turn['human'][:50]}...")
        print(f"Assistant: {turn['assistant'][:50]}...")
        
        observer.process_turn(
            turn["human"],
            turn["assistant"],
            thread_id
        )
    
    print("\n" + "="*60)
    print("MEMORIES RECORDED")
    print("="*60)
    
    for i, memory in enumerate(memories_db, 1):
        print(f"\n{i}. Entry:")
        print(f"   User: {memory.get('user', 'N/A')}")
        print(f"   Assistant: {memory.get('assistant', 'N/A')}")
    
    print("\n" + "="*60)
    print(f"Total memories recorded: {len(memories_db)}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        run_observer_demo()
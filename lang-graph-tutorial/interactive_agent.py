import os
from typing import TypedDict, Annotated, Sequence
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver
import operator

from tools import tools
from step1_state import AgentState


class InteractiveAgent:
    def __init__(self):
        self.llm = ChatOpenAI(
            model="gpt-4o-mini",
            temperature=0,
            api_key=os.getenv("OPENAI_API_KEY")
        )
        self.llm_with_tools = self.llm.bind_tools(tools)
        self.tool_node = ToolNode(tools)
        self.memory = MemorySaver()
        self.graph = self._build_graph()
        self.thread_id = "conversation-1"
        self.config = {"configurable": {"thread_id": self.thread_id}}
    
    def _build_graph(self):
        workflow = StateGraph(AgentState)
        
        workflow.add_node("agent", self.call_agent)
        workflow.add_node("tools", self.tool_node)
        
        workflow.set_entry_point("agent")
        
        workflow.add_conditional_edges(
            "agent",
            self.should_continue,
            {
                "continue": "tools",
                "end": END
            }
        )
        
        workflow.add_edge("tools", "agent")
        
        return workflow.compile(checkpointer=self.memory)
    
    def call_agent(self, state: AgentState):
        messages = state["messages"]
        response = self.llm_with_tools.invoke(messages)
        return {"messages": [response]}
    
    def should_continue(self, state: AgentState):
        last_message = state["messages"][-1]
        if last_message.tool_calls:
            return "continue"
        return "end"
    
    def chat(self, user_input: str):
        initial_state = {
            "messages": [HumanMessage(content=user_input)]
        }
        
        result = self.graph.invoke(initial_state, self.config)
        return result["messages"][-1].content
    
    def clear_memory(self):
        self.thread_id = f"conversation-{os.urandom(4).hex()}"
        self.config = {"configurable": {"thread_id": self.thread_id}}
        print("Memory cleared. Starting new conversation.")
    
    def show_conversation_history(self):
        state = self.graph.get_state(self.config)
        if state and state.values and "messages" in state.values:
            messages = state.values["messages"]
            print("\n=== Conversation History ===")
            for msg in messages:
                if isinstance(msg, HumanMessage):
                    print(f"User: {msg.content}")
                elif isinstance(msg, AIMessage) and msg.content:
                    print(f"Agent: {msg.content}")
            print("===========================\n")
        else:
            print("No conversation history yet.")


def run_interactive_session():
    print("="*50)
    print("INTERACTIVE AGENT")
    print("="*50)
    print("\nAvailable commands:")
    print("  /help     - Show this help message")
    print("  /clear    - Clear conversation memory")
    print("  /history  - Show conversation history")
    print("  /exit     - Exit the program")
    print("\nAvailable tools:")
    print("  - calculator: Evaluate math expressions")
    print("  - get_current_time: Get current date/time")
    print("  - get_random_fact: Get a random fact")
    print("\n" + "="*50)
    
    agent = InteractiveAgent()
    
    while True:
        try:
            user_input = input("\nYou: ").strip()
            
            if not user_input:
                continue
            
            if user_input.lower() == "/exit":
                print("Goodbye!")
                break
            elif user_input.lower() == "/help":
                print("\nCommands:")
                print("  /help     - Show this help message")
                print("  /clear    - Clear conversation memory")
                print("  /history  - Show conversation history")
                print("  /exit     - Exit the program")
                continue
            elif user_input.lower() == "/clear":
                agent.clear_memory()
                continue
            elif user_input.lower() == "/history":
                agent.show_conversation_history()
                continue
            
            print("\nAgent: ", end="", flush=True)
            response = agent.chat(user_input)
            print(response)
            
        except KeyboardInterrupt:
            print("\n\nGoodbye!")
            break
        except Exception as e:
            print(f"\nError: {e}")
            print("Try again or type /exit to quit.")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        run_interactive_session()
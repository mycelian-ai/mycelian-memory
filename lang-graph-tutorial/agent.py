import os
from typing import TypedDict, Annotated, Sequence
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, ToolMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
import operator

from tools import tools
from step1_state import AgentState


class SimpleAgent:
    def __init__(self):
        self.llm = ChatOpenAI(
            model="gpt-4o-mini",
            temperature=0,
            api_key=os.getenv("OPENAI_API_KEY")
        )
        self.llm_with_tools = self.llm.bind_tools(tools)
        self.tool_node = ToolNode(tools)
        self.graph = self._build_graph()
    
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
        
        return workflow.compile()
    
    def call_agent(self, state: AgentState):
        messages = state["messages"]
        response = self.llm_with_tools.invoke(messages)
        return {"messages": [response]}
    
    def should_continue(self, state: AgentState):
        last_message = state["messages"][-1]
        if last_message.tool_calls:
            return "continue"
        return "end"
    
    def run(self, user_input: str):
        initial_state = {
            "messages": [HumanMessage(content=user_input)]
        }
        
        result = self.graph.invoke(initial_state)
        return result["messages"][-1].content


def demonstrate_agent():
    agent = SimpleAgent()
    
    test_queries = [
        "What's 25 * 4?",
        "What time is it?",
        "Tell me an interesting fact",
        "Calculate (10 + 5) * 3 and tell me the current time"
    ]
    
    for query in test_queries:
        print(f"\nUser: {query}")
        response = agent.run(query)
        print(f"Agent: {response}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        demonstrate_agent()
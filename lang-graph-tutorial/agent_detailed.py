import os
from langchain_core.messages import HumanMessage, ToolMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode

from tools import tools
from step1_state import AgentState


def call_agent(state: AgentState):
    print("\n--- AGENT NODE ---")
    print(f"Received {len(state['messages'])} messages")
    
    llm = ChatOpenAI(
        model="gpt-4o-mini",
        temperature=0,
        api_key=os.getenv("OPENAI_API_KEY")
    )
    llm_with_tools = llm.bind_tools(tools)
    
    response = llm_with_tools.invoke(state["messages"])
    
    print(f"Agent response type: {type(response).__name__}")
    if response.tool_calls:
        print(f"Agent decided to call tools: {[tc['name'] for tc in response.tool_calls]}")
    else:
        print(f"Agent final response: {response.content[:100]}...")
    
    return {"messages": [response]}


def execute_tools(state: AgentState):
    print("\n--- TOOL NODE ---")
    tool_node = ToolNode(tools)
    result = tool_node.invoke(state)
    
    for msg in result["messages"]:
        if isinstance(msg, ToolMessage):
            print(f"Tool '{msg.name}' returned: {msg.content}")
    
    return result


def should_continue(state: AgentState):
    last_message = state["messages"][-1]
    decision = "continue" if last_message.tool_calls else "end"
    print(f"\n--- ROUTER ---")
    print(f"Decision: {decision}")
    return decision


def build_agent_graph():
    workflow = StateGraph(AgentState)
    
    workflow.add_node("agent", call_agent)
    workflow.add_node("tools", execute_tools)
    
    workflow.set_entry_point("agent")
    
    workflow.add_conditional_edges(
        "agent",
        should_continue,
        {
            "continue": "tools",
            "end": END
        }
    )
    
    workflow.add_edge("tools", "agent")
    
    return workflow.compile()


def run_agent(user_input: str):
    print("="*50)
    print(f"USER INPUT: {user_input}")
    print("="*50)
    
    graph = build_agent_graph()
    
    initial_state = {
        "messages": [HumanMessage(content=user_input)]
    }
    
    result = graph.invoke(initial_state)
    
    print("\n" + "="*50)
    print("FINAL RESULT:")
    print(result["messages"][-1].content)
    print("="*50)
    
    return result


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        run_agent("What is 15 + 27?")
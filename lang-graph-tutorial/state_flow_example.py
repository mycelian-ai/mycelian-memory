from typing import TypedDict, Annotated
import operator


class WorkflowState(TypedDict):
    steps_completed: Annotated[list, operator.add]
    total_score: Annotated[int, operator.add]
    current_status: str


def node_a(state: WorkflowState) -> dict:
    print(f"Node A received: {state}")
    return {
        "steps_completed": ["Node A processed"],
        "total_score": 10,
        "current_status": "A done"
    }


def node_b(state: WorkflowState) -> dict:
    print(f"Node B received: {state}")
    return {
        "steps_completed": ["Node B processed"],
        "total_score": 20,
        "current_status": "B done"
    }


def simulate_workflow():
    initial_state = {
        "steps_completed": [],
        "total_score": 0,
        "current_status": "started"
    }
    
    print("Initial State:", initial_state)
    print("\n--- Node A executes ---")
    update_a = node_a(initial_state)
    
    merged_state = {
        "steps_completed": initial_state["steps_completed"] + update_a["steps_completed"],
        "total_score": initial_state["total_score"] + update_a["total_score"],
        "current_status": update_a["current_status"]
    }
    print("State after Node A:", merged_state)
    
    print("\n--- Node B executes ---")
    update_b = node_b(merged_state)
    
    final_state = {
        "steps_completed": merged_state["steps_completed"] + update_b["steps_completed"],
        "total_score": merged_state["total_score"] + update_b["total_score"],
        "current_status": update_b["current_status"]
    }
    print("Final State:", final_state)
    
    print("\n" + "="*50)
    print("SUMMARY:")
    print(f"  Steps completed: {final_state['steps_completed']}")
    print(f"  Total score: {final_state['total_score']} (accumulated)")
    print(f"  Status: {final_state['current_status']} (replaced)")


if __name__ == "__main__":
    simulate_workflow()
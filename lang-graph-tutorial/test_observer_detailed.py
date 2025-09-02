import os
from observer_agent import ObserverAgent, memories_db


def test_detailed():
    print("="*60)
    print("DETAILED OBSERVER TEST")
    print("="*60)
    
    observer = ObserverAgent()
    thread_id = "test-001"
    
    # Turn 1
    print("\n--- Turn 1 ---")
    result = observer.process_turn(
        "Hi! I'm Sarah and I love hiking.",
        "Hello Sarah! It's great to meet a hiking enthusiast!",
        thread_id
    )
    print(f"Memories after turn 1: {len(memories_db)}")
    for mem in memories_db:
        print(f"  - {mem['memory'][:50]}...")
    
    # Turn 2  
    print("\n--- Turn 2 ---")
    result = observer.process_turn(
        "I especially enjoy mountain trails. I'm vegetarian too.",
        "Mountain trails offer amazing views! Being vegetarian on trails requires good planning for nutrition.",
        thread_id
    )
    print(f"Memories after turn 2: {len(memories_db)}")
    for mem in memories_db[-3:]:  # Show last 3 memories
        print(f"  - {mem['memory'][:50]}...")
    
    # Turn 3
    print("\n--- Turn 3 ---")
    result = observer.process_turn(
        "My favorite mountain is Mt. Fuji. I climbed it last year.",
        "Wow, climbing Mt. Fuji is quite an achievement! That must have been an incredible experience.",
        thread_id
    )
    print(f"Memories after turn 3: {len(memories_db)}")
    for mem in memories_db[-3:]:  # Show last 3 memories
        print(f"  - {mem['memory'][:50]}...")
    
    print("\n" + "="*60)
    print("ALL MEMORIES RECORDED")
    print("="*60)
    
    for i, memory in enumerate(memories_db, 1):
        print(f"\n{i}. {memory['memory']}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        test_detailed()
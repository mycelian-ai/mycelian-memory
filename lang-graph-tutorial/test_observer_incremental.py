import os
from observer_agent import ObserverAgent, memories_db


def test_incremental():
    print("="*60)
    print("INCREMENTAL OBSERVER TEST")
    print("="*60)
    
    # Clear previous memories
    memories_db.clear()
    
    observer = ObserverAgent()
    thread_id = "test-incremental"
    
    conversation = [
        ("Hi, I'm Alex", "Hello Alex! Nice to meet you."),
        ("I like pizza", "Pizza is great! What's your favorite topping?"),
        ("Pepperoni", "Classic choice! Pepperoni is very popular.")
    ]
    
    for i, (human, assistant) in enumerate(conversation, 1):
        print(f"\n--- Turn {i} ---")
        print(f"Human: {human}")
        print(f"Assistant: {assistant}")
        
        observer.process_turn(human, assistant, thread_id)
        
        print(f"\nMemories after turn {i}: {len(memories_db)}")
        if memories_db:
            latest = memories_db[-1]
            print(f"Latest entry:")
            print(f"  User: {latest['user']}")
            print(f"  Assistant: {latest['assistant']}")
    
    print("\n" + "="*60)
    print("ALL ENTRIES")
    print("="*60)
    
    for i, entry in enumerate(memories_db, 1):
        print(f"\n{i}. Entry:")
        print(f"   User: {entry['user']}")
        print(f"   Assistant: {entry['assistant']}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY")
    else:
        test_incremental()
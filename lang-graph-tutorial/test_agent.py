import os
from agent import SimpleAgent


def test_agent():
    agent = SimpleAgent()
    
    test_cases = [
        "What's 100 divided by 4?",
        "Give me a random fact and tell me what 7 squared is",
        "What's the time in a nice formatted way?",
        "I need to calculate 15% of 240",
        "Just say hello without using any tools"
    ]
    
    for query in test_cases:
        print(f"\n{'='*50}")
        print(f"User: {query}")
        response = agent.run(query)
        print(f"Agent: {response}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        test_agent()
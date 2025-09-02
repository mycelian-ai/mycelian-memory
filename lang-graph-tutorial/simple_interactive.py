import os
from agent import SimpleAgent


def run_simple_interactive():
    print("="*50)
    print("SIMPLE INTERACTIVE AGENT")
    print("="*50)
    print("Type 'exit' to quit")
    print("="*50)
    
    agent = SimpleAgent()
    
    while True:
        user_input = input("\nYou: ").strip()
        
        if user_input.lower() in ['exit', 'quit', 'bye']:
            print("Goodbye!")
            break
        
        if not user_input:
            continue
        
        response = agent.run(user_input)
        print(f"Agent: {response}")


if __name__ == "__main__":
    if not os.getenv("OPENAI_API_KEY"):
        print("Please set OPENAI_API_KEY environment variable")
    else:
        run_simple_interactive()
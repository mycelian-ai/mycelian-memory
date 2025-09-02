from langchain_core.tools import tool
from datetime import datetime
import random


@tool
def calculator(expression: str) -> float:
    """Evaluate a mathematical expression"""
    try:
        result = eval(expression, {"__builtins__": {}}, {})
        return float(result)
    except:
        return "Error: Invalid mathematical expression"


@tool  
def get_current_time() -> str:
    """Get the current date and time"""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


@tool
def get_random_fact() -> str:
    """Get a random interesting fact"""
    facts = [
        "Octopuses have three hearts",
        "Bananas are berries, but strawberries aren't",
        "A group of flamingos is called a 'flamboyance'",
        "Honey never spoils",
        "The Moon is moving away from Earth at 3.8 cm per year"
    ]
    return random.choice(facts)


tools = [calculator, get_current_time, get_random_fact]


if __name__ == "__main__":
    print("Available tools:")
    for t in tools:
        print(f"  - {t.name}: {t.description}")
    
    print("\nTesting tools:")
    print(f"Calculator: 2 + 3 * 4 = {calculator.invoke({'expression': '2 + 3 * 4'})}")
    print(f"Current time: {get_current_time.invoke({})}")
    print(f"Random fact: {get_random_fact.invoke({})}")
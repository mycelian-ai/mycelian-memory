#!/usr/bin/env python3
"""Simple script to test QA with the model directly."""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from langchain_openai import ChatOpenAI

# The context from the search
context = """
Context update: The user plans to organize their closet by season, starting with storing winter clothes in boxes (e.g., thick grey scarf, gloves) and bringing out spring items like a yellow sundress. They confirmed using boxes they already have to store winter pieces. They also mentioned having exchanged a pair of boots from Zara on 2/5 and needing to pick up the new pair, and asked for tips on staying organized with pickups and returns.

Entry summaries:
1. Used boxes the user has at home for seasonal storage; Provided six tracking tips: create a To-Pickup list, set reminders, designate a Returns spot, keep receipts organized, use a task-management app (Todoist, Trello, AnyList) for errands/shopping, and take a photo as backup.
2. User planned to return boots to Zara; boots were purchased on 2025-02-05, were too small, and were exchanged for a larger size; pickup had not yet occurred.
3. [plan] Advised user to prioritize picking up the new boots before the concert at Zara (exchanged on 2025-02-05), advised checking return/exchange policies to ensure time to return old boots, and offered to help with outfit ideas.
4. [plan] The user planned to pick up the navy blue blazer's dry cleaning, which they wore to a meeting a few weeks ago, and asked for best care practices for that blazer.
5. Designated a weekly laundry day for gym gear.

Additional context mentions: "keep in mind upcoming dry-cleaning task for navy blazer" and "remind to set a dry-cleaning reminder."
"""

def test_qa():
    llm = ChatOpenAI(
        model="gpt-5-2025-08-07",
        temperature=0,
        max_tokens=500
    )

    questions = [
        "Based on this context, count the total number of items of clothing the user needs to pick up or return from any store (including dry cleaners). List each item and then give the total count.",
        "Looking at the context, I see mentions of: 1) Zara boots to pick up, 2) possible old boots to return, 3) navy blazer at dry cleaner. How many total items is that?",
        "Is there anything about a navy blazer in the context?",
        "What needs to be picked up from the dry cleaner?"
    ]

    for q in questions:
        prompt = f"{context}\n\nQuestion: {q}\n\nAnswer based only on the context provided:"

        print(f"\n{'='*60}")
        print(f"Q: {q}")
        print("-"*60)

        try:
            response = llm.invoke(prompt)
            answer = response.content if hasattr(response, 'content') else str(response)
            print(f"A: {answer}")
        except Exception as e:
            print(f"Error: {e}")

if __name__ == "__main__":
    test_qa()

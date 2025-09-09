#!/usr/bin/env python3
"""Create minimal dataset for question 0a995998 with only relevant sessions."""

import json

# Load the subset dataset we already have
with open('longmemeval-datasets/longmemeval_0a995998_s10.json', 'r') as f:
    data = json.load(f)

question = data[0]

# We need:
# - Session 3 (index 3-10=0 in subset): navy blazer
# - Session 11 (index 11-10=1 in subset): green sweater & Zara boots  
# - Maybe 1-2 sessions before for context

# Let's keep sessions 0-3 (which are sessions 10-13 in original)
# This gives us 2 sessions before, the navy blazer session, and the Zara boots session

minimal_question = question.copy()

# Keep only first 4 sessions (indices 0-3, which are sessions 10-13 in original)
minimal_question['haystack_sessions'] = question['haystack_sessions'][:4]

# Adjust metadata
if 'haystack_session_ids' in minimal_question:
    minimal_question['haystack_session_ids'] = minimal_question['haystack_session_ids'][:4]
if 'haystack_dates' in minimal_question:
    minimal_question['haystack_dates'] = minimal_question['haystack_dates'][:4]

# The answer remains "3"
print(f"Original sessions: {len(question['haystack_sessions'])}")
print(f"Minimal sessions: {len(minimal_question['haystack_sessions'])}")
print(f"Question: {minimal_question['question']}")
print(f"Answer: {minimal_question['answer']}")

# Save minimal dataset
output_file = 'longmemeval-datasets/longmemeval_0a995998_minimal.json'
with open(output_file, 'w') as f:
    json.dump([minimal_question], f, indent=2)

print(f"\nCreated minimal dataset: {output_file}")
print("This includes sessions 10-13 from original:")
print("  - Session 10-11: Setup/context")
print("  - Session 12: Navy blazer mention")
print("  - Session 13: Green sweater & Zara boots mention")

#!/usr/bin/env python3
"""Create minimal dataset with only clothing item sessions."""

import json

with open('longmemeval-datasets/longmemeval_0a995998_s10.json', 'r') as f:
    data = json.load(f)

question = data[0]

# We need:
# Session 2 (navy blazer)
# Session 10 (green sweater & Zara boots) 
# Plus 1-2 sessions before each for context

# Let's take sessions 0-11 (covers all clothing mentions with minimal extras)
minimal_question = question.copy()
minimal_question['haystack_sessions'] = question['haystack_sessions'][:11]

# Adjust metadata
if 'haystack_session_ids' in minimal_question:
    minimal_question['haystack_session_ids'] = minimal_question['haystack_session_ids'][:11]
if 'haystack_dates' in minimal_question:
    minimal_question['haystack_dates'] = minimal_question['haystack_dates'][:11]

print(f"Original sessions: {len(question['haystack_sessions'])}")
print(f"Minimal sessions: {len(minimal_question['haystack_sessions'])}")
print(f"Sessions included: 0-10 (original 10-20)")
print(f"  - Session 2: Navy blazer")
print(f"  - Session 10: Green sweater & Zara boots")

# Save minimal dataset
output_file = 'longmemeval-datasets/longmemeval_0a995998_minimal.json'
with open(output_file, 'w') as f:
    json.dump([minimal_question], f, indent=2)

print(f"\nCreated: {output_file}")

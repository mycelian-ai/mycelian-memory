#!/usr/bin/env python3
"""Extract question 0a995998 with relevant sessions for clothing items."""

import json

# Load the full dataset
with open('longmemeval-datasets/longmemeval_5s.json', 'r') as f:
    data = json.load(f)

# Find question 0a995998
target_question = None
for q in data:
    if q['question_id'] == '0a995998':
        target_question = q
        break

if not target_question:
    print("Question 0a995998 not found!")
    exit(1)

# Create a subset with sessions from index 10 onwards (to include all clothing mentions)
subset_question = target_question.copy()
subset_question['haystack_sessions'] = target_question['haystack_sessions'][10:]

# Adjust the haystack_session_ids and dates accordingly
original_session_count = len(target_question['haystack_sessions'])
subset_session_count = len(subset_question['haystack_sessions'])
sessions_removed = original_session_count - subset_session_count

# Trim the session IDs and dates lists
if 'haystack_session_ids' in subset_question:
    subset_question['haystack_session_ids'] = subset_question['haystack_session_ids'][sessions_removed:]
if 'haystack_dates' in subset_question:
    subset_question['haystack_dates'] = subset_question['haystack_dates'][sessions_removed:]

# Fix answer_session_ids to match retained sessions
if 'answer_session_ids' in subset_question:
    valid_answer_ids = [aid for aid in subset_question['answer_session_ids'] 
                        if aid in subset_question.get('haystack_session_ids', [])]
    subset_question['answer_session_ids'] = valid_answer_ids if valid_answer_ids else []

# Create dataset with just this question
dataset = [subset_question]

# Save to new file
output_file = 'longmemeval-datasets/longmemeval_0a995998_s10.json'
with open(output_file, 'w') as f:
    json.dump(dataset, f, indent=2)

print(f"Created subset dataset: {output_file}")
print(f"Original sessions: {original_session_count}")
print(f"Subset sessions: {subset_session_count} (starting from session 10)")
print(f"Question: {subset_question['question']}")
print(f"Answer: {subset_question['answer']}")

# Check which sessions have clothing items
print("\nClothing items in subset:")
for i, session in enumerate(subset_question['haystack_sessions'], start=10):
    for turn in session:
        content = turn.get('content', '').lower()
        if 'navy blue blazer' in content and 'dry clean' in content:
            print(f"  Session {i}: Navy blue blazer (dry cleaning)")
            break
        if 'green sweater' in content and 'sister' in content:
            print(f"  Session {i}: Green sweater (borrowed from sister)")
            break
        if 'boots' in content and 'zara' in content and ('pick up' in content or 'exchanged' in content):
            print(f"  Session {i}: Boots from Zara (pickup after exchange)")
            break
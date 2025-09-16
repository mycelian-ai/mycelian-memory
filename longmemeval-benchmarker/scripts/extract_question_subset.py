#!/usr/bin/env python3
"""Extract a specific question with sessions from a given index onwards."""

import json
import sys

# Load the full dataset
with open('longmemeval-datasets/longmemeval_5s.json', 'r') as f:
    data = json.load(f)

# Find question 6a1eabeb
target_question = None
for q in data:
    if q['question_id'] == '6a1eabeb':
        target_question = q
        break

if not target_question:
    print("Question 6a1eabeb not found!")
    sys.exit(1)

# Create a subset with sessions from index 35 onwards
subset_question = target_question.copy()
subset_question['haystack_sessions'] = target_question['haystack_sessions'][35:]

# Adjust the haystack_session_ids and dates accordingly
# These lists should match the number of sessions
original_session_count = len(target_question['haystack_sessions'])
subset_session_count = len(subset_question['haystack_sessions'])
sessions_removed = original_session_count - subset_session_count

# Trim the session IDs and dates lists
if 'haystack_session_ids' in subset_question:
    subset_question['haystack_session_ids'] = subset_question['haystack_session_ids'][sessions_removed:]
if 'haystack_dates' in subset_question:
    subset_question['haystack_dates'] = subset_question['haystack_dates'][sessions_removed:]

# Fix answer_session_ids to match what's actually in the subset
# The answer is in session 40 (index 5 in our subset starting from 35)
# We need to check which answer_session_ids correspond to our retained sessions
if 'answer_session_ids' in subset_question:
    # For this specific question, we know session 40 has the answer
    # Check if any answer_session_ids are in our retained haystack_session_ids
    valid_answer_ids = [aid for aid in subset_question['answer_session_ids']
                        if aid in subset_question.get('haystack_session_ids', [])]

    # If no valid IDs, we need to identify which session has the answer
    if not valid_answer_ids:
        # Session 40 (index 5 in our subset) has the answer
        # Use the corresponding session ID
        if len(subset_question.get('haystack_session_ids', [])) > 5:
            subset_question['answer_session_ids'] = [subset_question['haystack_session_ids'][5]]
        else:
            subset_question['answer_session_ids'] = []
    else:
        subset_question['answer_session_ids'] = valid_answer_ids

# Create dataset with just this question
dataset = [subset_question]

# Save to new file
output_file = 'longmemeval-datasets/longmemeval_6a1eabeb_s35.json'
with open(output_file, 'w') as f:
    json.dump(dataset, f, indent=2)

print(f"Created subset dataset: {output_file}")
print(f"Original sessions: {original_session_count}")
print(f"Subset sessions: {subset_session_count} (starting from session 35)")
print(f"Question: {subset_question['question']}")
print(f"Answer: {subset_question['answer']}")

# Check which session has the answer
for i, session in enumerate(subset_question['haystack_sessions'], start=35):
    for turn in session:
        if 'has_answer' in turn and turn['has_answer']:
            print(f"Answer found in session {i}, turn {session.index(turn)}")
            if '25:50' in turn.get('content', '') or '25 minutes and 50 seconds' in turn.get('content', ''):
                print(f"  ✓ Correct answer (25:50) in session {i}")
            elif '27:12' in turn.get('content', ''):
                print(f"  ✗ Incorrect answer (27:12) in session {i}")

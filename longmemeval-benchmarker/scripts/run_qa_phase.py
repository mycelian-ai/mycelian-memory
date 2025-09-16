#!/usr/bin/env python3
"""Run QA phase on already ingested memories."""

import json
import sys
from debug_qa import search_and_qa

# Configuration
VAULT_ID = "be66fd4e-0256-4287-b16c-7c8e862d9b92"
RUN_ID = "1757058617"
MODEL = "openai:gpt-5-2025-08-07"
USE_TWO_PASS = True

# Map of question IDs to memory IDs from the ingestion run
MEMORY_MAP = {
    "e47becba": "28a68308-edf1-4e9f-abc0-4c362c395c7c",  # Question 1
    "0a995998": "7a745cb9-740e-4136-98a2-4140f24dfa61",  # Question 2
}

# Load the dataset
with open("longmemeval-datasets/longmemeval_5s.json", "r") as f:
    dataset = json.load(f)

# Run QA on each ingested question
results = []
for q in dataset[:2]:  # First 2 questions that were ingested
    qid = q["question_id"]
    question_text = q.get("question", "No question provided")

    if qid not in MEMORY_MAP:
        print(f"Skipping {qid} - no memory found")
        continue

    memory_id = MEMORY_MAP[qid]
    print(f"\n{'='*60}")
    print(f"Question ID: {qid}")
    print(f"Memory ID: {memory_id}")
    print(f"Question: {question_text}")
    print(f"{'='*60}")

    try:
        result = search_and_qa(
            memory_id=memory_id,
            vault_id=VAULT_ID,
            question=question_text,
            model=MODEL,
            use_two_pass=USE_TWO_PASS
        )

        print(f"\nAnswer: {result['answer']}")

        # Store result in LongMemEval format
        results.append({
            "question_id": qid,
            "hypothesis": result['answer']
        })

    except Exception as e:
        print(f"Error processing {qid}: {e}")
        results.append({
            "question_id": qid,
            "hypothesis": ""
        })

# Write results to hypotheses file
output_file = f"out/run_{RUN_ID}/hypotheses_qa_only.jsonl"
print(f"\n\nWriting results to {output_file}")

with open(output_file, "w") as f:
    for result in results:
        f.write(json.dumps(result) + "\n")

print(f"Wrote {len(results)} QA results")

# Display summary
print("\n" + "="*60)
print("QA PHASE SUMMARY")
print("="*60)
for result in results:
    qid = result['question_id']
    hyp = result['hypothesis'][:100] + "..." if len(result['hypothesis']) > 100 else result['hypothesis']
    print(f"{qid}: {hyp}")

#!/usr/bin/env python3
"""
Create a trimmed oracle file matching our 5-question subset.
"""

import json
import argparse
from pathlib import Path


def main():
    ap = argparse.ArgumentParser(description="Create trimmed oracle file for 5-question subset.")
    ap.add_argument("--subset-file", default="out/longmemeval_5s.json", 
                    help="Path to the 5-question subset file")
    ap.add_argument("--oracle-file", default="~/workspace/LongMemEval/data/longmemeval_oracle.json",
                    help="Path to the full oracle file")
    ap.add_argument("--output", default="out/longmemeval_oracle.json",
                    help="Output oracle file")
    args = ap.parse_args()

    # Expand ~ in paths
    oracle_path = Path(args.oracle_file).expanduser()
    subset_path = Path(args.subset_file)
    output_path = Path(args.output)

    print(f"Loading subset from: {subset_path}")
    with subset_path.open("r", encoding="utf-8") as f:
        subset = json.load(f)
    
    # Extract question IDs from our subset
    target_ids = set(q.get("question_id") for q in subset)
    print(f"Target question IDs: {sorted(target_ids)}")
    
    print(f"Loading full oracle from: {oracle_path}")
    if not oracle_path.exists():
        raise FileNotFoundError(f"Oracle file not found: {oracle_path}")
    
    with oracle_path.open("r", encoding="utf-8") as f:
        full_oracle = json.load(f)
    
    print(f"Full oracle contains {len(full_oracle)} entries")
    
    # Filter oracle to match our subset
    filtered_oracle = []
    found_ids = set()
    
    for entry in full_oracle:
        qid = entry.get("question_id")
        if qid in target_ids:
            filtered_oracle.append(entry)
            found_ids.add(qid)
    
    print(f"Found {len(filtered_oracle)} matching oracle entries")
    print(f"Matched IDs: {sorted(found_ids)}")
    
    missing_ids = target_ids - found_ids
    if missing_ids:
        print(f"WARNING: Missing oracle entries for: {sorted(missing_ids)}")
    
    # Save filtered oracle
    output_path.parent.mkdir(exist_ok=True)
    with output_path.open("w", encoding="utf-8") as f:
        json.dump(filtered_oracle, f, indent=2, ensure_ascii=False)
    
    print(f"Saved trimmed oracle: {output_path.resolve()}")
    
    # Show summary
    print("\nOracle subset summary:")
    for i, entry in enumerate(filtered_oracle, 1):
        qid = entry.get("question_id")
        qtype = entry.get("question_type", "unknown")
        question = entry.get("question", "")[:50] + "..."
        answer = str(entry.get("answer", ""))[:30] + "..."
        print(f"  {i}. {qid} ({qtype})")
        print(f"     Q: {question}")
        print(f"     A: {answer}")


if __name__ == "__main__":
    main()

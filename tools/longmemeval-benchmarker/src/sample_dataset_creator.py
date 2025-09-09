#!/usr/bin/env python3
"""
Create a subset from LongMemEval with questions from core abilities:
- IE (single-session-user)
- MR (multi-session)
- KU (knowledge-update)
- TR (temporal-reasoning)
- ABS (any with '_abs' in question_id)
- SP (single-session-preference)
- SA (single-session-assistant)

Sampling is systematic (first occurrence), to avoid cherry-picking.
"""

import json
import argparse
from collections import defaultdict
from pathlib import Path
from typing import List, Dict


def load_dataset(dataset_path: str) -> List[Dict]:
    path = Path(dataset_path)

    # If it's a direct file path, use it
    if path.is_file():
        with path.open("r", encoding="utf-8") as f:
            return json.load(f)

    # Otherwise, treat as directory and search for known files
    if path.is_dir():
        candidates = [
            path / "longmemeval_s.json",
            path / "data" / "longmemeval_s.json",
            path / "longmemeval_m.json",
            path / "data" / "longmemeval_m.json",
            path / "longmemeval_oracle.json",
            path / "data" / "longmemeval_oracle.json",
        ]
        src = next((p for p in candidates if p.exists()), None)
        if not src:
            raise FileNotFoundError(
                f"Could not find longmemeval_s.json, longmemeval_m.json, or longmemeval_oracle.json under {dataset_path}"
            )
        with src.open("r", encoding="utf-8") as f:
            return json.load(f)

    raise FileNotFoundError(f"Path does not exist: {dataset_path}")


def sample_questions(questions: List[Dict], target_types: List[str], num_questions: int) -> List[Dict]:
    by_type: Dict[str, List[Dict]] = defaultdict(list)
    for q in questions:
        by_type[q.get("question_type", "unknown")].append(q)

    sampled: List[Dict] = []
    seen_ids: set[str] = set()

    # First, try to get one from each target type
    for qtype in target_types:
        if len(sampled) >= num_questions:
            break
        items = by_type.get(qtype, [])
        if not items:
            print(f"Warning: no questions found for type '{qtype}'")
            continue
        sel = items[0]
        qid = sel.get("question_id", "")
        if qid not in seen_ids:
            sampled.append(sel)
            seen_ids.add(qid)
            print(f"Selected {qtype}: question_id={qid}")
        else:
            print(f"Skipped duplicate for type {qtype}: {qid}")

    # Add abstention question if we have room and haven't reached the limit
    if len(sampled) < num_questions:
        abs_q = next((q for q in questions if "_abs" in (q.get("question_id") or "")), None)
        if abs_q:
            qid = abs_q.get("question_id", "")
            if qid not in seen_ids:
                sampled.append(abs_q)
                seen_ids.add(qid)
                print(f"Selected ABS: question_id={qid}")
            else:
                print(f"ABS candidate duplicated existing selection: {qid}")
        else:
            print("Warning: no abstention ('_abs') question found")

    # If we still need more questions, add more from available types
    if len(sampled) < num_questions:
        for qtype in target_types:
            if len(sampled) >= num_questions:
                break
            items = by_type.get(qtype, [])
            for item in items[1:]:  # Skip first (already selected)
                if len(sampled) >= num_questions:
                    break
                qid = item.get("question_id", "")
                if qid not in seen_ids:
                    sampled.append(item)
                    seen_ids.add(qid)
                    print(f"Selected additional {qtype}: question_id={qid}")

    return sampled[:num_questions]


def main() -> None:
    ap = argparse.ArgumentParser(description="Create LongMemEval subset (systematic sampling).")
    ap.add_argument("dataset_path", help="Path to LongMemEval dataset file (e.g., longmemeval_s.json) or directory containing it")
    ap.add_argument("--num-questions", type=int, default=5, help="Number of questions to sample (default: 5)")
    ap.add_argument("--question-id", help="Filter to a specific question ID (overrides --num-questions)")
    ap.add_argument("--output", help="Output JSON filename (default: longmemeval_s_{num}.json or longmemeval_{question_id}.json)")
    ap.add_argument("--copy-as-s", action="store_true",
                    help="Also write a copy named 'longmemeval_s.json' for loaders that expect that name")
    args = ap.parse_args()

    # Handle question ID filtering vs sampling
    if args.question_id:
        if not args.output:
            args.output = f"longmemeval_{args.question_id}.json"

        print(f"Loading dataset from: {args.dataset_path}")
        questions = load_dataset(args.dataset_path)
        print(f"Total questions loaded: {len(questions)}")

        # Filter to specific question ID
        subset = [q for q in questions if q.get("question_id") == args.question_id]
        if not subset:
            print(f"Error: Question ID '{args.question_id}' not found in dataset")
            return
        print(f"Found question: {args.question_id} ({subset[0].get('question_type', 'unknown')})")
    else:
        if not args.output:
            args.output = f"longmemeval_s_{args.num_questions}.json"

        target_types = [
            "single-session-user",      # IE
            "multi-session",            # MR
            "knowledge-update",         # KU
            "temporal-reasoning",       # TR
            "single-session-preference", # SP
            "single-session-assistant", # SA
            # ABS handled by '_abs' in question_id
        ]

        print(f"Loading dataset from: {args.dataset_path}")
        questions = load_dataset(args.dataset_path)
        print(f"Total questions loaded: {len(questions)}")

        subset = sample_questions(questions, target_types, args.num_questions)
        print(f"Subset size: {len(subset)}")

    out_path = Path(args.output)
    with out_path.open("w", encoding="utf-8") as f:
        json.dump(subset, f, indent=2, ensure_ascii=False)
    print(f"Wrote: {out_path.resolve()}")

    if args.copy_as_s:
        alias = out_path.with_name("longmemeval_s.json")
        with alias.open("w", encoding="utf-8") as f:
            json.dump(subset, f, indent=2, ensure_ascii=False)
        print(f"Also wrote: {alias.resolve()} (for loader compatibility)")

    print("\nSubset summary:")
    for i, q in enumerate(subset, 1):
        question_text = q.get('question', 'No question text')
        if len(question_text) > 80:
            question_text = question_text[:77] + "..."
        print(f"{i}. {q.get('question_id')} ({q.get('question_type')}): {question_text}")


if __name__ == "__main__":
    main()

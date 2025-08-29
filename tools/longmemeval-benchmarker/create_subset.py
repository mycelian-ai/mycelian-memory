#!/usr/bin/env python3
"""
Create a 5-question subset from LongMemEval with one question per core ability:
- IE (single-session-user)
- MR (multi-session)
- KU (knowledge-update)
- TR (temporal-reasoning)
- ABS (any with '_abs' in question_id)

Sampling is systematic (first occurrence), to avoid cherry-picking.
"""

import json
import argparse
from collections import defaultdict
from pathlib import Path
from typing import List, Dict


def load_dataset(dataset_path: str) -> List[Dict]:
    root = Path(dataset_path)
    candidates = [
        root / "longmemeval_s.json",
        root / "data" / "longmemeval_s.json",
        root / "longmemeval_oracle.json",
        root / "data" / "longmemeval_oracle.json",
    ]
    src = next((p for p in candidates if p.exists()), None)
    if not src:
        raise FileNotFoundError(
            f"Could not find longmemeval_s.json or longmemeval_oracle.json under {dataset_path}"
        )
    with src.open("r", encoding="utf-8") as f:
        return json.load(f)


def sample_one_per_type(questions: List[Dict], target_types: List[str]) -> List[Dict]:
    by_type: Dict[str, List[Dict]] = defaultdict(list)
    for q in questions:
        by_type[q.get("question_type", "unknown")].append(q)

    sampled: List[Dict] = []
    seen_ids: set[str] = set()

    for qtype in target_types:
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

    # ABS: pick first with '_abs' in question_id
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

    # Keep exactly 5 if more were selected
    if len(sampled) > 5:
        sampled = sampled[:5]

    return sampled


def main() -> None:
    ap = argparse.ArgumentParser(description="Create 5-question LongMemEval subset (systematic sampling).")
    ap.add_argument("dataset_path", help="Path to LongMemEval dataset root (folder containing longmemeval_s.json)")
    ap.add_argument("--output", default="longmemeval_s_5.json", help="Output JSON filename")
    ap.add_argument("--copy-as-s", action="store_true",
                    help="Also write a copy named 'longmemeval_s.json' for loaders that expect that name")
    args = ap.parse_args()

    target_types = [
        "single-session-user",   # IE
        "multi-session",         # MR
        "knowledge-update",      # KU
        "temporal-reasoning",    # TR
        # ABS handled by '_abs' in question_id
    ]

    print(f"Loading dataset from: {args.dataset_path}")
    questions = load_dataset(args.dataset_path)
    print(f"Total questions loaded: {len(questions)}")

    subset = sample_one_per_type(questions, target_types)
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
        print(f"{i}. {q.get('question_id')} ({q.get('question_type')})")


if __name__ == "__main__":
    main()



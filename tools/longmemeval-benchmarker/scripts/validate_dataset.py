#!/usr/bin/env python3
"""
Dataset validator for LongMemEval datasets.
Validates that a dataset complies with the LongMemEval specification.
"""

import json
import sys
import argparse
from pathlib import Path
from typing import Dict, List, Any
from collections import Counter


class DatasetValidator:
    """Validator for LongMemEval dataset compliance."""

    # Valid question types per spec
    VALID_QUESTION_TYPES = {
        "single-session-user",
        "single-session-assistant",
        "single-session-preference",
        "temporal-reasoning",
        "knowledge-update",
        "multi-session"
    }

    # Required fields per spec
    REQUIRED_FIELDS = {
        "question_id",
        "question_type",
        "question",
        "answer",
        "question_date",
        "haystack_session_ids",
        "haystack_dates",
        "haystack_sessions",
        "answer_session_ids"
    }

    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self.errors = []
        self.warnings = []

    def validate_dataset(self, dataset_path: str) -> bool:
        """
        Validate an entire dataset file.

        Args:
            dataset_path: Path to the JSON dataset file

        Returns:
            True if valid, False otherwise
        """
        # Load dataset
        try:
            with open(dataset_path, 'r') as f:
                dataset = json.load(f)
        except json.JSONDecodeError as e:
            self.errors.append(f"Invalid JSON: {e}")
            return False
        except FileNotFoundError:
            self.errors.append(f"File not found: {dataset_path}")
            return False

        if not isinstance(dataset, list):
            self.errors.append("Dataset must be a JSON array")
            return False

        if len(dataset) == 0:
            self.warnings.append("Dataset is empty")
            return True

        print(f"Validating {len(dataset)} questions...")

        # Track statistics
        stats = {
            "total": len(dataset),
            "valid": 0,
            "invalid": 0,
            "question_types": Counter(),
            "abstention": 0,
            "total_sessions": 0,
            "non_string_answers": 0
        }

        # Validate each question
        for idx, question in enumerate(dataset):
            question_id = question.get("question_id", f"unknown_{idx}")

            if self.verbose:
                print(f"\nValidating question {idx + 1}/{len(dataset)}: {question_id}")

            is_valid, errors, warnings = self.validate_question(question)

            if is_valid:
                stats["valid"] += 1
                if self.verbose:
                    print(f"  ✓ Valid")
            else:
                stats["invalid"] += 1
                print(f"  ✗ Question {idx + 1} ({question_id}): INVALID")
                for error in errors:
                    print(f"    ERROR: {error}")
                    self.errors.append(f"Question {idx} ({question_id}): {error}")

            for warning in warnings:
                if self.verbose:
                    print(f"    WARNING: {warning}")
                self.warnings.append(f"Question {idx} ({question_id}): {warning}")

            # Collect stats
            if "question_type" in question:
                stats["question_types"][question["question_type"]] += 1
            if question_id.endswith("_abs"):
                stats["abstention"] += 1
            if "haystack_sessions" in question:
                stats["total_sessions"] += len(question["haystack_sessions"])
            if "answer" in question and not isinstance(question["answer"], str):
                stats["non_string_answers"] += 1

        return stats["invalid"] == 0, stats

    def validate_question(self, question: Dict[str, Any]) -> tuple[bool, List[str], List[str]]:
        """
        Validate a single question structure.

        Returns:
            Tuple of (is_valid, errors, warnings)
        """
        errors = []
        warnings = []

        # Check required fields
        for field in self.REQUIRED_FIELDS:
            if field not in question:
                errors.append(f"Missing required field: {field}")

        if errors:
            return False, errors, warnings

        # Validate question_id
        if not isinstance(question["question_id"], str) or len(question["question_id"]) == 0:
            errors.append("question_id must be a non-empty string")

        # Validate question_type
        qtype = question["question_type"]
        if qtype not in self.VALID_QUESTION_TYPES:
            errors.append(f"Invalid question_type: {qtype}. Must be one of: {', '.join(self.VALID_QUESTION_TYPES)}")

        # Check abstention naming convention
        if question["question_id"].endswith("_abs"):
            if len(question.get("answer_session_ids", [])) > 0:
                warnings.append("Abstention question has non-empty answer_session_ids")

        # Validate question and answer
        if not isinstance(question["question"], str):
            errors.append("question must be a string")

        # Answer can be string or int (per actual dataset)
        if not isinstance(question["answer"], (str, int)):
            errors.append(f"answer must be string or int, got {type(question['answer']).__name__}")
        elif not isinstance(question["answer"], str):
            warnings.append(f"answer is {type(question['answer']).__name__} (spec says string)")

        # Validate date
        if not isinstance(question["question_date"], str):
            errors.append("question_date must be a string")

        # Validate lists
        for field in ["haystack_session_ids", "haystack_dates", "haystack_sessions", "answer_session_ids"]:
            if not isinstance(question[field], list):
                errors.append(f"{field} must be a list")

        # Check list length consistency
        if not errors:  # Only if lists exist
            num_sessions = len(question["haystack_sessions"])

            if len(question["haystack_session_ids"]) != num_sessions:
                errors.append(f"haystack_session_ids length ({len(question['haystack_session_ids'])}) "
                            f"doesn't match haystack_sessions ({num_sessions})")

            if len(question["haystack_dates"]) != num_sessions:
                errors.append(f"haystack_dates length ({len(question['haystack_dates'])}) "
                            f"doesn't match haystack_sessions ({num_sessions})")

        # Validate session structure
        for sess_idx, session in enumerate(question.get("haystack_sessions", [])):
            if not isinstance(session, list):
                errors.append(f"Session {sess_idx} must be a list of turns")
                continue

            for turn_idx, turn in enumerate(session):
                if not isinstance(turn, dict):
                    errors.append(f"Session {sess_idx}, turn {turn_idx} must be a dict")
                    continue

                if "role" not in turn:
                    errors.append(f"Session {sess_idx}, turn {turn_idx} missing 'role'")
                elif turn["role"] not in ["user", "assistant"]:
                    errors.append(f"Session {sess_idx}, turn {turn_idx} invalid role: {turn['role']}")

                if "content" not in turn:
                    errors.append(f"Session {sess_idx}, turn {turn_idx} missing 'content'")
                elif not isinstance(turn["content"], str):
                    errors.append(f"Session {sess_idx}, turn {turn_idx} content must be string")

                if "has_answer" in turn and not isinstance(turn["has_answer"], bool):
                    errors.append(f"Session {sess_idx}, turn {turn_idx} has_answer must be boolean")

        # Validate answer_session_ids reference valid sessions
        for answer_id in question.get("answer_session_ids", []):
            if answer_id not in question.get("haystack_session_ids", []):
                errors.append(f"answer_session_id '{answer_id}' not in haystack_session_ids")

        return len(errors) == 0, errors, warnings


def print_stats(stats: Dict) -> None:
    """Print dataset statistics."""
    print("\n" + "="*60)
    print("DATASET STATISTICS")
    print("="*60)

    print(f"Total questions: {stats['total']}")
    print(f"Valid questions: {stats['valid']} ({stats['valid']/max(stats['total'], 1)*100:.1f}%)")
    print(f"Invalid questions: {stats['invalid']}")
    print(f"Total sessions: {stats['total_sessions']}")

    if stats['total'] > 0:
        print(f"Average sessions per question: {stats['total_sessions']/stats['total']:.1f}")

    print(f"\nQuestion types:")
    for qtype, count in sorted(stats['question_types'].items()):
        print(f"  - {qtype}: {count}")

    print(f"\nSpecial questions:")
    print(f"  - Abstention questions: {stats['abstention']}")
    print(f"  - Non-string answers: {stats['non_string_answers']}")


def main():
    """Main CLI entry point."""
    parser = argparse.ArgumentParser(
        description="Validate LongMemEval datasets for specification compliance"
    )
    parser.add_argument(
        "dataset",
        help="Path to the dataset JSON file to validate"
    )
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Show detailed validation output"
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat warnings as errors (strict mode)"
    )

    args = parser.parse_args()

    # Create validator
    validator = DatasetValidator(verbose=args.verbose)

    # Validate dataset
    is_valid, stats = validator.validate_dataset(args.dataset)

    # Print results
    print_stats(stats)

    # Print errors and warnings summary
    if validator.errors:
        print("\n" + "="*60)
        print(f"ERRORS ({len(validator.errors)})")
        print("="*60)
        for error in validator.errors[:10]:  # Show first 10
            print(f"  - {error}")
        if len(validator.errors) > 10:
            print(f"  ... and {len(validator.errors) - 10} more errors")

    if validator.warnings:
        print("\n" + "="*60)
        print(f"WARNINGS ({len(validator.warnings)})")
        print("="*60)
        for warning in validator.warnings[:10]:  # Show first 10
            print(f"  - {warning}")
        if len(validator.warnings) > 10:
            print(f"  ... and {len(validator.warnings) - 10} more warnings")

    # Final result
    print("\n" + "="*60)
    if is_valid and (not args.strict or not validator.warnings):
        print("✅ DATASET IS VALID")
        return 0
    elif is_valid and validator.warnings and args.strict:
        print("⚠️  DATASET HAS WARNINGS (failed in strict mode)")
        return 1
    else:
        print("❌ DATASET IS INVALID")
        return 1


if __name__ == "__main__":
    sys.exit(main())

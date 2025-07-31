from __future__ import annotations

import json
from typing import List

from .models import BenchmarkConversation, ConversationMessage, TestQuestion
from .hf_utils import ensure_dataset


def load_msc_dataset(local_path: str | None = None, hf_repo: str = "MemGPT/MSC-Self-Instruct", split: str = "train", max_conv: int | None = None) -> List[BenchmarkConversation]:
    """Load the MemGPT MSC-Self-Instruct benchmark dataset.

    Parameters
    ----------
    local_path: Optional path to a pre-downloaded JSONL file. If ``None`` the
        dataset will be downloaded from Hugging Face using :pyfunc:`hf_utils.ensure_dataset`.
    hf_repo:  Hugging Face dataset identifier. Defaults to the canonical
        "MemGPT/MSC-Self-Instruct" repo.
    split:    Dataset split to use (e.g. "train", "test"). Defaults to "train".
    max_conv: If provided, stop after this many conversations (handy for quick
        smoke-tests).
    """
    # Lazily fetch the dataset from HF if required
    if local_path is None:
        local_path = str(ensure_dataset(hf_repo, split=split))

    conversations: List[BenchmarkConversation] = []
    with open(local_path, "r", encoding="utf-8") as f:
        for line in f:
            record = json.loads(line)
            # Build message list across sessions
            messages: List[ConversationMessage] = []
            # Track message texts we have already added to prevent duplicates.
            seen: set[str] = set()

            session_id = 1
            for dialog in record.get("previous_dialogs", []):
                for idx, exch in enumerate(dialog.get("dialog", [])):
                    text = exch.get("text", "").strip()

                    # Skip duplicate texts that have already been processed. The MSC dataset
                    # repeats the full conversation history in each successive
                    # ``previous_dialogs`` entry, so without deduplication we would emit the
                    # same "Hi!" or similar greeting many times, causing the benchmark
                    # simulator to replay them as fresh user turns.
                    if text in seen:
                        continue
                    seen.add(text)

                    # Prefer explicit speaker annotation when present.
                    raw_id = exch.get("id", "").lower().strip()
                    if raw_id == "speaker 1":
                        speaker = "user"
                    elif raw_id == "speaker 2":
                        speaker = "assistant"
                    else:
                        # Fallback: alternate, starting with user inside each dialog.
                        speaker = "user" if idx % 2 == 0 else "assistant"

                    messages.append(
                        ConversationMessage(
                            speaker=speaker,
                            text=text,
                            session_id=session_id,
                        )
                    )
                session_id += 1

            # MSC uses the ``self_instruct`` field for cross-session QA
            questions: List[TestQuestion] = []
            if "self_instruct" in record:
                si = record["self_instruct"]
                questions.append(
                    TestQuestion(
                        question=si.get("B", ""),
                        expected_answer=si.get("A", ""),
                        question_type="cross_session",
                    )
                )

            conversations.append(
                BenchmarkConversation(
                    conversation_id=record.get("initial_data_id", f"conv_{len(conversations)}"),
                    messages=messages,
                    test_questions=questions,
                    metadata=record,
                )
            )

            if max_conv and len(conversations) >= max_conv:
                break

    return conversations 
import os
from pathlib import Path

import pytest

from benchmarks.python.msc_loader import load_msc_dataset

DATASET_PATH = Path(__file__).resolve().parents[3] / "benchmarks" / "python" / "datasets" / "MemGPT_MSC-Self-Instruct_train.json"

@pytest.mark.asyncio
async def test_role_annotation_on_msc_dataset():
    """Load a sample of MSC conversations and assert role alternation logic.

    Expectations per conversation:
    1. The very first message of every session is labelled ``user``.
    2. Roles strictly alternate user -> assistant -> user … within a session.
    """
    # Restrict to 30 conversations for reasonable test time.
    conversations = load_msc_dataset(local_path=str(DATASET_PATH), max_conv=30)

    for conv in conversations:
        prev_role = None
        current_session = conv.messages[0].session_id if conv.messages else None

        for msg in conv.messages:
            # Reset alternation check when a new session starts
            if msg.session_id != current_session:
                current_session = msg.session_id
                prev_role = None

            # First message in a session must come from user
            if prev_role is None:
                assert msg.speaker == "user", (
                    f"Conversation {conv.conversation_id} session {current_session} "
                    f"should start with user, got {msg.speaker}"
                )
            else:
                # Subsequent roles must alternate
                assert msg.speaker != prev_role, (
                    f"Conversation {conv.conversation_id} session {current_session} "
                    f"has two consecutive {msg.speaker} messages"
                )

            prev_role = msg.speaker

        # --- Per-conversation invariants ----------------------------------
        # 1. No duplicate message texts inside one conversation
        texts = [m.text.strip() for m in conv.messages]
        assert len(texts) == len(set(texts)), (
            f"Conversation {conv.conversation_id} contains duplicate message texts—"
            "possible role-assignment duplication bug."
        )

        # 2. User and assistant turn counts should differ by at most 1
        user_count = sum(1 for m in conv.messages if m.speaker == "user")
        assistant_count = len(conv.messages) - user_count
        assert abs(user_count - assistant_count) <= 1, (
            f"Conversation {conv.conversation_id} has unbalanced turns: "
            f"{user_count} user vs {assistant_count} assistant."
        ) 
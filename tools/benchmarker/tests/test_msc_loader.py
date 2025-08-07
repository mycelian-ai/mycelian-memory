import json
from pathlib import Path

from benchmarks.python.msc_loader import load_msc_dataset


def test_load_msc_dataset(tmp_path):
    sample = {
        "initial_data_id": "id-1",
        "previous_dialogs": [
            {"dialog": [{"text": "hello"}, {"text": "hi"}]}  # session 1
        ],
        "self_instruct": {"A": "columbus", "B": "capital of ohio"},
    }
    f = tmp_path / "msc.jsonl"
    with f.open("w", encoding="utf-8") as out:
        out.write(json.dumps(sample) + "\n")

    convs = load_msc_dataset(local_path=str(f))
    assert len(convs) == 1
    conv = convs[0]
    # Expect 2 messages
    assert len(conv.messages) == 2
    # First message should be user role
    assert conv.messages[0].speaker == "user"
    # One test question
    assert len(conv.test_questions) == 1
    tq = conv.test_questions[0]
    assert tq.expected_answer == "columbus" 
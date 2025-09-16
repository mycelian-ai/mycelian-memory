import os
import tempfile
import textwrap
import json
import pytest

# Package path
import sys
sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(__file__)), "src"))

from dataset_loader import load_longmemeval


def test_load_raises_on_invalid_path():
    with pytest.raises(ValueError):
        _ = list(load_longmemeval("/path/does/not/exist"))


def test_load_empty_on_valid_dir():
    with tempfile.TemporaryDirectory() as tmpdir:
        items = list(load_longmemeval(tmpdir))
        assert items == []


def test_load_jsonl_normalization():
    record = {
        "id": "Q-1",
        "answer": "42",
        "sessions": [
            {
                "messages": [
                    {"role": "user", "content": "hello"},
                    {"role": "assistant", "content": "hi"},
                    {"role": "user", "content": "question?"},
                ]
            }
        ],
    }
    with tempfile.TemporaryDirectory() as tmpdir:
        p = os.path.join(tmpdir, "qa.jsonl")
        with open(p, "w", encoding="utf-8") as f:
            f.write(json.dumps(record) + "\n")
        items = list(load_longmemeval(tmpdir))
        assert len(items) == 1
        q = items[0]
        assert q["question_id"] == "Q-1"
        assert q["expected"] == "42"
        assert len(q["sessions"]) == 1
        msgs = q["sessions"][0]["messages"]
        assert [m["role"] for m in msgs] == ["user", "assistant", "user"]
        assert [m["content"] for m in msgs][-1] == "question?"

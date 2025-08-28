from typing import Iterable, List, Dict, Any, Iterator
import os

DatasetMessage = Dict[str, Any]
DatasetSession = Dict[str, Any]
DatasetQuestion = Dict[str, Any]


def _read_jsonl(path: str) -> Iterator[Dict[str, Any]]:
    import json
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            yield json.loads(line)


def _normalize_record(rec: Dict[str, Any]) -> DatasetQuestion:
    # Expected schema (canonical):
    # {
    #   "question_id": str,
    #   "expected": str,
    #   "sessions": [
    #       {"session_id": str, "messages": [{"role": "user"|"assistant", "content": str}, ...]}
    #   ]
    # }
    qid = rec.get("question_id") or rec.get("id") or rec.get("qid")
    expected = rec.get("expected") or rec.get("answer") or ""
    sessions = rec.get("sessions") or []
    # Minimal validation
    if not isinstance(sessions, list):
        sessions = []
    norm_sessions: List[DatasetSession] = []
    for idx, s in enumerate(sessions):
        sid = s.get("session_id") or f"S{idx+1}"
        msgs = s.get("messages") or []
        if not isinstance(msgs, list):
            msgs = []
        norm_msgs: List[DatasetMessage] = []
        for m in msgs:
            role = m.get("role")
            content = m.get("content")
            if not isinstance(role, str) or not isinstance(content, str):
                continue
            norm_msgs.append({"role": role, "content": content})
        norm_sessions.append({"session_id": sid, "messages": norm_msgs})
    return {
        "question_id": str(qid) if qid is not None else "",
        "expected": expected,
        "sessions": norm_sessions,
    }


def load_longmemeval(dataset_repo_path: str) -> Iterable[DatasetQuestion]:
    """Yield questions from a local LongMemEval repo.

    The loader searches for a JSONL file in this order:
      1) <root>/qa.jsonl
      2) <root>/longmemeval.jsonl
      3) <root>/data/qa.jsonl
      4) <root>/data/longmemeval.jsonl

    Each JSONL record should match the canonical schema documented above.
    Non-conforming records are normalized best-effort and invalid messages are skipped.
    """
    if not dataset_repo_path or not os.path.isdir(dataset_repo_path):
        raise ValueError(f"Invalid dataset_repo_path: {dataset_repo_path}")

    candidates = [
        os.path.join(dataset_repo_path, "qa.jsonl"),
        os.path.join(dataset_repo_path, "longmemeval.jsonl"),
        os.path.join(dataset_repo_path, "data", "qa.jsonl"),
        os.path.join(dataset_repo_path, "data", "longmemeval.jsonl"),
    ]
    chosen = next((p for p in candidates if os.path.isfile(p)), None)
    if not chosen:
        # No file found – return empty iterator for now
        return []

    def _iter() -> Iterator[DatasetQuestion]:
        for rec in _read_jsonl(chosen):
            yield _normalize_record(rec)

    return _iter()

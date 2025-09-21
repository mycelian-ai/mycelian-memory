from typing import Iterable, List, Dict, Any, Iterator, Optional
import os
from datetime import datetime

DatasetMessage = Dict[str, Any]
DatasetSession = Dict[str, Any]
DatasetQuestion = Dict[str, Any]


def _parse_haystack_date(date_str: str) -> Optional[str]:
    """Parse haystack date format to ISO-8601.

    Input format: "2023/05/20 (Sat) 02:21"
    Output format: "2023-05-20T02:21:00Z"
    """
    if not date_str:
        return None

    try:
        # Direct parsing using standard strptime format
        # %a matches abbreviated weekday name (Mon, Tue, etc.)
        dt = datetime.strptime(date_str, "%Y/%m/%d (%a) %H:%M")
        # Return ISO-8601 format with Z suffix for UTC
        return dt.isoformat() + "Z"
    except (ValueError, TypeError):
        # If parsing fails, return None
        return None


def _read_jsonl(path: str) -> Iterator[Dict[str, Any]]:
    import json
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            yield json.loads(line)


def _read_json_array(path: str) -> Iterator[Dict[str, Any]]:
    import json
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
        if isinstance(data, list):
            for rec in data:
                if isinstance(rec, dict):
                    yield rec
        elif isinstance(data, dict):
            # Single-record object fallback
            yield data


def _normalize_record(rec: Dict[str, Any]) -> DatasetQuestion:
    # Expected schema (canonical):
    # {
    #   "question_id": str,
    #   "expected": str,
    #   "sessions": [
    #       {"session_id": str, "messages": [{"role": "user"|"assistant", "content": str}, ...], "conversation_time": str}
    #   ]
    # }
    qid = rec.get("question_id") or rec.get("id") or rec.get("qid")
    expected = rec.get("expected") or rec.get("answer") or ""
    question_text = rec.get("question") or rec.get("query") or ""

    # Extract haystack_dates if present
    haystack_dates = rec.get("haystack_dates", [])
    if not isinstance(haystack_dates, list):
        haystack_dates = []

    # Prefer canonical sessions; otherwise derive from haystack_sessions
    sessions_raw = rec.get("sessions")
    if not sessions_raw:
        sessions_raw = rec.get("haystack_sessions")
    sessions = sessions_raw or []
    # Minimal validation
    if not isinstance(sessions, list):
        sessions = []
    norm_sessions: List[DatasetSession] = []
    for idx, s in enumerate(sessions):
        # Support either dict sessions with messages, or list-of-message sessions (haystack)
        sid = None
        msgs: List[Dict[str, Any]] = []
        if isinstance(s, dict):
            sid = s.get("session_id")
            msgs = s.get("messages") or []
        elif isinstance(s, list):
            msgs = s  # haystack_sessions element is a list of {role, content}
        sid = sid or f"S{idx+1}"
        if not isinstance(msgs, list):
            msgs = []
        norm_msgs: List[DatasetMessage] = []
        for m in msgs:
            role = m.get("role")
            content = m.get("content")
            if not isinstance(role, str) or not isinstance(content, str):
                continue
            norm_msgs.append({"role": role, "content": content})

        # Extract and parse the timestamp for this session if available
        conversation_time = None
        if idx < len(haystack_dates):
            conversation_time = _parse_haystack_date(haystack_dates[idx])

        session_dict = {"session_id": sid, "messages": norm_msgs}
        if conversation_time:
            session_dict["conversation_time"] = conversation_time
        norm_sessions.append(session_dict)

    return {
        "question_id": str(qid) if qid is not None else "",
        "expected": expected,
        "question": question_text,
        "sessions": norm_sessions,
    }


def load_longmemeval_file(dataset_file_path: str) -> Iterable[DatasetQuestion]:
    """Yield questions from a specific LongMemEval dataset file.

    Args:
        dataset_file_path: Direct path to the dataset file (JSON or JSONL)
    """
    if not dataset_file_path or not os.path.isfile(dataset_file_path):
        raise ValueError(f"Dataset file not found: {dataset_file_path}")

    def _iter() -> Iterator[DatasetQuestion]:
        if dataset_file_path.endswith(".jsonl"):
            for rec in _read_jsonl(dataset_file_path):
                yield _normalize_record(rec)
        else:
            for rec in _read_json_array(dataset_file_path):
                yield _normalize_record(rec)

    return _iter()


def load_longmemeval(dataset_repo_path: str) -> Iterable[DatasetQuestion]:
    """Yield questions from a local LongMemEval repo.

    The loader searches for dataset files in this order (first match wins):
      JSONL (line-delimited):
        1) <root>/qa.jsonl
        2) <root>/longmemeval.jsonl
        3) <root>/data/qa.jsonl
        4) <root>/data/longmemeval.jsonl
      JSON (array):
        5) <root>/longmemeval_s.json
        6) <root>/longmemeval_m.json
        7) <root>/longmemeval_oracle.json
        8) <root>/data/longmemeval_s.json
        9) <root>/data/longmemeval_m.json
       10) <root>/data/longmemeval_oracle.json
       11) <root>/test_sample/longmemeval_test_5.json

    Each JSONL record should match the canonical schema documented above.
    Non-conforming records are normalized best-effort and invalid messages are skipped.
    """
    if not dataset_repo_path or not os.path.isdir(dataset_repo_path):
        raise ValueError(f"Invalid dataset_repo_path: {dataset_repo_path}")

    candidates = [
        # JSONL
        os.path.join(dataset_repo_path, "qa.jsonl"),
        os.path.join(dataset_repo_path, "longmemeval.jsonl"),
        os.path.join(dataset_repo_path, "data", "qa.jsonl"),
        os.path.join(dataset_repo_path, "data", "longmemeval.jsonl"),
        # JSON arrays (known LongMemEval variants)
        os.path.join(dataset_repo_path, "longmemeval_s.json"),
        os.path.join(dataset_repo_path, "longmemeval_m.json"),
        os.path.join(dataset_repo_path, "longmemeval_oracle.json"),
        os.path.join(dataset_repo_path, "data", "longmemeval_s.json"),
        os.path.join(dataset_repo_path, "data", "longmemeval_m.json"),
        os.path.join(dataset_repo_path, "data", "longmemeval_oracle.json"),
        os.path.join(dataset_repo_path, "test_sample", "longmemeval_test_5.json"),
    ]
    chosen = next((p for p in candidates if os.path.isfile(p)), None)
    if not chosen:
        # No file found – return empty iterator for now
        return []

    def _iter() -> Iterator[DatasetQuestion]:
        if chosen.endswith(".jsonl"):
            for rec in _read_jsonl(chosen):
                yield _normalize_record(rec)
        else:
            for rec in _read_json_array(chosen):
                yield _normalize_record(rec)

    return _iter()

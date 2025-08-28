from typing import Iterable, List, Dict, Any
import os

class DatasetMessage(Dict[str, Any]):
    pass

class DatasetSession(Dict[str, Any]):
    pass

class DatasetQuestion(Dict[str, Any]):
    pass


def load_longmemeval(dataset_repo_path: str) -> Iterable[DatasetQuestion]:
    """Yield questions from a local LongMemEval repo.

    For now, this is a stub that validates the path and yields an empty iterator.
    Replace with real parsing that returns items shaped as:
    {
      "question_id": str,
      "expected": str,
      "sessions": [
         {"session_id": str, "messages": [{"role": "user"|"assistant", "content": str}, ...]}
      ]
    }
    """
    if not dataset_repo_path or not os.path.isdir(dataset_repo_path):
        raise ValueError(f"Invalid dataset_repo_path: {dataset_repo_path}")
    return []

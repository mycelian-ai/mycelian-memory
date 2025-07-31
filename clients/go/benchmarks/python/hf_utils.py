from __future__ import annotations

import json
from pathlib import Path
from typing import List, Dict

try:
    from datasets import load_dataset  # type: ignore
except ImportError as exc:  # pragma: no cover
    raise ImportError("datasets package required for HuggingFace downloads. Run `pip install datasets`. ") from exc


_DATA_DIR = Path(__file__).resolve().parent / "datasets"
_DATA_DIR.mkdir(exist_ok=True)


def ensure_dataset(name: str, subset: str | None = None, split: str = "train") -> Path:
    """Download dataset from HF if not cached; return local JSON path.

    Parameters
    ----------
    name:  e.g. "synapse-ai-lab/dmr"
    subset: optional subset name (HF field)
    split:  dataset split, default "train"
    """
    ds_id = name + (f"/{subset}" if subset else "")
    local_file = _DATA_DIR / f"{name.replace('/', '_')}_{split}.json"
    if local_file.exists():
        return local_file

    ds = load_dataset(name, subset, split=split)
    with local_file.open("w", encoding="utf-8") as f:
        for record in ds:
            f.write(json.dumps(record, ensure_ascii=False) + "\n")
    return local_file 
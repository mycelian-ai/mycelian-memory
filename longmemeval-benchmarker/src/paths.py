#!/usr/bin/env python3
"""Shared path helpers for the LongMemEval benchmarker.

Provides a single benchmarker root and a resolver to normalize relative
paths under that root. This avoids dependence on process CWD.
"""

from __future__ import annotations

from pathlib import Path
from typing import Union


# This file lives at <bench-root>/src/paths.py, so parents[1] = <bench-root>
BENCH_ROOT: Path = Path(__file__).resolve().parents[1]


def resolve_under_root(path_like: Union[str, Path]) -> Path:
    """Return an absolute Path under BENCH_ROOT for relative inputs.

    If the input is already absolute, it is returned as-is (resolved).
    """
    p = Path(path_like)
    return p if p.is_absolute() else (BENCH_ROOT / p).resolve()

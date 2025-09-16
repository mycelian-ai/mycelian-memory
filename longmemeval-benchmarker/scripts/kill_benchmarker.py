#!/usr/bin/env python3
"""Kill all LongMemEval benchmarker processes safely.

This script terminates:
  - python -m src.orchestrator.tasks
  - python -m src.orchestrator

It sends SIGTERM to the entire process group first, waits briefly, then SIGKILL
if necessary. Optionally clears orchestrator state files.
"""

from __future__ import annotations

import argparse
import os
import signal
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, List, Set, Tuple


BENCH_ROOT = Path(__file__).resolve().parents[1]


PATTERNS: Tuple[str, ...] = (
    "python -m src.orchestrator.tasks",
    "python -m src.orchestrator",
)


@dataclass(frozen=True)
class Proc:
    pid: int
    pgid: int
    cmd: str


def _ps() -> List[Proc]:
    out = subprocess.check_output(["ps", "-eo", "pid,pgid,command"], text=True)
    procs: List[Proc] = []
    for line in out.splitlines()[1:]:  # skip header
        line = line.strip()
        if not line:
            continue
        try:
            # Split first two numeric columns, rest is command
            parts = line.split(None, 2)
            if len(parts) < 3:
                continue
            pid = int(parts[0])
            pgid = int(parts[1])
            cmd = parts[2]
            procs.append(Proc(pid=pid, pgid=pgid, cmd=cmd))
        except Exception:
            continue
    return procs


def _is_target(cmd: str) -> bool:
    return any(pat in cmd for pat in PATTERNS)


def _find_targets() -> Tuple[List[Proc], Set[int]]:
    procs = _ps()
    targets = [p for p in procs if _is_target(p.cmd)]
    pgids = {p.pgid for p in targets}
    return targets, pgids


def _pgid_alive(pgid: int) -> bool:
    try:
        # If any process still has this PGID, it's alive
        return any(p.pgid == pgid for p in _ps())
    except Exception:
        return False


def _kill_group(pgid: int, timeout: float = 5.0) -> str:
    try:
        os.killpg(pgid, signal.SIGTERM)
    except ProcessLookupError:
        return f"pgid {pgid}: already gone"
    except Exception as e:
        return f"pgid {pgid}: SIGTERM failed: {e}"

    # Wait for graceful exit
    deadline = time.time() + timeout
    while time.time() < deadline:
        if not _pgid_alive(pgid):
            return f"pgid {pgid}: terminated"
        time.sleep(0.2)

    # Force kill
    try:
        os.killpg(pgid, signal.SIGKILL)
    except ProcessLookupError:
        return f"pgid {pgid}: terminated after SIGTERM"
    except Exception as e:
        return f"pgid {pgid}: SIGKILL failed: {e}"

    # Give a moment for SIGKILL to take effect
    time.sleep(0.2)
    if _pgid_alive(pgid):
        return f"pgid {pgid}: WARNING still alive after SIGKILL"
    return f"pgid {pgid}: killed"


def _state_paths() -> List[Path]:
    data_dir = BENCH_ROOT / "data"
    return [
        data_dir / "orchestrator.db",
        data_dir / "orchestrator.db-shm",
        data_dir / "orchestrator.db-wal",
        data_dir / "progress.db",
        data_dir / "progress.db-shm",
        data_dir / "progress.db-wal",
    ]


def _clear_state() -> None:
    for p in _state_paths():
        try:
            if p.exists():
                p.unlink()
                print(f"removed {p}")
        except Exception as e:
            print(f"could not remove {p}: {e}")


def main() -> int:
    ap = argparse.ArgumentParser(description="Kill all benchmarker processes (orchestrator & workers)")
    ap.add_argument("--dry-run", action="store_true", help="List targets without killing")
    ap.add_argument("--force", action="store_true", help="Do not prompt for confirmation")
    ap.add_argument("--clear-state", action="store_true", help="Also delete huey/progress DB files")
    args = ap.parse_args()

    targets, pgids = _find_targets()
    if not targets:
        print("No benchmarker processes found.")
    else:
        print("Found processes:")
        for p in targets:
            print(f"  pid={p.pid} pgid={p.pgid} cmd={p.cmd}")

    if args.dry_run:
        if args.clear_state:
            print("\nClear-state would remove:")
            for p in _state_paths():
                print(f"  {p}")
        return 0

    if targets and not args.force:
        try:
            ans = input("Proceed to terminate these process groups? [y/N] ").strip().lower()
        except KeyboardInterrupt:
            print()
            return 1
        if ans != "y":
            print("Aborted.")
            return 1

    # Kill groups
    for pgid in sorted(pgids):
        msg = _kill_group(pgid)
        print(msg)

    if args.clear_state:
        print("\nClearing orchestrator state files…")
        _clear_state()

    return 0


if __name__ == "__main__":
    sys.exit(main())

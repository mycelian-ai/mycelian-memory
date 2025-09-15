"""Utilities for running coroutines on a persistent background asyncio loop.

This avoids using asyncio.run in long-lived worker processes, which can
shut down the default ThreadPoolExecutor and cause RuntimeError:
"cannot schedule new futures after shutdown" on subsequent calls.
"""

from __future__ import annotations

import asyncio
import threading
from typing import Any, Coroutine

_loop: asyncio.AbstractEventLoop | None = None
_thread: threading.Thread | None = None
_ready = threading.Event()


def _loop_runner(loop: asyncio.AbstractEventLoop) -> None:
    asyncio.set_event_loop(loop)
    _ready.set()
    loop.run_forever()


def get_loop() -> asyncio.AbstractEventLoop:
    global _loop, _thread
    if _loop and _loop.is_running():
        return _loop
    loop = asyncio.new_event_loop()
    t = threading.Thread(target=_loop_runner, args=(loop,), name="bench-asyncio-loop", daemon=True)
    t.start()
    _ready.wait()
    _loop = loop
    _thread = t
    return loop


def run(coro: Coroutine[Any, Any, Any]) -> Any:
    """Run a coroutine on the persistent loop and return its result synchronously."""
    loop = get_loop()
    fut = asyncio.run_coroutine_threadsafe(coro, loop)
    return fut.result()

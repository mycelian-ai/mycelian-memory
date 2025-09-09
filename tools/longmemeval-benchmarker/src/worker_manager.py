from __future__ import annotations

from typing import Any, Callable, Dict, List, Tuple


class WorkerManager:
    """Coordinate sequential/parallel execution of per-question work.

    Responsibilities:
    - Accept a list of (index, question) items and a `work_fn` that executes one question.
    - Manage per-question log file creation and pass the open stream to `work_fn`.
    - Run either sequentially or with a ThreadPool depending on `workers`.
    - Collect and return results in input order.

    Non-responsibilities:
    - No dataset parsing, vault resolution, or agent/memory details.
    - No QA prompt building; that is owned by the SingleQuestionRunner.
    """
    def __init__(self, workers: int, debug: bool = False):
        self.workers = max(1, int(workers))
        self.debug = bool(debug)

    def run(
        self,
        items: List[Tuple[int, Dict[str, Any]]],
        work_fn: Callable[[int, Dict[str, Any], Any], Dict[str, Any]],
        make_log_path: Callable[[int], str],
    ) -> List[Dict[str, Any]]:
        results: List[Dict[str, Any]] = []
        if self.workers == 1:
            for idx, q in items:
                log_path = make_log_path(idx)
                with open(log_path, "w", encoding="utf-8") as log:
                    results.append(work_fn(idx, q, log))
            return results

        from concurrent.futures import ThreadPoolExecutor

        def _do_one(idx: int, q: Dict[str, Any]) -> Dict[str, Any]:
            log_path = make_log_path(idx)
            with open(log_path, "w", encoding="utf-8") as log:
                return work_fn(idx, q, log)

        with ThreadPoolExecutor(max_workers=self.workers) as ex:
            futs = [ex.submit(_do_one, idx, q) for idx, q in items]
            for f in futs:
                results.append(f.result())
        return results

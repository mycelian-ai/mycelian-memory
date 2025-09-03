"""Benchmarker entrypoint.

Responsibilities:
- Parse config and CLI overrides.
- Load dataset and apply question/session/turn limits.
- Ensure `vault_id` via `MemoryManager` using a temporary agent.
- Prepare output/logs directories.
- Delegate per-question execution to `WorkerManager` + `SingleQuestionRunner`.
- Write `hypotheses.jsonl` at the end.

Non-responsibilities:
- No per-question message handling details (owned by SingleQuestionRunner).
- No concurrency implementation details (owned by WorkerManager).
"""

import argparse
import os
import json
import tomllib
import time
import logging
from typing import Any, Dict, List

from src.dataset_loader import load_longmemeval_file
from src.mycelian_memory_agent import create_mcp_client
from src.mycelian_memory_agent.build import build_agent_with_invoker
from src.memory_manager import MemoryManager
from src.single_question_runner import SingleQuestionRunner

# Minimal helpers (replacing the deleted runner module)
class _SimpleConfig:
    def __init__(self, raw_cfg: Dict[str, Any]):
        # Basic
        self.dataset_file_path = raw_cfg["dataset_file_path"]
        self.vault_title = raw_cfg["vault_title"]
        self.vault_id = raw_cfg.get("vault_id")
        self.memory_title_template = raw_cfg.get("memory_title_template", "{question_id}__{run_id}")
        self.run_id = str(int(time.time()))

        # Provider (keep as mapping)
        self.provider = raw_cfg["provider"]

        # Models (object with attributes agent, qa)
        self.models = type("Models", (), raw_cfg["models"])()

        # Params (object with required fields; defaults applied)
        params_raw = raw_cfg.get("params", {})
        self.params = type(
            "Params",
            (),
            {
                "top_k": params_raw.get("top_k", 10),
                "max_tool_calls_per_turn": params_raw.get("max_tool_calls_per_turn", 5),
                "question_limit": params_raw.get("question_limit"),
                "workers": params_raw.get("workers", 1),
                "max_sessions_per_question": params_raw.get("max_sessions_per_question"),
                "max_turns_per_session": params_raw.get("max_turns_per_session"),
                "dump_state": params_raw.get("dump_state", False)
            },
        )()


def parse_config(raw_cfg: Dict[str, Any]) -> _SimpleConfig:
    return _SimpleConfig(raw_cfg)


def _compute_out_dir(run_id: str) -> str:
    return f"out/run_{run_id}"


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)


def _apply_limits(q: Dict[str, Any], max_sessions: int, max_turns: int) -> Dict[str, Any]:
    out = dict(q)
    sessions = list(out.get("sessions", []))
    if max_sessions and max_sessions > 0:
        sessions = sessions[: max_sessions]
    trimmed_sessions = []
    for s in sessions:
        msgs = list(s.get("messages", []))
        if max_turns and max_turns > 0:
            msgs = msgs[: max_turns]
        trimmed_sessions.append({"messages": msgs})
    out["sessions"] = trimmed_sessions
    return out


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval benchmarker")
    parser.add_argument("config", help="Path to TOML config")
    parser.add_argument("--num-questions", type=int, default=None, help="Number of questions to process (overrides params.question_limit)")
    parser.add_argument("--workers", type=int, default=None, help="Number of parallel workers (overrides params.workers)")
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        raw_cfg: dict[str, Any] = tomllib.load(f)

    cfg = parse_config(raw_cfg)
    if args.num_questions is not None:
        cfg.params.question_limit = max(0, int(args.num_questions))
    if args.workers is not None:
        cfg.params.workers = max(1, int(args.workers))
    # Logging is always enabled (debug flag removed)

    # Configure root logging for terminal: keep quiet by default; per-question logs handled separately
    logging.basicConfig(
        level=logging.WARNING,
        format="%(asctime)s [%(filename)s:%(funcName)s] %(message)s",
    )

    # Load dataset
    ds: List[Dict[str, Any]] = list(load_longmemeval_file(cfg.dataset_file_path))
    if cfg.params.question_limit and cfg.params.question_limit > 0:
        ds = ds[: cfg.params.question_limit]
    if not ds:
        print("[benchmarker] no questions found – ensure dataset files are present")
        return

    # Create shared MCP client for administrative operations
    mcp_client = create_mcp_client()
    
    # Resolve vault once using MemoryManager
    memory_mgr = MemoryManager(mcp_client, debug=False)
    vault_id = memory_mgr.ensure_vault(cfg.vault_title, cfg.vault_id)

    # Directories
    out_dir = _compute_out_dir(cfg.run_id)
    os.makedirs(out_dir, exist_ok=True)
    tmp_dir = os.path.join(out_dir, "tmp")
    os.makedirs(tmp_dir, exist_ok=True)
    logs_dir = os.path.join(out_dir, "logs")
    os.makedirs(logs_dir, exist_ok=True)
    hyp_path = os.path.join(out_dir, "hypotheses.jsonl")

    print(f"[benchmarker] starting run with {len(ds)} question(s), workers={cfg.params.workers}")

    from src.worker_manager import WorkerManager
    wm = WorkerManager(workers=cfg.params.workers, debug=False)
    sqr = SingleQuestionRunner(cfg, mcp_client=mcp_client)

    def make_log_path(i: int) -> str:
        return os.path.join(logs_dir, f"question_{i:05d}.log")

    def work_fn(idx: int, q: Dict[str, Any], log) -> Dict[str, Any]:
        # Route runner/agent/tool logs to this question's log file only
        qhandler = logging.StreamHandler(log)
        qhandler.setFormatter(logging.Formatter("%(asctime)s [%(filename)s:%(funcName)s] %(message)s"))
        target_loggers = [
            logging.getLogger("lme.runner"),
            logging.getLogger("lme.agent"),
        ]
        prev = []
        for lg in target_loggers:
            prev.append((lg, list(lg.handlers), lg.propagate, lg.level))
            lg.setLevel(logging.INFO)  # Always log at INFO level
            lg.addHandler(qhandler)
            # Prevent console/root propagation; keep logs isolated per question file
            lg.propagate = False  # keep specifics in question log, not global
        try:
            q_limited = _apply_limits(q, cfg.params.max_sessions_per_question, cfg.params.max_turns_per_session)
            return sqr.run_question(q_limited, vault_id=vault_id, run_id=cfg.run_id, log=log)
        finally:
            # Restore logger states and remove handler
            for lg, handlers_snapshot, prop, lvl in prev:
                try:
                    lg.removeHandler(qhandler)
                except Exception:
                    pass
                lg.propagate = prop
                lg.setLevel(lvl)
            try:
                qhandler.flush()
            except Exception:
                pass

    items = list(enumerate(ds, start=1))
    results = wm.run(items, work_fn, make_log_path)

    # Write parts and merge
    wrote = 0
    with open(hyp_path, "w", encoding="utf-8") as out_f:
        for idx, res in enumerate(results, start=1):
            if not isinstance(res, dict):
                continue
            out_f.write(json.dumps(res) + "\n")
            wrote += 1
    print(f"[benchmarker] wrote {wrote} lines to {hyp_path}")


if __name__ == "__main__":
    main()



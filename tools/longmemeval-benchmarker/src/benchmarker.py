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

from dataset_loader import load_longmemeval_file
from mycelian_memory_agent import create_mcp_client
from mycelian_memory_agent.build import build_agent_with_invoker
from memory_manager import MemoryManager
from single_question_runner import SingleQuestionRunner

# Minimal helpers (replacing the deleted runner module)

def run_model_healthcheck(model_id: str, model_type: str = "agent") -> None:
    """Run a simple healthcheck on the model to ensure it's accessible.
    
    Args:
        model_id: The model identifier
        model_type: Type of model ("agent" or "qa")
        
    Raises:
        Exception: If the model is not accessible
    """
    import time
    from model_providers import get_chat_model
    
    start_time = time.time()
    print(f"[benchmarker] HEALTHCHECK START: {model_type} model: {model_id} at {time.strftime('%Y-%m-%d %H:%M:%S')}")
    
    try:
        # Use provider-agnostic model with LangChain's built-in retry
        print(f"[benchmarker] Creating LLM instance for {model_id}...")
        llm = get_chat_model(model_id)  # max_retries=6 is default
        print(f"[benchmarker] LLM instance created successfully")
        
        print(f"[benchmarker] Invoking healthcheck prompt for {model_id}...")
        result = llm.invoke("Hi, please respond with 'OK' if you're working.")
        print(f"[benchmarker] Received response from model: {str(result)[:100]}...")
        
        elapsed = time.time() - start_time
        print(f"[benchmarker] HEALTHCHECK SUCCESS: {model_type} model: {model_id} completed in {elapsed:.2f}s")
        print(f"[benchmarker] {model_type} model healthcheck passed")
    except Exception as e:
        elapsed = time.time() - start_time
        print(f"[benchmarker] HEALTHCHECK FAILED: {model_type} model: {model_id} after {elapsed:.2f}s")
        print(f"[benchmarker] ERROR: {model_type} model healthcheck failed: {e}")
        print(f"[benchmarker] Exception type: {type(e).__name__}")
        import traceback
        print(f"[benchmarker] Stack trace:\n{traceback.format_exc()}")
        raise

class _SimpleConfig:
    def __init__(self, raw_cfg: Dict[str, Any]):
        # Required fields
        self.dataset_file_path = raw_cfg["dataset_file_path"]
        self.vault_title = raw_cfg["vault_title"]
        
        # Models configuration - both ingest and qa are mandatory
        models_cfg = raw_cfg.get("models", {})
        if not models_cfg:
            # If no models section, use defaults
            models_cfg = {}
        
        # Ingest and QA models with defaults
        ingest_model = models_cfg.get("ingest", "openai:gpt-5-nano-2025-08-07")
        qa_model = models_cfg.get("qa", "openai:gpt-5-2025-08-07")
        
        self.models = type("Models", (), {
            "ingest": ingest_model,
            "qa": qa_model
        })()
        
        # Search configuration - optional with defaults
        search_cfg = raw_cfg.get("search", {})
        self.use_two_pass_search = search_cfg.get("use_two_pass", True)  # Default to True
        
        # Auto-generated fields
        self.run_id = str(int(time.time()))
        
        # Fixed defaults (removed from config)
        self.provider = {"type": "model-provider"}  # Supports OpenAI and Vertex AI
        self.vault_id = None  # Always auto-generated from vault_title
        self.memory_title_template = "{question_id}__{run_id}"  # Standard format
        
        # Fixed internal params (not exposed in config)
        self.params = type(
            "Params",
            (),
            {
                # Fixed internal defaults
                "top_k": 10,  # Search results limit
                "max_tool_calls_per_turn": 5,  # Legacy, unused
                "dump_state": False,  # Debug feature, removed
                "use_two_pass_search": self.use_two_pass_search,  # From config or default True
                
                # Removed limits - control via dataset file instead
                "question_limit": None,  # Always process all questions
                "max_sessions_per_question": None,  # Always process all sessions
                "max_turns_per_session": None,  # Always process all turns
            },
        )()


def parse_config(raw_cfg: Dict[str, Any]) -> _SimpleConfig:
    return _SimpleConfig(raw_cfg)


def _compute_out_dir(run_id: str) -> str:
    # If run_id already starts with "run_", don't add it again
    if run_id.startswith("run_"):
        return f"out/{run_id}"
    return f"out/run_{run_id}"


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)


# Removed _apply_limits function - always process all sessions and turns


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval benchmarker")
    # All arguments are now options (no positional arguments)
    parser.add_argument("--config", required=True, help="Path to TOML config")
    parser.add_argument("--run-id", required=True, help="Run identifier for this benchmark execution")
    parser.add_argument("--workers", type=int, default=1, help="Number of parallel workers (default: 1)")
    parser.add_argument("--mode", choices=["ingestion", "qa", "full"], default="full", 
                       help="Execution mode: ingestion (sessions only), qa (QA only), full (both)")
    # Single-question flags for orchestrator integration
    parser.add_argument("--question-id", help="Process only this question id (optional)")
    parser.add_argument("--start-session", type=int, default=0,
                       help="Start from this session index (for resume)")
    parser.add_argument("--memory-id", help="Memory ID for QA-only mode (skips memory creation)")
    parser.add_argument("--vault-id", help="Vault ID for QA-only mode (skips vault resolution)")
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        raw_cfg: dict[str, Any] = tomllib.load(f)

    cfg = parse_config(raw_cfg)
    # Override run_id from CLI argument
    cfg.run_id = args.run_id
    workers = max(1, args.workers)  # Use CLI workers directly
    # Logging is always enabled (debug flag removed)

    # Configure root logging for terminal: keep quiet by default; per-question logs handled separately
    logging.basicConfig(
        level=logging.WARNING,
        format="%(asctime)s [%(filename)s:%(funcName)s] %(message)s",
    )
    
    # Run model healthchecks before proceeding
    run_model_healthcheck(cfg.models.ingest, "ingest")
    if cfg.models.qa != cfg.models.ingest:
        run_model_healthcheck(cfg.models.qa, "qa")

    # Load dataset - always process all questions in the file
    ds: List[Dict[str, Any]] = list(load_longmemeval_file(cfg.dataset_file_path))
    # Filter to single question if requested
    if args.question_id:
        ds = [q for q in ds if q.get("question_id") == args.question_id]
        if not ds:
            print(f"[benchmarker] question_id {args.question_id} not found in dataset")
            return
        # Apply start-session slicing
        if args.start_session > 0:
            q0 = ds[0].copy()
            start = args.start_session
            q0["haystack_sessions"] = q0.get("haystack_sessions", [])[start:]
            if "haystack_session_ids" in q0:
                q0["haystack_session_ids"] = q0["haystack_session_ids"][start:]
            if "haystack_dates" in q0:
                q0["haystack_dates"] = q0["haystack_dates"][start:]
            ds[0] = q0
    if not ds:
        print("[benchmarker] no questions found – ensure dataset files are present")
        return

    # Create shared MCP client for administrative operations
    mcp_client = create_mcp_client()
    
    # Resolve vault - use provided vault_id if available, otherwise create/get from vault_title
    if args.vault_id:
        vault_id = args.vault_id
        print(f"[benchmarker] using provided vault_id: {vault_id}")
    else:
        memory_mgr = MemoryManager(mcp_client, debug=False)
        vault_id = memory_mgr.ensure_vault(cfg.vault_title, cfg.vault_id)
        print(f"[benchmarker] resolved vault_id: {vault_id}")

    # Directories
    out_dir = _compute_out_dir(cfg.run_id)
    os.makedirs(out_dir, exist_ok=True)
    tmp_dir = os.path.join(out_dir, "tmp")
    os.makedirs(tmp_dir, exist_ok=True)
    logs_dir = os.path.join(out_dir, "logs")
    os.makedirs(logs_dir, exist_ok=True)
    hyp_path = os.path.join(out_dir, "hypotheses.jsonl")

    print(f"[benchmarker] starting run with {len(ds)} question(s), workers={workers}")

    from worker_manager import WorkerManager
    wm = WorkerManager(workers=workers, debug=False)
    sqr = SingleQuestionRunner(cfg, mcp_client=mcp_client, mode=args.mode)

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
            # Always process all sessions and turns in the question
            # Pass memory_id for QA-only mode if provided
            return sqr.run_question(q, vault_id=vault_id, run_id=cfg.run_id, log=log, 
                                  memory_id=args.memory_id if args.mode == "qa" else None)
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

    # Write hypotheses only for QA or FULL modes
    if args.mode != "ingestion":
        wrote = 0
        # For single-question QA runs, append to avoid clobbering previous results
        open_mode = "a" if (args.mode == "qa" and args.question_id) else "w"
        with open(hyp_path, open_mode, encoding="utf-8") as out_f:
            for idx, res in enumerate(results, start=1):
                if not isinstance(res, dict):
                    continue
                # Write only the deliverable fields expected for hypotheses.jsonl
                if "hypothesis" in res:
                    line = {
                        "question_id": res.get("question_id"),
                        "hypothesis": res.get("hypothesis", "")
                    }
                    out_f.write(json.dumps(line) + "\n")
                    wrote += 1
        print(f"[benchmarker] wrote {wrote} lines to {hyp_path}")

    # If single-question mode, echo JSON result for orchestrator parsing
    if args.question_id and results:
        # results list preserves order; first element is our question
        res0 = results[0]
        if isinstance(res0, dict):
            print(json.dumps(res0))


if __name__ == "__main__":
    main()



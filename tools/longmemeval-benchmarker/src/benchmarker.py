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

def run_model_healthcheck(model_id: str, model_type: str = "agent") -> None:
    """Run a simple healthcheck on the model to ensure it's accessible.
    
    Args:
        model_id: The model identifier
        model_type: Type of model ("agent" or "qa")
        
    Raises:
        Exception: If the model is not accessible
    """
    from src.model_providers import get_chat_model
    
    print(f"[benchmarker] running healthcheck for {model_type} model: {model_id}")
    
    try:
        # Use provider-agnostic model with LangChain's built-in retry
        llm = get_chat_model(model_id)  # max_retries=6 is default
        result = llm.invoke("Hi, please respond with 'OK' if you're working.")
        print(f"[benchmarker] {model_type} model healthcheck passed")
    except Exception as e:
        print(f"[benchmarker] ERROR: {model_type} model healthcheck failed: {e}")
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
    return f"out/run_{run_id}"


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)


# Removed _apply_limits function - always process all sessions and turns


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval benchmarker")
    parser.add_argument("config", help="Path to TOML config")
    parser.add_argument("--workers", type=int, default=1, help="Number of parallel workers (default: 1)")
    parser.add_argument("--mode", choices=["ingestion", "qa", "full"], default="full", 
                       help="Execution mode: ingestion (sessions only), qa (QA only), full (both)")
    parser.add_argument("--memory-id", help="Memory ID for QA-only mode (skips memory creation)")
    parser.add_argument("--vault-id", help="Vault ID for QA-only mode (skips vault resolution)")
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        raw_cfg: dict[str, Any] = tomllib.load(f)

    cfg = parse_config(raw_cfg)
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
    if not ds:
        print("[benchmarker] no questions found – ensure dataset files are present")
        return

    # Create shared MCP client for administrative operations
    mcp_client = create_mcp_client()
    
    # Resolve vault once using MemoryManager (or use provided vault_id for QA-only mode)
    if args.mode == "qa" and args.vault_id:
        vault_id = args.vault_id
        print(f"[benchmarker] using provided vault_id for QA-only mode: {vault_id}")
    else:
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

    print(f"[benchmarker] starting run with {len(ds)} question(s), workers={workers}")

    from src.worker_manager import WorkerManager
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



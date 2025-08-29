import argparse
import os
import json
from datetime import datetime, timezone
import tomllib
from dataclasses import dataclass
from typing import Any, Dict

from .dataset_loader import load_longmemeval
from .mycelian_memory_agent import build_agent


@dataclass
class ProviderConfig:
    type: str  # "openai" | "bedrock"
    openai_base_url: str | None = None
    bedrock_region: str | None = None


@dataclass
class ModelsConfig:
    agent: str
    qa: str
    eval: str | None = None


@dataclass
class ParamsConfig:
    top_k: int = 10
    use_llm_judge: bool = True
    max_tool_calls_per_turn: int = 5
    debug: bool = False


@dataclass
class AppConfig:
    dataset_repo_path: str
    vault_title: str | None
    vault_id: str | None
    provider: ProviderConfig
    models: ModelsConfig
    params: ParamsConfig
    memory_title_template: str
    run_id: str


def _require(cond: bool, msg: str) -> None:
    if not cond:
        raise ValueError(msg)


def parse_config(cfg: Dict[str, Any]) -> AppConfig:
    # Required root fields
    dataset_repo_path = cfg.get("dataset_repo_path")
    _require(isinstance(dataset_repo_path, str) and dataset_repo_path, "dataset_repo_path is required")
    _require(os.path.isdir(dataset_repo_path), f"dataset_repo_path not found: {dataset_repo_path}")

    vault_title = cfg.get("vault_title")
    vault_id = cfg.get("vault_id")
    _require(bool(vault_title) or bool(vault_id), "provide either vault_title or vault_id")

    # Provider
    p = cfg.get("provider") or {}
    ptype = (p.get("type") or "").lower()
    _require(ptype in ("openai", "bedrock"), "provider.type must be 'openai' or 'bedrock'")
    provider = ProviderConfig(
        type=ptype,
        openai_base_url=(p.get("openai") or {}).get("base_url") or (p.get("openai_base_url")),
        bedrock_region=(p.get("bedrock") or {}).get("region") or (p.get("region")),
    )

    # Models
    m = cfg.get("models") or {}
    agent_model = m.get("agent")
    qa_model = m.get("qa")
    eval_model = m.get("eval")
    _require(isinstance(agent_model, str) and agent_model, "models.agent is required")
    _require(isinstance(qa_model, str) and qa_model, "models.qa is required")
    models = ModelsConfig(agent=agent_model, qa=qa_model, eval=eval_model if eval_model else None)

    # Params
    pr = cfg.get("params") or {}
    params = ParamsConfig(
        top_k=int(pr.get("top_k", 10)),
        use_llm_judge=bool(pr.get("use_llm_judge", True)),
        max_tool_calls_per_turn=int(pr.get("max_tool_calls_per_turn", 5)),
        debug=bool(pr.get("debug", False)),
    )

    # Memory title template
    memory_title_template = cfg.get("memory_title_template") or "{question_id}__{run_id}"

    # Run id
    run_id = cfg.get("run_id")
    if not run_id:
        import time
        run_id = str(int(time.time()))

    return AppConfig(
        dataset_repo_path=dataset_repo_path,
        vault_title=vault_title,
        vault_id=vault_id,
        provider=provider,
        models=models,
        params=params,
        memory_title_template=memory_title_template,
        run_id=run_id,
    )


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval benchmarker")
    parser.add_argument("config", help="Path to TOML config")
    parser.add_argument("--mode", choices=["ingestion", "qa", "eval"], default=None)
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        raw_cfg: dict[str, Any] = tomllib.load(f)

    cfg = parse_config(raw_cfg)
    # LangSmith (optional): enable tracing when API key is present
    if os.environ.get("LANGCHAIN_API_KEY"):
        os.environ.setdefault("LANGCHAIN_TRACING_V2", "true")
        os.environ.setdefault("LANGCHAIN_ENDPOINT", "https://api.smith.langchain.com")
        os.environ.setdefault("LANGCHAIN_PROJECT", f"longmemeval_{os.getpid()}_{int(cfg.run_id)}")
        print(f"[trace] LangSmith enabled project={os.environ.get('LANGCHAIN_PROJECT')}")
    mode = args.mode or "end_to_end"

    print(
        f"Loaded config: vault={'id:'+cfg.vault_id if cfg.vault_id else 'title:'+str(cfg.vault_title)} "
        f"provider={cfg.provider.type} agent={cfg.models.agent} qa={cfg.models.qa} eval={cfg.models.eval or 'EM'} "
        f"top_k={cfg.params.top_k} run_id={cfg.run_id} mode={mode}"
    )

    agent = build_agent(
        cfg.models.agent,
        max_tool_calls_per_turn=cfg.params.max_tool_calls_per_turn,
        provider_type=cfg.provider.type,
    )
    # Agent owns MCP tool usage. Runner only coordinates vault/memory binding.

    # Ingestion
    if mode in ("ingestion", "end_to_end"):
        count_q = 0
        total_turns = 0
        total_tool_calls = 0
        # Resolve vault once for the run via agent
        vault_id = agent.ensure_vault(cfg.vault_title, cfg.vault_id)
        for q in load_longmemeval(cfg.dataset_repo_path):
            count_q += 1
            qid = q.get("question_id", f"Q{count_q}")
            mem_title = build_memory_title(cfg.memory_title_template, qid, cfg.run_id)
            print(f"[ingest] question={qid} → memory_title={mem_title}")
            # Ensure memory, bind to agent, then stream sessions (agent decides tool usage per prompts)
            memory_id = agent.ensure_memory(vault_id, mem_title, memory_type="NOTES")
            agent.bind_memory(vault_id, memory_id)
            # Optional breadcrumb: write mapping for QA/Eval convenience
            out_dir = os.path.join(os.path.dirname(__file__), "..", "out", f"run_{cfg.run_id}")
            out_dir = os.path.normpath(out_dir)
            os.makedirs(out_dir, exist_ok=True)
            mapping_path = os.path.join(out_dir, "mapping.jsonl")
            mapping = {
                "question_id": qid,
                "vault_id": vault_id,
                "memory_id": memory_id,
                "memory_title": mem_title,
                "run_id": cfg.run_id,
                "ts": datetime.now(timezone.utc).isoformat(),
            }
            with open(mapping_path, "a", encoding="utf-8") as mf:
                mf.write(json.dumps(mapping) + "\n")
            for s in q.get("sessions", []):
                msgs = s.get("messages", [])
                if cfg.params.debug:
                    sid = s.get("session_id")
                    print(f"[coord][session] {sid} turns={len(msgs)}")
                    for idx, mm in enumerate(msgs, start=1):
                        rc = mm.get("role")
                        ct = mm.get("content", "")
                        preview = ct[:120].replace("\n", " ") if isinstance(ct, str) else str(ct)[:120]
                        print(f"[coord][turn] {idx} role={rc} len={len(ct) if isinstance(ct, str) else 'n/a'} preview={preview}")
                turns, tool_calls = agent.run_session(msgs)
                total_turns += turns
                total_tool_calls += tool_calls
                print(f"  [session] {s.get('session_id')} turns={turns} tool_calls={tool_calls}")
        if count_q == 0:
            print("[ingest] no questions found – ensure JSONL present in dataset_repo_path")
        else:
            print(f"[ingest] summary: questions={count_q} turns={total_turns} tool_calls={total_tool_calls}")

    if mode in ("qa", "end_to_end"):
        print("[qa] stub – will call search_memories and answer using QA model")

    if mode in ("eval", "end_to_end"):
        print("[eval] stub – will compute EM or LLM-judge and write JSONL")

    # Best-effort cleanup
    try:
        agent.close()
    except Exception:
        pass


if __name__ == "__main__":
    main()

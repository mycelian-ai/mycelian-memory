import argparse
import os
import tomllib
from dataclasses import dataclass
from typing import Any, Dict

from .dataset_loader import load_longmemeval


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
    mode = args.mode or "end_to_end"

    print(
        f"Loaded config: vault={'id:'+cfg.vault_id if cfg.vault_id else 'title:'+str(cfg.vault_title)} "
        f"provider={cfg.provider.type} agent={cfg.models.agent} qa={cfg.models.qa} eval={cfg.models.eval or 'EM'} "
        f"top_k={cfg.params.top_k} run_id={cfg.run_id} mode={mode}"
    )

    # Scaffold ingestion loop (no LangGraph yet)
    if mode in ("ingestion", "end_to_end"):
        count_q = 0
        for q in load_longmemeval(cfg.dataset_repo_path):
            count_q += 1
            mem_title = build_memory_title(cfg.memory_title_template, q.get("question_id", f"Q{count_q}"), cfg.run_id)
            print(f"[ingest] question={q.get('question_id')} → memory_title={mem_title}")
            for s in q.get("sessions", []):
                print(f"  [session] {s.get('session_id')}: {len(s.get('messages', []))} messages (stub)")
        if count_q == 0:
            print("[ingest] no questions found – ensure JSONL present in dataset_repo_path")

    if mode in ("qa", "end_to_end"):
        print("[qa] stub – will call search_memories and answer using QA model")

    if mode in ("eval", "end_to_end"):
        print("[eval] stub – will compute EM or LLM-judge and write JSONL")


if __name__ == "__main__":
    main()

import argparse
import os
import json
import tomllib
from dataclasses import dataclass
from typing import Any, Dict, List

from dataset_loader import load_longmemeval_file
from mycelian_memory_agent import build_agent


@dataclass
class ProviderConfig:
    type: str  # "openai" | "bedrock"
    openai_base_url: str | None = None
    bedrock_region: str | None = None


@dataclass
class ModelsConfig:
    agent: str
    qa: str


@dataclass
class ParamsConfig:
    top_k: int = 10
    max_tool_calls_per_turn: int = 5
    debug: bool = False
    # Testing controls
    question_limit: int = 0  # 0 = no limit
    max_sessions_per_question: int = 0  # 0 = no limit
    max_turns_per_session: int = 0  # 0 = no limit


@dataclass
class AppConfig:
    dataset_file_path: str
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
    dataset_file_path = cfg.get("dataset_file_path")
    _require(isinstance(dataset_file_path, str) and dataset_file_path, "dataset_file_path is required")
    _require(os.path.isfile(dataset_file_path), f"dataset_file_path not found: {dataset_file_path}")

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
    _require(isinstance(agent_model, str) and agent_model, "models.agent is required")
    _require(isinstance(qa_model, str) and qa_model, "models.qa is required")
    models = ModelsConfig(agent=agent_model, qa=qa_model)

    # Params
    pr = cfg.get("params") or {}
    params = ParamsConfig(
        top_k=int(pr.get("top_k", 10)),
        max_tool_calls_per_turn=int(pr.get("max_tool_calls_per_turn", 5)),
        debug=bool(pr.get("debug", False)),
        question_limit=int(pr.get("question_limit", 0)),
        max_sessions_per_question=int(pr.get("max_sessions_per_question", 0)),
        max_turns_per_session=int(pr.get("max_turns_per_session", 0)),
    )

    # Memory title template
    memory_title_template = cfg.get("memory_title_template") or "{question_id}__{run_id}"

    # Run id
    run_id = cfg.get("run_id")
    if not run_id:
        import time
        run_id = str(int(time.time()))

    return AppConfig(
        dataset_file_path=dataset_file_path,
        vault_title=vault_title,
        vault_id=vault_id,
        provider=provider,
        models=models,
        params=params,
        memory_title_template=memory_title_template,
        run_id=run_id,
    )


def _compute_out_dir(run_id: str) -> str:
    out_dir = os.path.join(os.path.dirname(__file__), "..", "out", f"run_{run_id}")
    return os.path.normpath(out_dir)


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)


def _derive_question_from_sessions(rec: Dict) -> str:
    sessions = rec.get("sessions") or []
    first_user = ""
    for s in sessions:
        msgs = s.get("messages") or []
        for m in msgs:
            if isinstance(m, dict) and m.get("role") == "user":
                content = m.get("content") or ""
                if isinstance(content, str) and content:
                    if "?" in content:
                        return content
                    if not first_user:
                        first_user = content
    return first_user


def _build_qa_context(search_result: Dict, top_k: int) -> str:
    latest_ctx = (search_result.get("latestContext") or search_result.get("latest_context") or "").strip()
    best_ctx = (search_result.get("bestContext") or search_result.get("best_context") or "").strip()
    entries = search_result.get("entries") or []
    entries_text: list[str] = []
    for e in entries[: top_k]:
        if isinstance(e, dict):
            txt = e.get("summary") or ""
            if txt:
                entries_text.append(str(txt))
    parts = [s for s in [latest_ctx, best_ctx, "\n\n".join(entries_text)] if s]
    return "\n\n".join(parts)


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval (simple): ingest first N questions and write hypotheses.jsonl")
    parser.add_argument("config", help="Path to TOML config")
    parser.add_argument("--num-questions", type=int, default=None, help="Number of questions to process (overrides params.question_limit)")
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        raw_cfg: dict[str, Any] = tomllib.load(f)

    cfg = parse_config(raw_cfg)
    # Apply CLI override for number of questions
    if args.num_questions is not None:
        cfg.params.question_limit = max(0, int(args.num_questions))

    # Print brief config summary (kept for tests)
    print(
        f"Loaded config: vault={'id:'+cfg.vault_id if cfg.vault_id else 'title:'+str(cfg.vault_title)} "
        f"provider={cfg.provider.type} agent={cfg.models.agent} qa={cfg.models.qa} "
        f"top_k={cfg.params.top_k} run_id={cfg.run_id} "
        f"limits:q={cfg.params.question_limit} sess/q={cfg.params.max_sessions_per_question} turns/sess={cfg.params.max_turns_per_session}"
    )

    # Load dataset
    ds: List[Dict[str, Any]] = list(load_longmemeval_file(cfg.dataset_file_path))
    if cfg.params.question_limit and cfg.params.question_limit > 0:
        ds = ds[: cfg.params.question_limit]
    if not ds:
        print("[simple] no questions found – ensure dataset files are present")
        return

    # Initialize agent
    agent = build_agent(
        cfg.models.agent,
        max_tool_calls_per_turn=cfg.params.max_tool_calls_per_turn,
        provider_type=cfg.provider.type,
    )

    # Resolve vault once
    vault_id = agent.ensure_vault(cfg.vault_title, cfg.vault_id)

    # Prepare QA model
    from langchain.chat_models import init_chat_model  # type: ignore
    qa_model_id = cfg.models.qa
    if str(cfg.provider.type).lower() == "bedrock" and not str(qa_model_id).startswith("bedrock:"):
        qa_model_id = f"bedrock:{qa_model_id}"
    qa_llm = init_chat_model(qa_model_id)

    # Output file
    out_dir = _compute_out_dir(cfg.run_id)
    os.makedirs(out_dir, exist_ok=True)
    hyp_path = os.path.join(out_dir, "hypotheses.jsonl")

    wrote = 0
    with open(hyp_path, "w", encoding="utf-8") as f:
        for idx, q in enumerate(ds, start=1):
            qid = q.get("question_id", f"Q{idx}")
            mem_title = build_memory_title(cfg.memory_title_template, qid, cfg.run_id)
            print(f"[simple] processing question={qid} → memory_title={mem_title}")

            # Create/bind memory
            memory_id = agent.ensure_memory(vault_id, mem_title, memory_type="NOTES")
            agent.bind_memory(vault_id, memory_id)

            # Stream sessions
            session_idx = 0
            for s in q.get("sessions", []):
                session_idx += 1
                if cfg.params.max_sessions_per_question and session_idx > cfg.params.max_sessions_per_question:
                    break
                msgs = s.get("messages", [])
                if cfg.params.max_turns_per_session and isinstance(msgs, list):
                    msgs = msgs[: cfg.params.max_turns_per_session]
                
                # Session start/end logging for debug
                print(f"[session] {session_idx} START ({len(msgs)} messages)")
                agent.run_session(msgs)
                print(f"[session] {session_idx} END")

            # Build query and run QA
            qtext = (q.get("question") or "") if isinstance(q.get("question"), str) else ""
            if not qtext:
                qtext = _derive_question_from_sessions(q)
            sr = agent.search_memories(memory_id, query=str(qtext or mem_title), top_k=cfg.params.top_k)
            qa_context = _build_qa_context(sr, cfg.params.top_k)
            prompt = (
                "You are a helpful assistant. Answer the question using the provided memory context.\n\n"
                + ("Context:\n" + qa_context + "\n\n" if qa_context else "")
                + "Question: " + (qtext or mem_title)
            )
            ans = qa_llm.invoke(prompt)  # type: ignore
            predicted = (getattr(ans, "content", str(ans)) or "").strip()

            # Write hypothesis line
            f.write(json.dumps({"question_id": qid, "hypothesis": predicted}) + "\n")
            wrote += 1

    print(f"[simple] wrote {wrote} lines to {hyp_path}")
    print("To score with LongMemEval:")
    print("  cd /Users/deesam/workspace/LongMemEval/src/evaluation")
    print(f"  python3 evaluate_qa.py gpt-4o {hyp_path} /Users/deesam/workspace/mycelian/mycelian-memory/tools/longmemeval-benchmarker/out/longmemeval_oracle.json")

    # Cleanup
    try:
        agent.close()
    except Exception:
        pass


if __name__ == "__main__":
    main()

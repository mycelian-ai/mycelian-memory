import os
import json
import argparse
from typing import Any, Dict, List


def load_dataset(path: str) -> List[Dict[str, Any]]:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def build_turns(rec: Dict[str, Any], max_sessions: int = 1, max_turns: int = 20) -> List[Dict[str, str]]:
    out: List[Dict[str, str]] = []
    sessions = rec.get("haystack_sessions", [])
    for sidx, sess in enumerate(sessions[: max_sessions]):
        for midx, m in enumerate(sess[: max_turns]):
            out.append({
                "role": str(m.get("role") or "user"),
                "content": str(m.get("content") or ""),
                "session_index": str(sidx+1),
                "turn_index": str(midx+1),
            })
    return out


def load_summary_prompt() -> str:
    here = os.path.dirname(__file__)
    p = os.path.normpath(os.path.join(here, "..", "..", "client", "prompts", "default", "chat", "summary_prompt.md"))
    try:
        with open(p, "r", encoding="utf-8") as f:
            return f.read()
    except Exception:
        return "You are the Summary Agent. Write a one-line summary in past tense."


def main() -> None:
    parser = argparse.ArgumentParser(description="Probe per-turn summaries with prior context")
    parser.add_argument("dataset_file", help="Path to LongMemEval dataset JSON")
    parser.add_argument("question_id", help="Question ID to sample")
    parser.add_argument("--model", default=None, help="LLM model id (e.g., openai:gpt-5-nano-2025-08-07)")
    parser.add_argument("--max-sessions", type=int, default=1)
    parser.add_argument("--max-turns", type=int, default=20)
    parser.add_argument("--out", default=None, help="Output path for summaries log")
    args = parser.parse_args()

    data = load_dataset(args.dataset_file)
    rec = next((r for r in data if r.get("question_id") == args.question_id), None)
    if not rec:
        raise SystemExit(f"question_id not found: {args.question_id}")

    turns = build_turns(rec, max_sessions=args.max_sessions, max_turns=args.max_turns)

    from langchain.chat_models import init_chat_model  # type: ignore
    from langchain_core.messages import SystemMessage, HumanMessage  # type: ignore

    model_id = args.model or os.environ.get("LME_AGENT_MODEL") or "openai:gpt-5-nano-2025-08-07"
    llm = init_chat_model(model_id)  # type: ignore

    summary_prompt = load_summary_prompt()

    history: List[Dict[str, str]] = []
    logs: List[str] = []

    for t in turns:
        history.append(t)
        # Build context text = prior turns (shortened) + current message
        ctx_lines: List[str] = []
        for h in history[-10:]:  # last 10 turns for context
            ctx_lines.append(f"{h['role']}: {h['content']}")
        context_text = "\n".join(ctx_lines)

        system = summary_prompt
        user = (
            "Summarise the CURRENT MESSAGE in the context below.\n\n"
            "Context (most recent last):\n" + context_text + "\n\n"
            "Return one micro-summary per the rules."
        )
        ans = llm.invoke([SystemMessage(content=system), HumanMessage(content=user)])  # type: ignore
        summary = getattr(ans, "content", str(ans))
        logs.append(f"S{t['session_index']} T{t['turn_index']} {t['role']}: {summary}")

    out_path = args.out or os.path.join(os.path.dirname(__file__), "..", "out", f"summary_probe_{args.question_id}.log")
    out_path = os.path.normpath(out_path)
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    with open(out_path, "w", encoding="utf-8") as f:
        f.write("\n".join(logs))
    print(f"Wrote {len(logs)} summaries to {out_path}")


if __name__ == "__main__":
    main()

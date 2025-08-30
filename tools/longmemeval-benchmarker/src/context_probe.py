import os
import json
import argparse
from typing import Any, Dict, List


def load_dataset(path: str) -> List[Dict[str, Any]]:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def build_conversation_messages(
    rec: Dict[str, Any], *, every: int = 6, max_sessions: int = 0, max_turns: int = 0,
    include_session_ids: List[str] | None = None,
) -> List[Dict[str, str]]:
    """Mirror runner._with_control_messages(): add start, periodic FLUSH, and end controls.

    Returns items as {"content": json-string} with schema:
    { type: "conversation" | "system", content: string, role?: "user" | "assistant" }.
    """
    import json as _json
    out: List[Dict[str, str]] = []

    # Session start
    out.append({
        "content": _json.dumps({
            "type": "system",
            "content": "SESSION_START: Call get_context, and then list_entries(limit=10) if resuming a session."
        })
    })

    session_count = 0
    sessions = rec.get("haystack_sessions", [])
    session_ids = rec.get("haystack_session_ids", [])
    for idx, s in enumerate(sessions):
        sid = session_ids[idx] if idx < len(session_ids) else None
        if include_session_ids and sid not in include_session_ids:
            continue
        session_count += 1
        if max_sessions and session_count > max_sessions:
            break
        msgs = list(s) if isinstance(s, list) else []
        if max_turns and isinstance(msgs, list):
            msgs = msgs[:max_turns]
        for i, m in enumerate(msgs):
            out.append({
                "content": _json.dumps({
                    "type": "conversation",
                    "role": m.get("role"),
                    "content": m.get("content", "")
                })
            })
            if (i + 1) % every == 0:
                out.append({
                    "content": _json.dumps({
                        "type": "system",
                        "content": "FLUSH_CONTEXT: Call await_consistency then put_context now."
                    })
                })
        # Session end
        out.append({
            "content": _json.dumps({
                "type": "system",
                "content": "SESSION_END: Call await_consistency then put_context to finalize."
            })
        })
    return out


def render_recent_messages(messages: List[Dict[str, str]]) -> str:
    """Human-readable block of recent conversation messages for the LLM."""
    import json as _json
    lines: List[str] = []
    for m in messages:
        try:
            obj = _json.loads(m.get("content", ""))
        except Exception:
            obj = {"type": "conversation", "content": m.get("content", "")}
        if obj.get("type") == "conversation":
            role = obj.get("role") or "user"
            content = obj.get("content", "")
            lines.append(f"- {role}: {content}")
    return "\n".join(lines)


def load_context_prompt() -> str:
    # Load local prompt file directly (simpler and avoids async MCP in this probe).
    here = os.path.dirname(__file__)
    local_path = os.path.normpath(os.path.join(here, "..", "..", "client", "prompts", "default", "chat", "context_prompt.md"))
    try:
        with open(local_path, "r", encoding="utf-8") as f:
            return f.read()
    except Exception:
        return "You are the Mycelian Context Maintenance Agent. Maintain a concise, durable context using bullets."


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate context for a LongMemEval question without writing to Mycelian.")
    parser.add_argument("dataset_file", help="Path to LongMemEval dataset JSON (e.g., tools/longmemeval-benchmarker/out/longmemeval_5s.json)")
    parser.add_argument("question_id", help="Question ID to probe (e.g., 0a995998)")
    parser.add_argument("--out", dest="out_path", default=None, help="Output file path (default: tools/.../out/context_probe_<qid>.md)")
    parser.add_argument("--every", type=int, default=6, help="Flush interval for simulated context updates")
    parser.add_argument("--max-sessions", type=int, default=0, help="Limit number of sessions to process (0=no limit)")
    parser.add_argument("--max-turns", type=int, default=0, help="Limit number of turns per session (0=no limit)")
    parser.add_argument("--model", default=None, help="LLM model id (default: env or bedrock:anthropic.claude-3-5-haiku-20241022-v1:0)")
    parser.add_argument("--system-prompt-file", dest="system_prompt_file", default=None, help="Optional path to a system prompt file to override the default context prompt")
    parser.add_argument("--single", action="store_true", help="Generate context in a single pass over all messages")
    parser.add_argument("--sessions", nargs="*", help="Specific session IDs to include (e.g., answer_afa9873b_1)")
    args = parser.parse_args()

    # Load dataset and target question
    data = load_dataset(args.dataset_file)
    rec = next((r for r in data if r.get("question_id") == args.question_id), None)
    if not rec:
        raise SystemExit(f"question_id not found: {args.question_id}")

    # Build message stream identical to runner
    msgs = build_conversation_messages(
        rec, 
        every=args.every, 
        max_sessions=args.max_sessions, 
        max_turns=args.max_turns,
        include_session_ids=args.sessions
    )
    # Basic diagnostics
    import json as _json
    conv_count = 0
    sys_count = 0
    for _m in msgs:
        try:
            _o = _json.loads(_m.get("content", ""))
            if _o.get("type") == "conversation":
                conv_count += 1
            elif _o.get("type") == "system":
                sys_count += 1
        except Exception:
            pass
    sel_sessions = args.sessions or []
    print(f"[probe] question={args.question_id} sessions_selected={len(sel_sessions)} conv_msgs={conv_count} system_msgs={sys_count} every={args.every}")
    if sel_sessions:
        print(f"[probe] include_session_ids={sel_sessions}")

    # Initialize model and prompt
    from langchain.chat_models import init_chat_model  # type: ignore
    model_id = args.model or os.environ.get("LME_AGENT_MODEL") or os.environ.get("LME_QA_MODEL") or "bedrock:anthropic.claude-3-5-haiku-20241022-v1:0"
    llm = init_chat_model(model_id)  # type: ignore
    print(f"[probe] model={model_id}")
    context_prompt = load_context_prompt()
    if args.system_prompt_file:
        try:
            with open(args.system_prompt_file, "r", encoding="utf-8") as spf:
                context_prompt = spf.read()
            print(f"[probe] using custom system prompt: {args.system_prompt_file}")
        except Exception as e:
            print(f"[probe] failed to read custom system prompt: {e}; falling back to default")

    # Accumulate context updates
    working_context = ""
    window: List[Dict[str, str]] = []

    def update_context_with(messages_block: List[Dict[str, str]]) -> None:
        nonlocal working_context
        system = context_prompt
        recent = render_recent_messages(messages_block)
        user = (
            "Update the context using the following recent messages.\n\n"
            f"Current context (may be empty):\n{working_context}\n\n"
            f"Recent messages:\n{recent}\n\n"
            "Return the full updated context document."
        )
        # Chat models expect a list of BaseMessages; construct explicitly
        from langchain_core.messages import SystemMessage, HumanMessage  # type: ignore
        print(f"[probe] invoking LLM: window_size={len(messages_block)} …", flush=True)
        ans = llm.invoke([SystemMessage(content=system), HumanMessage(content=user)])  # type: ignore
        working_context = getattr(ans, "content", str(ans))
        print(f"[probe] invoke done. context_len={len(working_context)}", flush=True)

    if args.single:
        # Single pass over all conversation messages
        window = [m for m in msgs if '"type": "conversation"' in m.get("content", "")]
        print(f"[probe] single-pass update with window_size={len(window)}")
        update_context_with(window)
    else:
        # Multi-pass: update on FLUSH and SESSION_END
        import json as _json
        for m in msgs:
            try:
                obj = _json.loads(m.get("content", ""))
            except Exception:
                obj = {"type": "conversation", "content": m.get("content", "")}
            if obj.get("type") == "conversation":
                window.append(m)
                continue
            # System control
            ctrl = str(obj.get("content", ""))
            if "FLUSH_CONTEXT" in ctrl or "SESSION_END" in ctrl:
                if window:
                    print(f"[probe] control={ctrl.split(':',1)[0]} triggering update with window_size={len(window)}")
                    update_context_with(window)
                    window.clear()

    # Write output
    default_out = os.path.normpath(os.path.join(os.path.dirname(__file__), "..", "out", f"context_probe_{args.question_id}.md"))
    out_path = args.out_path or default_out
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(working_context or "")
    print(f"Wrote context to {out_path}")


if __name__ == "__main__":
    main()



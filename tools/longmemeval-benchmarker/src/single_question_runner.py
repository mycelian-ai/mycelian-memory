from typing import Any, Dict, TextIO
import logging

from mycelian_memory_agent import build_agent
from memory_manager import MemoryManager


def _derive_question_from_sessions(rec: Dict[str, Any]) -> str:
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


def _build_qa_context(search_result: Dict[str, Any], top_k: int) -> str:
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


def _run_qa(model_id: str, question_text: str, context: str) -> str:
    from langchain.chat_models import init_chat_model  # type: ignore

    prompt = (
        "You are a helpful assistant. Answer the question using the provided memory context.\n\n"
        + ("Context:\n" + context + "\n\n" if context else "")
        + "Question: "
        + (question_text or "")
    )
    llm = init_chat_model(model_id)  # type: ignore
    ans = llm.invoke(prompt)  # type: ignore
    return (getattr(ans, "content", str(ans)) or "").strip()


class SingleQuestionRunner:
    """Run a single LongMemEval question end-to-end.

    Responsibilities:
    - Build an agent and bind it to a specific `vault_id`/memory via `MemoryManager`.
    - For each session: send system control messages (SESSION_START/SESSION_END).
    - For each conversation message: validate role/content and invoke the agent with
      `type=conversation`, including a strictly increasing `msg_idx` per session.
    - Perform a memory search to construct QA context and invoke the QA model.
    - Return a minimal result object: {"question_id", "hypothesis"}.

    Non-responsibilities:
    - No concurrency or worker orchestration.
    - No dataset or vault discovery (expects `vault_id` and `run_id` as inputs).
    - No global logging; writes to the provided `log` stream only.
    """
    def __init__(self, cfg: Any):
        self.cfg = cfg

    def run_question(self, q: Dict[str, Any], vault_id: str, run_id: str, log: TextIO) -> Dict[str, Any]:
        qid = q.get("question_id", "unknown")
        mem_title = (self.cfg.memory_title_template or "{question_id}__{run_id}").format(
            question_id=qid, run_id=run_id
        )
        runner_log = logging.getLogger("lme.runner")
        runner_log.info("RUN qid=%s run_id=%s vault_id=%s agent=%s qa=%s", qid, run_id, vault_id, self.cfg.models.agent, self.cfg.models.qa)

        ag = build_agent(
            self.cfg.models.agent,
            max_tool_calls_per_turn=self.cfg.params.max_tool_calls_per_turn,
            provider_type="openai",
            debug=self.cfg.params.debug,
        )
        try:
            # Route agent logs to the provided log stream if available
            try:
                ag.set_log_stream(log)  # type: ignore[attr-defined]
            except Exception:
                pass

            mm = MemoryManager(ag._mcp, debug=self.cfg.params.debug)
            memory_id = mm.ensure_memory(vault_id, mem_title, memory_type="NOTES")
            runner_log.info("MEMORY_BOUND qid=%s memory_id=%s title=%s", qid, memory_id, mem_title)
            ag.bind_memory(vault_id, memory_id)

            for s_idx, s in enumerate(q.get("sessions", []), start=1):
                thread_id = f"{memory_id}:s{s_idx}"
                runner_log.info("SESSION_START qid=%s s=%d memory_id=%s thread_id=%s", qid, s_idx, memory_id, thread_id)
                ag.invoke_message(type="system", content="SESSION_START", thread_id=thread_id)
                for msg_idx, m in enumerate(s.get("messages", []), start=1):
                    role = (m.get("role") or "").strip().lower()
                    content = m.get("content") or ""
                    if role in ("user", "assistant") and isinstance(content, str) and content.strip():
                        runner_log.info("MSG qid=%s s=%d msg=%d role=%s memory_id=%s", qid, s_idx, msg_idx, role, memory_id)
                        ag.invoke_message(
                            type="conversation",
                            role=role,
                            content=content,
                            thread_id=thread_id,
                            msg_idx=msg_idx,
                        )
                ag.invoke_message(type="system", content="SESSION_END", thread_id=thread_id)
                runner_log.info("SESSION_END qid=%s s=%d memory_id=%s thread_id=%s", qid, s_idx, memory_id, thread_id)
                # Optional in-memory state dump at session end
                try:
                    if getattr(self.cfg.params, "dump_state", False):
                        ag._dump_agent_state(thread_id)  # type: ignore[attr-defined]
                except Exception:
                    pass

            # Build QA and return hypothesis
            qtext = q.get("question") or _derive_question_from_sessions(q)
            sr = MemoryManager(ag._mcp, debug=self.cfg.params.debug).search_memories(
                memory_id, query=str(qtext or mem_title), top_k=self.cfg.params.top_k
            )
            ctx = _build_qa_context(sr, self.cfg.params.top_k)
            predicted = _run_qa(self.cfg.models.qa, qtext or mem_title, ctx)
            return {"question_id": qid, "hypothesis": predicted}
        finally:
            try:
                ag.close()
            except Exception:
                pass



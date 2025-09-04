"""SingleQuestionRunner using the clean agent implementation."""

from typing import Any, Dict, TextIO, Optional
import logging
import json
import os

from src.mycelian_memory_agent.build import build_agent_with_invoker
from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from src.memory_manager import MemoryManager


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
    from langchain.chat_models import init_chat_model
    from src.tenacious_agent_invoker import invoke_with_backoff

    prompt = (
        "You are a helpful assistant. Answer the question using the provided memory context.\n\n"
        + ("Context:\n" + context + "\n\n" if context else "")
        + "Question: "
        + (question_text or "")
    )
    
    def _qa_call():
        llm = init_chat_model(model_id, model_provider="openai")
        return llm.invoke(prompt)
    
    # Use retry logic for QA calls
    ans = invoke_with_backoff(_qa_call)
    return (getattr(ans, "content", str(ans)) or "").strip()


class SingleQuestionRunner:
    """Run a single LongMemEval question using the clean agent implementation.

    This version uses:
    - MycelianMemoryAgent with clean control-based protocol
    - MycelianAgentInvoker for message counting and flush logic
    - Simplified message processing without complex type checking
    """
    
    def __init__(self, cfg: Any, mcp_client: Any = None, mode: str = "all"):
        self.cfg = cfg
        self.mcp_client = mcp_client
        self.mode = mode  # "ingestion", "qa", or "all"

    def run_question(self, q: Dict[str, Any], vault_id: str, run_id: str, log: TextIO, 
                    memory_id: Optional[str] = None, qa_log_path: Optional[str] = None) -> Dict[str, Any]:
        qid = q.get("question_id", "unknown")
        mem_title = (self.cfg.memory_title_template or "{question_id}__{run_id}").format(
            question_id=qid, run_id=run_id
        )
        runner_log = logging.getLogger("lme.runner")
        runner_log.info("RUN qid=%s run_id=%s vault_id=%s agent=%s qa=%s", 
                        qid, run_id, vault_id, self.cfg.models.agent, self.cfg.models.qa)

        # Use provided MCP client or create one
        if self.mcp_client is None:
            self.mcp_client = create_mcp_client()
        
        # For QA-only mode, memory_id must be provided
        if self.mode == "qa":
            if not memory_id:
                raise ValueError("memory_id is required for QA-only mode")
            runner_log.info("QA_ONLY_MODE qid=%s using existing memory_id=%s", qid, memory_id)
            
            # Create a minimal invoker just for MCP client access
            invoker = build_agent_with_invoker(
                model_id=self.cfg.models.agent,
                vault_id=vault_id,
                memory_id=memory_id,
                mcp_client=self.mcp_client,
                max_tool_calls_per_turn=self.cfg.params.max_tool_calls_per_turn
            )
        else:
            # Use MemoryManager to ensure memory exists
            mm = MemoryManager(self.mcp_client, debug=False)
            memory_id = mm.ensure_memory(vault_id, mem_title, memory_type="NOTES")
            runner_log.info("MEMORY_BOUND qid=%s memory_id=%s title=%s", qid, memory_id, mem_title)
            
            # Build agent with clean implementation
            invoker = build_agent_with_invoker(
                model_id=self.cfg.models.agent,
                vault_id=vault_id,
                memory_id=memory_id,
                mcp_client=self.mcp_client,
                max_tool_calls_per_turn=self.cfg.params.max_tool_calls_per_turn
            )
            
            # Optionally set log stream (for compatibility)
            try:
                if hasattr(invoker, 'set_log_stream'):
                    invoker.set_log_stream(log)
            except Exception:
                pass
        
        try:
            # Skip ingestion for QA-only mode
            if self.mode != "qa":
                # Process all sessions
                for s_idx, s in enumerate(q.get("sessions", []), start=1):
                    thread_id = f"{memory_id}:s{s_idx}"
                    runner_log.info("SESSION_START qid=%s s=%d memory_id=%s thread_id=%s", 
                                  qid, s_idx, memory_id, thread_id)
                    
                    # Start session (retrieves context and recent entries)
                    invoker.start_session(thread_id)
                    
                    # Process all messages in the session
                    for msg_idx, m in enumerate(s.get("messages", []), start=1):
                        role = (m.get("role") or "").strip().lower()
                        content = m.get("content") or ""
                        
                        # Only process user and assistant messages with content
                        if role in ("user", "assistant") and isinstance(content, str) and content.strip():
                            runner_log.info("MSG qid=%s s=%d msg=%d role=%s memory_id=%s", 
                                          qid, s_idx, msg_idx, role, memory_id)
                            
                            # Process the message (handles flush every 6 messages automatically)
                            invoker.process_conversation_message(
                                role=role,
                                content=content,
                                thread_id=thread_id
                            )
                    
                    # End session (ensures consistency and saves context)
                    invoker.end_session(thread_id)
                    runner_log.info("SESSION_END qid=%s s=%d memory_id=%s thread_id=%s", 
                                  qid, s_idx, memory_id, thread_id)
                    
                    # Optional state dump for debugging
                    try:
                        if getattr(self.cfg.params, "dump_state", False) and hasattr(invoker, '_dump_agent_state'):
                            invoker._dump_agent_state(thread_id)
                    except Exception:
                        pass
            
            # Skip QA for ingestion-only mode
            if self.mode == "ingestion":
                runner_log.info("INGESTION_ONLY qid=%s skipping QA phase", qid)
                return {"question_id": qid, "hypothesis": ""}  # Empty hypothesis for ingestion-only

            # Build QA and return hypothesis
            qtext = q.get("question") or _derive_question_from_sessions(q)
            runner_log.info("QA_START qid=%s memory_id=%s question='%s'", 
                          qid, memory_id, qtext[:100] if qtext else "(derived)")
            
            # Use the MCP client from the invoker for search
            query_text = str(qtext or mem_title)
            runner_log.info("SEARCH_MEMORIES qid=%s memory_id=%s query='%s' top_k=%d", 
                          qid, memory_id, query_text[:100], self.cfg.params.top_k)
            
            sr = MemoryManager(invoker._mcp, debug=False).search_memories(
                memory_id, query=query_text, top_k=self.cfg.params.top_k
            )
            
            # Log search results
            entries_count = len(sr.get("entries", []) if isinstance(sr, dict) else [])
            has_latest = bool((sr.get("latestContext") or sr.get("latest_context")) if isinstance(sr, dict) else False)
            has_best = bool((sr.get("bestContext") or sr.get("best_context")) if isinstance(sr, dict) else False)
            runner_log.info("SEARCH_RESULT qid=%s entries=%d has_latest=%s has_best=%s", 
                          qid, entries_count, has_latest, has_best)
            
            # Build context and log it
            ctx = _build_qa_context(sr, self.cfg.params.top_k)
            ctx_preview = ctx[:500] if ctx else "(empty)"
            runner_log.info("QA_CONTEXT qid=%s context_len=%d preview='%s'", 
                          qid, len(ctx) if ctx else 0, ctx_preview)
            
            # Run QA and log
            runner_log.info("QA_INVOKE qid=%s model=%s", qid, self.cfg.models.qa)
            predicted = _run_qa(self.cfg.models.qa, qtext or mem_title, ctx)
            runner_log.info("QA_RESPONSE qid=%s response_len=%d response='%s'", 
                          qid, len(predicted), predicted[:200] if predicted else "(empty)")
            
            return {"question_id": qid, "hypothesis": predicted}
            
        finally:
            # Clean up if needed
            try:
                if hasattr(invoker, 'close'):
                    invoker.close()
            except Exception:
                pass
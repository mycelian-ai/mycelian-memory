"""SingleQuestionRunner using the clean agent implementation."""

from typing import Any, Dict, TextIO, Optional
import logging
import json
import os
import time

from mycelian_memory_agent.build import build_agent_with_invoker
from mycelian_memory_agent.mcp_utils import create_mcp_client
from memory_manager import MemoryManager
from pathlib import Path
import sqlite3


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
    from model_providers import get_chat_model
    
    prompt = (
        "You are a helpful assistant. Answer the question using the provided memory context.\n"
        "Before answering, carefully consider what the question is asking for.\n"
        "Evaluate each piece of relevant information in the context to determine if it should be part of your answer.\n\n"
        + ("Context:\n" + context + "\n\n" if context else "")
        + "Question: "
        + (question_text or "")
    )
    
    # Use provider-agnostic model with built-in retry
    llm = get_chat_model(model_id)  # max_retries=6 is default
    ans = llm.invoke(prompt)
    return (getattr(ans, "content", str(ans)) or "").strip()


def _two_pass_search(memory_manager: "MemoryManager", memory_id: str, question: str, 
                     model_id: str, logger: Optional[logging.Logger] = None) -> Dict[str, Any]:
    """Two-pass search algorithm for better retrieval.
    
    First pass: Search with original question using hybrid search on summaries and context.
    Analysis: Check if results contain relevant information or need refinement.
    Second pass (optional): Refined search based on first pass analysis.
    """
    from model_providers import get_chat_model
    
    if logger is None:
        logger = logging.getLogger("lme.runner")
    
    # First pass: Search with original question
    logger.info("TWO_PASS_SEARCH pass=1 query='%s'", question[:100])
    
    # Use higher limits for first pass to get broader results
    first_results = memory_manager.search_memories(
        memory_id, 
        query=question, 
        top_ke=10,  # Get more entries for analysis
        top_kc=3    # Get more context shards
    )
    
    # Analyze first pass results
    entries = first_results.get("entries", [])
    contexts = first_results.get("contexts", [])
    
    # Log what we found
    logger.info("TWO_PASS_SEARCH pass=1 found entries=%d contexts=%d", 
                len(entries), len(contexts))
    
    # Build summary of what we found for analysis
    summaries_text = "\n".join([
        f"- {e.get('summary', '')}" 
        for e in entries[:10] 
        if e.get('summary')
    ])
    
    # Check if we need a second pass
    if not summaries_text and not contexts:
        logger.info("TWO_PASS_SEARCH no_content_found, returning first results")
        return first_results
    
    # Use LLM to analyze if we need refinement
    analysis_prompt = f"""Analyze these search results for the question: "{question}"

Found summaries:
{summaries_text[:2000]}

Based on these search results, determine if a refined search would help find additional relevant information.
If the results seem comprehensive, respond with "SUFFICIENT".
If a refined search could help, respond with "REFINE: <refined query>"."""
    
    llm = get_chat_model(model_id)
    analysis = llm.invoke(analysis_prompt)
    analysis_text = (getattr(analysis, "content", str(analysis)) or "").strip()
    
    logger.info("TWO_PASS_SEARCH analysis='%s'", analysis_text[:200])
    
    # Check if second pass is needed
    if not analysis_text.startswith("REFINE:"):
        logger.info("TWO_PASS_SEARCH no_refinement_needed")
        return first_results
    
    # Extract refined query
    refined_query = analysis_text[7:].strip()
    if not refined_query:
        logger.info("TWO_PASS_SEARCH empty_refinement, using first results")
        return first_results
    
    # Second pass with refined query
    logger.info("TWO_PASS_SEARCH pass=2 refined_query='%s'", refined_query[:100])
    
    second_results = memory_manager.search_memories(
        memory_id,
        query=refined_query,
        top_ke=5,  # Focused search
        top_kc=2   # Focused context
    )
    
    # Merge results: combine unique entries and contexts from both passes
    merged = {
        "entries": [],
        "contexts": [],
        "latestContext": first_results.get("latestContext") or first_results.get("latest_context"),
        "bestContext": first_results.get("bestContext") or first_results.get("best_context")
    }
    
    # Track seen entry IDs to avoid duplicates
    seen_entry_ids = set()
    
    # Add first pass entries
    for entry in first_results.get("entries", []):
        entry_id = entry.get("id") or entry.get("entryId")
        if entry_id and entry_id not in seen_entry_ids:
            merged["entries"].append(entry)
            seen_entry_ids.add(entry_id)
    
    # Add second pass entries if unique
    for entry in second_results.get("entries", []):
        entry_id = entry.get("id") or entry.get("entryId")
        if entry_id and entry_id not in seen_entry_ids:
            merged["entries"].append(entry)
            seen_entry_ids.add(entry_id)
    
    # Combine contexts (these don't have IDs so check by content)
    seen_contexts = set()
    for ctx in first_results.get("contexts", []) + second_results.get("contexts", []):
        if isinstance(ctx, dict):
            ctx_text = ctx.get("context", "")
            if ctx_text and ctx_text not in seen_contexts:
                merged["contexts"].append(ctx)
                seen_contexts.add(ctx_text)
    
    logger.info("TWO_PASS_SEARCH merged entries=%d contexts=%d", 
                len(merged["entries"]), len(merged["contexts"]))
    
    return merged


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
        runner_log.info("RUN qid=%s run_id=%s vault_id=%s ingest=%s qa=%s", 
                        qid, run_id, vault_id, self.cfg.models.ingest, self.cfg.models.qa)

        start_ts = time.time()
        messages_processed = 0
        sessions_done = 0
        # Progress DB path (runner updates directly for fine-grained progress)
        # Use benchmarker root data/ directory (same as orchestrator)
        default_db = Path(__file__).resolve().parents[1] / "data" / "progress.db"
        progress_db_path = str(default_db)

        def _ensure_progress_schema_once() -> None:
            try:
                with sqlite3.connect(progress_db_path) as _conn:
                    cols = {row[1] for row in _conn.execute("PRAGMA table_info(question_progress)")}
                    if 'ingested_messages' not in cols:
                        _conn.execute("ALTER TABLE question_progress ADD COLUMN ingested_messages INTEGER DEFAULT 0")
                    if 'total_messages' not in cols:
                        _conn.execute("ALTER TABLE question_progress ADD COLUMN total_messages INTEGER DEFAULT 0")
                    if 'last_progress_at' not in cols:
                        _conn.execute("ALTER TABLE question_progress ADD COLUMN last_progress_at TIMESTAMP")
                    _conn.commit()
            except Exception:
                pass

        _ensure_progress_schema_once()

        def _write_progress(msgs: int, sessions: int | None = None) -> None:
            try:
                with sqlite3.connect(progress_db_path) as _conn:
                    if sessions is None:
                        _conn.execute(
                            """
                            UPDATE question_progress
                            SET ingested_messages = ?,
                                last_progress_at = strftime('%Y-%m-%d %H:%M:%S','now')
                            WHERE run_id = ? AND question_id = ?
                            """,
                            (msgs, run_id, qid),
                        )
                    else:
                        _conn.execute(
                            """
                            UPDATE question_progress
                            SET ingested_messages = ?,
                                completed_sessions = ?,
                                last_progress_at = strftime('%Y-%m-%d %H:%M:%S','now')
                            WHERE run_id = ? AND question_id = ?
                            """,
                            (msgs, sessions, run_id, qid),
                        )
                    _conn.commit()
            except Exception:
                pass

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
                model_id=self.cfg.models.ingest,
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
                model_id=self.cfg.models.ingest,
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
                            messages_processed += 1
                            # Periodically persist message-level progress (every 10 messages)
                            _write_progress(messages_processed)
                    
                    # End session (ensures consistency and saves context)
                    invoker.end_session(thread_id)
                    runner_log.info("SESSION_END qid=%s s=%d memory_id=%s thread_id=%s", 
                                  qid, s_idx, memory_id, thread_id)
                    sessions_done += 1
                    # Flush any remaining message increments since last batch
                    _write_progress(messages_processed, sessions_done)
                    
                    # Optional state dump for debugging
                    try:
                        if getattr(self.cfg.params, "dump_state", False) and hasattr(invoker, '_dump_agent_state'):
                            invoker._dump_agent_state(thread_id)
                    except Exception:
                        pass
            
            # Skip QA for ingestion-only mode
            if self.mode == "ingestion":
                runner_log.info(
                    "INGESTION_ONLY_DONE qid=%s sessions_completed=%d messages_processed=%d duration_sec=%.2f",
                    qid, sessions_done, messages_processed, time.time() - start_ts
                )
                return {
                    "status": "success",
                    "question_id": qid,
                    "vault_id": vault_id,
                    "memory_id": memory_id,
                    "sessions_completed": sessions_done,
                    "messages_processed": messages_processed
                }

            # Build QA and return hypothesis
            qtext = q.get("question") or _derive_question_from_sessions(q)
            runner_log.info("QA_START qid=%s memory_id=%s question='%s'", 
                          qid, memory_id, qtext[:100] if qtext else "(derived)")
            
            # Use the MCP client from the invoker for search
            query_text = str(qtext or mem_title)
            
            # Check if two-pass search is enabled (via config or default)
            use_two_pass = getattr(self.cfg.params, "use_two_pass_search", True)
            
            if use_two_pass:
                runner_log.info("SEARCH_MEMORIES qid=%s memory_id=%s mode=two_pass query='%s'", 
                              qid, memory_id, query_text[:100])
                sr = _two_pass_search(
                    MemoryManager(invoker._mcp, debug=False),
                    memory_id,
                    query_text,
                    self.cfg.models.qa,
                    runner_log
                )
            else:
                runner_log.info("SEARCH_MEMORIES qid=%s memory_id=%s mode=single query='%s' top_k=%d", 
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
            
            runner_log.info(
                "QUESTION_DONE qid=%s sessions_completed=%d messages_processed=%d duration_sec=%.2f",
                qid, sessions_done, messages_processed, time.time() - start_ts
            )
            return {
                "status": "success",
                "question_id": qid,
                "vault_id": vault_id,
                "memory_id": memory_id,
                "sessions_completed": sessions_done,
                "messages_processed": messages_processed,
                "hypothesis": predicted
            }
            
        finally:
            # Clean up if needed
            try:
                if hasattr(invoker, 'close'):
                    invoker.close()
            except Exception:
                pass
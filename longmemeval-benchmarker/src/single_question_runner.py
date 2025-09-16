"""SingleQuestionRunner using the clean agent implementation."""

from typing import Any, Dict, TextIO, Optional
import logging
import json
import os
import time

from src.mycelian_memory_agent.build import build_agent_with_invoker
from src.mycelian_memory_agent.mcp_utils import create_mcp_client
from src.memory_manager import MemoryManager
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


def _build_qa_context(search_result: Dict[str, Any]) -> str:
    import json

    # Get latestContext - handle JSON-encoded strings
    latest_ctx_raw = search_result.get("latestContext") or ""
    latest_ctx = ""
    if latest_ctx_raw:
        # Check if it's a JSON string (starts with '{')
        if isinstance(latest_ctx_raw, str) and latest_ctx_raw.strip().startswith('{'):
            try:
                parsed = json.loads(latest_ctx_raw)
                # Extract the actual context from JSON structure
                if isinstance(parsed, dict):
                    # Handle old format with activeContext field
                    latest_ctx = parsed.get("activeContext", "")
                    if not latest_ctx:
                        # If no activeContext, use the whole parsed object as string
                        latest_ctx = str(parsed)
            except json.JSONDecodeError:
                latest_ctx = latest_ctx_raw
        else:
            latest_ctx = str(latest_ctx_raw)
    latest_ctx = latest_ctx.strip()

    # Get context shards from contexts array (new API format)
    # Use all contexts returned by search (already limited by kc parameter)
    contexts = search_result.get("contexts") or []
    context_texts: list[str] = []
    for ctx in contexts:  # Use all context shards returned
        if isinstance(ctx, dict):
            ctx_text = ctx.get("context", "")
            if ctx_text:
                context_texts.append(str(ctx_text))

    # Get entry summaries
    # Use all entries returned by search (already limited by ke parameter)
    entries = search_result.get("entries") or []
    entries_text: list[str] = []
    for e in entries:  # Use all entries returned
        if isinstance(e, dict):
            txt = e.get("summary") or ""
            if txt:
                entries_text.append(str(txt))

    # Build final context from all parts
    # Priority: latestContext, then context shards, then entry summaries
    parts = []
    if latest_ctx:
        parts.append(latest_ctx)
    if context_texts:
        parts.append("\n\n".join(context_texts))
    if entries_text:
        parts.append("\n\n".join(entries_text))

    return "\n\n".join(parts) if parts else ""


def _run_qa(model_id: str, question_text: str, context: str) -> str:
    from src.model_providers import get_chat_model

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
    from src.model_providers import get_chat_model

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
    entries = (first_results.get("entries") or []) if isinstance(first_results, dict) else []
    contexts = (first_results.get("contexts") or []) if isinstance(first_results, dict) else []

    # Log what we found
    logger.info("TWO_PASS_SEARCH pass=1 found entries=%d contexts=%d",
                len(entries or []), len(contexts or []))

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
        "latestContext": first_results.get("latestContext") or first_results.get("latest_context")
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
        runner_log = logging.getLogger(f"lme.runner.{qid}")
        runner_log.info("RUN qid=%s run_id=%s vault_id=%s ingest=%s qa=%s",
                        qid, run_id, vault_id, self.cfg.models.ingest, self.cfg.models.qa)

        start_ts = time.time()
        messages_processed = 0
        sessions_done = 0
        # Progress DB path (runner updates directly for fine-grained progress)
        # Use benchmarker root data/ directory (same as orchestrator)
        default_db = Path(__file__).resolve().parents[1] / "data" / "progress.db"
        progress_db_path = str(default_db)

        # Bind per-question file handler to runner/agent loggers for this call
        qhandler: Optional[logging.Handler] = None
        logger_snapshots: list[tuple[logging.Logger, list[logging.Handler], bool, int]] = []

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
                    if 'vault_id' not in cols:
                        _conn.execute("ALTER TABLE question_progress ADD COLUMN vault_id TEXT")
                    if 'memory_id' not in cols:
                        _conn.execute("ALTER TABLE question_progress ADD COLUMN memory_id TEXT")
                    _conn.commit()
            except Exception:
                pass

        _ensure_progress_schema_once()

        # Attach file handler now that we have the output stream
        try:
            qhandler = logging.StreamHandler(log)
            qhandler.setFormatter(logging.Formatter("%(asctime)s [%(filename)s:%(funcName)s] %(message)s"))
            # Attach handler only to per-question runner logger
            lg = logging.getLogger(f"lme.runner.{qid}")
            logger_snapshots.append((lg, list(lg.handlers), lg.propagate, lg.level))
            lg.setLevel(logging.INFO)
            lg.addHandler(qhandler)
            lg.propagate = False
            # Defer agent logger handler until memory_id is known
            qhandler_agent = None
        except Exception:
            pass

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

        def _write_vault_memory_ids(vault: str, memory: str) -> None:
            try:
                with sqlite3.connect(progress_db_path) as _conn:
                    _conn.execute(
                        """
                        UPDATE question_progress
                        SET vault_id = ?, memory_id = ?, last_progress_at = strftime('%Y-%m-%d %H:%M:%S','now')
                        WHERE run_id = ? AND question_id = ?
                        """,
                        (vault, memory, run_id, qid),
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
            # Persist identifiers for orchestrator (vault_id known from arg)
            _write_vault_memory_ids(vault_id, memory_id)

            # Create a minimal invoker just for MCP client access
            # Check if config has context_only setting
            context_only = getattr(self.cfg, 'context_only', True)  # Default to True
            invoker = build_agent_with_invoker(
                model_id=self.cfg.models.ingest,
                vault_id=vault_id,
                memory_id=memory_id,
                mcp_client=self.mcp_client,
                max_tool_calls_per_turn=self.cfg.params.max_tool_calls_per_turn,
                context_only=context_only
            )
            # Attach agent logger for this memory_id only
            try:
                qhandler_agent = logging.StreamHandler(log)
                qhandler_agent.setFormatter(logging.Formatter("%(asctime)s [%(filename)s:%(funcName)s] %(message)s"))
                agent_logger = logging.getLogger(f"lme.agent.{memory_id}")
                logger_snapshots.append((agent_logger, list(agent_logger.handlers), agent_logger.propagate, agent_logger.level))
                agent_logger.setLevel(logging.INFO)
                agent_logger.addHandler(qhandler_agent)
                agent_logger.propagate = False
            except Exception:
                pass
        else:
            # Use MemoryManager to ensure memory exists
            mm = MemoryManager(self.mcp_client, debug=False)
            memory_id = mm.ensure_memory(vault_id, mem_title, memory_type="NOTES")
            runner_log.info("MEMORY_BOUND qid=%s memory_id=%s title=%s", qid, memory_id, mem_title)
            # Persist identifiers for orchestrator
            _write_vault_memory_ids(vault_id, memory_id)

            # Build agent with clean implementation
            # Check if config has context_only setting
            context_only = getattr(self.cfg, 'context_only', True)  # Default to True
            invoker = build_agent_with_invoker(
                model_id=self.cfg.models.ingest,
                vault_id=vault_id,
                memory_id=memory_id,
                mcp_client=self.mcp_client,
                max_tool_calls_per_turn=self.cfg.params.max_tool_calls_per_turn,
                context_only=context_only
            )
            # Attach agent logger for this memory_id only
            try:
                qhandler_agent = logging.StreamHandler(log)
                qhandler_agent.setFormatter(logging.Formatter("%(asctime)s [%(filename)s:%(funcName)s] %(message)s"))
                agent_logger = logging.getLogger(f"lme.agent.{memory_id}")
                logger_snapshots.append((agent_logger, list(agent_logger.handlers), agent_logger.propagate, agent_logger.level))
                agent_logger.setLevel(logging.INFO)
                agent_logger.addHandler(qhandler_agent)
                agent_logger.propagate = False
            except Exception:
                pass

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

                    try:
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
                    finally:
                        # Always attempt to end the session and advance counters,
                        # even if message processing raised.
                        try:
                            invoker.end_session(thread_id)
                        except Exception:
                            # Best-effort: continue to mark progress so orchestrator can resume safely
                            pass
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

            # Ensure all pending writes are complete before searching
            runner_log.info("AWAIT_CONSISTENCY qid=%s memory_id=%s", qid, memory_id)
            # Use MCP client directly to call await_consistency
            import asyncio
            from src.async_utils import run as run_async
            async def await_consistency():
                tools = await invoker._mcp.get_tools()
                for tool in tools:
                    if getattr(tool, "name", None) == "await_consistency":
                        return await tool.ainvoke({"memory_id": memory_id})
                raise RuntimeError("await_consistency tool not found")
            run_async(await_consistency())

            memory_manager = MemoryManager(invoker._mcp, debug=False)

            # Use the MCP client from the invoker for search
            query_text = str(qtext or mem_title)

            # Check if two-pass search is enabled (via config or default)
            use_two_pass = getattr(self.cfg.params, "use_two_pass_search", True)

            if use_two_pass:
                runner_log.info("SEARCH_MEMORIES qid=%s memory_id=%s mode=two_pass query='%s'",
                              qid, memory_id, query_text[:100])
                sr = _two_pass_search(
                    memory_manager,
                    memory_id,
                    query_text,
                    self.cfg.models.qa,
                    runner_log
                )
            else:
                runner_log.info("SEARCH_MEMORIES qid=%s memory_id=%s mode=single query='%s' top_ke=%d top_kc=%d",
                              qid, memory_id, query_text[:100], 5, 3)
                sr = memory_manager.search_memories(
                    memory_id, query=query_text, top_ke=5, top_kc=3
                )

            # Log search results
            entries_count = len((sr.get("entries") or []) if isinstance(sr, dict) else [])
            has_latest = bool((sr.get("latestContext") or sr.get("latest_context")) if isinstance(sr, dict) else False)
            contexts_count = len((sr.get("contexts") or []) if isinstance(sr, dict) else [])
            runner_log.info("SEARCH_RESULT qid=%s entries=%d has_latest=%s contexts=%d",
                          qid, entries_count, has_latest, contexts_count)

            # Build context and log it
            ctx = _build_qa_context(sr)
            ctx_preview = ctx[:500] if ctx else "(empty)"
            runner_log.info("QA_CONTEXT qid=%s context_len=%d preview='%s'",
                          qid, len(ctx) if ctx else 0, ctx_preview)

            # Run QA and log
            runner_log.info("QA_INVOKE qid=%s model=%s", qid, self.cfg.models.qa)
            predicted = _run_qa(self.cfg.models.qa, qtext or mem_title, ctx)
            runner_log.info("QA_RESPONSE qid=%s response_len=%d",
                          qid, len(predicted))

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
            # Restore loggers and remove file handler
            try:
                for lg, handlers_snapshot, prop, lvl in logger_snapshots:
                    try:
                        if qhandler:
                            lg.removeHandler(qhandler)
                        try:
                            qhandler_agent  # may be undefined if attach failed
                        except NameError:
                            qhandler_agent = None
                        if qhandler_agent:
                            lg.removeHandler(qhandler_agent)
                    except Exception:
                        pass
                    lg.propagate = prop
                    lg.setLevel(lvl)
                if qhandler:
                    try:
                        qhandler.flush()
                    except Exception:
                        pass
                try:
                    if qhandler_agent:
                        qhandler_agent.flush()
                except Exception:
                    pass
            except Exception:
                pass

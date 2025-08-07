from __future__ import annotations

import asyncio
import logging
import json
from datetime import datetime, timezone
import inspect
import random
import re
from typing import List, Dict, Any, Optional, Union, cast, TypedDict, Literal

# stdlib
import anthropic
import time
import os


class ContextData(TypedDict, total=False):
    """Type definition for context data structure."""
    last_updated: str
    message_count: int
    history_summary: str
    last_updated_by: str

from system_prompt_builder import PromptAssembler
from synapse_client import MycelianMemoryClient

# Marker the harness sends as last user turn in a session.
END_SESSION_TOKEN = "control:test_harness SESSION_END"

logger = logging.getLogger(__name__)

# Default to a 15-second interval between requests (~12 RPM). This is usually
# fast enough for local tests while remaining under Anthropic’s rate limits.
# Override via the SESSION_SIMULATOR_RATE_LIMIT_INTERVAL env var if you need a
# different pace.
_RATE_LIMIT_INTERVAL = float(os.getenv("SESSION_SIMULATOR_RATE_LIMIT_INTERVAL", "15.0"))
_last_call_ts: float = 0.0

async def _respect_rate_limit():
    """Sleep so that at most one Anthropic request is sent per interval."""
    global _last_call_ts
    now = time.monotonic()
    sleep_for = _RATE_LIMIT_INTERVAL - (now - _last_call_ts)
    if sleep_for > 0:
        await asyncio.sleep(sleep_for)
    _last_call_ts = time.monotonic()


class SessionSimulator:
    """Drive a single benchmark session through Claude with Mycelian tool calls.

    A new instance is created for every *session* (DMR has up to 5 per
    conversation).  Claude history is kept in-memory so the model preserves
    short-term context; long-term context is stored in Mycelian.
    """

    def __init__(
        self,
        anthropic_client,
        synapse_client: MycelianMemoryClient,
        system_builder: PromptAssembler,
        *,
        model_name: str = "claude-3-haiku-20240307",
    ) -> None:
        """Create a SessionSimulator.

        Tests sometimes pass a **string Anthropic API key** instead of an
        instantiated client to avoid heavy deps.  Accept either form for
        compatibility.
        """

        if isinstance(anthropic_client, str):
            self._ac = anthropic.Anthropic(api_key=anthropic_client)
        else:
            self._ac = anthropic_client
        self._sc = synapse_client
        self._system_builder = system_builder

        # Allow caller to override the Anthropic model for cost/perf testing
        self._model_name = model_name

        # Store system prompt separately per Anthropic API v1.
        self._system_prompt: str = system_builder.build()
        # History contains only user/assistant messages.
        self._history: List[Dict[str, str]] = []
        
        # Context management state
        self._message_counter = 0
        self._last_context_update = 0

        # Cache results of tool executions so we can echo them back to Claude
        # without issuing a second round-trip to the Mycelian CLI / HTTP API.
        self._tool_results: Dict[str, Any] = {}

        # Memory alias mapping: title/string -> UUID
        self._mem_aliases: Dict[str, str] = {}

        # Simple asset cache to prevent repeated fetches within one session
        self._asset_cache: Dict[str, str] = {}

        # Bootstrap compliance tracking for mandatory ctx_rules fetch
        self._boot_seen_ctx_rules = False
        self._boot_ctx_violation_asked = False

        # ------------------------------------------------------------------
        # Required static assets that Claude must fetch before we consider the
        # bootstrap sequence complete.  Without these chat-specific prompts
        # the model will not include `summary` in add_entry calls.
        # ------------------------------------------------------------------
        # Bootstrap now only requires governance rules; chat prompts load lazily.
        self._boot_required_assets: set[str] = {"ctx_rules"}
        self._boot_assets_downloaded: set[str] = set()

        # Bootstrap compliance tracking
        self._boot_seen_get_context = False
        self._boot_seen_list_entries = False
        self._boot_violation_asked = False

    # ------------------------------------------------------------------
    # Helper utilities
    # ------------------------------------------------------------------
    def _resolve_memory_id(self, candidate: Optional[str] = None) -> str:
        """Return a canonical memory_id.

        If *candidate* looks like a UUID (36 chars with hyphens), prefer it.
        Otherwise fall back to the memory_id stored on ``SystemPromptBuilder``.
        Raises ``ValueError`` if no valid ID can be determined.
        """
        # Direct UUID case
        if candidate and re.fullmatch(r"[0-9a-fA-F-]{36}", candidate):
            return candidate
        # Alias lookup (title -> id)
        if candidate and candidate in self._mem_aliases:
            return self._mem_aliases[candidate]
        mem_id = getattr(self._system_builder, "memory_id", None)
        if mem_id:
            return mem_id
        if candidate:
            # Last-ditch – allow caller value even if not UUID (may succeed)
            return candidate
        raise ValueError("Unable to resolve memory_id – none provided and none stored")

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------
    @property
    def tool_schema(self) -> List[Dict[str, Any]]:
        return self._system_builder.tool_schema

    # ------------------------------------------------------------------
    # Bootstrap status helper
    # ------------------------------------------------------------------
    @property
    def is_bootstrap_complete(self) -> bool:
        """Return True once the assistant has executed both mandatory bootstrap
        calls: `get_context()` and `list_entries(limit = 10)`.

        The ingestion harness waits for this flag before feeding the first
        conversation turn to avoid dropping messages that arrive before the
        context is initialized.
        """
        return (
            self._boot_seen_get_context
            and self._boot_seen_list_entries
            and self._boot_required_assets.issubset(self._boot_assets_downloaded)
        )

    async def step(self, user_text: str, max_tokens: int = 2000) -> str:
        """Send one user message, return Claude's visible reply text.
        
        Updates context every 5 messages or on critical updates.
        """
        # Log outgoing user message and current context length
        logger.debug("User message: %s", user_text)
        # ------------------------------------------------------------------
        # Emit INFO-level log so test harnesses can capture every outbound
        # user *or control* message without enabling DEBUG globally.
        # We tag control messages (sentinel tokens) with the prefix
        # [MSG][control] so they are clearly distinguishable in logs.
        # Control messages start with one of the defined sentinel tokens.
        # ------------------------------------------------------------------
        CONTROL_PREFIX = "control:"

        stripped = user_text.lstrip()
        is_control = stripped.lower().startswith(CONTROL_PREFIX)
        if is_control:
            # Reduce noise: control/bootstrap messages log at DEBUG to avoid clutter
            logger.debug("[CTRL] %s", user_text)
        else:
            logger.info("[MSG][user] %s", user_text)
        self._history.append({"role": "user", "content": user_text})
        
        # Increment message counter and check if we need to update context
        self._message_counter += 1
        logger.debug("Message counter: %d (last update: %d)", 
                    self._message_counter, self._last_context_update)
        


        # ------------------------------------------------------------------
        # DEBUG: dump outgoing payload so we can inspect control:test_harness SESSION_END tokens
        payload_preview: dict[str, Any] = {
            "model": self._model_name,
            "messages": self._history + ([{"role": "user", "content": user_text}] if user_text else []),
            "max_tokens": max_tokens,
        }
        logger.debug("[Anthropic PAYLOAD PREVIEW]\n%s", json.dumps(payload_preview, indent=2)[:2000])
        # ------------------------------------------------------------------

        # Build arguments for Anthropic messages.create dynamically so that
        # unit-test doubles that do *not* accept the new "system" parameter
        # still work.  Real clients (>= v1) *do* support it.
        create_fn = self._ac.messages.create  # may be coroutine function
        sig = inspect.signature(create_fn)
        kwargs: dict[str, Any] = {
            "model": self._model_name,
            "messages": self._history,
            "tools": self.tool_schema,
            "max_tokens": max_tokens,
        }
        if "system" in sig.parameters:
            kwargs["system"] = self._system_prompt

        # Respect global rate limit (≤1 request/sec) and add small jitter
        await _respect_rate_limit()
        await asyncio.sleep(random.uniform(0.05, 0.15))

        # ------------------------------------------------------------------
        # Robust retry loop for Anthropic overloads (HTTP 529)
        # Wait 60s on first overload, then exponential back-off capped at 5 min.
        # Give up after 10 minutes total.
        # ------------------------------------------------------------------
        max_deadline = time.monotonic() + 600  # 10-minute overall cap

        # Separate back-off timers for different failure classes
        overload_backoff = 60.0  # HTTP 529 – start with 1 min
        rate_backoff = 60.0      # HTTP 429 – start with 60 s for Claude Sonnet 4

        overload_attempt = 1
        rate_attempt = 1

        while True:
            try:
                resp = await create_fn(**kwargs)
                break  # success

            # ----------------------------------------------------------
            # Anthropic overload (529) – capacity error
            # ----------------------------------------------------------
            except anthropic.InternalServerError as e:
                err_code = getattr(e, "status_code", None)
                err_type = getattr(getattr(e, "error", None), "type", None)
                if err_code == 529 or err_code == 500:
                    now = time.monotonic()
                    if now + overload_backoff > max_deadline:
                        logger.error(
                            "Anthropic overload persisted beyond 10 min – aborting after %d attempts",
                            overload_attempt,
                        )
                        raise
                    logger.warning(
                        "Anthropic overload (HTTP %s) – attempt %d, retrying in %.0f s",
                        err_code,
                        overload_attempt,
                        overload_backoff,
                    )
                    await asyncio.sleep(overload_backoff)
                    overload_backoff = min(overload_backoff * 2, 300)  # cap at 5 min
                    overload_attempt += 1
                    continue
                # Any other InternalServerError → propagate
                raise

            # ----------------------------------------------------------
            # Rate-limit (429) – tokens-per-minute or requests-per-minute
            # ----------------------------------------------------------
            except anthropic.RateLimitError as e:  # type: ignore[attr-defined]
                err_code = getattr(e, "status_code", None)
                # Fallback for old client versions that don’t expose RateLimitError
                if err_code != 429 and err_code is not None:
                    raise  # re-raise unrelated errors

                now = time.monotonic()
                if now + rate_backoff > max_deadline:
                    logger.error(
                        "Anthropic rate-limit (HTTP 429) persisted beyond 10 min – aborting after %d attempts",
                        rate_attempt,
                    )
                    raise
                # Extract granular info when available (Anthropic SDK ≥ 0.19).
                limit_type = None
                # e.error may expose a nested object with limit_type ("tokens" | "requests").
                inner_err = getattr(e, "error", None)
                if inner_err is not None:
                    limit_type = getattr(inner_err, "limit_type", None)
                # Fall back to parsing message tokens.
                if limit_type is None:
                    msg_txt = str(getattr(e, "message", "")) or str(e)
                    if "token" in msg_txt.lower():
                        limit_type = "tokens"
                    elif "request" in msg_txt.lower():
                        limit_type = "requests"

                retry_after_hdr = None
                if hasattr(e, "headers") and isinstance(e.headers, dict):
                    retry_after_hdr = e.headers.get("retry-after") or e.headers.get("Retry-After")

                logger.warning(
                    "Anthropic rate-limit (HTTP 429, type=%s, retry-after=%s) – attempt %d, retrying in %.0f s",
                    limit_type or "unknown",
                    retry_after_hdr or "n/a",
                    rate_attempt,
                    rate_backoff,
                )
                await asyncio.sleep(rate_backoff)
                rate_backoff = min(rate_backoff * 2, 600)  # cap at 10 min
                rate_attempt += 1
                continue

            # ----------------------------------------------------------
            # Other API errors – propagate immediately
            # ----------------------------------------------------------
            except Exception:
                raise

        # ----------------------------------------------------------
        # Handle stop_reason values per Anthropic best-practice
        # ----------------------------------------------------------
        content_blocks: List[Any] = list(resp.content)

        while getattr(resp, "stop_reason", None) == "pause_turn":
            logger.info("pause_turn encountered – requesting additional tokens …")
            # Increase token budget conservatively (512) – caller can still set higher
            kwargs["max_tokens"] = kwargs.get("max_tokens", max_tokens) + 512
            # Respect rate-limit before retrying the **identical** prompt
            await _respect_rate_limit()
            await asyncio.sleep(random.uniform(0.05, 0.15))
            resp = await create_fn(**kwargs)
            content_blocks.extend(resp.content)

        final_stop_reason = getattr(resp, "stop_reason", None)
        logger.info("LLM stop_reason: %s", final_stop_reason)

        if final_stop_reason == "max_tokens":
            logger.warning(
                "Model hit max_tokens (%d). Consider increasing the limit or issuing a follow-up request.",
                kwargs.get("max_tokens", max_tokens),
            )

        logger.debug("LLM responded with %d blocks", len(content_blocks))

        tool_calls: List[Dict[str, Any]] = []
        reply_text: str | None = None
        tool_use_blocks: List[anthropic.types.MessageBlock] = []

        for idx, block in enumerate(content_blocks):
            block_type = getattr(block, "type", None)
            # ------------------------------------------------------------------
            # Handle both official Anthropic MessageBlock objects (have .type)
            # *and* lightweight test doubles that only expose `.text` and
            # `.tool_calls` attributes.
            # ------------------------------------------------------------------
            if block_type is None:
                # Legacy / fake block used in unit tests
                reply_text = getattr(block, "text", "").strip()
                logger.debug("Assistant legacy text block %d: %s", idx, reply_text[:120])
                tool_calls.extend(getattr(block, "tool_calls", []))
                continue

            if block_type == "text":
                # Visible assistant reply
                reply_text = getattr(block, "text", "").strip()
                logger.debug("Assistant text block %d: %s", idx, reply_text[:120])
                # INFO-level log so harness can capture assistant turns uniformly.
                logger.info("[MSG][assistant] %s", reply_text)
            elif block_type == "tool_use":
                # Track original tool_use block for history reconstruction
                tool_use_blocks.append(block)

                # Collect tool call details in legacy dict shape expected by dispatcher
                tool_calls.append({
                    "name": block.name,
                    "arguments": block.input or {},
                })
                logger.debug("Tool use block %d: %s", idx, tool_calls[-1])

        # Ensure we have some reply text (fallback)
        reply_text = reply_text or ""

        # Persist assistant visible text only if non-empty; empty content causes API 400 on next call
        if reply_text:
            # Already logged above; no duplicate INFO log needed.
            self._history.append({"role": "assistant", "content": reply_text})

        # Execute any tool calls
        for tc in tool_calls:
            logger.info("Tool call emitted: %s %s", tc.get("name"), tc.get("arguments"))

        if tool_calls:
            # Clear previous cache
            self._tool_results.clear()

            await self._dispatch_tool_calls(tool_calls)
            
            # Check if we should update context


            # ------------------------------------------------------------------
            # After successful execution, append tool_use and tool result blocks to history
            # so Claude sees that the call was made and completed.
            # ------------------------------------------------------------------
            # Reflect each original tool_use and its result per Anthropic protocol
            for tu in tool_use_blocks:
                # 1. Echo the tool_use block inside an assistant message
                self._history.append({
                    "role": "assistant",
                    "content": [{
                        "type": "tool_use",
                        "id": getattr(tu, "id", None),
                        "name": tu.name,
                        "input": tu.input or {},
                    }],
                })

                # 2. Provide corresponding tool_result block so Claude knows execution succeeded
                # Build rich result so the model can reference IDs or data.
                # Default status OK; override for enqueue-style tools.
                result_payload: dict[str, any] = {}

                if tc.get("name") in ("add_entry", "put_context"):
                    result_payload["status"] = "enqueued"
                else:
                    result_payload["status"] = "OK"

                if tc.get("name") == "create_memory":
                    result_payload["memory_id"] = self._mem_aliases.get(tc.get("arguments").get("title"), "")
                elif tc.get("name") == "get_context":
                    result_payload["context"] = self._tool_results.get("get_context", "")
                elif tc.get("name") == "list_entries":
                    result_payload["entries"] = self._tool_results.get("list_entries", [])
                elif tc.get("name") == "search_memories":
                    result_payload.update(self._tool_results.get("search_memories", {}))

                self._history.append({
                    "role": "user",
                    "content": [{
                        "type": "tool_result",
                        "tool_use_id": getattr(tu, "id", None),
                        "content": json.dumps(result_payload),
                    }],
                })

        return reply_text

    async def close_session(self, *, request_explanation: bool = True) -> str:
        """Finalize session by instructing assistant to flush entries and persist context.

        New simplified protocol:
        1. Send `control:test_harness SESSION_END` followed by a clear instruction:
           • Drain any remaining messages via `add_entry` (per entry_capture_prompt).
           • Then issue exactly **one** `put_context` call.
        2. Allow up to 10 assistant turns.  After each turn, exit early once
           `put_context` is detected (tracked by `_last_context_update`).
        3. If budget exhausted without `put_context`, optionally ask for an
           explanation and raise `RuntimeError`.
        """
        MAX_TURNS = 10

        # First attempt includes explicit shutdown instructions so the model
        # understands what actions are required. Subsequent retries only send
        # the sentinel token as a nudge.

        full_shutdown_instruction = (
            f"{END_SESSION_TOKEN}\n\n"
            "You are closing the session. Perform these steps in order:\n"
            "1. Persist any remaining dialogue via add_entry (follow entry_capture rules).\n"
            "2. Call await_consistency().\n"
            "3. Issue exactly **one** put_context() call whose `content`:\n"
            "   • Is wrapped in triple back-ticks (``` ```).\n"
            "   • Adheres to the section ordering defined in @context_prompt.md (asset `ctx_prompt_chat`).\n"
            "   • Is ≤ 5000 characters.\n"
            "After put_context succeeds, reply with `control:note_taker_assistant OK`."
        )

        attempt = 1
        prompt = full_shutdown_instruction

        while attempt <= MAX_TURNS:
            logger.info("Close-session turn %d/%d", attempt, MAX_TURNS)
            reply = await self.step(prompt)

            if self._last_context_update == self._message_counter:
                logger.info("put_context captured – session closed cleanly")
                return reply

            # Send reminder every 3 turns to avoid getting stuck
            if attempt % 3 == 0:
                prompt = (
                    f"{END_SESSION_TOKEN}\n\n"
                    "REMINDER: If you don't have any more entries to save, "
                    "call put_context() now to close the session."
                )
            else:
                # Next turns: send sentinel only as gentle nudge
                prompt = END_SESSION_TOKEN
            attempt += 1

        # Still no put_context – explanation & failure
        explanation: str | None = None
        if request_explanation:
            try:
                logger.warning("No put_context after %d attempts – requesting explanation …", MAX_TURNS)
                explanation = await self.step(
                    "You have not issued the required `put_context` call. Please explain why."
                )
            except Exception as e:
                logger.error("Failed to obtain final explanation: %s", e)

        raise RuntimeError(
            "Session ended without put_context after simplified close-session protocol. "
            + (f"Assistant explanation: {explanation}" if explanation else "")
        )

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------
    async def _dispatch_tool_calls(self, calls: List[Dict[str, Any]]) -> None:
        for call in calls:
            name: str = call.get("name")
            args: Dict[str, Any] = call.get("arguments", {})
            logger.debug("Dispatching tool call: %s with args: %s", name, args)

            try:
                if name == "add_entry":
                    await self._exec_add_entry(args)
                elif name == "put_context":
                    await self._exec_put_context(args)
                elif name == "list_entries":
                    await self._exec_list_entries(args)
                elif name == "search_memories":
                    await self._exec_search_memories(args)
                elif name == "create_memory":
                    await self._exec_create_memory(args)
                elif name == "get_context":
                    await self._exec_get_context(args)
                elif name == "get_user":
                    await self._exec_get_user(args)
                elif name == "get_memory":
                    await self._exec_get_memory(args)
                elif name == "await_consistency":
                    await self._exec_await_consistency(args)
                elif name == "list_assets":
                    await self._exec_list_assets(args)
                elif name == "get_asset":
                    await self._exec_get_asset(args)
                # Ignore unknown tool names to stay forward-compatible.
            except Exception as e:
                logger.error("Error in tool call %s: %s", name, str(e), exc_info=True)
                raise

    async def _exec_add_entry(self, args: Dict[str, Any]) -> None:
        # Validate mandatory fields per updated tool schema
        if "summary" not in args or not args["summary"]:
            # Intercept missing-summary calls: ask assistant to explain which rules it followed.
            logger.warning("add_entry missing summary – querying assistant for explanation")
            try:
                expl = await self.step(
                    "You attempted to call `add_entry` without the required `summary` field. "
                    "According to the tool schema (`add_entry.summary` is mandatory) and the project rules, "
                    "each stored entry must include a ≤512-char summary. "
                    "Which instructions or rules did you rely on when constructing this incomplete tool call?"
                )
                logger.error("Assistant explanation for missing summary: %s", expl)
            except Exception as e:
                logger.error("Failed to obtain explanation for missing summary: %s", e)
            raise ValueError("add_entry tool call missing mandatory 'summary' field")
        if "raw_entry" not in args or not args["raw_entry"]:
            raise ValueError("add_entry tool call missing 'raw_entry' field")

        role_value = args.get("role")
        tags_dict = args.get("tags") or {}
        if not role_value and isinstance(tags_dict, dict):
            role_value = tags_dict.pop("role", None)
 
        if not role_value:
            # Attempt to infer from last user prefix
            for h in reversed(self._history):
                if h.get("role") == "user":
                    txt = str(h.get("content", ""))
                    if txt.startswith("benchmark_conversation:"):
                        role_value = txt.split(" ", 1)[0].split(":", 1)[1]  # after colon
                        break
            if not role_value:
                raise ValueError("add_entry tool call missing 'role' field")

        # Normalize legacy 'speaker 1'/'speaker 2' → 'speaker_1'/'speaker_2'
        if role_value in {"speaker 1", "speaker 2"}:
            role_value = role_value.replace(" ", "_")
            logger.info("Normalised legacy role tag to '%s'", role_value)

        # Validate role string to avoid drift
        allowed_roles = {"speaker_1", "speaker_2"}
        if role_value not in allowed_roles:
            logger.warning("Unexpected role '%s' in add_entry; requesting retry with canonical role", role_value)
            try:
                await self.step(
                    f"`add_entry.tags.role` must be one of {sorted(allowed_roles)}. "
                    f"You sent '{role_value}'. Please resend the tool call with the canonical role value."
                )
            except Exception:
                pass
            raise ValueError("add_entry received non-canonical role tag")

        # Determine target memory_id (accept candidate or stored)
        memory_id = self._resolve_memory_id(args.get("memory_id"))

        # Ensure 'role' is also included inside tags for backend compatibility.
        tags_dict = tags_dict or {}
        tags_dict["role"] = role_value

        self._sc.add_entry(
            memory_id,
            args["raw_entry"],
            args["summary"],
            role=role_value,
            tags=tags_dict,
        )

        # Detect bootstrap violation: add_entry before list_entries(limit=10)
        if (
            self._boot_seen_get_context
            and not self._boot_seen_list_entries
            and not self._boot_violation_asked
        ):
            self._boot_violation_asked = True
            logger.warning(
                "Assistant issued add_entry before list_entries(limit=10). Querying explanation …"
            )
            try:
                explanation = await self.step(
                    "As part of the session bootstrap you must call `list_entries(limit=10)` before storing new entries. "
                    "You have just stored an entry without listing recent entries first. Why did you skip the required step?"
                )
                logger.error("Assistant bootstrap violation explanation: %s", explanation)
            except Exception as e:
                logger.error("Failed to obtain explanation for bootstrap violation: %s", e)

    async def _exec_list_entries(self, args: Dict[str, Any]) -> None:
        """Handle list_entries tool call (read-only)."""
        memory_id = self._resolve_memory_id(args.get("memory_id"))
        limit = int(args.get("limit", 10))
        # Track bootstrap compliance
        if limit == 10:
            self._boot_seen_list_entries = True

        # Guard: the rules mandate limit=10 on bootstrap. If we see 5, treat as violation.
        if limit == 5:
            logger.error("Rule violation: assistant called list_entries with limit=5 (expected 10). Requesting explanation …")
            try:
                explanation = await self.step(
                    "You just issued `list_entries` with `limit = 5`, but the project rules specify `limit = 10` on session bootstrap. Why did you deviate from the rule?"
                )
                logger.error("Assistant explanation: %s", explanation)
                try:
                    follow = await self.step(
                        "What additional information or clarity would help you follow the bootstrap rule (`list_entries(limit=10)` before any `add_entry`) in future sessions?"
                    )
                    logger.error("Assistant compliance-help reply: %s", follow)
                except Exception as e:
                    logger.error("Failed to obtain compliance-help reply: %s", e)
            except Exception as e:
                logger.error("Failed to obtain explanation from assistant: %s", e)
            raise RuntimeError("Assistant violated bootstrap rule by using list_entries(limit=5)")

        entries = self._sc.list_entries(memory_id, limit=limit)
        self._tool_results["list_entries"] = entries

    async def _exec_search_memories(self, args: Dict[str, Any]) -> None:
        """Handle search_memories tool call (read-only)."""
        memory_id = self._resolve_memory_id(args.get("memory_id"))
        query = args["query"]
        top_k = int(args.get("top_k", 5))
        res = self._sc.search_memories(memory_id, query=query, top_k=top_k)
        self._tool_results["search_memories"] = res

    async def _exec_create_memory(self, args: Dict[str, Any]) -> None:
        """Handle create_memory tool call and update internal memory_id."""
        title = args["title"]
        memory_type = args.get("memory_type", "conversation")
        description = args.get("description", "")
        mem_id = self._sc.create_memory(title, memory_type=memory_type, description=description)
        # Small delay to allow backend replicas to serve the default context row
        await asyncio.sleep(0.3)
        # Store mapping so later calls using the title are resolved
        self._mem_aliases[title] = mem_id
        setattr(self._system_builder, "memory_id", mem_id)

    async def _exec_get_context(self, args: Dict[str, Any]) -> None:
        memory_id = self._resolve_memory_id(args.get("memory_id"))
        logger.info("Fetching context for memory %s", memory_id)

        try:
            ctx = self._sc.get_context(memory_id)
            logger.debug("Context retrieved (%d chars)", len(ctx) if ctx else 0)
            # Echo the retrieved context so callers can verify backend default
            if ctx:
                # Attempt to pretty-print if the content is valid JSON.
                pretty_ctx = ctx
                try:
                    parsed = json.loads(ctx)
                    pretty_ctx = json.dumps(parsed, indent=2, ensure_ascii=False)
                except json.JSONDecodeError:
                    # Leave as-is if not JSON.
                    pass
                print(f"[get_context] Retrieved (pretty):\n{pretty_ctx}\n")
            else:
                print("[get_context] (empty)")

            # Cache for tool_result
            self._tool_results["get_context"] = ctx
            # Mark bootstrap sequence step
            self._boot_seen_get_context = True
        except Exception as e:
            logger.error("Failed to fetch context for %s: %s", memory_id, e, exc_info=True)
            raise

    async def _exec_get_user(self, args: Dict[str, Any]) -> None:
        """No-op: user profile fetched out-of-band for now."""
        return

    # ... (rest of the code remains the same)

    async def _exec_get_memory(self, args: Dict[str, Any]) -> None:
        """Placeholder until SDK exposes get_memory; ignored."""
        return

    async def _exec_await_consistency(self, args: Dict[str, Any]) -> None:
        """Approximate eventual consistency with small sleep."""
        # Delegate to client helper which uses CLI barrier when available
        memory_id = self._resolve_memory_id(args.get("memory_id")) if args else self._resolve_memory_id()
        try:
            self._sc.await_consistency(memory_id)
        except Exception:
            # Fallback: short sleep in case of client error
            await asyncio.sleep(0.5)

    # ------------------------------------------------------------------
    # Asset tool helpers
    # ------------------------------------------------------------------
    async def _exec_list_assets(self, args: Dict[str, Any]) -> None:
        """Return list of available asset IDs (read-only)."""
        ids = self._sc.list_assets()
        self._tool_results["list_assets"] = ids

    async def _exec_get_asset(self, args: Dict[str, Any]) -> None:
        """Fetch static asset text, respecting session cache to avoid spam."""
        asset_id = args.get("id") or args.get("asset_id")  # tolerate either arg name
        if not asset_id:
            raise ValueError("get_asset requires 'id' argument")

        if asset_id in self._asset_cache:
            logger.debug("Returning cached asset %s", asset_id)
            self._tool_results["get_asset"] = self._asset_cache[asset_id]
            return

        txt = self._sc.get_asset(asset_id)
        if not txt:
            raise RuntimeError(f"get_asset failed or returned empty for id={asset_id}")
        # Cache and expose via tool_results
        self._asset_cache[asset_id] = txt
        self._tool_results["get_asset"] = txt

        # Track required asset downloads for bootstrap gate
        if asset_id == "ctx_rules":
            self._boot_seen_ctx_rules = True

        self._boot_assets_downloaded.add(asset_id)

    # ------------------------------------------------------------------
    # Legacy helper methods kept for unit-test compatibility
    # ------------------------------------------------------------------
    def _get_current_context(self, memory_id: str) -> Dict[str, Any]:
        """Return current context dict; never raises."""
        try:
            ctx = self._sc.get_context(memory_id)
            if isinstance(ctx, dict):
                return ctx
            logger.warning("Context for %s is not a dict – returning empty", memory_id)
            return {}
        except Exception as e:
            logger.warning("Failed to get current context: %s", e)
            return {}

    def _prepare_updated_context(self, current_ctx: Dict[str, Any]) -> Dict[str, Any]:
        """Merge bookkeeping fields into context for saving."""
        ts = datetime.now(timezone.utc).isoformat()
        updated = dict(current_ctx)
        updated.update(
            last_updated=ts,
            message_count=self._message_counter,
            history_summary=f"{len(self._history)} messages in history",
            last_updated_by="SessionSimulator._update_context",
        )
        return cast(Dict[str, Any], updated)

    def _save_context(self, memory_id: str, context: Dict[str, Any]) -> None:
        """Persist context JSON; bubbles up exceptions for tests."""
        try:
            self._sc.put_context(memory_id, json.dumps(context))
        except Exception as e:
            logger.error("Failed to save context: %s", e)
            raise

    # ------------------------------------------------------------------
    # Context helpers (kept minimal – explicit put_context by model)
    # ------------------------------------------------------------------

    # ------------------------------------------------------------------
    async def _exec_put_context(self, args: Dict[str, Any]) -> None:
        """Handle put_context tool call with additional logging."""
        content = args.get("content") or "{}"  # CLI requires non-empty
        memory_id = self._resolve_memory_id(args.get("memory_id"))

        logger.debug("Updating context for memory %s (%d chars)", memory_id, len(content))

        # Persist context
        self._sc.put_context(memory_id, content)
        self._last_context_update = self._message_counter  # track

        # ---------------------- bootstrap compliance ----------------------
        if not self._boot_seen_ctx_rules and not self._boot_ctx_violation_asked:
            self._boot_ctx_violation_asked = True
            logger.warning("Assistant called put_context before loading ctx_rules. Requesting explanation …")
            try:
                explanation = await self.step(
                    "You must load @context_summary_rules.md via get_asset(\"ctx_rules\") before put_context. Why did you skip?"
                )
                logger.error("Assistant ctx_rules bootstrap violation explanation: %s", explanation)
            except Exception as e:
                logger.error("Failed to obtain explanation for ctx_rules bootstrap violation: %s", e)

        # ---------------------- empty-context reasoning -------------------
        if (
            content.strip() in ("{}", "")
            and self._message_counter > 5
            and not getattr(self, "_asked_empty_ctx_reason", False)
        ):
            setattr(self, "_asked_empty_ctx_reason", True)
            logger.warning("Assistant persisted empty context; requesting reasoning …")
            try:
                # First, ask the assistant to recap the conversation so far.
                summary = await self.step(
                    "Please provide a concise summary (≤200 words) of the conversation so far between speaker 1 and speaker 2."
                )
                logger.info("Assistant conversation summary: %s", summary)

                # Then, ask why this summary (or any context) was not persisted.
                reason = await self.step(
                    "Given the recap you just provided, why did you persist an **empty** context document instead of a context that includes that information?"
                )
                logger.info("Assistant reasoning for empty context: %s", reason)
            except Exception as e:
                logger.error("Failed to obtain summary/reasoning for empty context: %s", e)


# ----------------------------------------------------------------------
# Convenience helper for synchronous testing
# ----------------------------------------------------------------------

def run_sync(coro):  # pragma: no cover – helper for notebooks
    return asyncio.get_event_loop().run_until_complete(coro) 
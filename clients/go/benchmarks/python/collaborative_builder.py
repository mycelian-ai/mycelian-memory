from __future__ import annotations

import logging
from typing import Dict, Any, List

import anthropic

from .models import BenchmarkConversation, ConversationMessage
from .synapse_client import SynapseMemoryClient
from .system_prompt_builder import PromptAssembler
# Message annotation prefixes (see docs/design/msc_dataset_note_taker.md)
CONTROL_PREFIX = "control:test_harness"
SPEAKER1_PREFIX = "benchmark_conversation:speaker_1"
SPEAKER2_PREFIX = "benchmark_conversation:speaker_2"

from .session_simulator import SessionSimulator

logger = logging.getLogger(__name__)

# ------------------------------------------------------------------
# One-time reminder to enforce bootstrap sequence (get_context → list_entries(10))
# Sent with no prior tool calls emitted.
# ------------------------------------------------------------------
SESSION_BOOTSTRAP_PROMPT = (
    f"{CONTROL_PREFIX} SESSION_BOOTSTRAP\n\n"
    "### Startup sequence (MUST follow in order)\n"
    "1. Fetch static assets – call get_asset() once for each id **in this order**:\n"
    "   • ctx_rules\n"
    "   • ctx_prompt_chat\n"
    "   • entry_prompt_chat\n"
    "   • summary_prompt_chat\n"
    "2. get_context() – retrieve stored document.\n"
    "3. list_entries(limit = 10) – merge recent facts.\n\n"
    "### Harness command semantics\n"
    "• Any message whose first token starts with `control:` is a harness command.\n"
    "  It MUST NOT be persisted with add_entry; treat it as out-of-band instruction.\n\n"
    "### Session termination sentinel\n"
    "When you receive the message `control:test_harness SESSION_END` you MUST:\n"
    "  1. Persist any remaining raw dialogue messages via add_entry (follow entry_capture rules).\n"
    "  2. Call await_consistency() to ensure all previous writes are durable.\n"
    "  3. Issue exactly **one** put_context() call containing the updated context, formatted per ctx_prompt_chat (≤ 5000 chars).\n"
    "After completing these steps, reply with `control:note_taker_assistant OK`.\n\n"
    "These instructions apply only to this benchmark; they override more generic prompt guidance."
)


class ConversationIngestor:
    """Build Synapse memory by replaying a conversation via SessionSimulator."""

    def __init__(
        self,
        anthropic_client: anthropic.Anthropic,
        memory_client: SynapseMemoryClient,
        *,
        model_name: str = "claude-3-haiku-20240307",
    ):
        self._ac = anthropic_client
        self._mc = memory_client
        self._model_name = model_name

    # ------------------------------------------------------------------
    async def process_conversation(
        self,
        conv: BenchmarkConversation,
        benchmark_name: str = "DMR",
        max_messages_per_session: int | None = None,
    ) -> str:
        """Create memory, replay conversation sessions, return memory_id."""
        memory_id = self._mc.create_memory(
            title=f"{benchmark_name} Conversation {conv.conversation_id}",
            description="Benchmark ingestion conversation",
        )
        logger.info("Created memory %s for conv %s", memory_id, conv.conversation_id)

        # Group by session_id
        by_session: Dict[int, List[ConversationMessage]] = {}
        for m in conv.messages:
            by_session.setdefault(m.session_id, []).append(m)

        # Replay each session in order
        for session_id in sorted(by_session):
            logger.info("Session %s with %d turns", session_id, len(by_session[session_id]))

            context_doc = self._mc.get_context(memory_id)
            sys_builder = PromptAssembler(
                benchmark_name=benchmark_name,
                user_id=self._mc.user_id,
                memory_id=memory_id,
                context_doc=context_doc,
                recent_entries=[],
            )

            simulator = SessionSimulator(self._ac, self._mc, sys_builder, model_name=self._model_name)

            # ------------------------------------------------------------------
            # 1. Bootstrap – enforce explicit sequence reminder
            # ------------------------------------------------------------------
            await simulator.step(SESSION_BOOTSTRAP_PROMPT)

            # Wait until the assistant has actually executed get_context() and
            # list_entries(limit = 10) to avoid racing the first dataset turn.
            MAX_WAIT_TURNS = 6
            turns_waited = 0
            while not simulator.is_bootstrap_complete and turns_waited < MAX_WAIT_TURNS:
                # Mark keep-alive polling messages as control signals so the
                # assistant ignores them for entry persistence.
                await simulator.step(f"{CONTROL_PREFIX} POLL")
                turns_waited += 1

            if not simulator.is_bootstrap_complete:
                logger.error("Assistant did not complete bootstrap within %d turns", MAX_WAIT_TURNS)
                raise RuntimeError("Bootstrap sequence not completed – ingestion aborted")

            # ------------------------------------------------------------------
            # 2. Replay conversation turns (both user and assistant)
            # ------------------------------------------------------------------
            # Optionally truncate messages for fast test mode
            session_msgs = by_session[session_id]
            if max_messages_per_session is not None:
                session_msgs = session_msgs[:max_messages_per_session]

            for msg in session_msgs:
                if msg.speaker == "user":  # speaker 1
                    await simulator.step(f"{SPEAKER1_PREFIX} {msg.text}")
                else:  # speaker 2
                    await simulator.step(f"{SPEAKER2_PREFIX} {msg.text}")

            # ------------------------------------------------------------------
            # 3. Close session to trigger put_context flush
            # ------------------------------------------------------------------
            await simulator.close_session()

            # TODO: we could verify context was put, but leave for future

        return memory_id 
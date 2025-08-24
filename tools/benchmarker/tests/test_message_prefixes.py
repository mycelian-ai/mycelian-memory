import pytest

from benchmarks.python.session_simulator import SessionSimulator
from benchmarks.python.system_prompt_builder import PromptAssembler
from benchmarks.python.mycelian_client import MycelianMemoryClient

# ---------------------------------------------------------------------------
# Fake Anthropic client that always emits an add_entry tool call for any
# benchmark_conversation:* message and no tool calls for control:* messages.
# ---------------------------------------------------------------------------
class _FakeAnthropicMessageBlock:
    """Lightweight stand-in for Anthropic MessageBlock."""

    def __init__(self, text: str, tool_calls=None):
        self.type = None  # legacy / test path (SessionSimulator checks .type is None)
        self.text = text
        self.tool_calls = tool_calls or []


class _FakeAnthropic:
    class _Messages:
        async def create(self, *, model, messages, tools, max_tokens):  # noqa: D401
            last_user = messages[-1]["content"]
            tool_calls = []
            # Only emit add_entry tool call for benchmark messages
            if last_user.startswith("benchmark_conversation:"):
                tool_calls.append(
                    {
                        "name": "add_entry",
                        "arguments": {
                            "memory_id": "mem-1",
                            "raw_entry": last_user,
                            "summary": "summary",
                            "role": "speaker_1" if "speaker_1" in last_user else "speaker_2",
                        },
                    }
                )
            return type(
                "Resp",
                (),
                {"content": [_FakeAnthropicMessageBlock("OK", tool_calls)]},
            )()

    def __init__(self):
        self.messages = self._Messages()


class _FakeMycelian(MycelianMemoryClient := object):  # simple duck-type
    def __init__(self):
        self.added = []
        self.user_id = "user-1"

    def add_entry(self, memory_id, raw_entry, summary, *, role=None, tags=None):
        self.added.append((memory_id, raw_entry, summary))

    # Unused interface methods
    def get_context(self, m):
        return ""

    def list_entries(self, m, limit=10):
        return []

    def put_context(self, memory_id, content):
        pass


@pytest.mark.asyncio
async def test_message_annotation_prefix_handling():
    fake_ac = _FakeAnthropic()
    fake_sc = _FakeMycelian()
    spb = PromptAssembler("MSC", fake_sc.user_id, "mem-1", "", [])
    sim = SessionSimulator(fake_ac, fake_sc, spb)

    # 1. Control message – should NOT trigger add_entry
    await sim.step("control:test_harness SESSION_BOOTSTRAP")
    assert not fake_sc.added

    # 2. Speaker 1 message – should be persisted with role user (we just assert call happened)
    await sim.step("benchmark_conversation:speaker_1 Hello")
    assert len(fake_sc.added) == 1
    assert fake_sc.added[0][1].startswith("benchmark_conversation:speaker_1")

    # 3. Speaker 2 message – should be persisted with role assistant
    await sim.step("benchmark_conversation:speaker_2 Hi there")
    assert len(fake_sc.added) == 2
    assert fake_sc.added[1][1].startswith("benchmark_conversation:speaker_2")

    # 4. Another control message – still no new entries
    await sim.step("control:test_harness POLL")
    assert len(fake_sc.added) == 2 
import pytest

from benchmarks.python.session_simulator import SessionSimulator, END_SESSION_TOKEN
from benchmarks.python.system_prompt_builder import PromptAssembler


class FakeAnthropicMessageBlock:
    def __init__(self, text, tool_calls=None):
        self.text = text
        self.tool_calls = tool_calls or []


class FakeAnthropic:
    class _Messages:
        async def create(self, *, model, messages, tools, max_tokens):
            # Always return a single assistant reply with one add_entry tool call and, if END_SESSION_TOKEN in user, a put_context.
            last_user = messages[-1]["content"]
            tool_calls = []
            if "hello" in last_user:
                tool_calls.append({
                    "name": "add_entry",
                    "arguments": {
                        "memory_id": "mem-1",
                        "raw_entry": last_user,
                        "summary": "user said hello",
                        "role": "speaker_1",
                    },
                })
            if END_SESSION_TOKEN in last_user:
                tool_calls.append({
                    "name": "put_context",
                    "arguments": {"memory_id": "mem-1", "content": "ctx"},
                })
            return type(
                "Resp",
                (),
                {"content": [FakeAnthropicMessageBlock("assistant reply", tool_calls)]},
            )()

    def __init__(self):
        self.messages = self._Messages()


class FakeMycelian(MycelianMemoryClient := object):  # simple duck type
    def __init__(self):
        self.added = []
        self.context_puts = []
        self.user_id = "user-1"

    def add_entry(self, memory_id, raw_entry, summary, *, role=None, tags=None):
        self.added.append((memory_id, raw_entry, summary))

    def put_context(self, memory_id, content):
        self.context_puts.append((memory_id, content))

    # unused methods
    def get_context(self, m):
        return ""

    def list_recent_entries(self, m, limit=5):
        return []


@pytest.mark.asyncio
async def test_session_simulator_executes_tool_calls():
    fake_ac = FakeAnthropic()
    fake_sc = FakeMycelian()
    spb = PromptAssembler("DMR", fake_sc.user_id, "mem-1", "", [])
    sim = SessionSimulator(fake_ac, fake_sc, spb)

    await sim.step("hello there")
    await sim.close_session()

    # Verify tool calls executed
    assert fake_sc.added, "add_entry should be called"
    assert fake_sc.context_puts, "put_context should be called on END_SESSION" 
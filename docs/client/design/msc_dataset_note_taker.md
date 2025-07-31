# MSC-Self-Instruct Conversation Format and Note-Taker Semantics

Status: draft  
Last updated: 2025-07-07  
Author: Benchmark Team

---

## 1.  Why this document?

Recent debugging sessions revealed confusion around the *role* of the benchmark agent versus the two speakers encoded in the MemGPT **MSC-Self-Instruct** dataset.  This note records what we learned and specifies how **SessionSimulator** and related tests should treat the speaker labels going forward.

---

## 2.  Dataset recap

* **Source**   MemGPT / *MSC-Self-Instruct* (`MemGPT_MSC-Self-Instruct_train.json`).
* **Structure**  Each top-level record represents **one conversation** that can span multiple “sessions”.
* **Fields of interest**
  * `previous_dialogs` – array of session objects.
    * `dialog` – ordered list of message objects
      * `id` – string, usually `"speaker 1"` or `"speaker 2"`
      * `text` – utterance text
  * `self_instruct` – cross-session QA pair used by evaluation harness.

### 2.1  Synthetic origin
The dialogues are *not* real user⇄assistant chats.  They were generated offline to probe multi-session memory.  Consequently neither speaker is “the AI”.

---

## 3.  Mapping in `msc_loader.py`

```python
if raw_id == "speaker 1":
    speaker = "user"
elif raw_id == "speaker 2":
    speaker = "assistant"
else:
    # fallback: alternate turns starting with user
    speaker = "user" if idx % 2 == 0 else "assistant"
```

Highlights:

* The mapping is **purely conventional**—it satisfies the re-player which expects exactly two roles (`user` and `assistant`).
* The loader does **not** imply that `speaker 2` was produced by an LLM.

---

## 4.  Benchmark-agent’s role

The agent we run today (Claude, GPT-4, etc.) **is not either speaker**.  It acts as a *neutral note-taker* whose job is to

1. Re-inject the historical utterances into Synapse via `add_entry` (using `role = user/assistant` according to the mapping above).
2. Generate a ≤ 512-char summary for each entry (see `summary_prompt_chat`).
3. Periodically update the session context (see `context_prompt_chat`).

We therefore issue control messages:

* `<ASSISTANT_REPLAY>\n<text>` for historical assistant turns
* Plain text for historical user turns

and require the note-taker to reply only with `add_entry` tool calls followed by `OK`.

---

## 5.  Test implications

* **Unit tests** should keep using the `speaker 1` → `user`, `speaker 2` → `assistant` mapping so that assertions on stored roles remain stable.
* **Integration harness** sends **both** roles to the agent; the agent *never* fabricates new dialogue.
* The bootstrap logic (`get_context` → `list_entries` → asset fetches…) remains unchanged.

---

## 6.  FAQ

**Q – Does the agent ever speak as the assistant?**  
No.  It only stores historical text.  Its visible reply to the test harness is always `OK`.

**Q – Why keep the `assistant` role at all?**  
Because Synapse’s memory schema distinguishes user and assistant messages.  This helps downstream search and evaluation.

**Q – Will this change token usage?**  
No; we already replay both sides.  This document only clarifies semantics and test conventions.

---

## 7.  Next steps

* Update README of benchmark directory to link to this note.
* Ensure future datasets document speaker semantics explicitly. 

---

## 3.5  Unified message annotation – `TYPE:ROLE`

To remove ad-hoc sentinel tokens and make every message self-describing, **all strings exchanged between the test harness and the note-taker MUST start with one of the following prefixes**:

| TYPE | ROLE | Example prefix | Persisted? |
|------|------|----------------|------------|
| `control` | `test_harness` | `control:test_harness` | No |
| `control` | `note_taker_assistant` | `control:note_taker_assistant` | No |
| `benchmark_conversation` | `speaker 1` | `benchmark_conversation:speaker 1` | Yes – store as `user` |
| `benchmark_conversation` | `speaker 2` | `benchmark_conversation:speaker 2` | Yes – store as `assistant` |

### Rules
1. **Exactly one prefix** per message, followed by a single space, then the free-form body.
2. Messages with a `control:*` prefix **must never be persisted** via `add_entry`.
3. The note-taker must respond to every successfully persisted message with
   `control:note_taker_assistant OK` (no extra text).
4. The old sentinel tokens (`<SESSION_BOOTSTRAP>`, `<ASSISTANT_REPLAY>`, `<END_SESSION>`) are **deprecated** and will be removed once the new format ships.

### Minimal bootstrap sequence (example)
```
control:test_harness SESSION_BOOTSTRAP – fetch assets, get_context, list_entries
control:test_harness POLL – keep-alive while waiting for bootstrap completion
benchmark_conversation:speaker 1 Hi there!
control:note_taker_assistant OK
benchmark_conversation:speaker 2 Hello, how can I help?
control:note_taker_assistant OK
control:test_harness SESSION_END – flush remaining entries & call put_context
```

The table and example above supersede the temporary `<ASSISTANT_REPLAY>` mechanism described in earlier drafts.

--- 
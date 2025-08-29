from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol, List, Dict, Any, Tuple, Optional


class Store(Protocol):
    def ensure_vault(self, title: Optional[str], vault_id: Optional[str]) -> str: ...
    def ensure_memory(self, vault_id: str, title: str, memory_type: str = "NOTES") -> str: ...
    def add_entry(self, vault_id: str, memory_id: str, raw: str, summary: str) -> None: ...
    def put_context(self, vault_id: str, memory_id: str, content: str) -> None: ...
    def await_consistency(self, memory_id: str) -> None: ...


@dataclass
class IngestStats:
    questions: int = 0
    sessions: int = 0
    user_messages: int = 0
    entries_written: int = 0
    contexts_written: int = 0


class InMemoryStore:
    def __init__(self) -> None:
        self.vault_id = "vault-1"
        self.created_memories: List[Tuple[str, str]] = []  # (vault_id, title)
        self.entries: List[Tuple[str, str, str]] = []  # (memory_id, role, raw)
        self.contexts: List[Tuple[str, str]] = []  # (memory_id, content)

    def ensure_vault(self, title: Optional[str], vault_id: Optional[str]) -> str:
        return self.vault_id if not vault_id else vault_id

    def ensure_memory(self, vault_id: str, title: str, memory_type: str = "NOTES") -> str:
        mem_id = f"mem-{len(self.created_memories)+1}"
        self.created_memories.append((vault_id, title))
        return mem_id

    def add_entry(self, vault_id: str, memory_id: str, raw: str, summary: str) -> None:
        self.entries.append((memory_id, "user", raw))

    def put_context(self, vault_id: str, memory_id: str, content: str) -> None:
        self.contexts.append((memory_id, content))

    def await_consistency(self, memory_id: str) -> None:
        return None


class Ingester:
    def __init__(self, store: Store):
        self.store = store

    def ingest_question(self, vault_title: Optional[str], vault_id: Optional[str], memory_title: str, question: Dict[str, Any]) -> IngestStats:
        stats = IngestStats(questions=1)
        v_id = self.store.ensure_vault(vault_title, vault_id)
        m_id = self.store.ensure_memory(v_id, memory_title, memory_type="NOTES")
        # For each session, write entries for user messages, then a simple context snapshot
        for sess in question.get("sessions", []):
            stats.sessions += 1
            user_texts: List[str] = [m.get("content", "") for m in sess.get("messages", []) if m.get("role") == "user"]
            for raw in user_texts:
                if not raw:
                    continue
                stats.user_messages += 1
                self.store.add_entry(v_id, m_id, raw=raw, summary=raw[:120])
                stats.entries_written += 1
            # Simple context: join user texts (truncated)
            if user_texts:
                ctx = "\n".join(user_texts)
                if len(ctx) > 4800:
                    ctx = ctx[:4800] + "…"
                self.store.put_context(v_id, m_id, ctx)
                stats.contexts_written += 1
            self.store.await_consistency(m_id)
        return stats

    def ingest_many(self, vault_title: Optional[str], vault_id: Optional[str], title_template: str, run_id: str, questions: List[Dict[str, Any]], limit: int = 2) -> IngestStats:
        agg = IngestStats()
        for idx, q in enumerate(questions[: max(0, limit)]):
            mem_title = title_template.format(question_id=q.get("question_id", f"Q{idx+1}"), run_id=run_id)
            s = self.ingest_question(vault_title, vault_id, mem_title, q)
            agg.questions += s.questions
            agg.sessions += s.sessions
            agg.user_messages += s.user_messages
            agg.entries_written += s.entries_written
            agg.contexts_written += s.contexts_written
        return agg

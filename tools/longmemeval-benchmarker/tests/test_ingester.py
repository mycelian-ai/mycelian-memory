from longmemeval_benchmarker.ingester import Ingester, InMemoryStore


def _sample_question() -> dict:
    return {
        "question_id": "Q-1",
        "expected": "E",
        "sessions": [
            {
                "session_id": "S1",
                "messages": [
                    {"role": "user", "content": "A"},
                    {"role": "assistant", "content": "B"},
                    {"role": "user", "content": "C"},
                ],
            },
            {
                "session_id": "S2",
                "messages": [
                    {"role": "user", "content": "D"},
                ],
            },
        ],
    }


def test_ingest_question_inmemory():
    store = InMemoryStore()
    ing = Ingester(store)
    s = ing.ingest_question("vault-title", None, "mem-title", _sample_question())
    assert s.questions == 1
    assert s.sessions == 2
    assert s.user_messages == 3
    assert s.entries_written == 3
    assert s.contexts_written == 2


def test_ingest_many_limit_2():
    store = InMemoryStore()
    ing = Ingester(store)
    qlist = [_sample_question(), _sample_question(), _sample_question()]
    s = ing.ingest_many("vault-title", None, "{question_id}__{run_id}", "RUN", qlist, limit=2)
    assert s.questions == 2
    assert s.sessions == 4
    assert s.entries_written == 6

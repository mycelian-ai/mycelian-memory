# ADR-018: LongMemEval Resumable Benchmarker

## Status
Proposed

## Context
The LongMemEval benchmarker processes up to 500 questions from the dataset, with each question containing multiple sessions (ranging from ~50 to 500+ depending on the dataset variant). A full benchmark run can take many hours or days, especially with larger models. Currently:

1. **No resume capability** - If the benchmarker crashes after processing 250/500 questions, we must restart from the beginning
2. **Sequential file naming** - Logs use `question_00001.log` which doesn't map to actual question IDs
3. **Monolithic output** - All hypotheses written at the end, losing progress if interrupted
4. **No progress visibility** - Can't query current status without parsing logs
5. **Worker coordination issues** - Multiple workers could conflict when writing shared files

## Decision
Implement a resumable benchmarker using SQLite for progress tracking and individual files per question.

### Core Design Elements

#### 1. Progress Tracking with SQLite
Use SQLite database (`progress.db`) to track progress at the question level.

**Important**: We track both `vault_id` and `memory_id` (UUIDs returned by Mycelian) because:
- Memory titles follow pattern `{run_id}_{question_id}` but Mycelian returns UUID identifiers
- QA phase needs the actual memory_id UUID to query against, not just the title
- Tracking vault_id enables multi-vault support and complete traceability

```sql
-- Question-level tracking with session progress
CREATE TABLE benchmark_progress (
    run_id TEXT NOT NULL,
    question_id TEXT NOT NULL,          -- 8-char hex from LongMemEval
    question_type TEXT,                  -- e.g., "multi-session"
    total_sessions INTEGER,              -- Dynamically set from dataset (varies: ~50 to 500+)
    sessions_completed INTEGER DEFAULT 0, -- Number of sessions ingested so far
    ingestion_status TEXT DEFAULT 'pending',  -- pending/running/completed/failed
    qa_status TEXT DEFAULT 'pending',
    vault_id TEXT,                       -- UUID of Mycelian vault containing the memory
    memory_id TEXT,                      -- UUID returned by Mycelian when memory is created
    ingestion_started_at TIMESTAMP,
    ingestion_completed_at TIMESTAMP,
    qa_started_at TIMESTAMP,
    qa_completed_at TIMESTAMP,
    ingestion_duration_seconds REAL,
    qa_duration_seconds REAL,
    agent_model TEXT,                    -- e.g., "openai:gpt-5-nano-2025-08-07"
    qa_model TEXT,                       -- e.g., "vertex-ai:gemini-2.5-flash-lite"
    next_session_index INTEGER DEFAULT 0, -- Next session to process (for resume)
    error_message TEXT,
    worker_id INTEGER,
    PRIMARY KEY (run_id, question_id)
);

CREATE INDEX idx_ingestion_status ON benchmark_progress(run_id, ingestion_status);
CREATE INDEX idx_qa_status ON benchmark_progress(run_id, qa_status);
```

#### 2. File Structure Changes
Move from sequential numbering to question ID-based naming:

**Current:**
```
out/run_1733430000/
├── logs/
│   └── question_00001.log
└── hypotheses.jsonl
```

**New:**
```
out/run_1733430000/
├── progress.db                    # SQLite progress tracking
├── logs/
│   └── q_0a995998.log             # Named by question_id
└── hypotheses/
    └── h_0a995998.json            # Individual hypothesis files
```

#### 3. Resume Logic

```python
# On startup
tracker = ProgressTracker(run_dir, run_id, agent_model, qa_model)

# Get all questions that need work (pending + failed)
pending = tracker.get_pending_questions(dataset)  # Returns pending AND failed questions

for question in pending:
    question_id = question['question_id']
    status = tracker.get_question_status(question_id)

    if status['ingestion_status'] != 'completed':
        # Resume from next_session_index
        next_idx = status.get('next_session_index', 0)
        run_ingestion(question, start_from_session=next_idx)

    if status['qa_status'] != 'completed':
        # Use stored vault_id and memory_id for QA
        vault_id = status['vault_id']
        memory_id = status['memory_id']
        run_qa(question, vault_id, memory_id)
```

```python
def run_ingestion(question, start_from_session=0):
    sessions = question['haystack_sessions']
    question_id = question['question_id']
    total_sessions = len(sessions)
    run_id = tracker.run_id

    # Create memory with deterministic name, store returned UUIDs
    memory_title = f"{run_id}_{question_id}"
    vault_id, memory_id = create_or_get_memory(memory_title)

    # Initialize or update question progress with Mycelian IDs
    tracker.init_or_update_question(question_id, total_sessions, vault_id, memory_id)

    for idx in range(start_from_session, total_sessions):
        session = sessions[idx]

        try:
            # Process all messages in the session
            for message in session.get('messages', []):
                process_message(message)

            # Update progress after successful session completion
            tracker.update_next_session_index(question_id, idx + 1)
            tracker.increment_sessions_completed(question_id)

            # Log progress periodically
            if idx % 10 == 0:
                log.info(f"Progress: {idx+1}/{total_sessions} sessions")

        except Exception as e:
            tracker.set_error(question_id, str(e))
            raise  # Stop processing this question

    tracker.update_status(question_id, 'ingestion_status', 'completed')
```

#### 4. Concurrency Safety
- SQLite with WAL mode handles concurrent updates from multiple workers
- Individual hypothesis files eliminate write contention
- File locking handled by SQLite, not application code

## Consequences

### Positive
1. **Granular crash recovery** - Resume at exact session within a question (e.g., session 25/50 or 250/500)
2. **Progress visibility** - Query SQLite for real-time status at question and session level
3. **Performance analysis** - Track duration and model performance per question and session
4. **Parallel safety** - Multiple workers can process different questions without conflicts
5. **Debugging** - Easy to find logs/results for specific questions
6. **Selective retry** - Can retry only failed questions or sessions
7. **Model comparison** - Track which model configuration was used
8. **Memory efficiency** - Process questions with many sessions without holding all in memory
9. **Dataset flexibility** - Handles different session counts dynamically (50 to 500+)

### Negative
1. **Complexity** - Adds SQLite dependency and progress tracking logic
2. **Migration** - Existing runs can't be resumed (one-time issue)
3. **Final assembly** - Need to combine individual hypothesis files for evaluation

### Neutral
1. **Storage** - Small increase from progress.db file (~100KB for 500 questions)
2. **Performance** - SQLite operations are negligible compared to LLM calls

## Implementation Notes

### Memory Management
- **Memory naming**: Each question gets a memory with title `{run_id}_{question_id}` for deterministic identification
- **UUID tracking**: Store the vault_id and memory_id UUIDs returned by Mycelian for direct API access
- **Resume support**: On resume, check if memory_id exists in progress DB to avoid recreating
- **QA queries**: Use stored vault_id and memory_id directly, no title lookups needed

### CLI Changes
```bash
# Normal run
python -m src.benchmarker config.toml --workers 4

# Resume interrupted run (auto-detects progress.db, includes failed questions)
python -m src.benchmarker config.toml --workers 4 --resume

# Check progress
python -m src.benchmarker status --run-dir out/run_1733430000
```

### Progress Queries
```sql
-- Overall progress
SELECT
    COUNT(*) as total_questions,
    SUM(sessions_completed) as total_sessions_done,
    SUM(total_sessions) as total_sessions_expected,
    SUM(CASE WHEN ingestion_status = 'completed' THEN 1 ELSE 0 END) as questions_ingested,
    SUM(CASE WHEN qa_status = 'completed' THEN 1 ELSE 0 END) as questions_completed,
    AVG(ingestion_duration_seconds) as avg_ingestion_time,
    AVG(qa_duration_seconds) as avg_qa_time,
    SUM(ingestion_duration_seconds) as total_ingestion_time,
    SUM(qa_duration_seconds) as total_qa_time
FROM benchmark_progress WHERE run_id = ?;

-- Question-level progress
SELECT
    question_id,
    sessions_completed || '/' || total_sessions as session_progress,
    ROUND(CAST(sessions_completed AS FLOAT) / total_sessions * 100, 2) as percent_complete,
    next_session_index,
    ingestion_status,
    qa_status,
    ingestion_duration_seconds,
    qa_duration_seconds
FROM benchmark_progress
WHERE run_id = ? AND question_id = ?;

-- Ingestion performance breakdown
SELECT
    question_id,
    total_sessions,
    sessions_completed,
    ingestion_duration_seconds,
    ROUND(ingestion_duration_seconds / NULLIF(sessions_completed, 0), 2) as avg_seconds_per_session,
    agent_model
FROM benchmark_progress
WHERE run_id = ? AND sessions_completed > 0
ORDER BY ingestion_duration_seconds DESC;

-- Questions needing resume
SELECT
    question_id,
    ingestion_status,
    qa_status,
    sessions_completed || '/' || total_sessions as progress,
    next_session_index,
    error_message
FROM benchmark_progress
WHERE run_id = ?
  AND (ingestion_status IN ('pending', 'failed', 'running')
       OR qa_status IN ('pending', 'failed', 'running'))
ORDER BY question_id;

-- Model performance comparison
SELECT
    agent_model,
    COUNT(*) as questions_processed,
    AVG(ingestion_duration_seconds) as avg_ingestion_time,
    AVG(CAST(sessions_completed AS FLOAT) / NULLIF(total_sessions, 0)) as avg_completion_rate
FROM benchmark_progress
WHERE run_id = ?
GROUP BY agent_model;
```

### Hypothesis Assembly
For final evaluation, combine individual files:
```bash
cat out/run_xyz/hypotheses/h_*.json | jq -c . > hypotheses.jsonl
```

## Alternatives Considered

1. **JSON progress file** - Rejected due to concurrent write issues with multiple workers
2. **JSONL append log** - Rejected as it doesn't support updates, only appends
3. **DuckDB** - Considered but SQLite is in standard library, no extra dependencies
4. **Directory markers** - Too complex for querying and statistics

## References
- SQLite WAL mode: https://www.sqlite.org/wal.html
- LongMemEval dataset structure: Contains question_id as 8-character hex strings
- Python sqlite3 module: Standard library support for SQLite

# Huey Task Queue Prototype

This prototype demonstrates how Huey can be used for processing LongMemEval questions with parallel workers and progress tracking.

## What is Huey?

Huey is a lightweight task queue for Python that:
- Manages a queue of tasks to be processed
- Prevents multiple workers from processing the same task
- Provides automatic retries on failure
- Persists tasks to SQLite (survives crashes)
- Handles worker coordination automatically

## Demo Features

1. **Task Processing**: Each question from LongMemEval is a task
2. **Parallel Workers**: Multiple processes can work on different questions
3. **Progress Tracking**: Combines Huey with custom SQLite tracking for session-level progress
4. **No Collision**: Huey ensures each task is processed exactly once
5. **Resume Capability**: Can resume from where it left off after interruption

## Quick Start

### Run the All-in-One Demo
```bash
python demo.py
```

This will:
1. Set up the environment
2. Enqueue 10 questions from the dataset
3. Start 3 worker processes
4. Monitor progress
5. Show results

### Manual Operation

#### 1. Enqueue Work
```bash
# Process first 5 questions
python enqueue_work.py ../longmemeval-datasets/longmemeval_s.json 5
```

#### 2. Start Workers
```bash
# Terminal 1
python run_worker.py

# Terminal 2 (optional - for parallel processing)
python run_worker.py

# Terminal 3 (optional)
python run_worker.py
```

#### 3. Monitor Progress
```bash
# In another terminal
python monitor.py
```

## Files

- `tasks.py` - Huey task definitions
- `enqueue_work.py` - Script to add tasks to queue
- `run_worker.py` - Worker process that executes tasks
- `monitor.py` - Real-time progress monitoring
- `demo.py` - All-in-one demonstration
- `requirements.txt` - Python dependencies

## Databases

- `huey_tasks.db` - Huey's internal task queue
- `progress.db` - Our custom progress tracking
- `results.jsonl` - Output results (one JSON per line)

## Key Concepts Demonstrated

### 1. Task Definition
```python
@huey.task(retries=2, retry_delay=5)
def process_question(question_data):
    # Process the question
    # Huey handles queuing, distribution, retries
```

### 2. Progress Tracking
We combine Huey's task queue with our own SQLite table to track:
- Which sessions have been processed
- How many messages were counted
- Which worker processed each question

### 3. Worker Coordination
Huey automatically ensures:
- No two workers process the same task
- Failed tasks are retried
- Tasks persist even if workers crash

## Comparison with Pure SQLite Approach

### Huey Advantages
✅ Automatic task distribution - no manual locking code
✅ Built-in retries with exponential backoff
✅ Task persistence handled automatically
✅ Clean worker management

### Pure SQLite Advantages
✅ No external dependencies (Huey is a dependency)
✅ Full control over progress tracking
✅ Simpler mental model for our use case
✅ One less abstraction layer

### Verdict for LongMemEval

For the LongMemEval benchmarker with:
- Only 5-500 questions to process
- Need for session-level progress tracking
- Requirement to resume from specific session

**Recommendation**: The pure SQLite approach is simpler and sufficient. Huey adds value for larger-scale task processing but is overkill for processing a fixed set of questions with detailed progress tracking needs.

## What We Learned

1. **Huey excels at task distribution** - No worry about worker collisions
2. **Still need custom progress tracking** - Huey only knows task is done/not done
3. **Good for fire-and-forget tasks** - Less ideal when you need granular progress
4. **Best for larger task volumes** - Overkill for just 5-50 questions

## Testing Different Scenarios

### Test Resume Capability
```bash
# Start processing
python enqueue_work.py ../longmemeval-datasets/longmemeval_s.json 5
python run_worker.py

# Kill worker (Ctrl+C) after a few seconds
# Then restart
python run_worker.py  # Will resume from where it left off
```

### Test Multiple Workers
```bash
# Terminal 1
python monitor.py  # Watch the progress

# Terminal 2
python enqueue_work.py ../longmemeval-datasets/longmemeval_s.json 20

# Terminals 3, 4, 5 (run simultaneously)
python run_worker.py
python run_worker.py
python run_worker.py
```

Watch in the monitor how different workers claim different questions with no collisions!
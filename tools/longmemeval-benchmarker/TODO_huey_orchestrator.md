# TODO: Huey-based LongMemEval Orchestrator

## Overview
Implement a robust, resumable benchmarker using Huey task queue to replace the current multi-threading approach with process-based workers.

## Phase 1: Core Infrastructure
- [ ] Set up Huey configuration with SqliteHuey backend
  - [ ] Create `huey_config.py` with proper settings
  - [ ] Configure task retries and error handling
  - [ ] Set up proper logging configuration

- [ ] Design progress tracking schema
  - [ ] Create SQLite schema for benchmark runs
  - [ ] Track question-level progress (total sessions, completed sessions)
  - [ ] Track session-level progress (messages processed) - don't need this. if a session fails we will process it from the start, it's not that expansive
  - [ ] Add status fields (pending, in_progress, completed, failed)

- [ ] Implement task definitions
  - [ ] Create `process_question` task
  - [ ] Create `process_session` subtask  
  - [ ] Add proper error handling and retries
  - [ ] Implement progress updates after each session

## Phase 2: Mycelian Integration [non needed - all of this is encapsulated under current implementation, we just need to add an additional parameter to ingest session id so that the single question runner can start from there]
- how will we record memory created for every question for the run, it will be deterministic name run_id_question_id so we don't need to persist it
- [ ] Create Mycelian memory setup tasks
  - [ ] Task to create/verify vault exists
  - [ ] Task to create/verify memory exists
  - [ ] Task to clear existing memory if requested
  - [ ] Add connection pooling and retry logic

- [ ] Implement session processing [all of this is encapsulated huegy just treats the current python process as a worker]
  - [ ] Convert session messages to Mycelian entries
  - [ ] Handle context updates appropriately
  - [ ] Track which sessions have been stored
  - [ ] Add idempotency (skip already processed sessions)

## Phase 3: Orchestration Layer
- [ ] Create main orchestrator script
  - [ ] Parse command-line arguments
  - [ ] Load and validate dataset
  - [ ] Initialize progress database
  - [ ] Enqueue all questions as tasks
  - [ ] Monitor overall progress

- [ ] Implement worker management
  - [ ] Create worker launch script
  - [ ] Add graceful shutdown handling
  - [ ] Implement worker health checks
  - [ ] Add automatic worker scaling (optional)

## Phase 4: Monitoring & Resumability
- [ ] Add progress monitoring
  - [ ] Create real-time progress display
  - [ ] Show per-question and overall statistics
  - [ ] Display worker status and throughput
  - [ ] Add ETA calculations

- [ ] Implement resume capability
  - [ ] Detect incomplete runs on startup
  - [ ] Resume from last completed session per question
  - [ ] Handle partially processed sessions
  - [ ] Clean up stale locks from crashed workers

## Phase 5: QA Runner Integration
- [ ] Create QA phase orchestration
  - [ ] Task to run QA for completed questions
  - [ ] Store QA results separately
  - [ ] Support different QA modes (quick vs full)
  - [ ] Track QA completion status

- [ ] Implement result aggregation
  - [ ] Collect results from all questions
  - [ ] Calculate accuracy metrics
  - [ ] Generate summary reports
  - [ ] Export results in standard format

## Phase 6: Testing & Validation
- [ ] Unit tests
  - [ ] Test task definitions
  - [ ] Test progress tracking
  - [ ] Test resume logic
  - [ ] Test error handling

- [ ] Integration tests
  - [ ] Test with small dataset subset
  - [ ] Test worker coordination
  - [ ] Test resume after simulated crash
  - [ ] Validate against Huey prototype

- [ ] Performance testing
  - [ ] Benchmark throughput with various worker counts
  - [ ] Test memory usage patterns
  - [ ] Optimize database queries
  - [ ] Profile and remove bottlenecks

## Phase 7: Production Readiness
- [ ] Add operational features
  - [ ] Configuration file support
  - [ ] Environment variable overrides
  - [ ] Proper logging with rotation
  - [ ] Metrics export (Prometheus format?)

- [ ] Documentation
  - [ ] Write user guide
  - [ ] Document configuration options
  - [ ] Add troubleshooting guide
  - [ ] Create architecture diagram

- [ ] Deployment automation
  - [ ] Create Docker image
  - [ ] Add docker-compose configuration
  - [ ] Create systemd service files
  - [ ] Add health check endpoints

## Migration Path
- [ ] Create migration script from old format
  - [ ] Convert old progress files
  - [ ] Import partial results
  - [ ] Validate data integrity

## Known Issues to Address
1. **Log file collision**: Use question_id in log filenames instead of sequential index
2. **Data contamination**: Ensure correct session data loaded per question  
3. **Thread safety**: Eliminate by using separate processes
4. **Dataset compliance**: Handle both string and integer answers

## Success Criteria
- [ ] Can process full LongMemEval dataset without errors
- [ ] Supports graceful resume after interruption
- [ ] Multiple workers coordinate without conflicts
- [ ] Progress tracking accurate to session level
- [ ] QA phase produces valid evaluation metrics
- [ ] Performance better than multi-threaded version

## Notes
- Start with Phase 1 & 2 for MVP
- Huey prototype in `huey-prototype/` serves as reference
- Use SQLite for both Huey backend and progress tracking
- Consider using Click for CLI interface
- Each worker processes one question at a time to avoid complexity
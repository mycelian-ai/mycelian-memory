# Conversation Time Implementation - TODO Tracker

## ✅ Completed Tasks (6/15 - 40%)

### Infrastructure Layer
- [x] Create storage layer tests for conversation_time
- [x] Create API handler tests for conversation_time
- [x] Add future date validation for conversation_time
- [x] Create client library tests for conversation_time
- [x] Update MCP handler tests for conversation_time
- [x] Create E2E tests for conversation_time

## 📋 Pending Tasks (9/15 - 60%)

### LongMemEval Integration (Priority 1)
- [ ] Implement non-context-only mode with conversation_time in add_entry
- [ ] Update LongMemEval dataset loader to parse dates
- [ ] Update single_question_runner to pass conversation_time
- [ ] Update context builder to create Timeline from conversation_time
- [ ] Update prompt files for conversation_time handling
- [ ] Test end-to-end with temporal questions

### Search & Compliance (Priority 2)
- [ ] Update store layer compliance tests
- [ ] Update SearchHit model with ConversationTime field
- [ ] Update search implementation to handle conversationTime

## Progress Notes

### Current Status
- Infrastructure ready: conversation_time flows through entire stack
- Database schema updated with indexes
- API validation implemented (rejects future dates)
- All tests passing

### Key Findings
- LongMemEval dataset has session-level timestamps in `haystack_dates`
- Currently runs in context-only mode (doesn't persist)
- Need to enable non-context-only mode to use timestamps

### Dataset Timestamp Structure
```json
{
  "question_date": "2025-09-12",
  "haystack_dates": [
    "2025-09-11T14:31:10Z",  // Session 0 timestamp
    "2025-09-11T14:34:15Z"   // Session 1 timestamp
  ],
  "haystack_sessions": [
    [ // Session 0 - all messages share the session timestamp
      {"role": "user", "content": "..."},
      {"role": "assistant", "content": "..."}
    ],
    [ // Session 1 - all messages share the session timestamp
      {"role": "user", "content": "..."},
      {"role": "assistant", "content": "..."}
    ]
  ]
}
```

### Implementation Strategy
1. **Phase 1: Enable Persistence**
   - Switch from context-only to full persistence mode
   - Ensure add_entry calls include conversation_time

2. **Phase 2: Dataset Integration**
   - Parse haystack_dates from dataset
   - Map session timestamps to messages
   - Pass timestamps through agent pipeline

3. **Phase 3: Temporal Features**
   - Build timeline from conversation_time
   - Update prompts for temporal reasoning
   - Test with temporal questions

### Relevant Files
- `/server/internal/storage/postgres/schema.sql` - Database schema with conversation_time
- `/server/internal/services/memory.go` - Service layer validation
- `/longmemeval-benchmarker/src/dataset_loader.py` - Dataset parsing
- `/longmemeval-benchmarker/src/single_question_runner.py` - Message processing
- `/longmemeval-benchmarker/src/mycelian_memory_agent/agent.py` - Agent implementation

### Git Commits
All changes are on branch: `conversation-time`

Recent commits:
- `4bb7f62` - test: add E2E integration tests for conversation_time
- `6b8ced9` - test: add MCP handler tests for conversation_time
- `f0f02cf` - test: add client library tests for conversation_time
- `f5d54d5` - feat: add API handler tests and future date validation
- `72e27df` - test: add storage layer tests for conversation_time

### Next Steps
1. Enable non-context-only mode in benchmarker config
2. Update dataset loader to extract and normalize timestamps
3. Modify agent to accept and use conversation_time parameter
4. Test with a small dataset to verify timestamps are persisted correctly
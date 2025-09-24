# Conversation Time Implementation - TODO Tracker

## ✅ Completed Tasks (12/21 - 57%)

### Infrastructure Layer
- [x] Create storage layer tests for conversation_time
- [x] Create API handler tests for conversation_time
- [x] Add future date validation for conversation_time
- [x] Create client library tests for conversation_time
- [x] Update MCP handler tests for conversation_time
- [x] Create E2E tests for conversation_time

### LongMemEval Integration
- [x] Implement non-context-only mode with conversation_time in add_entry
- [x] Update LongMemEval dataset loader to parse dates
- [x] Update single_question_runner to pass conversation_time
- [x] Update context builder to create Timeline from conversation_time
- [x] Update prompt files for conversation_time handling
- [x] Test end-to-end with temporal questions (smoke test)

## 📋 Pending Tasks (9/21 - 43%)

### Search & Compliance (Priority 1)
- [ ] Update SearchHit model with ConversationTime field
- [ ] Update search implementation to handle conversationTime parameter
- [ ] Add unit tests for SearchHit model with ConversationTime
- [ ] Add search tests for conversation_time filtering
- [ ] Add search tests for conversation_time in results
- [ ] Check if outbox worker needs updates for conversation_time
- [ ] Update store layer compliance tests for search with conversation_time
- [ ] Add integration tests for search API with conversation_time
- [ ] Test search API with temporal queries from LongMemEval

## Progress Notes

### Current Status
- ✅ Infrastructure complete: conversation_time flows through entire stack
- ✅ Database schema updated with indexes
- ✅ API validation implemented (rejects future dates)
- ✅ LongMemEval integration complete: dataset timestamps parsed and used
- ✅ Context Timeline now uses conversation_time from dataset
- ⏳ Search API integration pending

### Key Accomplishments
- Successfully parse and normalize haystack_dates to ISO-8601
- Pass conversation_time through entire session lifecycle (start, process, end)
- Timeline entries in context use dataset timestamps instead of current date
- Add_entry tool calls include conversation_time parameter
- Debug logging added for troubleshooting timestamp flow

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
1. **Phase 1: Enable Persistence** ✅ COMPLETE
   - Switched to entry+context mode (context_only=false)
   - add_entry calls include conversation_time parameter

2. **Phase 2: Dataset Integration** ✅ COMPLETE
   - Parse haystack_dates from dataset (both formats supported)
   - Map session timestamps to messages
   - Pass timestamps through agent pipeline

3. **Phase 3: Temporal Features** ✅ COMPLETE
   - Timeline built from conversation_time in context
   - Prompts updated for temporal reasoning
   - Tested with smoke dataset

4. **Phase 4: Search Integration** ⏳ IN PROGRESS
   - Update search models and API
   - Add comprehensive test coverage
   - Enable temporal search queries

### Relevant Files
- `/server/internal/storage/postgres/schema.sql` - Database schema with conversation_time
- `/server/internal/services/memory.go` - Service layer validation
- `/longmemeval-benchmarker/src/dataset_loader.py` - Dataset parsing
- `/longmemeval-benchmarker/src/single_question_runner.py` - Message processing
- `/longmemeval-benchmarker/src/mycelian_memory_agent/agent.py` - Agent implementation

### Git Commits
All changes are on branch: `conversation-time`

Recent commits:
- `22afb95` - fix: ensure conversation_time flows through entire session lifecycle
- `d911cbb` - feat: implement conversation_time support in LongMemEval benchmarker
- `3be0023` - docs: add persistent TODO tracker for conversation_time implementation
- `4bb7f62` - test: add E2E integration tests for conversation_time
- `6b8ced9` - test: add MCP handler tests for conversation_time
- `f0f02cf` - test: add client library tests for conversation_time

### Next Steps
1. Implement SearchHit model updates with ConversationTime field
2. Update search implementation to handle temporal queries
3. Add comprehensive test coverage for search functionality
4. Test with full LongMemEval dataset for temporal question accuracy

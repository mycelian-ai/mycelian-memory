d# Agent Implementation Todo List

## Overview
Complete rewrite of the Mycelian Memory Agent using control-based state management pattern from our working prototype. The goal is to create a clean, reliable agent that doesn't get stuck in infinite loops and properly observes conversations.

## Phase 1: Core Structure
- [ ] **Design ObserverState TypedDict with control field and necessary state tracking**
  - Define state structure with control field
  - Add message tracking fields
  - Include thread_id and msg_idx tracking
  
- [ ] **Create ObserverAgent class skeleton with __init__ and core structure**
  - Basic class with dependencies (llm, tools, prompt_builder, checkpointer)
  - Store config and helpers
  - Set up constants (FLUSH_INTERVAL=6, RECURSION_LIMIT=12)
  
- [ ] **Implement _build_graph method with observe node and tool node**
  - Create StateGraph with ObserverState
  - Add observe and tools nodes
  - Set up routing from observe → tools → observe
  - Compile with checkpointer
  
- [ ] **Implement _observe_node with tool execution completion checks**
  - Check if returning from tool execution
  - Different completion logic for each control state
  - Build appropriate prompt and invoke LLM
  
- [ ] **Implement _determine_control method to map messages to control states**
  - SESSION_START → "start_session"
  - SESSION_END → "end_session"
  - Conversation with idx % 6 == 0 → "process_and_flush"
  - Regular conversation → "process_turn"

## Phase 2: Prompt Building
- [ ] **Build control-specific prompts for start_session operation**
  - Instruction to call get_context
  - Then call list_entries with limit=10
  - Include context_prompt for guidance
  
- [ ] **Build control-specific prompts for process_turn operation**
  - Extract current conversation message
  - Instruction to call add_entry
  - Include entry_capture_prompt for quality
  
- [ ] **Build control-specific prompts for process_and_flush operation**
  - Process current message with add_entry
  - Call await_consistency
  - Call put_context with updated context
  - Include both entry and context prompts
  
- [ ] **Build control-specific prompts for end_session operation**
  - Optional put_context for final save
  - Include context_prompt for comprehensive summary

## Phase 3: Integration
- [ ] **Implement invoke_message entry point with validation**
  - Validate message parameters per message_spec.md
  - Create appropriate LangChain message
  - Set up state with control
  - Configure and invoke graph
  
- [ ] **Create simplified agent_factory with MCP client setup**
  - Setup MCP client (injected or created)
  - Load tools from MCP
  - Create LLM instance
  - Build PromptBuilder
  - Create checkpointer
  - Wire everything together
  
- [ ] **Update __init__.py to export new components**
  - Export ObserverAgent
  - Export build_agent factory function
  - Keep helper classes
  - Maintain backward compatibility during transition

## Phase 4: Testing
- [ ] **Test SESSION_START without infinite loops**
  - Verify get_context is called
  - Verify list_entries is called
  - Confirm clean exit after both tools
  
- [ ] **Test conversation message processing with add_entry**
  - Verify add_entry is called for user messages
  - Verify add_entry is called for assistant messages
  - Confirm no duplicate processing
  
- [ ] **Test flush logic at msg_idx % 6 == 0**
  - Verify add_entry is called first
  - Verify await_consistency is called
  - Verify put_context is called
  - Confirm proper sequencing
  
- [ ] **Test SESSION_END with final context save**
  - Verify put_context is called if needed
  - Confirm clean exit
  
- [ ] **Verify state accumulation and checkpointer integration**
  - Confirm messages accumulate properly
  - Verify thread_id isolation
  - Test state retrieval

## Phase 5: Polish
- [ ] **Apply Clean Code principles and refactor**
  - Single Responsibility Principle
  - DRY (Don't Repeat Yourself)
  - Clear, intentional naming
  - Reduce cyclomatic complexity
  - Extract magic numbers to constants
  
- [ ] **Run full benchmarker test suite**
  - Run test_message_flow.py
  - Verify no infinite loops
  - Check tool call accuracy
  - Measure performance
  
- [ ] **Document the new architecture and control flow**
  - Explain control-based approach
  - Document state management
  - Provide usage examples
  - Create migration guide from old agent

## Key Design Principles
1. **Control field drives behavior** - Not prompt-based detection
2. **Phase-specific prompts** - Not one giant prompt with all rules
3. **Clear tool completion checks** - Prevent infinite loops
4. **Simple, focused methods** - Each method does one thing
5. **Explicit over implicit** - Clear state transitions

## Success Criteria
- [ ] No infinite loops on SESSION_START
- [ ] Proper add_entry for all conversation messages
- [ ] Correct flush behavior every 6 messages
- [ ] Clean SESSION_END handling
- [ ] All tests pass
- [ ] Code is clean and maintainable

## Notes
- Based on working prototype from lang-graph-tutorial/session_context_observer.py
- Adapts pattern to benchmarker's message format (SystemMessage for control, ChatMessage for conversation)
- Maintains compatibility with existing helper classes
- Focuses on simplicity and reliability over cleverness
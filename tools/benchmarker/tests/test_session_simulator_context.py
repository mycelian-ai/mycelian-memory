"""Unit tests for SessionSimulator context management."""
"""Unit tests for SessionSimulator context management."""
import json
import pytest
from unittest.mock import AsyncMock, MagicMock

import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(__file__)))
from session_simulator import SessionSimulator, END_SESSION_TOKEN
from mycelian_client import MycelianMemoryClient
from system_prompt_builder import PromptAssembler

@pytest.fixture
def mock_anthropic():
    mock = MagicMock()
    mock.messages = MagicMock()
    mock.messages.create = AsyncMock()
    return mock

@pytest.fixture
def mock_mycelian_client():
    client = MagicMock(spec=MycelianMemoryClient)
    client.get_context.return_value = {}
    client.put_context.return_value = True
    client.add_entry.return_value = True
    return client

@pytest.fixture
def mock_prompt_builder():
    builder = MagicMock(spec=PromptAssembler)
    builder.build.return_value = "Test system prompt"
    builder.tool_schema = [{"name": "test_tool"}]
    builder.memory_id = "test-memory-123"
    return builder

@pytest.fixture
def session_simulator(mock_anthropic, mock_mycelian_client, mock_prompt_builder):
    return SessionSimulator(
        anthropic_client=mock_anthropic,
        mycelian_client=mock_mycelian_client,
        system_builder=mock_prompt_builder
    )

@pytest.mark.asyncio
async def test_message_counter_increments(session_simulator, mock_anthropic):
    """Test that message counter increments with each step."""
    # Mock Claude response
    mock_response = MagicMock()
    mock_response.content = [MagicMock(type="text", text="Test response")]
    mock_anthropic.messages.create.return_value = mock_response
    
    # Initial state
    assert session_simulator._message_counter == 0
    
    # First message
    await session_simulator.step("Hello")
    assert session_simulator._message_counter == 1
    
    # Second message
    await session_simulator.step("How are you?")
    assert session_simulator._message_counter == 2

@pytest.mark.asyncio
async def test_context_update_after_5_messages(session_simulator, mock_anthropic, mock_mycelian_client, mock_prompt_builder):
    """Test that context updates after 5 messages."""
    # Set up mock prompt builder with memory_id
    mock_prompt_builder.memory_id = "test-memory-123"
    
    # Mock Claude response
    mock_response = MagicMock()
    mock_response.content = [MagicMock(type="text", text="Test response")]
    mock_anthropic.messages.create.return_value = mock_response
    
    # Reset call count before test
    mock_mycelian_client.put_context.reset_mock()
    
    # Send 5 messages (should trigger update on 5th)
    for i in range(1, 6):
        await session_simulator.step(f"Message {i}")
        
        # After 5th message, we should have a context update
        if i == 5:
            # Verify context was updated
            assert mock_mycelian_client.put_context.call_count > 0, "Context should be updated after 5 messages"
            
            # Get the last call to put_context
            args, _ = mock_mycelian_client.put_context.call_args
            assert args[0] == "test-memory-123"
            context = json.loads(args[1])
            assert "message_count" in context
            assert context["message_count"] == 5
            assert "last_updated" in context
            assert "history_summary" in context

@pytest.mark.asyncio
async def test_critical_update_triggers_context_save(session_simulator, mock_anthropic, mock_mycelian_client, mock_prompt_builder):
    """Test that critical updates trigger immediate context save."""
    # Mock Claude response with tool call
    mock_tool_call = MagicMock()
    mock_tool_call.type = "tool_use"
    mock_tool_call.name = "add_entry"
    mock_tool_call.input = {
        "memory_id": "test-memory-123",
        "summary": "Test summary", 
        "raw_entry": "Test entry"
    }
    
    mock_response = MagicMock()
    mock_response.content = [mock_tool_call]
    mock_anthropic.messages.create.return_value = mock_response
    
    # First message with critical update
    await session_simulator.step("Add important note")
    
    # Should call add_entry with correct parameters
    mock_mycelian_client.add_entry.assert_called_once_with(
        "test-memory-123",
        "Test entry",
        "Test summary",
        {}
    )

@pytest.mark.asyncio
async def test_session_end_updates_context(session_simulator, mock_anthropic, mock_mycelian_client):
    """Test that session end triggers final context update."""
    # Mock Claude response
    mock_response = MagicMock()
    mock_response.content = [MagicMock(type="text", text="Final response")]
    mock_anthropic.messages.create.return_value = mock_response
    
    # Reset call count
    mock_mycelian_client.put_context.reset_mock()
    
    # Send some messages
    for i in range(3):
        await session_simulator.step(f"Message {i}")
    
    # Close session – expect RuntimeError because mock model never issues put_context
    with pytest.raises(RuntimeError):
        await session_simulator.close_session()

    # Ensure we did *not* attempt to persist context client-side (behaviour changed)
    assert mock_mycelian_client.put_context.call_count == 0

@pytest.mark.asyncio
async def test_context_update_error_handling(session_simulator, mock_anthropic, mock_mycelian_client, caplog, mock_prompt_builder):
    """Test that context update errors are handled gracefully."""
    # Set up mock prompt builder with memory_id
    mock_prompt_builder.memory_id = "test-memory-123"
    
    # Mock Claude response
    mock_response = MagicMock()
    mock_response.content = [MagicMock(type="text", text="Test response")]
    mock_anthropic.messages.create.return_value = mock_response
    
    # Make get_context raise an error
    test_error = Exception("Test error")
    mock_mycelian_client.get_context.side_effect = test_error
    
    # Clear any existing logs
    caplog.clear()
    
    # This should not raise
    try:
        # Send enough messages to trigger context update (5th message)
        for i in range(5):
            await session_simulator.step(f"Test message {i}")
    except Exception as e:
        pytest.fail(f"Unexpected exception: {e}")
    
    # Verify we tried to get the context
    mock_mycelian_client.get_context.assert_called()
    
    # Check that we logged the error
    error_logged = any(
        record.levelname == "WARNING" and 
        "Failed to get current context" in str(record.message) and
        "Test error" in str(record.message)
        for record in caplog.records
    )
    assert error_logged, f"Expected error log not found in records. Logs: {[str(r) for r in caplog.records]}"

@pytest.mark.asyncio
async def test_get_default_prompts_bootstrap_marks_assets(session_simulator, mock_mycelian_client):
    """get_default_prompts should mark required assets as satisfied."""
    mock_mycelian_client.get_prompts.return_value = {
        "version": "v1",
        "context_summary_rules": "RULES",
        "templates": {
            "context_prompt": "CTX",
            "entry_capture_prompt": "ENTRY",
            "summary_prompt": "SUMMARY",
        },
    }
    await session_simulator._exec_get_default_prompts({"memory_type": "chat"})
    assert session_simulator._boot_seen_ctx_rules is True
    assert {
        "ctx_rules",
        "ctx_prompt_chat",
        "entry_prompt_chat",
        "summary_prompt_chat",
    }.issubset(session_simulator._boot_assets_downloaded)

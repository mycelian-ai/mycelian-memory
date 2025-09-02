"""Test compliance of the agent."""
import json
from src.agent.agent_builder import AgentBuilder
from src.agent.mcp_utils import create_mcp_client

# Create agent
mcp_client = create_mcp_client()
builder = AgentBuilder(
    model_id="gpt-5-nano-2025-08-07",
    vault_id="test-vault",
    memory_id="test-memory",
    debug=True,
    mcp_client=mcp_client
)
agent = builder.build()

# Test message creation
print("Testing message creation with idx field...")

# Create a conversation message
msg = agent._create_langchain_message(
    message_type="conversation",
    content="Test message",
    role="user",
    msg_idx=6
)

print(f"Message created: {type(msg).__name__}")
print(f"Message idx: {getattr(msg, 'idx', 'NOT FOUND')}")
print(f"Message role: {getattr(msg, 'role', 'NOT FOUND')}")

# Test dynamic prompt
print("\nTesting dynamic prompt generation...")
prompt = agent.agent._build_dynamic_prompt("conversation", 6)
if "FLUSH REQUIRED" in prompt:
    print("✓ Flush requirement detected for message 6")
else:
    print("✗ Flush requirement NOT detected for message 6")

print("\nSuccess!")

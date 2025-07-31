"""
Test script to validate context management behavior.
Verifies that Claude correctly follows the context update rules.
"""
import asyncio
import pytest
from datetime import datetime
from typing import Dict, List, Optional

class TestContextManager:
    def __init__(self):
        self.message_counter = 0
        self.context: Dict = {
            'participants': {},
            'key_facts': {},
            'topics': {},
            'timeline': [],
            'decisions': {},
            'open_tasks': {}
        }
        self.logs: List[str] = []
    
    async def get_context(self) -> Dict:
        """Simulate getting context from memory."""
        self._log("INFO", "Fetching context from memory")
        return self.context
    
    async def put_context(self, context: Dict) -> bool:
        """Simulate saving context to memory."""
        self._log("SAVE", f"Saving context (size: {len(str(context))} chars)")
        self.context = context
        return True
    
    async def process_message(self, message: str, message_type: str = "user") -> str:
        """Process a message and update context accordingly."""
        self.message_counter += 1
        self._log("MSG", f"[{message_type.upper()}] {message}")
        
        # Update context based on message
        self._update_context(message)
        
        # Check if we need to save
        if self.message_counter % 5 == 0 or self._is_critical(message):
            await self.put_context(self.context)
            self._log("INFO", f"Context saved (trigger: {'critical' if self._is_critical(message) else '5th message'})")
        
        return f"Processed message #{self.message_counter}"
    
    def _update_context(self, message: str) -> None:
        """Update context based on message content."""
        # Simulate context updates
        if "introduce" in message.lower() and "as " in message.lower():
            name = message.split("as ")[1].split()[0]
            self.context['participants'][name] = {
                'role': 'participant',
                'first_mentioned': self.message_counter
            }
            self._add_timeline_entry(f"{name} joined the conversation")
        
        # Add more context update rules as needed
        
    def _add_timeline_entry(self, event: str) -> None:
        """Add an entry to the timeline."""
        timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        self.context['timeline'].append({
            'timestamp': timestamp,
            'event': event,
            'message_id': self.message_counter
        })
    
    def _is_critical(self, message: str) -> bool:
        """Check if message contains critical information."""
        critical_keywords = ["important", "decision", "urgent", "save context"]
        return any(keyword in message.lower() for keyword in critical_keywords)
    
    def _log(self, level: str, message: str) -> None:
        """Log a message with timestamp."""
        timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]
        log_entry = f"[{timestamp}] [{level}] {message}"
        self.logs.append(log_entry)
        print(log_entry)

# pytest-asyncio ensures async test function is awaited automatically
@pytest.mark.asyncio
async def test_context_updates():
    """Test context update behavior."""
    print("=== Starting Context Management Test ===")
    manager = TestContextManager()
    
    # Initial context load
    await manager.get_context()
    
    # Simulate conversation
    messages = [
        ("user", "Hello, I'm Alice"),
        ("assistant", "Hi Alice! How can I help you today?"),
        ("user", "Let's talk about the project"),
        ("assistant", "Sure, what would you like to know?"),
        ("user", "Please introduce me as the project manager"),  # Should trigger save (5th message)
        ("assistant", "Got it, you're the project manager."),
        ("user", "Important: The deadline is next Friday"),  # Critical update
        ("assistant", "I've noted the deadline."),
        ("user", "Let's schedule a meeting"),
        ("assistant", "When works for you?")  # Should trigger save (10th message)
    ]
    
    for role, content in messages:
        await manager.process_message(content, role)
    
    # Final save
    await manager.put_context(manager.context)
    print("\n=== Test Complete ===")
    print(f"Total messages processed: {manager.message_counter}")
    print(f"Context saves: {len([l for l in manager.logs if '[SAVE]' in l])}")

if __name__ == "__main__":
    asyncio.run(test_context_updates())

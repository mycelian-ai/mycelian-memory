"""MycelianAgentInvoker - Encapsulates message building and control determination."""

import json
import logging
from datetime import datetime
from typing import Optional
from langchain_core.messages import ChatMessage

from .control_state import ControlState
from .agent import MycelianMemoryAgent

# Setup logger for audit trail
# Using "lme.agent" to integrate with benchmarker's logging system
logger = logging.getLogger("lme.agent")


class MycelianAgentInvoker:
    """Encapsulates message building and control determination.
    
    This invoker wraps the MycelianMemoryAgent to provide a clean API that:
    - Handles message counting for flush logic
    - Converts raw strings to ChatMessage objects
    - Determines ControlState internally
    - Provides simple methods: start_session, process_conversation_message, end_session
    """
    
    def __init__(self, agent: MycelianMemoryAgent):
        """Initialize the invoker with an agent.
        
        Args:
            agent: The MycelianMemoryAgent to wrap
        """
        self.agent = agent
        self.msg_count = 0    
    def start_session(self, thread_id: str) -> None:
        """Start a new session.
        
        This retrieves previous context and recent entries.
        
        Args:
            thread_id: Unique identifier for this conversation thread
        """
        self.msg_count = 0
        
        logger.info(json.dumps({
                "event": "invoker_start_session",
                "timestamp": datetime.utcnow().isoformat(),
                "thread_id": thread_id,
                "msg_count_reset": True
            }))
        
        result = self.agent.invoke(
            control=ControlState.START_SESSION,
            thread_id=thread_id
        )
        return result
    
    def process_conversation_message(self, role: str, content: str, thread_id: str) -> None:
        """Process a conversation message, handling flush automatically.
        
        Args:
            role: The role of the message (user, assistant, etc.)
            content: The content of the message
            thread_id: Unique identifier for this conversation thread
        """
        self.msg_count += 1
        
        # Build message internally
        message = ChatMessage(role=role, content=content)
        
        # Determine control internally - flush every 6 messages
        if self.msg_count % 6 == 0:
            control = ControlState.PROCESS_MESSAGE_AND_FLUSH
        else:
            control = ControlState.PROCESS_MESSAGE
        
        logger.info(json.dumps({
                "event": "invoker_process_message",
                "timestamp": datetime.utcnow().isoformat(),
                "thread_id": thread_id,
                "msg_count": self.msg_count,
                "control": control.value,
                "role": role,
                "content_preview": content[:200] if content else None,
                "will_flush": control == ControlState.PROCESS_MESSAGE_AND_FLUSH
            }))
        
        result = self.agent.invoke(
            control=control,
            thread_id=thread_id,
            to_process=message
        )
        return result
    
    def end_session(self, thread_id: str) -> None:
        """End the session.
        
        This ensures consistency and saves the final context.
        
        Args:
            thread_id: Unique identifier for this conversation thread
        """
        logger.info(json.dumps({
                "event": "invoker_end_session",
                "timestamp": datetime.utcnow().isoformat(),
                "thread_id": thread_id,
                "final_msg_count": self.msg_count
            }))
        
        result = self.agent.invoke(
            control=ControlState.END_SESSION,
            thread_id=thread_id
        )
        return result
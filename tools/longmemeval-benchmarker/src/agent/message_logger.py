"""MessageLogger class for handling message-specific logging."""

import json
import logging
from typing import Any, Dict, Optional


class MessageLogger:
    """Specialized logger for message handling and formatting."""
    
    # Constants
    MESSAGE_LOG_LIMIT = 200
    MESSAGE_PREVIEW_LIMIT = 197
    TRUNCATION_SUFFIX = "..."
    TRUNCATION_SUFFIX_LEN = 3
    
    def __init__(self, debug: bool = False):
        """Initialize MessageLogger with debug mode setting."""
        self.debug = debug
        self.logger = logging.getLogger("lme.agent")
    
    def log_system_message(self, thread_id: str, content: str) -> None:
        """Log a system message with appropriate formatting.
        
        Args:
            thread_id: Thread identifier for the session
            content: System message content
        """
        if not self.debug:
            return
            
        try:
            preview = self._truncate_preview(content)
            self.logger.info("SYS thread_id=%s content=%s", thread_id, preview)
        except (AttributeError, TypeError):
            pass  # Silent failure for logging
    
    def log_conversation_message(
        self, 
        thread_id: str, 
        role: str, 
        msg_idx: int, 
        content: str
    ) -> None:
        """Log a conversation message with role and index information.
        
        Args:
            thread_id: Thread identifier for the session
            role: Message role ('user' or 'assistant')
            msg_idx: 1-based message index in the conversation
            content: Message content
        """
        if not self.debug:
            return
            
        try:
            preview = self._truncate_preview(content)
            self.logger.info(
                "MSG thread_id=%s msg_idx=%d role=%s chars=%d preview=%s",
                thread_id,
                int(msg_idx or 0),
                str(role or ""),
                len(content or ""),
                preview,
            )
        except (AttributeError, TypeError, ValueError):
            pass  # Silent failure for logging
    
    def log_message(
        self,
        message_type: str,
        content: str,
        thread_id: str,
        role: Optional[str] = None,
        msg_idx: Optional[int] = None
    ) -> None:
        """Log a message based on its type.
        
        This is a dispatcher method that routes to the appropriate
        specialized logging method based on message type.
        
        Args:
            message_type: Type of message ('system' or 'conversation')
            content: Message content
            thread_id: Thread identifier
            role: Optional role for conversation messages
            msg_idx: Optional index for conversation messages
        """
        if not self.debug:
            return
            
        if message_type == "system":
            self.log_system_message(thread_id, content)
        elif message_type == "conversation" and role and msg_idx is not None:
            self.log_conversation_message(thread_id, role, msg_idx, content)
    
    def log_result(self, thread_id: str, message_count: Optional[int] = None) -> None:
        """Log the result summary after agent invocation.
        
        Args:
            thread_id: Thread identifier
            message_count: Optional count of messages in the state
        """
        if not self.debug:
            return
            
        try:
            if message_count is not None:
                self.logger.info("RESULT thread_id=%s messages=%s", thread_id, message_count)
        except (AttributeError, TypeError):
            pass  # Silent failure for logging
    
    def log_error(self, error_message: str, limit: Optional[int] = None) -> None:
        """Log an error message with truncation.
        
        Args:
            error_message: The error message to log
            limit: Optional character limit for the message
        """
        if not self.debug:
            return
            
        try:
            if limit:
                truncated = self._truncate_text(error_message, limit)
            else:
                truncated = self._truncate_preview(error_message)
            self.logger.info("AINVOKE_ERROR %s", truncated)
        except (AttributeError, TypeError):
            pass  # Silent failure for logging
    
    def _truncate_preview(self, text: Optional[str]) -> str:
        """Truncate text for preview with standard limits.
        
        Args:
            text: Text to truncate
            
        Returns:
            Truncated text suitable for logging preview
        """
        if not text:
            return ""
            
        text = (text or "").strip()
        
        # If text is within log limit, return as is
        if len(text) <= self.MESSAGE_LOG_LIMIT:
            return text
            
        # Truncate to preview limit
        return self._truncate_text(text, self.MESSAGE_PREVIEW_LIMIT)
    
    def _truncate_text(self, text: str, limit: int) -> str:
        """Truncate text to specified limit with suffix.
        
        Args:
            text: Text to truncate
            limit: Maximum length
            
        Returns:
            Truncated text with suffix if needed
        """
        if not text or len(text) <= limit:
            return text or ""
            
        return text[:limit - self.TRUNCATION_SUFFIX_LEN] + self.TRUNCATION_SUFFIX
    
    def format_structured_log(self, log_data: Dict[str, Any]) -> str:
        """Format a dictionary as a structured log entry.
        
        Args:
            log_data: Dictionary of log data
            
        Returns:
            JSON-formatted string for structured logging
        """
        try:
            return json.dumps(log_data, ensure_ascii=False, default=str)
        except (TypeError, ValueError):
            return str(log_data)
    
    def enable_structured_logging(self) -> None:
        """Enable structured logging format for all messages."""
        # This could be extended to configure JSON formatters
        # for the logger handlers, but keeping it simple for now
        self.logger.info("Structured logging enabled")
    
    def set_debug(self, debug: bool) -> None:
        """Update debug mode setting.
        
        Args:
            debug: Whether debug logging is enabled
        """
        self.debug = debug
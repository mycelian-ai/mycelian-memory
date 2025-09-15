"""Control state definitions for the MycelianMemoryAgent protocol."""

from enum import Enum


class ControlState(Enum):
    """Control states for agent execution.

    These control states determine the operation sequence the agent will perform.
    """
    START_SESSION = "start_session"
    PROCESS_MESSAGE = "process_message"
    FLUSH = "flush"
    END_SESSION = "end_session"

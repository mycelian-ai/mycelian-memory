"""Conversation Processor Node - Stateless processing of haystack sessions."""

from typing import Dict, Any, List
import structlog

from ..types import LongMemEvalState, ConversationTurn
from ..agents.memory_agent import MycelianMemoryAgent

logger = structlog.get_logger(__name__)


class ConversationProcessorNode:
    """
    Stateless LangGraph node that processes haystack sessions and sends
    individual turns to the Mycelian Memory Agent for observation and storage.
    
    This node orchestrates the data flow without maintaining state itself.
    """

    def __init__(self, memory_agent: MycelianMemoryAgent):
        """
        Initialize the conversation processor.
        
        Args:
            memory_agent: Initialized Mycelian Memory Agent
        """
        self.memory_agent = memory_agent

    async def process_conversation(self, state: LongMemEvalState) -> LongMemEvalState:
        """
        Process all haystack sessions for a conversation through the Memory Agent.
        
        This is the main LangGraph node function that will be called by the workflow.
        
        Args:
            state: Current workflow state containing haystack sessions and metadata
            
        Returns:
            Updated state with ingestion_complete flag set
        """
        logger.info(
            "Starting conversation processing", 
            conversation_id=state["conversation_id"],
            num_sessions=len(state["haystack_sessions"])
        )

        try:
            # Initialize memory session with the Memory Agent
            await self.memory_agent.initialize_session(
                conversation_id=state["conversation_id"],
                memory_id=state["memory_id"]
            )

            # Process each session
            for session_idx, session in enumerate(state["haystack_sessions"]):
                await self._process_session(
                    session_idx=session_idx + 1,  # 1-based indexing
                    session=session,
                    conversation_id=state["conversation_id"],
                    memory_id=state["memory_id"]
                )

            # Finalize the memory session
            await self.memory_agent.finalize_session(
                conversation_id=state["conversation_id"],
                memory_id=state["memory_id"]
            )

            # Mark ingestion as complete
            state["ingestion_complete"] = True
            
            logger.info(
                "Conversation processing complete",
                conversation_id=state["conversation_id"]
            )

        except Exception as e:
            logger.error(
                "Error during conversation processing",
                conversation_id=state["conversation_id"],
                error=str(e)
            )
            state["ingestion_complete"] = False
            raise

        return state

    async def _process_session(
        self,
        session_idx: int,
        session: List[Dict[str, Any]],
        conversation_id: str,
        memory_id: str
    ) -> None:
        """
        Process a single session by sending each turn to the Memory Agent.
        
        Args:
            session_idx: 1-based session index
            session: List of conversation turns in the session
            conversation_id: ID of the conversation
            memory_id: ID of the memory to store turns in
        """
        logger.debug(
            "Processing session",
            session_idx=session_idx,
            conversation_id=conversation_id,
            turn_count=len(session)
        )

        # Signal session start to Memory Agent
        await self.memory_agent.start_session(
            session_idx=session_idx,
            conversation_id=conversation_id,
            memory_id=memory_id
        )

        # Process each turn in the session
        for turn_idx, turn in enumerate(session):
            await self._process_turn(
                turn=turn,
                turn_idx=turn_idx,
                session_idx=session_idx,
                memory_id=memory_id
            )

        logger.debug(
            "Session processing complete",
            session_idx=session_idx,
            conversation_id=conversation_id
        )

    async def _process_turn(
        self,
        turn: Dict[str, Any],
        turn_idx: int,
        session_idx: int,
        memory_id: str
    ) -> None:
        """
        Process a single conversation turn through the Memory Agent.
        
        Args:
            turn: Individual turn data with 'role' and 'content' keys
            turn_idx: 0-based turn index within session
            session_idx: 1-based session index
            memory_id: ID of the memory to store the turn in
        """
        role = turn.get("role", "unknown")
        content = turn.get("content", "")
        
        if not content.strip():
            logger.warning(
                "Skipping empty turn",
                turn_idx=turn_idx,
                session_idx=session_idx,
                role=role
            )
            return

        logger.info(
            "Processing turn - CALLING Memory Agent observe_turn",
            turn_idx=turn_idx,
            session_idx=session_idx,
            role=role,
            content_length=len(content),
            memory_id=memory_id,
            content_preview=content[:100]
        )

        try:
            # Send turn to Memory Agent for observation
            response = await self.memory_agent.observe_turn(
                role=role,
                content=content,
                memory_id=memory_id
            )

            logger.info(
                "Turn processed successfully - Memory Agent response received",
                turn_idx=turn_idx,
                session_idx=session_idx,
                role=role,
                response_type=type(response).__name__
            )

        except Exception as e:
            logger.error(
                "Error processing turn",
                turn_idx=turn_idx,
                session_idx=session_idx,
                role=role,
                error=str(e)
            )
            # Continue processing other turns even if one fails
            pass


# Convenience function to create the processor node function for LangGraph
async def create_conversation_processor_node(
    memory_agent: MycelianMemoryAgent
) -> callable:
    """
    Create a conversation processor node function for use in LangGraph workflows.
    
    Args:
        memory_agent: Initialized Mycelian Memory Agent
        
    Returns:
        Async function that can be used as a LangGraph node
    """
    processor = ConversationProcessorNode(memory_agent)
    
    async def processor_node(state: LongMemEvalState) -> LongMemEvalState:
        """LangGraph node function for conversation processing."""
        return await processor.process_conversation(state)
    
    return processor_node


# Alternative implementation using a simple function approach
async def conversation_processor_node_simple(
    state: LongMemEvalState,
    memory_agent: MycelianMemoryAgent
) -> LongMemEvalState:
    """
    Simple stateless function implementation of conversation processing.
    
    This is a more direct approach that can be used directly as a LangGraph node
    without the class wrapper.
    
    Args:
        state: Current workflow state
        memory_agent: Initialized Mycelian Memory Agent
        
    Returns:
        Updated state with ingestion complete
    """
    logger.info(
        "Processing haystack sessions",
        conversation_id=state["conversation_id"],
        session_count=len(state["haystack_sessions"])
    )

    # Initialize memory session
    await memory_agent.initialize_session(
        conversation_id=state["conversation_id"],
        memory_id=state["memory_id"]
    )

    # Process each session and turn
    for session_idx, session in enumerate(state["haystack_sessions"]):
        # Signal session start
        await memory_agent.start_session(
            session_idx=session_idx + 1,
            conversation_id=state["conversation_id"],
            memory_id=state["memory_id"]
        )

        # Process each turn
        for turn in session:
            role = turn.get("role", "unknown")
            content = turn.get("content", "")
            
            if content.strip():
                await memory_agent.observe_turn(
                    role=role,
                    content=content,
                    memory_id=state["memory_id"]
                )

    # Finalize memory session
    await memory_agent.finalize_session(
        conversation_id=state["conversation_id"],
        memory_id=state["memory_id"]
    )

    # Mark ingestion complete
    state["ingestion_complete"] = True
    
    logger.info(
        "Haystack processing complete",
        conversation_id=state["conversation_id"]
    )

    return state
"""QA Agent - Stateful agent for answering questions using stored memories."""

from typing import Dict, Any, Optional
from langchain_mcp_adapters.client import MultiServerMCPClient
from langgraph.prebuilt import create_react_agent
import structlog

from ..types import QAAgentResponse
from ..utils.model_factory import create_chat_model

logger = structlog.get_logger(__name__)


class QAAgent:
    """
    Stateful LangGraph agent that uses stored memories to answer questions.
    
    This agent is separate from the Mycelian Memory Agent and focuses solely
    on retrieving information from stored memories to answer benchmark questions.
    """

    def __init__(
        self,
        mcp_url: str = "http://localhost:11546/mcp",
        model: str = "gpt-4o-2024-08-06",
        transport: str = "streamable_http",
        aws_region: str = "us-west-2"
    ):
        """
        Initialize the QA Agent.
        
        Args:
            mcp_url: URL of the Mycelian MCP server
            model: High-capability LLM model for reasoning
            transport: MCP transport type
            aws_region: AWS region for Bedrock models
        """
        self.mcp_url = mcp_url
        self.model = model
        self.transport = transport
        self.aws_region = aws_region
        self._agent = None
        self._mcp_client = None

    async def initialize(self) -> None:
        """Initialize MCP client and create the LangGraph agent."""
        # Configure MCP client for memory operations
        self._mcp_client = MultiServerMCPClient({
            "mycelian": {
                "url": self.mcp_url,
                "transport": self.transport
            }
        })

        # Get available MCP tools
        qa_tools = await self._mcp_client.get_tools()

        # Create the agent with QA-focused system prompt
        system_prompt = self._build_system_prompt()
        
        # Create model instance using factory (supports both OpenAI and Bedrock)
        model_instance = create_chat_model(
            model_id=self.model,
            provider="auto",  # Auto-detect based on model_id
            aws_region=self.aws_region
        )
        
        self._agent = create_react_agent(
            model=model_instance,
            tools=qa_tools,
            prompt=system_prompt
        )

        logger.info("QA Agent initialized", model=self.model)

    def _build_system_prompt(self) -> str:
        """
        Build the system prompt for the QA Agent.
        
        Following the design document's QA agent specification.
        """
        return """You are a question answering agent. Use stored memories to answer questions.

For each question:
1. Call get_context() to retrieve current context
2. Call list_entries(limit=10) to review recent memories  
3. Call search_memories() with the question to find relevant information
4. Provide your answer based on retrieved information
5. If insufficient information, clearly state this for abstention

Always be thorough in your memory retrieval before answering. Use multiple approaches to find relevant information."""

    async def answer_question(
        self, 
        question: str, 
        memory_id: str,
        vault_id: Optional[str] = None
    ) -> QAAgentResponse:
        """
        Answer a question using stored memories.
        
        Args:
            question: The question to answer
            memory_id: ID of the memory containing relevant information
            vault_id: Optional vault ID for additional context
            
        Returns:
            QAAgentResponse with the answer and metadata
        """
        if not self._agent:
            raise RuntimeError("Agent not initialized. Call initialize() first.")

        logger.info(
            "Answering question",
            question_length=len(question),
            memory_id=memory_id
        )

        try:
            # Construct the QA message with memory context
            qa_message = self._build_qa_message(question, memory_id, vault_id)

            # Invoke the agent to answer the question
            response = await self._agent.ainvoke({
                "messages": [{"role": "user", "content": qa_message}]
            })

            # Extract the answer from the response
            answer = self._extract_answer(response)

            # Parse any reasoning or confidence if available
            reasoning = self._extract_reasoning(response)
            sources = self._extract_sources(response)

            qa_response = QAAgentResponse(
                answer=answer,
                reasoning=reasoning,
                sources=sources
            )

            logger.info(
                "Question answered successfully",
                answer_length=len(answer),
                memory_id=memory_id
            )

            return qa_response

        except Exception as e:
            logger.error(
                "Error answering question",
                memory_id=memory_id,
                error=str(e)
            )
            
            # Return error response
            return QAAgentResponse(
                answer="I encountered an error while trying to answer this question.",
                reasoning=f"Error: {str(e)}",
                sources=[]
            )

    def _build_qa_message(
        self, 
        question: str, 
        memory_id: str, 
        vault_id: Optional[str] = None
    ) -> str:
        """
        Build the message to send to the QA agent.
        
        Args:
            question: The question to answer
            memory_id: Memory ID containing information
            vault_id: Optional vault ID
            
        Returns:
            Formatted message for the agent
        """
        message_parts = [
            f"Answer this question using stored memories:",
            f"Question: {question}",
            f"Memory ID: {memory_id}"
        ]

        if vault_id:
            message_parts.append(f"Vault ID: {vault_id}")

        return "\n\n".join(message_parts)

    def _extract_answer(self, response: Dict[str, Any]) -> str:
        """
        Extract the final answer from the agent response.
        
        Args:
            response: Raw response from the LangGraph agent
            
        Returns:
            The answer text
        """
        try:
            # Get the last message from the agent
            messages = response.get("messages", [])
            if messages:
                last_message = messages[-1]
                return last_message.get("content", "No answer provided.")
            
            return "No response generated."
            
        except Exception as e:
            logger.warning("Error extracting answer", error=str(e))
            return "Error extracting answer from response."

    def _extract_reasoning(self, response: Dict[str, Any]) -> Optional[str]:
        """
        Extract reasoning or chain of thought from the response.
        
        Args:
            response: Raw response from the agent
            
        Returns:
            Reasoning text if available
        """
        # This could be enhanced to parse specific reasoning patterns
        # from the agent's response
        return None

    def _extract_sources(self, response: Dict[str, Any]) -> list:
        """
        Extract information about sources used in the answer.
        
        Args:
            response: Raw response from the agent
            
        Returns:
            List of source references
        """
        # This could be enhanced to identify which MCP tool calls
        # were made and what information was retrieved
        sources = []
        
        # Look for tool calls in the response
        try:
            messages = response.get("messages", [])
            for message in messages:
                if message.get("type") == "tool_use" or "tool_calls" in message:
                    # Add tool call information as sources
                    tool_name = message.get("name", "unknown_tool")
                    sources.append(f"Retrieved from {tool_name}")
        except Exception:
            pass
            
        return sources

    async def close(self) -> None:
        """Close the MCP client connection."""
        # Note: MultiServerMCPClient doesn't have a close() method
        logger.info("QA Agent cleanup complete")


# Convenience function for creating and initializing the QA agent
async def create_qa_agent(
    mcp_url: str = "http://localhost:11546/mcp",
    model: str = "gpt-4o-2024-08-06",
    transport: str = "streamable_http"
) -> QAAgent:
    """
    Create and initialize a QA Agent.
    
    Args:
        mcp_url: URL of the Mycelian MCP server
        model: LLM model for the agent (high-capability for reasoning)
        transport: MCP transport type
        
    Returns:
        Initialized QAAgent instance
    """
    agent = QAAgent(mcp_url=mcp_url, model=model, transport=transport)
    await agent.initialize()
    return agent


# Function to create QA agent node for LangGraph workflows
async def create_qa_agent_node(qa_agent: QAAgent) -> callable:
    """
    Create a QA agent node function for use in LangGraph workflows.
    
    Args:
        qa_agent: Initialized QA Agent
        
    Returns:
        Async function that can be used as a LangGraph node
    """
    async def qa_agent_node(state) -> dict:
        """LangGraph node function for QA processing."""
        try:
            # Answer the question using the QA agent
            qa_response = await qa_agent.answer_question(
                question=state["question"],
                memory_id=state["memory_id"],
                vault_id=state.get("vault_id")
            )
            
            # Update state with the response
            state["qa_response"] = qa_response.answer
            
            # Store additional metadata if needed
            if hasattr(state, 'qa_metadata'):
                state["qa_metadata"] = {
                    "reasoning": qa_response.reasoning,
                    "sources": qa_response.sources
                }

            logger.info(
                "QA node processing complete",
                conversation_id=state["conversation_id"]
            )

        except Exception as e:
            logger.error(
                "Error in QA agent node",
                conversation_id=state["conversation_id"],
                error=str(e)
            )
            state["qa_response"] = "Error occurred during question answering."

        return state
    
    return qa_agent_node
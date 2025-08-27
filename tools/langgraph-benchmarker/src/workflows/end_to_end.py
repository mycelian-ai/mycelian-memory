"""End-to-End LangGraph Workflow for LongMemEval benchmarking."""

from typing import Dict, Any, List
from langgraph.graph import StateGraph, START, END
import structlog

from ..types import LongMemEvalState, BenchmarkConfig
from ..agents.memory_agent import MycelianMemoryAgent
from ..agents.qa_agent import QAAgent
from ..nodes.setup_node import create_setup_node
from ..nodes.conversation_processor import create_conversation_processor_node
from ..nodes.evaluation_node import LLMEvaluator, create_evaluation_node
from ..nodes.results_node import create_results_node
from langchain_mcp_adapters.client import MultiServerMCPClient

logger = structlog.get_logger(__name__)


class LongMemEvalWorkflow:
    """
    End-to-End LangGraph workflow orchestrating the complete LongMemEval benchmark process.
    
    This workflow manages the three main phases:
    1. Setup: Initialize vault and memory
    2. Ingestion: Process haystack sessions through Memory Agent  
    3. QA: Answer questions using stored memories
    4. Evaluation: Judge answers with Claude Haiku
    5. Results: Aggregate final benchmark metrics
    """

    def __init__(self, config: BenchmarkConfig):
        """
        Initialize the workflow with configuration.
        
        Args:
            config: Benchmark configuration including models, URLs, etc.
        """
        self.config = config
        self.workflow = None
        self._components = {}

    async def initialize(self) -> None:
        """Initialize all workflow components and create the LangGraph workflow."""
        logger.info("Initializing LongMemEval workflow")

        try:
            # Initialize MCP client
            mcp_client = MultiServerMCPClient({
                "mycelian": {
                    "url": self.config.mcp_url,
                    "transport": self.config.mcp_transport
                }
            })
            self._components['mcp_client'] = mcp_client

            # Initialize Mycelian Memory Agent
            memory_agent = MycelianMemoryAgent(
                mcp_url=self.config.mcp_url,
                model=self.config.memory_agent_model,
                transport=self.config.mcp_transport,
                aws_region=self.config.aws_region
            )
            await memory_agent.initialize()
            self._components['memory_agent'] = memory_agent

            # Initialize QA Agent
            qa_agent = QAAgent(
                mcp_url=self.config.mcp_url,
                model=self.config.qa_agent_model,
                transport=self.config.mcp_transport,
                aws_region=self.config.aws_region
            )
            await qa_agent.initialize()
            self._components['qa_agent'] = qa_agent

            # Initialize LLM Evaluator
            evaluator = LLMEvaluator(
                model_id=self.config.evaluation_model,
                aws_region=self.config.aws_region,
                provider="auto"
            )
            self._components['evaluator'] = evaluator

            # Build the workflow graph
            await self._build_workflow()

            logger.info("LongMemEval workflow initialized successfully")

        except Exception as e:
            logger.error("Failed to initialize workflow", error=str(e))
            raise

    async def _build_workflow(self) -> None:
        """Build the LangGraph workflow with all nodes and edges."""
        
        # Create the state graph
        workflow = StateGraph(LongMemEvalState)

        # Create node functions with initialized components
        setup_node = create_setup_node(
            mcp_client=self._components['mcp_client'],
            vault_name=self.config.vault_name
        )

        conversation_processor_node = await create_conversation_processor_node(
            memory_agent=self._components['memory_agent']
        )

        qa_agent_node = await self._create_qa_agent_node()

        evaluation_node = create_evaluation_node(
            evaluator=self._components['evaluator']
        )

        results_node = create_results_node(include_detailed_metrics=True)

        # Add nodes to the workflow
        workflow.add_node("setup", setup_node)
        workflow.add_node("ingestion", conversation_processor_node)  
        workflow.add_node("qa", qa_agent_node)
        workflow.add_node("evaluation", evaluation_node)
        workflow.add_node("results", results_node)

        # Define the workflow edges (sequential flow)
        workflow.add_edge(START, "setup")
        workflow.add_edge("setup", "ingestion")
        workflow.add_edge("ingestion", "qa")
        workflow.add_edge("qa", "evaluation")
        workflow.add_edge("evaluation", "results")
        workflow.add_edge("results", END)

        # Compile the workflow
        self.workflow = workflow.compile()

        logger.info("LangGraph workflow compiled successfully")

    async def _create_qa_agent_node(self) -> callable:
        """Create the QA agent node function."""
        qa_agent = self._components['qa_agent']

        async def qa_agent_node(state: LongMemEvalState) -> LongMemEvalState:
            """QA agent node that answers questions using stored memories."""
            logger.info(
                "Starting QA processing",
                conversation_id=state["conversation_id"]
            )

            try:
                # Answer the question using the QA agent
                qa_response = await qa_agent.answer_question(
                    question=state["question"],
                    memory_id=state["memory_id"],
                    vault_id=state.get("vault_id")
                )

                # Update state with the response
                state["qa_response"] = qa_response.answer

                logger.info(
                    "QA processing complete",
                    conversation_id=state["conversation_id"],
                    answer_length=len(qa_response.answer)
                )

            except Exception as e:
                logger.error(
                    "Error in QA processing",
                    conversation_id=state["conversation_id"],
                    error=str(e)
                )
                state["qa_response"] = f"Error occurred during question answering: {str(e)}"

            return state

        return qa_agent_node

    async def _create_benchmark_memory(self) -> tuple[str, str]:
        """
        Create a shared memory for the entire benchmark run.
        
        Returns:
            Tuple of (vault_id, memory_id) for the shared benchmark memory
        """
        from ..utils.mcp_client import get_or_create_vault
        import time
        
        mcp_client = self._components['mcp_client']
        vault_name = self.config.vault_name
        
        logger.info("Creating shared benchmark memory", vault_name=vault_name)
        
        try:
            # Create or get the benchmark vault
            vault_id = await get_or_create_vault(
                client=mcp_client,
                vault_name=vault_name,
                description=f"Vault for LongMemEval benchmark runs"
            )
            
            # Create shared memory for this benchmark run
            timestamp = int(time.time())
            memory_title = f"longmemeval-run-{timestamp}"
            memory_id = await mcp_client.create_memory(
                vault_id=vault_id,
                title=memory_title,
                memory_type="PROJECT",
                description=f"Shared memory for LongMemEval benchmark run at {timestamp}"
            )
            
            logger.info(
                "Shared benchmark memory created",
                vault_id=vault_id,
                memory_id=memory_id,
                memory_title=memory_title
            )
            
            return vault_id, memory_id
            
        except Exception as e:
            logger.error("Failed to create benchmark memory", error=str(e))
            raise

    async def run_single_conversation(
        self, 
        conversation_data: Dict[str, Any],
        run_id: str
    ) -> Dict[str, Any]:
        """
        Run the workflow for a single conversation.
        
        Args:
            conversation_data: Single conversation from LongMemEval dataset
            run_id: Unique identifier for this benchmark run
            
        Returns:
            Benchmark results for this conversation
        """
        if not self.workflow:
            raise RuntimeError("Workflow not initialized. Call initialize() first.")

        conversation_id = conversation_data["question_id"]
        
        logger.info(
            "Running workflow for conversation",
            conversation_id=conversation_id
        )

        try:
            # Prepare initial state
            initial_state = LongMemEvalState(
                conversation_id=conversation_id,
                haystack_sessions=conversation_data["haystack_sessions"],
                question=conversation_data["question"],
                expected_answer=conversation_data["answer"],
                memory_id="",  # Will be set by setup node
                vault_id="",   # Will be set by setup node
                run_id=run_id, # Unique run identifier
                ingestion_complete=False,
                qa_response="",
                evaluation_score=0.0,
                benchmark_results={}
            )

            # Execute the workflow with higher recursion limit for Memory Agent
            final_state = await self.workflow.ainvoke(
                initial_state,
                config={"recursion_limit": 100}  # Allow Memory Agent to complete its tool calls
            )

            # Return the benchmark results
            results = final_state["benchmark_results"]
            
            logger.info(
                "Workflow completed successfully",
                conversation_id=conversation_id,
                score=results.get("evaluation_score", 0.0)
            )

            return results

        except Exception as e:
            logger.error(
                "Error running workflow",
                conversation_id=conversation_id,
                error=str(e)
            )
            
            # Return error result
            return {
                "conversation_id": conversation_id,
                "error": str(e),
                "evaluation_score": 0.0,
                "correct": False
            }

    async def run_full_benchmark(
        self, 
        dataset: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """
        Run the complete benchmark on a LongMemEval dataset.
        
        Args:
            dataset: Full LongMemEval dataset
            
        Returns:
            Aggregated benchmark results
        """
        # Generate unique run ID for this benchmark run
        import uuid
        run_id = str(uuid.uuid4())[:8]  # Short UUID for readability
        
        logger.info(
            "Starting full benchmark",
            run_id=run_id,
            total_conversations=len(dataset)
        )

        results = []
        correct_count = 0
        
        for idx, conversation in enumerate(dataset):
            logger.info(
                "Processing conversation",
                index=idx + 1,
                total=len(dataset),
                conversation_id=conversation["question_id"]
            )

            try:
                # Run workflow for this conversation (each gets its own memory)
                result = await self.run_single_conversation(conversation, run_id)
                results.append(result)
                
                # Track correct answers
                if result.get("correct", False):
                    correct_count += 1

            except Exception as e:
                logger.error(
                    "Error processing conversation",
                    index=idx + 1,
                    conversation_id=conversation.get("question_id", "unknown"),
                    error=str(e)
                )
                
                # Add error result
                results.append({
                    "conversation_id": conversation.get("question_id", f"error_{idx}"),
                    "error": str(e),
                    "evaluation_score": 0.0,
                    "correct": False
                })

        # Calculate overall metrics
        total_questions = len(results)
        overall_accuracy = correct_count / total_questions if total_questions > 0 else 0.0

        # Group results by question type
        accuracy_by_type = {}
        counts_by_type = {}
        
        for result in results:
            qtype = result.get("question_type", "unknown")
            if qtype not in accuracy_by_type:
                accuracy_by_type[qtype] = []
                counts_by_type[qtype] = 0
            
            accuracy_by_type[qtype].append(result.get("correct", False))
            counts_by_type[qtype] += 1

        # Calculate per-type accuracy
        final_accuracy_by_type = {}
        for qtype, correct_list in accuracy_by_type.items():
            final_accuracy_by_type[qtype] = sum(correct_list) / len(correct_list)

        benchmark_results = {
            "run_id": run_id,
            "total_questions": total_questions,
            "correct_answers": correct_count,
            "overall_accuracy": overall_accuracy,
            "accuracy_by_type": final_accuracy_by_type,
            "counts_by_type": counts_by_type,
            "question_results": results,
            "config": self.config
        }

        logger.info(
            "Full benchmark completed",
            total_questions=total_questions,
            correct_answers=correct_count,
            overall_accuracy=overall_accuracy
        )

        return benchmark_results

    async def close(self) -> None:
        """Clean up workflow components."""
        logger.info("Closing workflow components")

        if 'memory_agent' in self._components:
            await self._components['memory_agent'].close()
        
        if 'qa_agent' in self._components:
            await self._components['qa_agent'].close()
            
        # Note: MultiServerMCPClient doesn't have a close() method

        logger.info("Workflow cleanup complete")


async def create_longmemeval_workflow(config: BenchmarkConfig) -> LongMemEvalWorkflow:
    """
    Create and initialize a LongMemEval workflow.
    
    Args:
        config: Benchmark configuration
        
    Returns:
        Initialized LongMemEvalWorkflow
    """
    workflow = LongMemEvalWorkflow(config)
    await workflow.initialize()
    return workflow
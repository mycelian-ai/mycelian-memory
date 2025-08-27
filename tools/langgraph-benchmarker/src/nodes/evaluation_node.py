"""Evaluation Node - Multi-provider judge for answer evaluation."""

import json
from typing import Dict, Any, Optional
import structlog

from ..types import LongMemEvalState, EvaluationResult
from ..utils.model_factory import create_chat_model

logger = structlog.get_logger(__name__)


class LLMEvaluator:
    """
    Evaluation service supporting multiple LLM providers.
    
    Uses model factory to support both OpenAI and Bedrock models for evaluation.
    """

    def __init__(
        self,
        model_id: str = "anthropic.claude-3-haiku-20240307-v1:0",
        aws_region: str = "us-east-1",
        provider: str = "auto"
    ):
        """
        Initialize the evaluator.
        
        Args:
            model_id: Model identifier (e.g., "gpt-4o-mini", "anthropic.claude-3-haiku-20240307-v1:0")
            aws_region: AWS region for Bedrock models
            provider: Provider name or "auto" for auto-detection
        """
        self.model_id = model_id
        self.aws_region = aws_region
        self.provider = provider
        self._model = None
        
        logger.info(
            "LLM evaluator initialized",
            model_id=model_id,
            provider=provider
        )

    def _get_model(self):
        """Lazy initialization of the model."""
        if self._model is None:
            self._model = create_chat_model(
                model_id=self.model_id,
                provider=self.provider,
                aws_region=self.aws_region
            )
        return self._model

    async def evaluate_answer(
        self,
        question: str,
        expected_answer: str,
        model_response: str,
        question_type: str,
        is_abstention: bool = False
    ) -> EvaluationResult:
        """
        Evaluate a model response against the expected answer.
        
        Args:
            question: The original question
            expected_answer: Expected/correct answer
            model_response: Response from the QA agent
            question_type: Type of question for appropriate evaluation prompt
            is_abstention: Whether this is an abstention question
            
        Returns:
            EvaluationResult with score and reasoning
        """
        try:
            # Generate evaluation prompt based on LongMemEval methodology
            eval_prompt = self._get_evaluation_prompt(
                question=question,
                expected_answer=expected_answer,
                model_response=model_response,
                question_type=question_type,
                is_abstention=is_abstention
            )
            
            # Call LLM via LangChain
            model = self._get_model()
            response = await model.ainvoke(eval_prompt)
            
            # Parse the evaluation response (LangChain returns a message object)
            response_text = response.content if hasattr(response, 'content') else str(response)
            score = self._parse_evaluation_score(response_text)
            
            result = EvaluationResult(
                score=score,
                reasoning=response_text,
                model_used=self.model_id,
                evaluation_prompt=eval_prompt
            )
            
            logger.debug(
                "Answer evaluated",
                score=score,
                question_type=question_type,
                is_abstention=is_abstention
            )
            
            return result
            
        except Exception as e:
            logger.error("Error during evaluation", error=str(e))
            return EvaluationResult(
                score=0.0,
                reasoning=f"Evaluation error: {str(e)}",
                model_used=self.model_id,
                evaluation_prompt=""
            )

    def _get_evaluation_prompt(
        self,
        question: str,
        expected_answer: str,
        model_response: str,
        question_type: str,
        is_abstention: bool
    ) -> str:
        """
        Generate evaluation prompt based on LongMemEval methodology.
        
        This follows the same evaluation logic as the original evaluate_qa.py
        but adapted for Claude Haiku.
        """
        if is_abstention:
            template = """I will give you an unanswerable question, an explanation, and a response from a model. Please answer yes if the model correctly identifies the question as unanswerable. The model could say that the information is incomplete, or some other information is given but the asked information is not.

Question: {question}

Explanation: {expected_answer}

Model Response: {model_response}

Does the model correctly identify the question as unanswerable? Answer yes or no only."""
            
        elif question_type in ['single-session-user', 'single-session-assistant', 'multi-session']:
            template = """I will give you a question, a correct answer, and a response from a model. Please answer yes if the response contains the correct answer. Otherwise, answer no. If the response is equivalent to the correct answer or contains all the intermediate steps to get the correct answer, you should also answer yes. If the response only contains a subset of the information required by the answer, answer no.

Question: {question}

Correct Answer: {expected_answer}

Model Response: {model_response}

Is the model response correct? Answer yes or no only."""

        elif question_type == 'temporal-reasoning':
            template = """I will give you a question, a correct answer, and a response from a model. Please answer yes if the response contains the correct answer. Otherwise, answer no. If the response is equivalent to the correct answer or contains all the intermediate steps to get the correct answer, you should also answer yes. If the response only contains a subset of the information required by the answer, answer no. In addition, do not penalize off-by-one errors for the number of days. If the question asks for the number of days/weeks/months, etc., and the model makes off-by-one errors (e.g., predicting 19 days when the answer is 18), the model's response is still correct.

Question: {question}

Correct Answer: {expected_answer}

Model Response: {model_response}

Is the model response correct? Answer yes or no only."""

        elif question_type == 'knowledge-update':
            template = """I will give you a question, a correct answer, and a response from a model. Please answer yes if the response contains the correct answer. Otherwise, answer no. If the response contains some previous information along with an updated answer, the response should be considered as correct as long as the updated answer is the required answer.

Question: {question}

Correct Answer: {expected_answer}

Model Response: {model_response}

Is the model response correct? Answer yes or no only."""

        elif question_type == 'single-session-preference':
            template = """I will give you a question, a rubric for desired personalized response, and a response from a model. Please answer yes if the response satisfies the desired response. Otherwise, answer no. The model does not need to reflect all the points in the rubric. The response is correct as long as it recalls and utilizes the user's personal information correctly.

Question: {question}

Rubric: {expected_answer}

Model Response: {model_response}

Is the model response correct? Answer yes or no only."""

        else:
            # Default template for unknown question types
            template = """I will give you a question, a correct answer, and a response from a model. Please answer yes if the response contains the correct answer. Otherwise, answer no.

Question: {question}

Correct Answer: {expected_answer}

Model Response: {model_response}

Is the model response correct? Answer yes or no only."""

        return template.format(
            question=question,
            expected_answer=expected_answer,
            model_response=model_response
        )

    async def _call_bedrock(self, prompt: str) -> str:
        """
        Call Claude Haiku via AWS Bedrock.
        
        Args:
            prompt: Evaluation prompt
            
        Returns:
            Response from Claude Haiku
        """
        # Construct the request body for Claude
        body = {
            "anthropic_version": "bedrock-2023-05-31",
            "max_tokens": 10,
            "temperature": 0,
            "messages": [
                {
                    "role": "user",
                    "content": prompt
                }
            ]
        }

        try:
            response = self.bedrock_client.invoke_model(
                modelId=self.model_id,
                body=json.dumps(body),
                contentType="application/json"
            )
            
            # Parse response
            response_body = json.loads(response["body"].read())
            
            if "content" in response_body and len(response_body["content"]) > 0:
                return response_body["content"][0]["text"].strip()
            else:
                raise ValueError("No content in Bedrock response")
                
        except Exception as e:
            logger.error("Bedrock API call failed", error=str(e))
            raise

    def _parse_evaluation_score(self, response: str) -> float:
        """
        Parse the evaluation score from Claude's response.
        
        Args:
            response: Raw response from Claude
            
        Returns:
            Score as 1.0 (correct) or 0.0 (incorrect)
        """
        response_lower = response.lower().strip()
        
        # Look for "yes" in the response (indicating correct answer)
        if "yes" in response_lower:
            return 1.0
        else:
            return 0.0


async def evaluation_node(
    state: LongMemEvalState,
    evaluator: LLMEvaluator
) -> LongMemEvalState:
    """
    Evaluate QA response using Claude Haiku judge.
    
    This LangGraph node function evaluates the QA agent's response
    against the expected answer using Claude Haiku via AWS Bedrock.
    
    Args:
        state: Current workflow state with QA response
        evaluator: Initialized Claude Haiku evaluator
        
    Returns:
        Updated state with evaluation score
    """
    conversation_id = state["conversation_id"]
    
    logger.info(
        "Evaluating answer",
        conversation_id=conversation_id,
        response_length=len(state["qa_response"])
    )

    try:
        # Determine question type and abstention status
        question_type = _determine_question_type(conversation_id)
        is_abstention = "_abs" in conversation_id
        
        # Evaluate the response
        evaluation_result = await evaluator.evaluate_answer(
            question=state["question"],
            expected_answer=state["expected_answer"],
            model_response=state["qa_response"],
            question_type=question_type,
            is_abstention=is_abstention
        )
        
        # Update state with evaluation score
        state["evaluation_score"] = evaluation_result.score
        
        logger.info(
            "Evaluation complete",
            conversation_id=conversation_id,
            score=evaluation_result.score,
            question_type=question_type
        )

    except Exception as e:
        logger.error(
            "Error during evaluation",
            conversation_id=conversation_id,
            error=str(e)
        )
        state["evaluation_score"] = 0.0

    return state


def _determine_question_type(conversation_id: str) -> str:
    """
    Determine question type from conversation_id based on LongMemEval patterns.
    
    Args:
        conversation_id: The conversation/question identifier
        
    Returns:
        Question type string
    """
    # Handle abstention questions
    if conversation_id.endswith('_abs'):
        return 'abstention'
    
    # Map based on common patterns in LongMemEval
    type_mapping = {
        'single_hop': 'single-session-user',
        'implicit_preference_v2': 'single-session-preference',
        'assistant_previnfo': 'single-session-assistant',
        'two_hop': 'multi-session',
        'multi_session_synthesis': 'multi-session',
        'temp_reasoning_implicit': 'temporal-reasoning',
        'temp_reasoning_explicit': 'temporal-reasoning',
        'knowledge_update': 'knowledge-update'
    }
    
    # Try to match based on conversation_id patterns
    for pattern, qtype in type_mapping.items():
        if pattern in conversation_id:
            return qtype
    
    # Default fallback
    return 'multi-session'


def create_evaluation_node(evaluator: LLMEvaluator) -> callable:
    """
    Create an evaluation node function for LangGraph workflows.
    
    Args:
        evaluator: Initialized Claude Haiku evaluator
        
    Returns:
        Async function for use as LangGraph node
    """
    async def node_function(state: LongMemEvalState) -> LongMemEvalState:
        return await evaluation_node(state, evaluator)
    
    return node_function
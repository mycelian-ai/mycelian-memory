"""Model factory for creating language models from different providers."""

from typing import Dict, Any
from langchain_core.language_models.chat_models import BaseChatModel
import structlog

logger = structlog.get_logger(__name__)


def create_chat_model(
    model_id: str,
    provider: str = "auto",
    aws_region: str = "us-west-2",
    **kwargs
) -> BaseChatModel:
    """
    Create a chat model from different providers.
    
    Args:
        model_id: Model identifier (e.g., "gpt-4o-mini", "anthropic.claude-3-haiku-20240307-v1:0")
        provider: Provider name ("openai", "bedrock", or "auto" for auto-detection)
        aws_region: AWS region for Bedrock models
        **kwargs: Additional model parameters
        
    Returns:
        Initialized chat model instance
        
    Raises:
        ValueError: If provider cannot be determined or is unsupported
    """
    # Auto-detect provider if not specified
    if provider == "auto":
        provider = _detect_provider(model_id)
    
    logger.info(
        "Creating chat model",
        model_id=model_id,
        provider=provider,
        aws_region=aws_region if provider == "bedrock" else None
    )
    
    if provider == "openai":
        return _create_openai_model(model_id, **kwargs)
    elif provider == "bedrock":
        return _create_bedrock_model(model_id, aws_region, **kwargs)
    else:
        raise ValueError(f"Unsupported provider: {provider}")


def _detect_provider(model_id: str) -> str:
    """
    Auto-detect provider based on model ID patterns.
    
    Args:
        model_id: Model identifier
        
    Returns:
        Provider name
    """
    if model_id.startswith("gpt-") or model_id.startswith("o1-"):
        return "openai"
    elif model_id.startswith("anthropic.") or model_id.startswith("meta.") or model_id.startswith("amazon."):
        return "bedrock"
    else:
        # Default fallback - try OpenAI format first
        return "openai"


def _create_openai_model(model_id: str, **kwargs) -> BaseChatModel:
    """Create OpenAI chat model."""
    try:
        from langchain_openai import ChatOpenAI
        
        return ChatOpenAI(
            model=model_id,
            **kwargs
        )
    except ImportError as e:
        raise ImportError(
            "langchain-openai is required for OpenAI models. "
            "Install with: pip install langchain-openai"
        ) from e


def _create_bedrock_model(model_id: str, aws_region: str, **kwargs) -> BaseChatModel:
    """Create AWS Bedrock chat model."""
    try:
        from langchain_aws import ChatBedrock
        
        return ChatBedrock(
            model_id=model_id,
            region_name=aws_region,
            **kwargs
        )
    except ImportError as e:
        raise ImportError(
            "langchain-aws is required for Bedrock models. "
            "Install with: pip install langchain-aws"
        ) from e


def get_supported_providers() -> Dict[str, Dict[str, Any]]:
    """
    Get information about supported providers.
    
    Returns:
        Dictionary with provider info and example models
    """
    return {
        "openai": {
            "description": "OpenAI models via API",
            "examples": ["gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"],
            "required_env": ["OPENAI_API_KEY"],
            "required_packages": ["langchain-openai"]
        },
        "bedrock": {
            "description": "AWS Bedrock models",
            "examples": [
                "anthropic.claude-3-haiku-20240307-v1:0",
                "anthropic.claude-3-sonnet-20240229-v1:0",
                "meta.llama3-70b-instruct-v1:0"
            ],
            "required_env": ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"],
            "required_packages": ["langchain-aws", "boto3"]
        }
    }


def validate_model_config(model_id: str, provider: str = "auto") -> Dict[str, Any]:
    """
    Validate model configuration and return provider info.
    
    Args:
        model_id: Model identifier
        provider: Provider name or "auto"
        
    Returns:
        Dictionary with validation results and provider info
    """
    detected_provider = _detect_provider(model_id) if provider == "auto" else provider
    providers_info = get_supported_providers()
    
    if detected_provider not in providers_info:
        return {
            "valid": False,
            "provider": detected_provider,
            "error": f"Unsupported provider: {detected_provider}"
        }
    
    provider_info = providers_info[detected_provider]
    
    return {
        "valid": True,
        "provider": detected_provider,
        "model_id": model_id,
        "provider_info": provider_info
    }
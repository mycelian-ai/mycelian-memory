"""Multi-provider model support for LongMemEval benchmarker.

Supports:
- OpenAI (GPT models)
- Google Vertex AI (Gemini models)
- OpenRouter (Multiple model providers via unified API)

Provider specification format: "provider:model"
Examples:
  - "openai:gpt-5-nano-2025-08-07"
  - "openai:gpt-5-2025-08-07"
  - "vertex-ai:gemini-2.5-flash-lite"
  - "openrouter:anthropic/claude-3.5-sonnet"
  - "openrouter:google/gemini-pro-1.5"
  - "gpt-5-nano-2025-08-07" (backward compatible, defaults to OpenAI)
"""

import os
from typing import Optional
from langchain.chat_models import init_chat_model
from langchain_core.language_models.chat_models import BaseChatModel
from pathlib import Path

# Load environment variables from .env file if it exists
try:
    from dotenv import load_dotenv
    # Look for .env in the benchmarker directory
    env_path = Path(__file__).parent.parent / '.env'
    if env_path.exists():
        load_dotenv(env_path)
    else:
        # Also try current working directory
        load_dotenv()
except ImportError:
    # python-dotenv not installed, rely on system environment
    pass


def get_chat_model(model_spec: str, **kwargs) -> BaseChatModel:
    """Get a chat model from a model specification.

    Args:
        model_spec: Model specification string
                   Format: "provider:model" or just "model" (defaults to OpenAI)
                   Examples:
                   - "openai:gpt-5-nano-2025-08-07"
                   - "vertex-ai:gemini-2.5-flash-lite"
                   - "gpt-5-nano-2025-08-07" (defaults to OpenAI)
        **kwargs: Additional arguments to pass to the model

    Returns:
        Chat model instance with built-in retry
    """
    # Parse model specification
    if ":" in model_spec:
        provider, model_id = model_spec.split(":", 1)
    else:
        # Backward compatibility - default to OpenAI
        provider, model_id = "openai", model_spec

    # Set default max_retries if not provided
    if "max_retries" not in kwargs:
        kwargs["max_retries"] = 6

    # Handle different providers
    if provider == "openai":
        # Ensure API key is set
        api_key = os.getenv("OPENAI_API_KEY")
        if not api_key:
            raise ValueError("OPENAI_API_KEY environment variable not set")

        # Use init_chat_model which will use langchain_openai.ChatOpenAI
        return init_chat_model(
            model_id,
            model_provider="openai",
            api_key=api_key,
            **kwargs
        )

    elif provider == "openrouter":
        # OpenRouter uses OpenAI-compatible API with different base URL
        api_key = os.getenv("OPENROUTER_API_KEY")
        if not api_key:
            raise ValueError("OPENROUTER_API_KEY environment variable not set")

        # OpenRouter requires specific headers and base URL
        from langchain_openai import ChatOpenAI

        # Get optional site name and app name for OpenRouter tracking
        site_name = os.getenv("OPENROUTER_SITE_NAME", "longmemeval-benchmarker")
        app_name = os.getenv("OPENROUTER_APP_NAME", "LongMemEval")

        return ChatOpenAI(
            model=model_id,
            openai_api_key=api_key,
            openai_api_base="https://openrouter.ai/api/v1",
            default_headers={
                "HTTP-Referer": site_name,
                "X-Title": app_name,
            },
            **kwargs
        )

    elif provider == "vertex-ai":
        # Check for required environment variables
        project_id = os.getenv("VERTEX_AI_PROJECT_ID") or os.getenv("GCP_PROJECT_ID")
        if not project_id:
            raise ValueError(
                "VERTEX_AI_PROJECT_ID or GCP_PROJECT_ID environment variable not set. "
                "Please set one of these to your Google Cloud project ID."
            )

        # Import here to avoid requiring google-cloud libs if not using Vertex
        try:
            from langchain_google_vertexai import ChatVertexAI
        except ImportError:
            raise ImportError(
                "langchain-google-vertexai not installed. "
                "Install with: pip install langchain-google-vertexai"
            )

        # Map model names to Vertex AI model IDs
        model_map = {
            "gemini-2.5-flash-lite": "gemini-2.5-flash-lite",
            "gemini-1.5-flash": "gemini-1.5-flash",
            "gemini-1.5-pro": "gemini-1.5-pro",
        }
        vertex_model_id = model_map.get(model_id, model_id)

        # Create Vertex AI model
        return ChatVertexAI(
            model=vertex_model_id,
            project=project_id,
            location="global",  # Global endpoint
            **kwargs
        )

    else:
        raise ValueError(
            f"Unknown provider: {provider}. "
            f"Supported providers: 'openai', 'vertex-ai', 'openrouter'"
        )

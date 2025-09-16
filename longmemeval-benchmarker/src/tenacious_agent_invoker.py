"""
DEPRECATED: This module is currently unused in favor of LangChain's built-in retry mechanisms.

We are using LangChain's built-in retry handling (max_retries parameter) which is configured
on all model initializations. We will evaluate if this default handling is sufficient for our
needs. If we encounter issues with rate limiting or need more sophisticated retry logic
(exponential backoff, custom error detection, retry logging), we may reactivate and use this
module based on our results.

This module provides:
- Custom exponential backoff with jitter
- Configurable retry schedules via environment variables
- Detailed retry logging
- Support for both OpenAI and Vertex AI error patterns

To reactivate if needed:
1. Import invoke_with_backoff from this module
2. Wrap model calls with invoke_with_backoff
3. Configure LME_LLM_BACKOFF_SCHEDULE environment variable for custom schedules
"""

from __future__ import annotations

from typing import Callable, List, Optional, Any
import os
import time as _time
import random as _random


DEFAULT_BACKOFF_SCHEDULE: List[float] = [5.0, 30.0, 120.0, 300.0, 600.0]


# LLM retryable error patterns (OpenAI and Vertex AI)
RETRYABLE_LLM_PATTERNS = {
    # OpenAI patterns
    "rate_limit",
    "429",
    "500",
    "502",
    "503",
    "504",
    "timeout",
    "connection",
    "server_error",
    "internal_server_error",
    "bad_gateway",
    "service_unavailable",
    "gateway_timeout",
    # Vertex AI/Google patterns
    "resource_exhausted",
    "resource has been exhausted",
    "quota exceeded",
    "deadline exceeded",
    "unavailable",
    "aborted",
    # Note: insufficient_quota is typically not quickly recoverable; handle separately below
}


def backoff_schedule_from_env(env_key: str = "LME_LLM_BACKOFF_SCHEDULE") -> List[float]:
    raw = os.environ.get(env_key, "").strip()
    if not raw:
        return list(DEFAULT_BACKOFF_SCHEDULE)
    try:
        vals = [float(s.strip()) for s in raw.split(",") if s.strip()]
        return vals if vals else list(DEFAULT_BACKOFF_SCHEDULE)
    except Exception:
        return list(DEFAULT_BACKOFF_SCHEDULE)


def is_retryable_llm_error(exc: Exception) -> bool:
    """Check if an exception is a retryable LLM error (OpenAI or Vertex AI).

    Checks for:
    - OpenAI RateLimitError (from openai package)
    - Google/Vertex AI exceptions (ResourceExhausted, ServiceUnavailable, etc.)
    - HTTP status codes in error messages (429, 5xx)
    - Common error patterns (rate_limit, timeout, etc.)
    - LangChain model provider inference errors (likely transient)
    """
    exc_type = type(exc).__name__
    exc_str = str(exc).lower()

    # Check for LangChain model provider inference error (likely transient/throttling)
    if "unable to infer model provider" in exc_str:
        return True

    # Check for Google/Vertex AI exception types
    google_retryable_types = [
        "ResourceExhausted",  # 429 equivalent
        "ServiceUnavailable",  # 503 equivalent
        "DeadlineExceeded",  # timeout
        "Internal",  # 500 equivalent
        "Aborted",  # transient error
        "Unavailable"  # service unavailable
    ]
    if any(err_type in exc_type for err_type in google_retryable_types):
        return True

    # Check for OpenAI-specific exception types
    if "ratelimiterror" in exc_type.lower():
        return True
    if "timeout" in exc_type.lower():
        return True
    if "connectionerror" in exc_type.lower():
        return True
    if "apierror" in exc_type.lower() and any(p in exc_str for p in ["500", "502", "503", "504"]):
        return True

    # Check for HTTP status codes in the error message
    if "429" in exc_str or "rate" in exc_str and "limit" in exc_str:
        return True
    if any(f"50{i}" in exc_str for i in range(5)):  # 500-504
        return True

    # Check for common error patterns (excluding insufficient_quota here)
    for pattern in RETRYABLE_LLM_PATTERNS:
        if pattern in exc_str:
            return True
    # Treat insufficient_quota as non-retryable (or handle with one-off long delay in caller)
    if "insufficient_quota" in exc_str:
        return False

    # Check for openai.APIStatusError with retryable status codes
    if hasattr(exc, 'status_code'):
        status = getattr(exc, 'status_code', 0)
        if status == 429 or (500 <= status < 600):
            return True

    # Check for response attribute (some OpenAI errors have this)
    if hasattr(exc, 'response'):
        try:
            response = getattr(exc, 'response')
            if hasattr(response, 'status_code'):
                status = response.status_code
                if status == 429 or (500 <= status < 600):
                    return True
        except Exception:
            pass

    return False


def invoke_with_backoff(call_fn: Callable[[], Any], debug: bool = False, log: Optional[Callable[[str], None]] = None) -> Any:
    """DEPRECATED: Invoke call_fn with LLM-aware backoff (OpenAI and Vertex AI).

    This function is currently unused. We're using LangChain's built-in retry via max_retries.
    Keeping this for potential future use if we need more sophisticated retry logic.

    Schedule: from LME_LLM_BACKOFF_SCHEDULE (CSV, seconds) or DEFAULT_BACKOFF_SCHEDULE.
    Retries only for transient LLM errors (rate limits, server errors, timeouts).
    """
    schedule = backoff_schedule_from_env()
    # attempts = 1 immediate + len(schedule) retries with sleeps
    for attempt in range(1, len(schedule) + 2):
        try:
            return call_fn()
        except Exception as e:
            if not is_retryable_llm_error(e) or attempt > len(schedule):
                raise

            base_wait = schedule[attempt - 1]
            jitter = 1.0 + _random.uniform(-0.15, 0.15)
            sleep_for = max(0.1, base_wait * jitter)

            # Log retries unconditionally when a logger is supplied
            if log is not None:
                error_type = type(e).__name__
                log(f"[agent][llm] retryable error ({error_type}): retry {attempt}/{len(schedule)+1} after {sleep_for:.2f}s")

            _time.sleep(sleep_for)

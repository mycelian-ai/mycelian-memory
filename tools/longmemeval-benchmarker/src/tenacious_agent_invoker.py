from __future__ import annotations

from typing import Callable, List, Optional, Any
import os
import time as _time
import random as _random


DEFAULT_BACKOFF_SCHEDULE: List[float] = [5.0, 30.0, 120.0, 300.0, 600.0]


# OpenAI retryable error patterns
RETRYABLE_OPENAI_PATTERNS = {
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


def is_retryable_openai_error(exc: Exception) -> bool:
    """Check if an exception is a retryable OpenAI error.
    
    Checks for:
    - OpenAI RateLimitError (from openai package)
    - HTTP status codes in error messages (429, 5xx)
    - Common error patterns (rate_limit, timeout, etc.)
    - LangChain model provider inference errors (likely transient)
    """
    exc_type = type(exc).__name__
    exc_str = str(exc).lower()
    
    # Check for LangChain model provider inference error (likely transient/throttling)
    if "unable to infer model provider" in exc_str:
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
    for pattern in RETRYABLE_OPENAI_PATTERNS:
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
    """Invoke call_fn with OpenAI-aware backoff.

    Schedule: from LME_LLM_BACKOFF_SCHEDULE (CSV, seconds) or DEFAULT_BACKOFF_SCHEDULE.
    Retries only for transient OpenAI errors (rate limits, server errors, timeouts).
    """
    schedule = backoff_schedule_from_env()
    # attempts = 1 immediate + len(schedule) retries with sleeps
    for attempt in range(1, len(schedule) + 2):
        try:
            return call_fn()
        except Exception as e:
            if not is_retryable_openai_error(e) or attempt > len(schedule):
                raise
            
            base_wait = schedule[attempt - 1]
            jitter = 1.0 + _random.uniform(-0.15, 0.15)
            sleep_for = max(0.1, base_wait * jitter)
            
            # Log retries unconditionally when a logger is supplied
            if log is not None:
                error_type = type(e).__name__
                log(f"[agent][llm] retryable error ({error_type}): retry {attempt}/{len(schedule)+1} after {sleep_for:.2f}s")
            
            _time.sleep(sleep_for)
from __future__ import annotations

from typing import Callable, List, Optional, Any
import os
import time as _time
import random as _random


DEFAULT_BACKOFF_SCHEDULE: List[float] = [5.0, 30.0, 120.0, 300.0, 600.0]


RETRYABLE_BEDROCK_CODES = {
    "ThrottlingException",
    "TooManyRequestsException",
    "ModelTimeoutException",
    "InternalServerException",
    "ServiceUnavailableException",
    "RateLimitExceededException",
    "RateExceeded",
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


def extract_bedrock_error_code(exc: Exception) -> str:
    # boto3 ClientError shape
    try:
        code = getattr(exc, "response", {}).get("Error", {}).get("Code")  # type: ignore[attr-defined]
        if code:
            return str(code)
    except Exception:
        pass
    # Fallback: parse from string
    msg = str(exc)
    for code in list(RETRYABLE_BEDROCK_CODES) | {
        "ValidationException",
        "AccessDeniedException",
        "ResourceNotFoundException",
        "ServiceQuotaExceededException",
        "ModelErrorException",
    }:
        if code in msg:
            return code
    if "429" in msg or "Too many requests" in msg or "Rate limit" in msg:
        return "TooManyRequestsException"
    if "503" in msg or "Service Unavailable" in msg:
        return "ServiceUnavailableException"
    if "Throttling" in msg:
        return "ThrottlingException"
    return ""


def is_retryable_bedrock_code(code: str) -> bool:
    return bool(code) and (code in RETRYABLE_BEDROCK_CODES)


def invoke_with_backoff(call_fn: Callable[[], Any], debug: bool = False, log: Optional[Callable[[str], None]] = None) -> Any:
    """Invoke call_fn with Bedrock-aware backoff.

    Schedule: from LME_LLM_BACKOFF_SCHEDULE (CSV, seconds) or DEFAULT_BACKOFF_SCHEDULE.
    Retries only for transient Bedrock errors.
    """
    schedule = backoff_schedule_from_env()
    # attempts = 1 immediate + len(schedule) retries with sleeps
    for attempt in range(1, len(schedule) + 2):
        try:
            return call_fn()
        except Exception as e:
            code = extract_bedrock_error_code(e)
            if not is_retryable_bedrock_code(code) or attempt > len(schedule):
                raise
            base_wait = schedule[attempt - 1]
            jitter = 1.0 + _random.uniform(-0.15, 0.15)
            sleep_for = max(0.1, base_wait * jitter)
            if debug and log is not None:
                log(f"[agent][llm] retryable error ({code or 'unknown'}): retry {attempt}/{len(schedule)+1} after {sleep_for:.2f}s")
            _time.sleep(sleep_for)



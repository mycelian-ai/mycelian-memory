"""Tests for tenacious_agent_invoker with mock providers and fast clock."""

import unittest
from unittest.mock import Mock, patch, call
import time
from typing import List

from src.tenacious_agent_invoker import (
    invoke_with_backoff,
    is_retryable_openai_error,
    backoff_schedule_from_env,
    DEFAULT_BACKOFF_SCHEDULE
)


class TestTenaciousInvoker(unittest.TestCase):
    """Test the tenacious invoker with mock providers."""

    def setUp(self):
        """Set up test fixtures."""
        self.sleep_calls: List[float] = []
        self.log_calls: List[str] = []

    def mock_sleep(self, duration: float):
        """Mock sleep that records call durations."""
        self.sleep_calls.append(duration)
        # Don't actually sleep - makes tests fast

    def mock_log(self, msg: str):
        """Mock logger that records messages."""
        self.log_calls.append(msg)

    def test_is_retryable_openai_error(self):
        """Test error classification for OpenAI errors."""
        # Rate limit errors
        self.assertTrue(is_retryable_openai_error(Exception("Rate limit exceeded")))
        self.assertTrue(is_retryable_openai_error(Exception("Error 429: Too many requests")))

        # Server errors
        self.assertTrue(is_retryable_openai_error(Exception("Internal server error 500")))
        self.assertTrue(is_retryable_openai_error(Exception("Bad gateway 502")))
        self.assertTrue(is_retryable_openai_error(Exception("Service unavailable 503")))

        # Timeout/connection errors
        self.assertTrue(is_retryable_openai_error(Exception("Request timeout")))
        self.assertTrue(is_retryable_openai_error(Exception("Connection error")))

        # LangChain model provider error (our addition)
        self.assertTrue(is_retryable_openai_error(
            ValueError("Unable to infer model provider for model='gpt-5-nano-2025-08-07'")
        ))

        # Non-retryable errors
        self.assertFalse(is_retryable_openai_error(Exception("Invalid API key")))
        self.assertFalse(is_retryable_openai_error(Exception("Model not found")))
        self.assertFalse(is_retryable_openai_error(Exception("Insufficient_quota")))

    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_successful_call_no_retry(self, mock_sleep):
        """Test that successful calls don't trigger retries."""
        mock_sleep.side_effect = self.mock_sleep

        mock_fn = Mock(return_value="success")

        result = invoke_with_backoff(mock_fn, log=self.mock_log)

        self.assertEqual(result, "success")
        self.assertEqual(mock_fn.call_count, 1)
        self.assertEqual(len(self.sleep_calls), 0)  # No sleeps for success
        self.assertEqual(len(self.log_calls), 0)  # No logs for success

    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_retryable_error_with_recovery(self, mock_sleep):
        """Test retry logic with eventual success."""
        mock_sleep.side_effect = self.mock_sleep

        # Fail twice with rate limit, then succeed
        mock_fn = Mock(side_effect=[
            Exception("Error 429: Rate limit exceeded"),
            Exception("Error 429: Rate limit exceeded"),
            "success"
        ])

        result = invoke_with_backoff(mock_fn, debug=True, log=self.mock_log)

        self.assertEqual(result, "success")
        self.assertEqual(mock_fn.call_count, 3)
        self.assertEqual(len(self.sleep_calls), 2)  # Two retries = two sleeps

        # Check backoff durations (with jitter, so approximate)
        self.assertAlmostEqual(self.sleep_calls[0], 5.0, delta=1.0)  # ~5s
        self.assertAlmostEqual(self.sleep_calls[1], 30.0, delta=5.0)  # ~30s

        # Check log messages
        self.assertEqual(len(self.log_calls), 2)
        self.assertIn("retry 1/6", self.log_calls[0])
        self.assertIn("retry 2/6", self.log_calls[1])

    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_non_retryable_error_immediate_failure(self, mock_sleep):
        """Test that non-retryable errors fail immediately."""
        mock_sleep.side_effect = self.mock_sleep

        mock_fn = Mock(side_effect=Exception("Invalid API key"))

        with self.assertRaises(Exception) as ctx:
            invoke_with_backoff(mock_fn, log=self.mock_log)

        self.assertIn("Invalid API key", str(ctx.exception))
        self.assertEqual(mock_fn.call_count, 1)  # Only one attempt
        self.assertEqual(len(self.sleep_calls), 0)  # No retries

    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_exhausted_retries(self, mock_sleep):
        """Test that persistent errors eventually fail after all retries."""
        mock_sleep.side_effect = self.mock_sleep

        # Always fail with rate limit
        mock_fn = Mock(side_effect=Exception("Error 429: Rate limit exceeded"))

        with self.assertRaises(Exception) as ctx:
            invoke_with_backoff(mock_fn, log=self.mock_log)

        self.assertIn("429", str(ctx.exception))
        # 1 initial attempt + 5 retries (default schedule length)
        self.assertEqual(mock_fn.call_count, 6)
        self.assertEqual(len(self.sleep_calls), 5)

        # Verify exponential backoff pattern
        expected_durations = [5.0, 30.0, 120.0, 300.0, 600.0]
        for i, (actual, expected) in enumerate(zip(self.sleep_calls, expected_durations)):
            self.assertAlmostEqual(actual, expected, delta=expected * 0.2,
                                 msg=f"Sleep {i} duration mismatch")

    @patch.dict('os.environ', {'LME_LLM_BACKOFF_SCHEDULE': '1,2,3'})
    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_custom_backoff_schedule(self, mock_sleep):
        """Test custom backoff schedule from environment."""
        mock_sleep.side_effect = self.mock_sleep

        # Verify schedule is loaded from env
        schedule = backoff_schedule_from_env()
        self.assertEqual(schedule, [1.0, 2.0, 3.0])

        # Test with custom schedule
        mock_fn = Mock(side_effect=[
            Exception("Error 429"),
            Exception("Error 429"),
            "success"
        ])

        result = invoke_with_backoff(mock_fn, log=self.mock_log)

        self.assertEqual(result, "success")
        self.assertEqual(len(self.sleep_calls), 2)
        self.assertAlmostEqual(self.sleep_calls[0], 1.0, delta=0.3)  # ~1s
        self.assertAlmostEqual(self.sleep_calls[1], 2.0, delta=0.5)  # ~2s

    @patch('src.tenacious_agent_invoker._time.sleep')
    def test_langchain_model_provider_error_retryable(self, mock_sleep):
        """Test that LangChain model provider errors are retried."""
        mock_sleep.side_effect = self.mock_sleep

        # Simulate the specific LangChain error
        langchain_error = ValueError(
            "Unable to infer model provider for model='gpt-5-nano-2025-08-07', "
            "please specify model_provider directly."
        )

        # Fail once with LangChain error, then succeed
        mock_fn = Mock(side_effect=[langchain_error, "success"])

        result = invoke_with_backoff(mock_fn, log=self.mock_log)

        self.assertEqual(result, "success")
        self.assertEqual(mock_fn.call_count, 2)
        self.assertEqual(len(self.sleep_calls), 1)  # One retry

    def test_backoff_schedule_from_env_edge_cases(self):
        """Test edge cases for environment variable parsing."""
        # Empty env var
        with patch.dict('os.environ', {'LME_LLM_BACKOFF_SCHEDULE': ''}):
            self.assertEqual(backoff_schedule_from_env(), DEFAULT_BACKOFF_SCHEDULE)

        # Invalid format
        with patch.dict('os.environ', {'LME_LLM_BACKOFF_SCHEDULE': 'not,numbers'}):
            self.assertEqual(backoff_schedule_from_env(), DEFAULT_BACKOFF_SCHEDULE)

        # Mixed valid/invalid
        with patch.dict('os.environ', {'LME_LLM_BACKOFF_SCHEDULE': '1,bad,3'}):
            self.assertEqual(backoff_schedule_from_env(), DEFAULT_BACKOFF_SCHEDULE)

        # Valid with spaces
        with patch.dict('os.environ', {'LME_LLM_BACKOFF_SCHEDULE': ' 1.5 , 2.5 , 3.5 '}):
            self.assertEqual(backoff_schedule_from_env(), [1.5, 2.5, 3.5])


if __name__ == '__main__':
    unittest.main()

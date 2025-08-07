"""
Tests for vault creation and validation logic in benchmark harness.
"""
import unittest
from unittest.mock import Mock, patch, MagicMock
import pytest
import re
from datetime import datetime

# Import our modules
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(__file__)))
from synapse_client import SynapseMemoryClient


class TestVaultTitleValidation(unittest.TestCase):
    """Test vault title validation constraints."""

    def setUp(self):
        """Set up a mock client for testing."""
        with patch('subprocess.run'):
            self.client = SynapseMemoryClient("http://localhost:8080", "test_user")

    def test_valid_vault_titles(self):
        """Test valid vault titles that should pass validation."""
        valid_titles = [
            "benchmark",
            "MSC-bench",
            "test-vault-123",
            "Benchmark-Run-2025",
            "a",  # minimum length
            "A" * 50,  # maximum length
            "simple123",
            "Test-Vault",
        ]
        
        for title in valid_titles:
            with self.subTest(title=title):
                self.assertTrue(
                    self.client._is_valid_vault_title(title), 
                    f"Title '{title}' should be valid"
                )

    def test_invalid_vault_titles(self):
        """Test invalid vault titles that should fail validation."""
        invalid_titles = [
            "",  # empty
            "A" * 51,  # too long
            "vault with spaces",  # spaces not allowed
            "vault_with_underscores",  # underscores not allowed
            "vault@special!",  # special characters
            "vault.dots",  # dots not allowed
            "vault's",  # apostrophes not allowed
            "vault/slash",  # slashes not allowed
            "vault\\backslash",  # backslashes not allowed
        ]
        
        for title in invalid_titles:
            with self.subTest(title=title):
                self.assertFalse(
                    self.client._is_valid_vault_title(title), 
                    f"Title '{title}' should be invalid"
                )

    def test_generate_unique_vault_title(self):
        """Test unique vault title generation."""
        # Test default base name
        title = self.client.generate_unique_vault_title()
        
        # Should be valid
        self.assertTrue(self.client._is_valid_vault_title(title))
        
        # Should contain 'benchmark' and 'run'
        self.assertIn("benchmark", title)
        self.assertIn("run", title)
        
        # Should match expected pattern
        pattern = r'^benchmark-run-\d{8}-\d{6}$'
        self.assertRegex(title, pattern)
        
        # Should be unique (contains timestamp)
        self.assertGreater(len(title), len("benchmark-run-"))

    def test_generate_unique_vault_title_custom_base(self):
        """Test unique vault title generation with custom base name."""
        title = self.client.generate_unique_vault_title("MSC-test")
        
        # Should be valid
        self.assertTrue(self.client._is_valid_vault_title(title))
        
        # Should contain custom base
        self.assertIn("MSC-test", title)
        self.assertIn("run", title)

    def test_generate_unique_vault_title_long_base(self):
        """Test unique vault title generation with long base name that needs truncation."""
        long_base = "very-long-base-name-that-exceeds-limits"
        title = self.client.generate_unique_vault_title(long_base)
        
        # Should be valid (≤50 chars)
        self.assertTrue(self.client._is_valid_vault_title(title))
        self.assertLessEqual(len(title), 50)
        
        # Should still contain 'run' 
        self.assertIn("run", title)

    def test_generate_unique_vault_title_invalid_chars(self):
        """Test unique vault title generation cleans invalid characters."""
        title = self.client.generate_unique_vault_title("test_with_invalid@chars!")
        
        # Should be valid
        self.assertTrue(self.client._is_valid_vault_title(title))
        
        # Should not contain invalid characters
        self.assertNotIn("_", title)
        self.assertNotIn("@", title)
        self.assertNotIn("!", title)


class TestVaultCreation(unittest.TestCase):
    """Test vault creation functionality."""

    def setUp(self):
        """Set up mock client for testing."""
        with patch('subprocess.run'):
            self.client = SynapseMemoryClient("http://localhost:8080", "test_user")

    @patch('subprocess.run')
    def test_create_vault_cli_success(self, mock_run):
        """Test successful vault creation via CLI."""
        # Mock successful CLI output
        mock_run.return_value = Mock(
            returncode=0,
            stdout="Vault created: 12345678-1234-1234-1234-123456789abc (test-vault)\n",
            stderr=""
        )
        
        vault_id = self.client.create_vault("test-vault", "Test description")
        
        # Should return parsed vault ID
        self.assertEqual(vault_id, "12345678-1234-1234-1234-123456789abc")
        
        # Should have called CLI with correct arguments
        mock_run.assert_called_once()
        call_args = mock_run.call_args[0][0]
        self.assertEqual(call_args[0], "synapse")
        self.assertIn("create-vault", call_args)
        self.assertIn("--title", call_args)
        self.assertIn("test-vault", call_args)

    def test_create_vault_invalid_title(self):
        """Test vault creation with invalid title."""
        with self.assertRaises(ValueError) as cm:
            self.client.create_vault("invalid title with spaces")
        
        self.assertIn("Invalid vault title", str(cm.exception))

    @patch('subprocess.run')
    @patch.object(SynapseMemoryClient, '_run_cli')
    @patch('requests.Session.post')
    def test_create_vault_cli_fallback_to_http(self, mock_post, mock_run_cli, mock_run):
        """Test vault creation falls back to HTTP when CLI fails."""
        # Mock CLI failure 
        mock_run_cli.side_effect = RuntimeError("CLI failed")
        
        # Mock successful HTTP response
        mock_response = Mock()
        mock_response.status_code = 201
        mock_response.json.return_value = {"vaultId": "http-vault-id"}
        mock_response.raise_for_status.return_value = None
        mock_post.return_value = mock_response
        
        vault_id = self.client.create_vault("test-vault")
        
        # Should return HTTP vault ID
        self.assertEqual(vault_id, "http-vault-id")
        
        # Should have attempted HTTP POST
        mock_post.assert_called_once()


class TestMemoryCreationWithVault(unittest.TestCase):
    """Test memory creation with vault integration."""

    def setUp(self):
        """Set up mock client for testing."""
        with patch('subprocess.run'):
            self.client = SynapseMemoryClient("http://localhost:8080", "test_user")

    @patch.object(SynapseMemoryClient, '_run_cli')
    @patch.object(SynapseMemoryClient, 'create_vault')
    def test_create_memory_without_vault_creates_vault(self, mock_create_vault, mock_run_cli):
        """Test that create_memory creates a vault when none provided."""
        # Mock vault creation
        mock_create_vault.return_value = "auto-vault-id"
        
        # Mock memory creation CLI
        mock_run_cli.return_value = "Memory created: 12345678-1234-1234-1234-123456789123 - test-memory\n"
        
        memory_id = self.client.create_memory("test-memory")
        
        # Should return correct memory ID
        self.assertEqual(memory_id, "12345678-1234-1234-1234-123456789123")
        
        # Should have created a vault
        mock_create_vault.assert_called_once()
        
        # Should have called CLI to create memory with vault
        mock_run_cli.assert_called()
        call_args = mock_run_cli.call_args[0]
        self.assertIn("create-memory", call_args)
        self.assertIn("--vault-id", call_args)
        self.assertIn("auto-vault-id", call_args)

    @patch.object(SynapseMemoryClient, '_run_cli')
    def test_create_memory_with_existing_vault(self, mock_run_cli):
        """Test create_memory with existing vault ID."""
        # Mock memory creation CLI
        mock_run_cli.return_value = "Memory created: 12345678-1234-1234-1234-123456789456 - test-memory\n"
        
        memory_id = self.client.create_memory("test-memory", vault_id="existing-vault")
        
        # Should return correct memory ID
        self.assertEqual(memory_id, "12345678-1234-1234-1234-123456789456")
        
        # Should use provided vault ID  
        call_args = mock_run_cli.call_args[0]
        self.assertIn("create-memory", call_args)
        self.assertIn("--vault-id", call_args)
        self.assertIn("existing-vault", call_args)


if __name__ == "__main__":
    unittest.main()
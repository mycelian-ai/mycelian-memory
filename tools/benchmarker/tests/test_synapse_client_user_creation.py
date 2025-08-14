"""
Unit tests for SynapseMemoryClient user creation logic.
"""
import pytest
from unittest.mock import Mock, patch, MagicMock
import requests

from benchmarks.python.synapse_client import SynapseMemoryClient


class TestSynapseClientUserCreation:
    """Test SynapseMemoryClient user creation and local_user logic."""

    @patch('benchmarks.python.synapse_client.SynapseMemoryClient._check_local_user_exists')
    @patch('benchmarks.python.synapse_client.SynapseMemoryClient._create_user_via_cli')
    def test_uses_local_user_when_available(self, mock_create_user, mock_check_user):
        """When local_user exists, should use it instead of creating new user."""
        mock_check_user.return_value = True
        mock_create_user.return_value = "new-user-id"
        
        # Create client without specifying user_id
        client = SynapseMemoryClient("http://localhost:11545")
        
        # Should use local_user
        assert client.user_id == "local_user"
        mock_check_user.assert_called_once()
        mock_create_user.assert_not_called()

    @patch('benchmarks.python.synapse_client.SynapseMemoryClient._check_local_user_exists')
    @patch('benchmarks.python.synapse_client.SynapseMemoryClient._create_user_via_cli')
    def test_creates_new_user_when_local_user_not_available(self, mock_create_user, mock_check_user):
        """When local_user doesn't exist, should create new user."""
        mock_check_user.return_value = False
        mock_create_user.return_value = "new-user-123"
        
        # Create client without specifying user_id
        client = SynapseMemoryClient("http://localhost:11545")
        
        # Should create new user
        assert client.user_id == "new-user-123"
        mock_check_user.assert_called_once()
        mock_create_user.assert_called_once()

    def test_uses_provided_user_id(self):
        """When user_id is provided, should use it directly without checking local_user."""
        with patch('benchmarks.python.synapse_client.SynapseMemoryClient._check_local_user_exists') as mock_check_user, \
             patch('benchmarks.python.synapse_client.SynapseMemoryClient._create_user_via_cli') as mock_create_user:
            
            # Create client with explicit user_id
            client = SynapseMemoryClient("http://localhost:11545", user_id="explicit-user")
            
            # Should use provided user_id
            assert client.user_id == "explicit-user"
            mock_check_user.assert_not_called()
            mock_create_user.assert_not_called()

    @patch('requests.Session.get')
    def test_check_local_user_exists_success(self, mock_get):
        """_check_local_user_exists should return True when API returns 200."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_get.return_value = mock_response
        
        client = SynapseMemoryClient("http://localhost:11545", user_id="test-user")
        result = client._check_local_user_exists()
        
        assert result is True
        mock_get.assert_called_once_with("http://localhost:11545/v0/users/local_user", timeout=10)

    @patch('requests.Session.get')
    def test_check_local_user_exists_not_found(self, mock_get):
        """_check_local_user_exists should return False when API returns 404."""
        mock_response = Mock()
        mock_response.status_code = 404
        mock_get.return_value = mock_response
        
        client = SynapseMemoryClient("http://localhost:11545", user_id="test-user")
        result = client._check_local_user_exists()
        
        assert result is False
        mock_get.assert_called_once_with("http://localhost:11545/v0/users/local_user", timeout=10)

    @patch('requests.Session.get')
    def test_check_local_user_exists_handles_exception(self, mock_get):
        """_check_local_user_exists should return False when API call fails."""
        mock_get.side_effect = requests.RequestException("Connection error")
        
        client = SynapseMemoryClient("http://localhost:11545", user_id="test-user")
        result = client._check_local_user_exists()
        
        assert result is False
        mock_get.assert_called_once_with("http://localhost:11545/v0/users/local_user", timeout=10)

    @patch('benchmarks.python.synapse_client.SynapseMemoryClient._run_cli')
    @patch('benchmarks.python.synapse_client.uuid.uuid4')
    def test_create_user_via_cli_includes_user_id(self, mock_uuid, mock_run_cli):
        """_create_user_via_cli should include --user-id flag in CLI command."""
        # Create a mock UUID object
        mock_uuid_obj = Mock()
        mock_uuid_obj.hex = "abc12345"
        mock_uuid_obj.__str__ = Mock(return_value="12345678-1234-1234-1234-123456789abc")
        mock_uuid.return_value = mock_uuid_obj
        
        # The CLI should return the user ID that matches our regex pattern
        mock_run_cli.return_value = "User created: 12345678-1234-1234-1234-123456789abc (benchmark-abc12345@example.com)"
        
        client = SynapseMemoryClient("http://localhost:11545", user_id="test-user")
        result = client._create_user_via_cli()
        
        assert result == "12345678-1234-1234-1234-123456789abc"
        
        # Verify CLI was called with correct arguments
        args, _ = mock_run_cli.call_args
        args_list = list(args)
        assert "create-user" in args_list
        assert "--user-id" in args_list
        assert "12345678-1234-1234-1234-123456789abc" in args_list
        assert "--email" in args_list
        assert "--display-name" in args_list
        assert "Benchmark User" in args_list
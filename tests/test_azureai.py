"""Tests for the Azure AI plugin"""

import os
import sys
import pytest
from unittest.mock import Mock, patch
import requests
import json

# Add src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.AIModels.AzureAI.AzureAI import AzureAI

@pytest.fixture
def mock_env(monkeypatch):
    """Mock environment variables"""
    monkeypatch.setenv("AZURE_OPENAI_API_KEY", "test_key")
    monkeypatch.setenv("AZURE_OPENAI_ENDPOINT", "https://test-endpoint.openai.azure.com")

@pytest.fixture
def plugin(mock_env):
    """Create plugin instance with mocked environment"""
    return AzureAI()

def test_init_missing_env():
    """Test initialization fails without required environment variables"""
    with pytest.raises(ValueError):
        AzureAI()

def test_init_success(plugin):
    """Test successful initialization"""
    assert plugin.api_key == "test_key"
    assert plugin.endpoint == "https://test-endpoint.openai.azure.com"

@patch("requests.request")
def test_chat_completion(mock_request, plugin):
    """Test chat completion request"""
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "choices": [
            {
                "message": {
                    "content": "Test response"
                }
            }
        ]
    }
    mock_request.return_value = mock_response

    messages = [{"role": "user", "content": "Hello"}]
    result = plugin.chat_completion("gpt-4", messages)

    assert result == "Test response"
    mock_request.assert_called_once()
    called_url = mock_request.call_args[0][1]
    assert "gpt-4" in called_url
    assert "chat/completions" in called_url

@patch("requests.request")
def test_chat_completion_rate_limit(mock_request, plugin):
    """Test handling of rate limit errors"""
    mock_response = Mock()
    mock_response.status_code = 429
    mock_response.json.return_value = {"error": {"message": "Rate limit exceeded"}}
    mock_request.return_value = mock_response

    messages = [{"role": "user", "content": "Hello"}]
    
    with pytest.raises(RuntimeError) as exc_info:
        plugin.chat_completion("gpt-4", messages)
    
    assert "Rate limit exceeded" in str(exc_info.value)
    assert mock_request.call_count == plugin.max_retries + 1

@patch("requests.request")
def test_get_available_models(mock_request, plugin):
    """Test fetching available models"""
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "data": [
            {"id": "gpt-4"},
            {"id": "gpt-35-turbo"}
        ]
    }
    mock_request.return_value = mock_response

    models = plugin.get_available_models()

    assert "gpt-4" in models
    assert "gpt-35-turbo" in models
    mock_request.assert_called_once()

def test_execute_pipeline_step(plugin):
    """Test pipeline step execution"""
    with patch.object(plugin, "chat_completion", return_value="Test response"):
        result = plugin.execute_pipeline_step({
            "config": {
                "deployment_id": "gpt-4",
                "messages": [{"role": "user", "content": "Hello"}]
            },
            "output": "response"
        }, {})

        assert result == {"response": "Test response"}

def test_execute_pipeline_step_error(plugin):
    """Test pipeline step error handling"""
    with patch.object(plugin, "chat_completion", side_effect=Exception("Test error")):
        result = plugin.execute_pipeline_step({
            "config": {
                "deployment_id": "gpt-4",
                "messages": [{"role": "user", "content": "Hello"}]
            },
            "output": "response"
        }, {})

        assert "Azure AI ERROR" in result["response"]
        assert "Test error" in result["response"]

@patch("requests.request")
def test_request_retry_success(mock_request, plugin):
    """Test successful retry after temporary failure"""
    fail_response = Mock()
    fail_response.status_code = 503
    fail_response.json.return_value = {"error": {"message": "Service unavailable"}}

    success_response = Mock()
    success_response.status_code = 200
    success_response.json.return_value = {"success": True}

    mock_request.side_effect = [fail_response, success_response]

    result = plugin._make_request("GET", "test_url", {})

    assert result == {"success": True}
    assert mock_request.call_count == 2

@patch("requests.request")
def test_text_completion_converts_to_chat(mock_request, plugin):
    """Test that text completion converts to chat format"""
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "choices": [
            {
                "message": {
                    "content": "Test response"
                }
            }
        ]
    }
    mock_request.return_value = mock_response

    result = plugin.text_completion("gpt-4", "Hello")

    assert result == "Test response"
    # Verify request was made in chat format
    called_json = mock_request.call_args[1]["json"]
    assert "messages" in called_json
    assert called_json["messages"][0]["role"] == "user"
    assert called_json["messages"][0]["content"] == "Hello"
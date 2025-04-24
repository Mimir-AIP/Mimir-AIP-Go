"""
Test suite for the OpenRouter plugin
"""

import os
import pytest
from Plugins.AIModels.OpenRouter.OpenRouter import OpenRouter
from unittest.mock import patch, MagicMock
import time

def test_openrouter_initialization():
    """Test OpenRouter plugin initialization"""
    try:
        plugin = OpenRouter()
        assert isinstance(plugin, OpenRouter)
        assert plugin.api_key is not None
        assert plugin.base_url == "https://openrouter.ai/api/v1"
        print(f"\nPlugin initialized successfully")
        print(f"API Key: {'*' * 20}")
        print(f"Base URL: {plugin.base_url}")
    except ValueError as e:
        pytest.skip(str(e))

def test_openrouter_chat_completion():
    """Test OpenRouter chat completion with a simple prompt"""
    try:
        plugin = OpenRouter()
        
        # Simple test prompt
        messages = [
            {
                "role": "user",
                "content": "What is the capital of France?"
            }
        ]
        
        # Try different models
        models = [
            "agentica-org/deepcoder-14b-preview:free",
            "meta-llama/llama-4-maverick:free",
            "meta-llama/llama-3.1-8b-instruct:free"
        ]
        
        for model in models:
            print(f"\nTesting model: {model}")
            print(f"Request data: {messages}")
            print(f"API Key: {'*' * 20}")
            print(f"Base URL: {plugin.base_url}")
            
            try:
                response = plugin.chat_completion(model=model, messages=messages)
                assert isinstance(response, str)
                assert len(response) > 0
                print(f"Response: {response}")
            except Exception as e:
                print(f"Error with model {model}: {e}")
                continue
    except Exception as e:
        pytest.fail(f"Test failed: {str(e)}")

def test_get_available_models_success():
    """Test get_available_models returns correct IDs on valid API response."""
    plugin = OpenRouter()
    fake_response = {
        "data": [
            {"id": "model-a", "name": "Model A"},
            {"id": "model-b", "name": "Model B"}
        ]
    }
    with patch("requests.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = fake_response
        mock_get.return_value = mock_resp
        result = plugin.get_available_models()
        assert result == ["model-a", "model-b"]


def test_get_available_models_api_error():
    """Test get_available_models raises RuntimeError on HTTP error."""
    plugin = OpenRouter()
    with patch("requests.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.status_code = 500
        mock_resp.text = "Internal Server Error"
        mock_get.return_value = mock_resp
        try:
            plugin.get_available_models()
        except RuntimeError as e:
            assert "OpenRouter API error" in str(e)
        else:
            assert False, "Expected RuntimeError"


def test_get_available_models_invalid_json():
    """Test get_available_models raises RuntimeError on invalid JSON structure."""
    plugin = OpenRouter()
    with patch("requests.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"unexpected": []}
        mock_get.return_value = mock_resp
        try:
            plugin.get_available_models()
        except RuntimeError as e:
            assert "Invalid response format" in str(e)
        else:
            assert False, "Expected RuntimeError"


def test_get_available_models_network_error():
    """Test get_available_models raises RuntimeError on network error."""
    plugin = OpenRouter()
    with patch("requests.get", side_effect=Exception("Network down")):
        try:
            plugin.get_available_models()
        except RuntimeError as e:
            assert "Could not fetch models from OpenRouter" in str(e)
        else:
            assert False, "Expected RuntimeError"

def test_live_get_available_models():
    """
    Integration test: Make a real API call to OpenRouter to fetch available models.
    Skips if OPENROUTER_API_KEY is not set.
    """
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        pytest.skip("OPENROUTER_API_KEY not set; skipping live API test.")
    plugin = OpenRouter()
    models = plugin.get_available_models()
    print(f"Live models returned: {models}")
    assert isinstance(models, list)
    assert len(models) > 0
    assert all(isinstance(mid, str) and mid for mid in models)


def test_live_chat_completion():
    """
    Integration test: Make a real chat completion call to OpenRouter using a free model.
    Skips if OPENROUTER_API_KEY is not set.
    """
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        pytest.skip("OPENROUTER_API_KEY not set; skipping live API test.")
    plugin = OpenRouter()
    # Use a simple, free model and prompt
    models = plugin.get_available_models()
    free_model = None
    for mid in models:
        if ":free" in mid:
            free_model = mid
            break
    if not free_model:
        pytest.skip("No free model found in live model list.")
    prompt = "What is the capital of France?"
    messages = [{"role": "user", "content": prompt}]
    response = plugin.chat_completion(model=free_model, messages=messages)
    print(f"Live chat completion response: {response}")
    assert isinstance(response, str)
    assert "Paris".lower() in response.lower()
    #(avoid rate limits)
    time.sleep(1)

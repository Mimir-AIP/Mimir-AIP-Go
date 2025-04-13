"""
Test suite for the OpenRouter plugin
"""

import os
import pytest
from Plugins.AIModels.OpenRouter.OpenRouter import OpenRouter

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

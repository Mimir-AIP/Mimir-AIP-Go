import pytest
import os
import sys
from unittest.mock import Mock

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Data_Processing.LLMFunction.LLMFunction import LLMFunction
from Plugins.AIModels.OpenRouter.OpenRouter import OpenRouter

def test_llm_function_initialization():
    """Test LLMFunction plugin initialization"""
    mock_plugin_manager = Mock()
    mock_logger = Mock()
    plugin = LLMFunction(plugin_manager=mock_plugin_manager, logger=mock_logger)
    assert isinstance(plugin, LLMFunction)
    assert plugin.llm_plugin is None
    assert isinstance(plugin.plugin_manager, Mock)
    assert isinstance(plugin.logger, Mock)

def test_llm_function_set_llm_plugin():
    """Test setting LLM plugin"""
    plugin = LLMFunction()
    
    # Mock the plugin manager
    mock_plugin_manager = Mock()
    mock_openrouter = Mock()
    mock_plugin_manager.get_plugin.return_value = mock_openrouter
    plugin.plugin_manager = mock_plugin_manager
    
    # Set the LLM plugin
    plugin.set_llm_plugin("OpenRouter")
    
    assert plugin.llm_plugin == mock_openrouter
    mock_plugin_manager.get_plugin.assert_called_once_with("AIModel", "OpenRouter")

def test_llm_function_execute_pipeline_step():
    """Test executing pipeline step with LLM function"""
    # Initialize OpenRouter
    openrouter = OpenRouter()
    
    # Initialize LLMFunction with OpenRouter
    plugin = LLMFunction(llm_plugin=openrouter)
    
    # Test with a simple prompt
    config = {
        "plugin": "OpenRouter",
        "model": "meta-llama/llama-4-maverick:free",
        "function": "Summarize the following text in one sentence.",
        "format": "response"
    }
    
    # Test with a sample input
    input_data = "The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris, France. It is named after the engineer Gustave Eiffel, whose company designed and built the tower."
    
    # Create step configuration
    step_config = {
        "config": config,
        "input": "text",
        "output": "summary"
    }
    
    # Create context
    context = {"text": input_data}
    
    # Execute the pipeline step
    result = plugin.execute_pipeline_step(step_config, context)
    
    assert "summary" in result
    assert isinstance(result["summary"], str)
    assert len(result["summary"]) > 0
    
    # Verify the response is a summary
    assert "Eiffel Tower" in result["summary"]
    assert "Paris" in result["summary"]
    assert len(result["summary"]) < len(input_data)

def test_llm_function_execute_pipeline_step_with_format():
    """Test executing pipeline step with custom format"""
    # Mock OpenRouter and its chat_completion
    mock_openrouter = Mock()
    # Simulate a response that matches the expected format
    mock_openrouter.chat_completion.return_value = [
        {'topic': 'Paris Landmarks', 'description': 'Famous landmarks in Paris.'},
        {'topic': 'French Architecture', 'description': 'Notable architectural styles in France.'},
        {'topic': 'Tourism in France', 'description': 'Popular tourist destinations in France.'}
    ]
    plugin = LLMFunction(llm_plugin=mock_openrouter)
    config = {
        "plugin": "OpenRouter",
        "model": "meta-llama/llama-4-maverick:free",
        "function": "Generate a list of 3 related topics.",
        "format": "response"
    }
    input_data = "The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris, France."
    step_config = {
        "config": config,
        "input": "text",
        "output": "related_topics"
    }
    context = {"text": input_data}
    result = plugin.execute_pipeline_step(step_config, context)
    assert "related_topics" in result
    assert isinstance(result["related_topics"], list)
    assert len(result["related_topics"]) == 3
    for topic in result["related_topics"]:
        assert "topic" in topic
        assert "description" in topic
        assert isinstance(topic["topic"], str)
        assert isinstance(topic["description"], str)

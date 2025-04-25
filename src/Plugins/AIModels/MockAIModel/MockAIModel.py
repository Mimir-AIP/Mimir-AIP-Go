"""
Mock AI model plugin for testing purposes

This plugin provides simple responses without requiring an API key
"""

from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
from Plugins.BasePlugin import BasePlugin

class MockAIModel(BaseAIModel, BasePlugin):
    """Mock AI model plugin for testing"""

    plugin_type = "AIModels"
    # Explicitly declare for plugin discovery
    pass

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """
        Execute a pipeline step for the MockAIModel plugin.
        Extracts config, generates a mock response, and stores it in the context under the output key.
        Args:
            step_config (dict): Pipeline step configuration (must include 'config' and 'output')
            context (dict): Current pipeline context
        Returns:
            dict: Updated context with mock AI model output
        """
        config = step_config.get("config", {})
        output_key = step_config.get("output", "mock_ai_output")
        # Use chat_completion if messages provided, else text_completion if prompt provided
        if "messages" in config:
            # Use a default model name for mock
            response = self.chat_completion(config.get("model", "mock-model-1"), config["messages"])
        elif "prompt" in config:
            response = self.text_completion(config.get("model", "mock-model-1"), config["prompt"])
        else:
            response = "[MockAIModel] No prompt or messages provided."
        context[output_key] = response
        return {output_key: response}

    def __init__(self):
        """Initialize the mock AI model"""
        self.name = "MockAIModel"

    def chat_completion(self, model, messages):
        """
        Generate a mock chat completion

        Args:
            model (str): Model identifier (ignored in mock)
            messages (list): List of message dictionaries

        Returns:
            str: Mock response text
        """
        # Extract the last user message
        last_message = next((m for m in reversed(messages) if m["role"] == "user"), None)
        if not last_message:
            return "No user message found"

        # Generate a simple mock response
        response = f"Mock response to: {last_message['content'][:50]}..."
        return response

    def text_completion(self, model, prompt):
        """
        Generate a mock text completion

        Args:
            model (str): Model identifier (ignored in mock)
            prompt (str): Text prompt

        Returns:
            str: Mock completion text
        """
        return f"Mock completion for: {prompt[:50]}..."

    def get_available_models(self):
        """
        Get list of available models

        Returns:
            list: List of mock model identifiers
        """
        return ["mock-model-1", "mock-model-2"]

# Aliases for PluginManager compatibility
Mockaimodel = MockAIModel
MockaimodelPlugin = MockAIModel

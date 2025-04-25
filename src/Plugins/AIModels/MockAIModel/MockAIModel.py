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
        """Stub implementation to satisfy BasePlugin ABC"""
        return context

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

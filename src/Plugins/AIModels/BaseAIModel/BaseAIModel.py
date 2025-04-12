"""
Base class for AI model plugins
"""

from abc import ABC, abstractmethod

class BaseAIModel(ABC):
    """
    Abstract base class for AI model plugins
    Defines the interface that all AI model plugins must implement
    """

    plugin_type = "AIModels"

    @abstractmethod
    def chat_completion(self, model, messages):
        """
        Generate a chat completion using the specified model

        Args:
            model (str): Model identifier to use for completion
            messages (list): List of message dicts with 'role' and 'content' keys
                           Example: [{"role": "user", "content": "Hello!"}]

        Returns:
            str: Generated response text
        """
        pass

    @abstractmethod
    def text_completion(self, model, prompt):
        """
        Generate a text completion using the specified model

        Args:
            model (str): Model identifier to use for completion
            prompt (str): Text prompt to complete

        Returns:
            str: Generated completion text
        """
        pass

    @abstractmethod
    def get_available_models(self):
        """
        Get list of available models

        Returns:
            list: List of model identifiers that can be used with this plugin
        """
        pass

"""
OpenRouter AI model plugin for accessing various LLM APIs
"""

import os
import requests
from dotenv import load_dotenv
from ..BaseAIModel import BaseAIModel

class OpenRouter(BaseAIModel):
    """
    Plugin for accessing various LLM APIs through OpenRouter
    """

    def __init__(self, base_url="https://openrouter.ai/api/v1"):
        """
        Initialize the OpenRouter plugin

        Args:
            base_url (str): Base URL for the OpenRouter API
        """
        super().__init__()
        self.base_url = base_url
        load_dotenv()
        self.api_key = os.getenv("OPENROUTER_API_KEY")
        if not self.api_key:
            raise ValueError("OPENROUTER_API_KEY environment variable not set")

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
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "HTTP-Referer": "https://github.com/CiaranMcAleer/Mimir-AIP",
            "Content-Type": "application/json"
        }

        data = {
            "model": model,
            "messages": messages
        }

        try:
            response = requests.post(
                f"{self.base_url}/chat/completions",
                headers=headers,
                json=data
            )
            response.raise_for_status()
            result = response.json()
            return result["choices"][0]["message"]["content"]
        except requests.exceptions.RequestException as err:
            print(f"API Error: {err}")
            return None

    def text_completion(self, model, prompt):
        """
        Generate a text completion using the specified model

        Args:
            model (str): Model identifier to use for completion
            prompt (str): Text prompt to complete

        Returns:
            str: Generated completion text
        """
        messages = [{"role": "user", "content": prompt}]
        return self.chat_completion(model, messages)

    def get_available_models(self):
        """
        Get list of available models

        Returns:
            list: List of model identifiers that can be used with this plugin
        """
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "HTTP-Referer": "https://github.com/CiaranMcAleer/Mimir-AIP"
        }

        try:
            response = requests.get(
                f"{self.base_url}/models",
                headers=headers
            )
            response.raise_for_status()
            result = response.json()
            return [model["id"] for model in result["data"]]
        except requests.exceptions.RequestException as err:
            print(f"API Error: {err}")
            return []

if __name__ == "__main__":
    # Example usage
    plugin = OpenRouter()
    
    try:
        # Get available models
        models = plugin.get_available_models()
        print("Available models:", models)
        
        if models:
            # Test chat completion
            messages = [
                {"role": "user", "content": "What is the capital of France?"}
            ]
            response = plugin.chat_completion(models[0], messages)
            print("\nChat completion response:", response)
            
            # Test text completion
            prompt = "Complete this sentence: The Eiffel Tower is located in"
            response = plugin.text_completion(models[0], prompt)
            print("\nText completion response:", response)
    except Exception as e:
        print(f"Error: {str(e)}")
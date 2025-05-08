"""
Ollama plugin for accessing locally running Ollama server.

This plugin provides access to Ollama's local API server for running various LLMs.
"""

import os
import json
import logging
import requests
from typing import List, Dict, Any, Optional, Union
import dotenv
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel

class Ollama(BaseAIModel):
    """Ollama plugin for accessing local Ollama server"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize the Ollama plugin"""
        # Load configuration from local .env file
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        
        # Get base URL from env or use default
        self.base_url = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
        
        self.logger = logging.getLogger("Plugins.AIModels.Ollama")
        self.logger.setLevel(logging.DEBUG)
        self.logger.debug(f"Ollama plugin initialized with base URL: {self.base_url}")

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a pipeline step using Ollama"""
        try:
            config = step_config["config"]
            if "messages" in config:
                response = self.chat_completion(
                    model=config.get("model", "llama2"),
                    messages=config["messages"],
                    return_full_response=config.get("return_full_response", False)
                )
            elif "prompt" in config:
                response = self.text_completion(
                    model=config.get("model", "llama2"),
                    prompt=config["prompt"]
                )
            else:
                raise ValueError("Either 'messages' or 'prompt' must be provided in config")

            return {step_config["output"]: response}
        except Exception as e:
            self.logger.error(f"Error in execute_pipeline_step: {str(e)}", exc_info=True)
            return {step_config.get("output", "error"): f"Ollama ERROR: {str(e)}"}

    def chat_completion(self, model: str, messages: List[Dict[str, str]], return_full_response: bool = False) -> Union[str, Dict[str, Any]]:
        """
        Generate a chat completion using Ollama's API
        
        Args:
            model: The Ollama model to use (e.g., 'llama2', 'mistral', etc.)
            messages: List of message dictionaries with 'role' and 'content'
            return_full_response: Whether to return the full API response
            
        Returns:
            Generated completion text or full response dictionary
        """
        headers = {
            "Content-Type": "application/json"
        }
        
        # Convert messages to Ollama format
        formatted_messages = []
        for msg in messages:
            role = msg.get("role", "user")
            # Ollama uses 'assistant' for 'system' role
            if role == "system":
                role = "assistant"
            formatted_messages.append({
                "role": role,
                "content": msg.get("content", "")
            })
        
        data = {
            "model": model,
            "messages": formatted_messages,
            "stream": False  # We don't handle streaming for now
        }

        try:
            response = requests.post(
                f"{self.base_url}/chat",
                headers=headers,
                json=data
            )
            response.raise_for_status()
            json_response = response.json()

            if return_full_response:
                return json_response

            return json_response.get("message", {}).get("content", "")

        except Exception as e:
            self.logger.error(f"Error in chat_completion: {str(e)}")
            raise

    def text_completion(self, model: str, prompt: str) -> str:
        """
        Generate a text completion using Ollama's API
        
        Args:
            model: The Ollama model to use (e.g., 'llama2', 'mistral', etc.)
            prompt: The text prompt
            
        Returns:
            Generated completion text
        """
        headers = {
            "Content-Type": "application/json"
        }
        
        data = {
            "model": model,
            "prompt": prompt,
            "stream": False  # We don't handle streaming for now
        }

        try:
            response = requests.post(
                f"{self.base_url}/generate",
                headers=headers,
                json=data
            )
            response.raise_for_status()
            json_response = response.json()
            return json_response.get("response", "")

        except Exception as e:
            self.logger.error(f"Error in text_completion: {str(e)}")
            raise

    def get_available_models(self) -> List[str]:
        """
        Get list of available Ollama models
        
        Returns:
            List of model identifiers
        """
        try:
            response = requests.get(f"{self.base_url}/api/tags")
            response.raise_for_status()
            data = response.json()
            # Extract model names from the response
            return [model["name"] for model in data.get("models", [])]
        except Exception as e:
            self.logger.error(f"Error fetching available models: {e}")
            # Return common models as fallback
            return [
                "llama2",
                "mistral",
                "codellama",
                "stable-beluga"
            ]
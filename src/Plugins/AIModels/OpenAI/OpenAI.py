"""
OpenAI plugin for accessing OpenAI's API services.

This plugin provides access to OpenAI models via their official API.
"""

import os
import json
import logging
import requests
from typing import List, Dict, Any, Optional, Union
import dotenv
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel

class OpenAI(BaseAIModel):
    """OpenAI plugin for accessing OpenAI API services"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize the OpenAI plugin"""
        # Load API key from local .env file
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        self.api_key = os.getenv("OPENAI_API_KEY")
        if not self.api_key:
            raise ValueError("OPENAI_API_KEY environment variable not set")
        
        self.base_url = "https://api.openai.com/v1"
        self.logger = logging.getLogger("Plugins.AIModels.OpenAI")
        self.logger.setLevel(logging.DEBUG)
        self.logger.debug(f"OpenAI plugin initialized with API key: {'*' * 20}")

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a pipeline step using OpenAI"""
        try:
            config = step_config["config"]
            if "messages" in config:
                response = self.chat_completion(
                    model=config["model"],
                    messages=config["messages"],
                    return_full_response=config.get("return_full_response", False)
                )
            elif "prompt" in config:
                response = self.text_completion(
                    model=config["model"],
                    prompt=config["prompt"]
                )
            else:
                raise ValueError("Either 'messages' or 'prompt' must be provided in config")

            return {step_config["output"]: response}
        except Exception as e:
            self.logger.error(f"Error in execute_pipeline_step: {str(e)}", exc_info=True)
            return {step_config.get("output", "error"): f"OpenAI ERROR: {str(e)}"}

    def chat_completion(self, model: str, messages: List[Dict[str, str]], return_full_response: bool = False) -> Union[str, Dict[str, Any]]:
        """
        Generate a chat completion using OpenAI's API
        
        Args:
            model: The OpenAI model to use
            messages: List of message dictionaries with 'role' and 'content'
            return_full_response: Whether to return the full API response
            
        Returns:
            Generated completion text or full response dictionary
        """
        headers = {
            "Authorization": f"Bearer {self.api_key}",
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
            json_response = response.json()

            if return_full_response:
                return json_response

            if "choices" in json_response and len(json_response["choices"]) > 0:
                return json_response["choices"][0]["message"]["content"]
            else:
                raise ValueError("No completion choices in response")

        except Exception as e:
            self.logger.error(f"Error in chat_completion: {str(e)}")
            raise

    def text_completion(self, model: str, prompt: str) -> str:
        """
        Generate a text completion using OpenAI's API
        
        Args:
            model: The OpenAI model to use
            prompt: The text prompt
            
        Returns:
            Generated completion text
        """
        # Convert to chat format since OpenAI is deprecating non-chat endpoints
        messages = [{"role": "user", "content": prompt}]
        return self.chat_completion(model, messages)

    def get_available_models(self) -> List[str]:
        """
        Get list of available OpenAI models
        
        Returns:
            List of model identifiers
        """
        headers = {
            "Authorization": f"Bearer {self.api_key}"
        }

        try:
            response = requests.get(
                f"{self.base_url}/models",
                headers=headers
            )
            response.raise_for_status()
            data = response.json()
            return [model["id"] for model in data["data"]]
        except Exception as e:
            self.logger.error(f"Error fetching available models: {e}")
            # Return common models as fallback
            return [
                "gpt-4-turbo-preview",
                "gpt-4",
                "gpt-3.5-turbo",
                "gpt-3.5-turbo-16k"
            ]
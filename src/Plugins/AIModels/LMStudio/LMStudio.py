"""
LMStudio plugin for accessing locally running LM Studio server.

This plugin provides access to LM Studio's local server which uses OpenAI-compatible API format.
"""

import os
import json
import logging
import requests
from typing import List, Dict, Any, Optional, Union
import dotenv
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel

class LMStudio(BaseAIModel):
    """LMStudio plugin for accessing local LM Studio server"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize the LMStudio plugin"""
        # Load configuration from local .env file
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        
        # Get base URL from env or use default
        self.base_url = os.getenv("LMSTUDIO_BASE_URL", "http://localhost:1234/v1")
        
        self.logger = logging.getLogger("Plugins.AIModels.LMStudio")
        self.logger.setLevel(logging.DEBUG)
        self.logger.debug(f"LMStudio plugin initialized with base URL: {self.base_url}")

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a pipeline step using LMStudio"""
        try:
            config = step_config["config"]
            if "messages" in config:
                response = self.chat_completion(
                    model=config.get("model", "local-model"),
                    messages=config["messages"],
                    return_full_response=config.get("return_full_response", False)
                )
            elif "prompt" in config:
                response = self.text_completion(
                    model=config.get("model", "local-model"),
                    prompt=config["prompt"]
                )
            else:
                raise ValueError("Either 'messages' or 'prompt' must be provided in config")

            return {step_config["output"]: response}
        except Exception as e:
            self.logger.error(f"Error in execute_pipeline_step: {str(e)}", exc_info=True)
            return {step_config.get("output", "error"): f"LMStudio ERROR: {str(e)}"}

    def chat_completion(self, model: str, messages: List[Dict[str, str]], return_full_response: bool = False) -> Union[str, Dict[str, Any]]:
        """
        Generate a chat completion using LM Studio's local server
        
        Args:
            model: Model identifier (usually ignored by LM Studio)
            messages: List of message dictionaries with 'role' and 'content'
            return_full_response: Whether to return the full API response
            
        Returns:
            Generated completion text or full response dictionary
        """
        headers = {
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
        Generate a text completion using LM Studio's local server
        
        Args:
            model: Model identifier (usually ignored by LM Studio)
            prompt: The text prompt
            
        Returns:
            Generated completion text
        """
        # Convert to chat format since that's what LM Studio expects
        messages = [{"role": "user", "content": prompt}]
        return self.chat_completion(model, messages)

    def get_available_models(self) -> List[str]:
        """
        Get list of available models
        
        Returns:
            List of model identifiers
        """
        try:
            response = requests.get(f"{self.base_url}/models")
            response.raise_for_status()
            data = response.json()
            return [model["id"] for model in data["data"]]
        except Exception as e:
            self.logger.error(f"Error fetching available models: {e}")
            # Return generic local model identifier since LM Studio typically runs one model at a time
            return ["local-model"]
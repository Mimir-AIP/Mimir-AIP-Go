"""
Azure AI model plugin for accessing Azure OpenAI services.

This plugin is designed for compatibility with the Mimir-AIP system and provides
access to Azure-hosted OpenAI models via the Azure OpenAI API.

Design notes:
- Uses Azure OpenAI API with endpoint URL and API key authentication
- Supports both chat completion and text completion interfaces
- Implements robust error handling, rate limiting, and logging
- Follows project conventions for plugin_type and method signatures
"""

import os
import json
import time
import logging
import requests
from typing import List, Dict, Any, Optional, Union
import dotenv
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel

class AzureAI(BaseAIModel):
    """Azure AI plugin for accessing Azure OpenAI services"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize the Azure AI plugin"""
        # Load configuration from .env file in the same directory
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        
        # Get required environment variables
        self.api_key = os.getenv("AZURE_OPENAI_API_KEY")
        self.endpoint = os.getenv("AZURE_OPENAI_ENDPOINT")
        if not self.api_key or not self.endpoint:
            raise ValueError("AZURE_OPENAI_API_KEY and AZURE_OPENAI_ENDPOINT environment variables must be set")
        
        # Set up logging
        self.logger = logging.getLogger("Plugins.AIModels.AzureAI")
        self.logger.setLevel(logging.DEBUG)
        self.logger.debug("Azure AI plugin initialized")
        
        # Default retry settings
        self.max_retries = 3
        self.retry_delay = 1  # seconds
        
        # Initialize rate limiting state
        self.last_request_time = 0
        self.min_request_interval = 0.1  # seconds between requests

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a pipeline step using this plugin

        Args:
            step_config: Dictionary containing step configuration
            context: Pipeline execution context

        Returns:
            dict: Step execution results

        Example step_config:
        {
            "plugin": "AzureAI",
            "config": {
                "deployment_id": "gpt-4",
                "messages": [
                    {"role": "user", "content": "Hello, how are you?"}
                ],
                "temperature": 0.7,
                "max_tokens": 800
            },
            "output": "response"
        }
        """
        try:
            config = step_config["config"]
            deployment_id = config.get("deployment_id")
            if not deployment_id:
                raise ValueError("deployment_id must be specified in config")

            if "messages" in config:
                response = self.chat_completion(
                    model=deployment_id,
                    messages=config["messages"],
                    temperature=config.get("temperature", 0.7),
                    max_tokens=config.get("max_tokens", 800),
                    return_full_response=config.get("return_full_response", False)
                )
            elif "prompt" in config:
                response = self.text_completion(
                    model=deployment_id,
                    prompt=config["prompt"],
                    temperature=config.get("temperature", 0.7),
                    max_tokens=config.get("max_tokens", 800)
                )
            else:
                raise ValueError("Either 'messages' or 'prompt' must be provided in config")

            return {step_config["output"]: response}
            
        except Exception as e:
            self.logger.error(f"Error in execute_pipeline_step: {str(e)}", exc_info=True)
            return {step_config.get("output", "error"): f"Azure AI ERROR: {str(e)}"}

    def chat_completion(
        self,
        model: str,
        messages: List[Dict[str, str]],
        temperature: float = 0.7,
        max_tokens: int = 800,
        return_full_response: bool = False
    ) -> Union[str, Dict[str, Any]]:
        """Generate a chat completion using the specified model

        Args:
            model: The deployment ID of the model to use
            messages: List of message dictionaries with 'role' and 'content'
            temperature: Sampling temperature (0-1)
            max_tokens: Maximum tokens to generate
            return_full_response: If True, return the full API response

        Returns:
            str: Generated completion text, or full response dict if return_full_response=True
        """
        url = f"{self.endpoint}/openai/deployments/{model}/chat/completions?api-version=2024-02-15-preview"
        
        headers = {
            "Content-Type": "application/json",
            "api-key": self.api_key
        }
        
        data = {
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens
        }

        response = self._make_request("POST", url, headers=headers, json=data)
        
        if return_full_response:
            return response
            
        if "choices" in response and len(response["choices"]) > 0:
            return response["choices"][0]["message"]["content"]
        else:
            raise RuntimeError("No choices in response")

    def text_completion(
        self,
        model: str,
        prompt: str,
        temperature: float = 0.7,
        max_tokens: int = 800
    ) -> str:
        """Generate a text completion using the specified model
        
        Args:
            model: The deployment ID of the model to use
            prompt: The text prompt to complete
            temperature: Sampling temperature (0-1)
            max_tokens: Maximum tokens to generate
            
        Returns:
            str: Generated completion text
        """
        # Convert to chat format since newer models use chat completions
        messages = [{"role": "user", "content": prompt}]
        return self.chat_completion(model, messages, temperature, max_tokens)

    def get_available_models(self) -> List[str]:
        """Get list of available models/deployments
        
        Returns:
            list: List of available model deployment IDs
        """
        try:
            url = f"{self.endpoint}/openai/deployments?api-version=2024-02-15-preview"
            headers = {"api-key": self.api_key}
            
            response = self._make_request("GET", url, headers=headers)
            return [deployment["id"] for deployment in response.get("data", [])]
            
        except Exception as e:
            self.logger.error(f"Error fetching available models: {e}")
            # Return common deployment IDs as fallback
            return ["gpt-4", "gpt-35-turbo", "gpt-4-32k"]

    def _make_request(
        self,
        method: str,
        url: str,
        headers: Dict[str, str],
        json: Optional[Dict[str, Any]] = None,
        retry_count: int = 0
    ) -> Dict[str, Any]:
        """Make an HTTP request to the Azure OpenAI API with retries and rate limiting
        
        Args:
            method: HTTP method ("GET" or "POST")
            url: Request URL
            headers: Request headers
            json: Optional JSON payload
            retry_count: Current retry attempt number
            
        Returns:
            dict: Parsed JSON response
            
        Raises:
            RuntimeError: If request fails after all retries
        """
        # Apply rate limiting
        now = time.time()
        time_since_last = now - self.last_request_time
        if time_since_last < self.min_request_interval:
            time.sleep(self.min_request_interval - time_since_last)
        
        try:
            response = requests.request(method, url, headers=headers, json=json)
            self.last_request_time = time.time()
            
            response.raise_for_status()
            return response.json()
            
        except requests.exceptions.HTTPError as e:
            if retry_count < self.max_retries:
                if response.status_code in (429, 500, 502, 503, 504):  # Retryable status codes
                    wait_time = self.retry_delay * (2 ** retry_count)  # Exponential backoff
                    self.logger.warning(f"Request failed with {response.status_code}, retrying in {wait_time}s...")
                    time.sleep(wait_time)
                    return self._make_request(method, url, headers, json, retry_count + 1)
                    
            # Extract error details from response if possible
            error_detail = ""
            try:
                error_json = response.json()
                if "error" in error_json:
                    error_detail = f": {error_json['error'].get('message', '')}"
            except:
                pass
                
            raise RuntimeError(f"Azure OpenAI API error (HTTP {response.status_code}){error_detail}")
            
        except requests.exceptions.RequestException as e:
            if retry_count < self.max_retries:
                wait_time = self.retry_delay * (2 ** retry_count)
                self.logger.warning(f"Request failed with {str(e)}, retrying in {wait_time}s...")
                time.sleep(wait_time)
                return self._make_request(method, url, headers, json, retry_count + 1)
            raise RuntimeError(f"Request failed: {str(e)}")
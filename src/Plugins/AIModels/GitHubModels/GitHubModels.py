"""
GitHubModels AI model plugin for accessing models hosted on Azure with GitHub PAT authentication.

This plugin is structured for compatibility with the Mimir-AIP system (mirrors OpenRouter plugin).

Design notes:
- Loads GitHub PAT from a .env file in the plugin directory.
- Accepts endpoint URL and deployment/model name via config.
- Implements robust error handling and logging.
- Follows project conventions for plugin_type, docstrings, and method signatures.
"""

import os
import requests
import logging
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
import dotenv

class GitHubModels(BaseAIModel):
    """GitHubModels plugin for accessing models hosted on Azure with GitHub PAT authentication"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize the GitHubModels plugin"""
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        self.github_pat = os.getenv("GITHUB_PAT")
        if not self.github_pat:
            raise ValueError("GITHUB_PAT environment variable not set")
        self.logger = logging.getLogger(__name__)
        self.logger.debug("GitHubModels plugin initialized with PAT: {}".format('*' * 20))

    def execute_pipeline_step(self, step_config, context):
        """
        Execute a pipeline step for this plugin.
        Expected step_config format:
        {
            "plugin": "GitHubModels",
            "config": {
                "endpoint": "https://models.github.ai/inference",
                "model": "openai/gpt-4.1",
                "messages": [
                    {"role": "user", "content": "<prompt>"}
                ],
                "temperature": 1.0,  # optional
                "top_p": 1.0          # optional
            },
            "output": "response"
        }
        If temperature or top_p are not specified, defaults will be used (temperature=1.0, top_p=1.0).
        """
        config = step_config["config"]
        endpoint = config.get("endpoint", "https://models.github.ai/inference")
        model = config["model"]
        messages = config["messages"]
        # Use defaults if not specified
        temperature = config["temperature"] if "temperature" in config and config["temperature"] is not None else 1.0
        top_p = config["top_p"] if "top_p" in config and config["top_p"] is not None else 1.0
        response = self.chat_completion(endpoint, model, messages, temperature, top_p)
        return {step_config["output"]: response}

    def get_available_models(self):
        """
        Returns a hardcoded list of currently available models on GitHub Models Marketplace (as of April 2025).
        This list should be periodically updated as new models are added or removed. For the latest, see:
        https://github.com/marketplace/models

        Returns:
            list[str]: Model identifiers usable in the 'model' parameter for chat completion.
        """
        return [
            "openai/gpt-4.1",
            "openai/o1",
            "openai/o3",
            "openai/o3-mini",
            "openai/o4-mini",
            "anthropic/claude-3-5-sonnet",
            "anthropic/claude-3-7-sonnet",
            "google/gemini-2.0-flash",
            "google/gemini-2.5-pro"
        ]

    def text_completion(self, *args, **kwargs):
        """
        Stub implementation for abstract method from BaseAIModel.
        This plugin uses chat_completion instead; text_completion is not supported.
        """
        raise NotImplementedError("text_completion is not implemented for GitHubModels. Use chat_completion instead.")

    def chat_completion(self, endpoint, model, messages, temperature=1.0, top_p=1.0, return_full_response=False):
        """
        Send a chat completion request to the Azure AI endpoint using GitHub PAT authentication.

        Args:
            endpoint (str): The Azure AI inference endpoint URL.
            model (str): The deployment/model name to use.
            messages (list): List of message dicts with 'role' and 'content'.
            temperature (float): Sampling temperature.
            top_p (float): Nucleus sampling parameter.
            return_full_response (bool): If True, return the full API response JSON.

        Returns:
            str or dict: Model response text or full response.
        """
        headers = {
            "Authorization": f"Bearer {self.github_pat}",
            "Content-Type": "application/json"
        }
        data = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "top_p": top_p
        }
        try:
            url = endpoint.rstrip("/") + "/chat/completions"
            resp = requests.post(url, headers=headers, json=data)
            self.logger.debug(f"Request to {url} | Status: {resp.status_code} | Response: {resp.text}")
            if resp.status_code != 200:
                self.logger.error(f"Error from Azure AI endpoint: {resp.status_code} {resp.text}")
                # Try to extract error from response
                try:
                    err = resp.json().get("error")
                    if err:
                        raise RuntimeError(err)
                except Exception:
                    pass
                raise RuntimeError(f"Azure AI endpoint error: {resp.status_code}")
            resp_json = resp.json()
            if "choices" not in resp_json or not resp_json["choices"]:
                self.logger.error(f"Unexpected response format: {resp_json}")
                raise RuntimeError("No choices returned from model.")
            if return_full_response:
                return resp_json
            return resp_json["choices"][0]["message"]["content"]
        except Exception as e:
            self.logger.error(f"Error during chat completion: {e}")
            raise RuntimeError(f"GitHubModels chat completion failed: {e}")

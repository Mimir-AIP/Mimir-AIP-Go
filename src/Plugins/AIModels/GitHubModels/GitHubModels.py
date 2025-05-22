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
    """GitHubModels plugin for accessing GitHub-hosted AI models"""

    plugin_type = "AIModels"
    BASE_URL = "https://models.github.ai"
    API_VERSION = "2022-11-28"
    CACHE_DURATION = 3600  # 1 hour cache for model list

    def __init__(self):
        """Initialize the GitHubModels plugin"""
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        self.github_pat = os.getenv("GITHUB_PAT")
        if not self.github_pat:
            raise ValueError("GITHUB_PAT environment variable not set")
        self.logger = logging.getLogger(__name__)
        self.logger.debug("GitHubModels plugin initialized with PAT: {}".format('*' * 20))

    def _get_headers(self):
        """Get common headers for API requests"""
        return {
            "Authorization": f"Bearer {self.github_pat}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": self.API_VERSION,
            "Content-Type": "application/json"
        }

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
                "top_p": 1.0,        # optional
                "organization": None, # optional
                "seed": None,        # optional
                "stream": False      # optional
            },
            "output": "response"
        }
        """
        config = step_config["config"]
        endpoint = config.get("endpoint", "https://models.github.ai/inference")
        model = config["model"]
        messages = config["messages"]
        org = config.get("organization")
        seed = config.get("seed")
        stream = config.get("stream", False)
        temperature = config["temperature"] if "temperature" in config and config["temperature"] is not None else 1.0
        top_p = config["top_p"] if "top_p" in config and config["top_p"] is not None else 1.0
        response = self.chat_completion(endpoint, model, messages, temperature, top_p, organization=org)
        return {step_config["output"]: response}
  
    def get_available_models(self):
        """
        Fetch available models from GitHub Models catalog API.

        Returns:
            list[str]: Model identifiers usable in the 'model' parameter for chat completion.
        """
        url = f"{self.BASE_URL}/catalog/models"
        response = requests.get(url, headers=self._get_headers())
        if response.status_code == 200:
            models = response.json()
            return [model["id"] for model in models]
        raise RuntimeError(f"Failed to fetch models: {response.status_code}")


    def text_completion(self, *args, **kwargs):
        """
        Stub implementation for abstract method from BaseAIModel.
        This plugin uses chat_completion instead; text_completion is not supported.
        """
        raise NotImplementedError("text_completion is not implemented for GitHubModels. Use chat_completion instead.")

    def chat_completion(self, endpoint, model, messages, temperature=1.0, top_p=1.0, return_full_response=False, organization=None):
        """
        Send a chat completion request to GitHub Models API.

        Args:
            endpoint (str): The inference endpoint URL.
            model (str): The model name to use.
            messages (list): List of message dicts with 'role' and 'content'.
            temperature (float): Sampling temperature.
            top_p (float): Nucleus sampling parameter.
            return_full_response (bool): If True, return the full API response JSON.
            organization (str): Optional GitHub organization name.

        Returns:
            str or dict: Model response text or full response.
        """
        headers = self._get_headers()
        data = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "top_p": top_p
        }
        try:
            url = f"{self.BASE_URL}/{'orgs/' + organization + '/' if organization else ''}inference/chat/completions"
            resp = requests.post(url, headers=headers, json=data)
            self.logger.debug(f"Request to {url} | Status: {resp.status_code} | Response: {resp.text}")
            if resp.status_code != 200:
                self.logger.error(f"Error from GitHub Models API: {resp.status_code} {resp.text}")
                # Try to extract error from response
                try:
                    err = resp.json().get("error")
                    if err:
                        raise RuntimeError(err)
                except Exception:
                    pass
                raise RuntimeError(f"GitHub Models API error: {resp.status_code}")
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

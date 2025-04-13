"""
OpenRouter AI model plugin for accessing various LLM models

Example usage:
    plugin = OpenRouter()
    result = plugin.execute_pipeline_step({
        "config": {
            "model": "meta-llama/llama-3-8b-instruct:free",
            "messages": [
                {"role": "user", "content": "Hello, how are you?"}
            ]
        },
        "output": "response"
    }, {})
"""

import os
import requests
import logging
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
import dotenv

class OpenRouter(BaseAIModel):
    """OpenRouter plugin for accessing various LLM models"""

    plugin_type = "AIModel"

    def __init__(self):
        """Initialize the OpenRouter plugin"""
        # Load API key from local .env file
        dotenv.load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))
        self.api_key = os.getenv("OPENROUTER_API_KEY")
        if not self.api_key:
            raise ValueError("OPENROUTER_API_KEY environment variable not set")
        self.base_url = "https://openrouter.ai/api/v1"
        
        # Add debug logging
        logging.basicConfig(level=logging.DEBUG)
        self.logger = logging.getLogger(__name__)
        self.logger.debug(f"OpenRouter initialized with API key: {'*' * 20}")
        self.logger.debug(f"Base URL: {self.base_url}")

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "OpenRouter",
            "config": {
                "model": "meta-llama/llama-3-8b-instruct:free",
                "messages": [
                    {"role": "user", "content": "Hello, how are you?"}
                ]
            },
            "output": "response"
        }
        """
        config = step_config["config"]
        self.logger.debug(f"Executing pipeline step with config: {config}")
        response = self.chat_completion(
            model=config["model"],
            messages=config["messages"]
        )
        self.logger.debug(f"Pipeline step response: {response}")
        return {step_config["output"]: response}

    def get_available_models(self):
        """
        Get list of available models
        
        Returns:
            list: List of available model identifiers
        """
        return [
            "agentica-org/deepcoder-14b-preview:free",
            "meta-llama/llama-4-maverick:free",
            "meta-llama/llama-3.1-8b-instruct:free"
        ]

    def chat_completion(self, model, messages):
        """
        Send a chat completion request to OpenRouter
        
        Args:
            model (str): Model identifier (e.g., 'meta-llama/llama-3-8b-instruct:free')
            messages (list): List of message dictionaries with 'role' and 'content'
        
        Returns:
            str: Model response text
        """
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "HTTP-Referer": "https://github.com/CiaranMcAleer/Mimir-AIP",
        }

        data = {
            "model": model,
            "messages": messages
        }

        self.logger.debug(f"\nSending request to OpenRouter")
        self.logger.debug(f"API Key: {'*' * 20}")
        self.logger.debug(f"Model: {model}")
        self.logger.debug(f"Messages: {messages}")
        self.logger.debug(f"Headers: {headers}")
        
        response = requests.post(
            f"{self.base_url}/chat/completions",
            headers=headers,
            json=data
        )

        self.logger.debug(f"\nOpenRouter response:")
        self.logger.debug(f"Status code: {response.status_code}")
        self.logger.debug(f"Headers: {response.headers}")
        self.logger.debug(f"Response text: {response.text}")

        if response.status_code == 200:
            return response.json()["choices"][0]["message"]["content"]
        else:
            raise ValueError(f"Error from OpenRouter API: {response.text}")

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


if __name__ == "__main__":
    # Test the plugin
    plugin = OpenRouter()
    
    test_config = {
        "plugin": "OpenRouter",
        "config": {
            "model": "meta-llama/llama-3-8b-instruct:free",
            "messages": [
                {"role": "user", "content": "Write a one-sentence summary of OpenRouter."}
            ]
        },
        "output": "response"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Response: {result['response']}")
    except ValueError as e:
        print(f"Error: {e}")
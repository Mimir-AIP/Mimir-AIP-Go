# src/Plugins/AIModels/OpenRouter/OpenRouter.py
import requests
import os
from dotenv import load_dotenv
from Plugins.AIModels.BaseAIModel import BaseAIModel


class OpenRouter(BaseAIModel):
    def __init__(self, base_url="https://openrouter.ai/api/v1"):
        """
        Initialize the OpenRouter LLM plugin with BaseAIModel interface.
        
        :param base_url: Base URL for the OpenRouter API.
        """
        load_dotenv()
        self.api_key = os.getenv("OPENROUTER_API_KEY")
        if not self.api_key:
            raise ValueError("OPENROUTER_API_KEY environment variable not set")
        
        self.base_url = base_url

    def get_name(self) -> str:
        """
        Return the name of the plugin.
        """
        return "OpenRouter"

    def generate_response(self, prompt: str) -> str:
        """
        Generate a response from the LLM plugin using a single user prompt.

        :param prompt: User input prompt.
        :return: The response content from the model.
        """
        # For consistency with more advanced models, wrap the prompt in a "user" role message.
        messages = [{"role": "user", "content": prompt}]
        return self.query_chat_model(
            model="meta-llama/llama-3-8b-instruct:free",  # Default model, you can dynamically pass this as needed.
            messages=messages
        )

    def query_chat_model(self, model, messages, max_tokens=1000, temperature=0.7, referer=None, title=None) -> str:
        """
        Send a query to the specified chat-based model.

        :param model: The model name to query (e.g., "openai/gpt-4", "anthropic/claude-v1").
        :param messages: A list of message dictionaries in the format [{"role": "user", "content": "your message"}].
        :param max_tokens: Maximum number of tokens to generate.
        :param temperature: Sampling temperature for randomness.
        :param referer: (Optional) URL of your site for OpenRouter rankings.
        :param title: (Optional) Title of your site for OpenRouter rankings.
        :return: The response content from the model.
        """
        url = f"{self.base_url}/chat/completions"
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }

        # Add optional headers if provided
        if referer:
            headers["HTTP-Referer"] = referer
        if title:
            headers["X-Title"] = title

        payload = {
            "model": model,
            "messages": messages,
            "max_tokens": max_tokens,
            "temperature": temperature,
        }

        try:
            response = requests.post(url, headers=headers, json=payload)
            response.raise_for_status()
            data = response.json()

            # Return the generated content
            return data.get("choices", [{}])[0].get("message", {}).get("content", "")
        except requests.exceptions.HTTPError as errh:
            print("HTTP Error:", errh)
        except requests.exceptions.ConnectionError as errc:
            print("Error Connecting:", errc)
        except requests.exceptions.Timeout as errt:
            print("Timeout Error:", errt)
        except requests.exceptions.RequestException as err:
            print("API Error:", err)
        return None
"""
Enhanced Mock AI model plugin for testing and development

Features:
- Configurable response patterns
- Context-aware responses
- Simulated latency
- Error injection
- Multiple model personalities
- Canned responses for common queries
- Realistic API error responses
- Configurable context size limits
"""

import random
import time
import json
import logging
from typing import Dict, List, Union, Any, Optional
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
from Plugins.BasePlugin import BasePlugin

class MockAIModel(BaseAIModel, BasePlugin):
    """Enhanced Mock AI model plugin with realistic behaviors for testing"""

    plugin_type = "AIModels"

    def __init__(self):
        """Initialize with default configuration"""
        self.name = "MockAIModel"
        self.config = {
            "response_style": "helpful",  # helpful, concise, creative, technical
            "response_format": "text",    # text, json
            "latency_ms": (50, 200),      # min/max response delay in ms
            "error_rate": 0.05,           # 5% chance of error
            "error_types": ["RuntimeError"],  # Types of errors to simulate
            "max_tokens": 500,            # Max response length
            "personality": "neutral",     # neutral, enthusiastic, skeptical
            "support_keywords": True,     # Enable keyword-based responses
            "context_persistence": True,  # Persist context across calls
            "max_context_size": 2000      # Max context size in tokens
        }
        self.context_history = []
        self.model_capabilities = {
            "mock-model-1": {"max_tokens": 500, "supports_images": False, "max_context_size": 1000},
            "mock-model-2": {"max_tokens": 1000, "supports_images": True, "max_context_size": 2000},
            "mock-model-pro": {"max_tokens": 2000, "supports_images": True, "max_context_size": 4000}
        }
        self.call_count = 0
        self.error_count = 0
        self.response_lengths = []
        self.canned_responses = {
            "What is the capital of France?": "Paris",
            "what is ai": "Artificial Intelligence is the simulation of human intelligence in machines",
            "who created python": "Python was created by Guido van Rossum",
            "what is the meaning of life": "42"
        }
        self.logger = logging.getLogger(__name__)

    def configure(self, **kwargs):
        """Update model configuration

        Args:
            **kwargs: Configuration parameters to update
        """
        self.config.update(kwargs)
        self.logger.info(f"Model configured with: {self.config}")

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Enhanced pipeline step execution with context tracking

        Args:
            step_config: Configuration for this pipeline step
            context: Shared context dictionary

        Returns:
            dict: Updated context with AI response
        """
        try:
            self.call_count += 1

            # Simulate processing delay
            delay = random.uniform(*self.config["latency_ms"]) / 1000
            time.sleep(delay)

            config = step_config.get("config", {})
            output_key = step_config.get("output", "mock_ai_output")

            # Check context size limit
            current_context_size = sum(len(str(item)) for item in self.context_history)
            model_name = config.get("model", "mock-model-1")
            max_context_size = self.model_capabilities[model_name]["max_context_size"]

            if current_context_size > max_context_size:
                error_msg = (
                    f"Context size exceeded limit of {max_context_size} tokens. "
                    f"Current size: {current_context_size} tokens."
                )
                self.logger.error(error_msg)
                return {"error": error_msg}

            # Store context for future reference if enabled
            if self.config["context_persistence"]:
                self.context_history.append({
                    "config": config,
                    "context": context.copy()
                })

            # Check for errors
            if random.random() < self.config["error_rate"]:
                error_type = random.choice(self.config["error_types"])
                error_msg = f"Simulated {error_type}: API request failed"
                self.error_count += 1
                self.logger.error(error_msg)
                raise eval(error_type)(error_msg)

            # Process input and generate response
            if "messages" in config:
                response = self.chat_completion(model_name, config["messages"])
            elif "prompt" in config:
                response = self.text_completion(model_name, config["prompt"])
            else:
                response = self._generate_response("No specific prompt provided")

            # Track response length
            self.response_lengths.append(len(response))

            # Format response based on configuration
            if self.config["response_format"] == "json":
                try:
                    response_json = json.dumps({"response": response})
                except Exception as e:
                    response_json = json.dumps({"response": str(e)})
                response = response_json

            context[output_key] = response
            return {output_key: response}

        except Exception as e:
            self.error_count += 1
            self.logger.error(f"Error in execute_pipeline_step: {str(e)}")
            return {"error": f"MockAIModel error: {str(e)}"}

    def chat_completion(self, model: str, messages: List[Dict[str, str]]) -> str:
        """Enhanced chat completion with context awareness

        Args:
            model: Name of the model to use
            messages: List of message dictionaries with 'role' and 'content' keys

        Returns:
            str: Generated response

        Raises:
            ValueError: If messages is not a list of dicts
        """
        try:
            if not isinstance(messages, list):
                raise ValueError("messages must be a list of dicts")

            # Extract conversation history
            conversation = "\n".join(
                f"{m.get('role', 'user')}: {m.get('content', '')}"
                for m in messages
            )

            # Check for canned responses
            for key in self.canned_responses:
                if key in conversation.lower():
                    return self.canned_responses[key]

            # Generate context-aware response based on configuration
            last_content = messages[-1].get('content', '')[:100]
            response = self._generate_response(last_content, len(messages))

            return self._truncate_response(response, model)

        except Exception as e:
            self.logger.error(f"chat_completion error: {e}")
            return f"Error: {e}"

    def text_completion(self, model: str, prompt: str) -> str:
        """Enhanced text completion with personality

        Args:
            model: Name of the model to use
            prompt: Text prompt for completion

        Returns:
            str: Generated response
        """
        try:
            # Check for canned responses
            prompt_lower = prompt.lower()
            if any(key in prompt_lower for key in self.canned_responses):
                for key in self.canned_responses:
                    if key in prompt_lower:
                        return self.canned_responses[key]

            response = self._generate_response(prompt[:100])
            return self._truncate_response(response, model)

        except Exception as e:
            self.logger.error(f"text_completion error: {e}")
            return f"Error: {e}"

    def _generate_response(self, prompt: str, message_count: int = 1) -> str:
        """Generate response based on configured style and personality

        Args:
            prompt: Text prompt to generate response for
            message_count: Number of messages in the conversation

        Returns:
            str: Generated response
        """
        base = f"Regarding '{prompt}':\n"

        # Include part of the prompt in the response
        prompt_snippet = prompt[:50]

        if self.config["response_style"] == "helpful":
            response = (
                f"{base}Here's a detailed response considering your query. "
                f"I've processed {message_count} messages in this conversation. "
                f"You mentioned: '{prompt_snippet}'"
            )
        elif self.config["response_style"] == "concise":
            response = f"{base}Response: {prompt_snippet}"
        elif self.config["response_style"] == "technical":
            response = (
                f"{base}Technical analysis:\n"
                f"- Input length: {len(prompt)}\n"
                f"- Context items: {message_count}\n"
                f"- Keywords identified: {prompt_snippet}\n"
                f"- Suggested approach: Consider multiple factors"
            )
        else:  # creative
            response = (
                f"{base}*Creative interpretation*\n"
                f"Let me suggest an innovative perspective on this topic... "
                f"Based on your input: '{prompt_snippet}'"
            )

        # Add personality flavor
        if self.config["personality"] == "enthusiastic":
            response = f"Great question! {response} I'm excited to help with this!"
        elif self.config["personality"] == "skeptical":
            response = f"Hmm, interesting. {response} But have you considered alternatives?"

        return response

    def _truncate_response(self, response: str, model: str) -> str:
        """Ensure response doesn't exceed model limits

        Args:
            response: Response text to truncate
            model: Name of the model to apply limits for

        Returns:
            str: Truncated response
        """
        max_len = self.model_capabilities.get(model, {}).get("max_tokens", 500)
        return response[:max_len]

    def get_available_models(self) -> List[str]:
        """Get list of available models with capabilities

        Returns:
            List[str]: List of model names
        """
        return list(self.model_capabilities.keys())

    def get_model_info(self, model_name: str) -> Dict[str, Any]:
        """Get detailed capabilities for a specific model

        Args:
            model_name: Name of the model to get info for

        Returns:
            Dict[str, Any]: Model capabilities
        """
        return self.model_capabilities.get(model_name, {})

    def get_metrics(self) -> Dict[str, Any]:
        """Get metrics about model usage

        Returns:
            Dict[str, Any]: Metrics including call count, error count, and response lengths
        """
        return {
            "call_count": self.call_count,
            "error_count": self.error_count,
            "average_response_length": sum(self.response_lengths) / len(self.response_lengths) if self.response_lengths else 0,
            "max_response_length": max(self.response_lengths) if self.response_lengths else 0,
            "min_response_length": min(self.response_lengths) if self.response_lengths else 0
        }

    def reset_metrics(self) -> None:
        """Reset all metrics counters"""
        self.call_count = 0
        self.error_count = 0
        self.response_lengths = []

# Aliases for PluginManager compatibility
Mockaimodel = MockAIModel
MockaimodelPlugin = MockAIModel

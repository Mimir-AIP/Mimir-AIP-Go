"""
Enhanced Mock AI model plugin for testing and development

Features:
- Configurable response patterns
- Context-aware responses
- Simulated latency
- Error injection
- Multiple model personalities
"""

import random
import time
import json
from typing import Dict, List, Union
from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
from Plugins.BasePlugin import BasePlugin

class MockAIModel(BaseAIModel, BasePlugin):
    """Enhanced Mock AI model plugin with realistic behaviors"""

    plugin_type = "AIModels"
    
    def __init__(self):
        """Initialize with default configuration"""
        self.name = "MockAIModel"
        self.config = {
            "response_style": "helpful",  # helpful, concise, creative, technical
            "latency_ms": (50, 200),     # min/max response delay in ms
            "error_rate": 0.05,          # 5% chance of error
            "max_tokens": 500,           # Max response length
            "personality": "neutral"     # neutral, enthusiastic, skeptical
        }
        self.context_history = []
        self.model_capabilities = {
            "mock-model-1": {"max_tokens": 500, "supports_images": False},
            "mock-model-2": {"max_tokens": 1000, "supports_images": True},
            "mock-model-pro": {"max_tokens": 2000, "supports_images": True}
        }

    def configure(self, **kwargs):
        """Update model configuration"""
        self.config.update(kwargs)

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Enhanced pipeline step execution with context tracking"""
        try:
            # Simulate processing delay
            delay = random.uniform(*self.config["latency_ms"]) / 1000
            time.sleep(delay)
            
            config = step_config.get("config", {})
            output_key = step_config.get("output", "mock_ai_output")
            
            # Store context for future reference
            self.context_history.append({
                "config": config,
                "context": context.copy()
            })
            
            if random.random() < self.config["error_rate"]:
                raise RuntimeError("Simulated random error")
                
            if "messages" in config:
                response = self.chat_completion(config.get("model", "mock-model-1"),
                                              config["messages"])
            elif "prompt" in config:
                response = self.text_completion(config.get("model", "mock-model-1"),
                                              config["prompt"])
            else:
                response = self._generate_response("No specific prompt provided")
                
            context[output_key] = response
            return {output_key: response}
            
        except Exception as e:
            return {"error": f"MockAIModel error: {str(e)}"}

    def chat_completion(self, model, messages):
        """Enhanced chat completion with context awareness"""
        try:
            if not isinstance(messages, list):
                raise ValueError("messages must be a list of dicts")
                
            # Extract conversation history
            conversation = "\n".join(
                f"{m.get('role', 'user')}: {m.get('content', '')}"
                for m in messages
            )
            
            # Generate context-aware response based on configuration
            last_content = messages[-1].get('content', '')[:100]
            response = self._generate_response(last_content, len(messages))
            
            return self._truncate_response(response, model)

        except Exception as e:
            import logging
            logging.getLogger(__name__).error(f"[MockAIModel] chat_completion error: {e}")
            return f"[MockAIModel] Error: {e}"

    def text_completion(self, model, prompt):
        """Enhanced text completion with personality"""
        try:
            response = self._generate_response(prompt[:100])
            return self._truncate_response(response, model)
        except Exception as e:
            return f"[MockAIModel] Error: {e}"

    def _generate_response(self, prompt, message_count=1):
        """Generate response based on configured style and personality"""
        base = f"Regarding '{prompt}':\n"
        
        if self.config["response_style"] == "helpful":
            response = (f"{base}Here's a detailed response considering your query. "
                      f"I've processed {message_count} messages in this conversation.")
        elif self.config["response_style"] == "concise":
            response = f"{base}Response: {prompt[:50]}"
        elif self.config["response_style"] == "technical":
            response = (f"{base}Technical analysis:\n"
                       f"- Input length: {len(prompt)}\n"
                       f"- Context items: {message_count}\n"
                       f"- Suggested approach: Consider multiple factors")
        else:  # creative
            response = (f"{base}*Creative interpretation*\n"
                       f"Let me suggest an innovative perspective on this topic...")
        
        # Add personality flavor
        if self.config["personality"] == "enthusiastic":
            response = f"Great question! {response} I'm excited to help with this!"
        elif self.config["personality"] == "skeptical":
            response = f"Hmm, interesting. {response} But have you considered alternatives?"
            
        return response

    def _truncate_response(self, response, model):
        """Ensure response doesn't exceed model limits"""
        max_len = self.model_capabilities.get(model, {}).get("max_tokens", 500)
        return response[:max_len]

    def get_available_models(self):
        """Get list of available models with capabilities"""
        return list(self.model_capabilities.keys())

    def get_model_info(self, model_name):
        """Get detailed capabilities for a specific model"""
        return self.model_capabilities.get(model_name, {})

# Aliases for PluginManager compatibility
Mockaimodel = MockAIModel
MockaimodelPlugin = MockAIModel

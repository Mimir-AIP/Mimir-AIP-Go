"""
Pollinations AI model plugin providing access to multiple AI models including text, image, and audio generation.
"""

import requests
import logging
from typing import Dict, List, Any, Optional
from urllib.parse import urlencode
import base64
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))))
from src.Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel


class Pollinations(BaseAIModel):
    """Pollinations AI model plugin"""
    
    plugin_type = "AIModels"
    
    def __init__(self):
        """Initialize the Pollinations plugin"""
        self.text_base_url = "https://text.pollinations.ai"
        self.image_base_url = "https://image.pollinations.ai"
        self.logger = logging.getLogger("Plugins.AIModels.Pollinations")
        self.logger.setLevel(logging.DEBUG)
        self.logger.propagate = True
        self.logger.debug("Pollinations plugin initialized")
        
    def get_available_models(self) -> Dict[str, List[str]]:
        """
        Get list of available models for all modalities
        
        Returns:
            dict: Dictionary containing lists of available models for text, image, and audio
        """
        try:
            # Get text/audio models
            text_models = self._get_text_models()
            # Get image models
            image_models = self._get_image_models()
            # Combine and format
            return {
                "text_models": text_models,
                "image_models": image_models,
                "audio_models": ["openai-audio"],
                "available_voices": ["alloy", "echo", "fable", "onyx", "nova", "shimmer"]
            }
        except Exception as e:
            self.logger.error(f"Error fetching models: {e}")
            return {
                "text_models": ["openai", "mistral"],
                "image_models": ["flux"],
                "audio_models": ["openai-audio"],
                "available_voices": ["alloy", "echo", "fable", "onyx", "nova", "shimmer"]
            }
            
    def _get_text_models(self) -> List[str]:
        """Get available text and audio models"""
        try:
            response = requests.get(f"{self.text_base_url}/models")
            response.raise_for_status()
            models = response.json()
            return [{"name": m, "description": f"{m} model"} for m in models]
        except Exception as e:
            self.logger.error(f"Error fetching text models: {e}")
            return [{"name": "openai", "description": "OpenAI model"}, {"name": "mistral", "description": "Mistral model"}]
            
    def _get_image_models(self) -> List[str]:
        """Get available image models"""
        try:
            response = requests.get(f"{self.image_base_url}/models")
            response.raise_for_status()
            models = response.json()
            return [{"name": m, "description": f"{m} model"} for m in models]
        except Exception as e:
            self.logger.error(f"Error fetching image models: {e}")
            return [{"name": "flux", "description": "Flux model"}]
            
    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """
        Execute a pipeline step for this plugin
        
        Args:
            step_config: Configuration for the step
            context: Pipeline execution context
            
        Returns:
            dict: Step execution result
        """
        try:
            config = step_config["config"]
            output_type = config.get("output_type", "text")
            
            if output_type == "image":
                return self._handle_image_generation(config)
            elif output_type == "audio":
                return self._handle_audio_generation(config)
            elif output_type == "multimodal" and config.get("image"):
                return self._handle_multimodal_request(config)
            else:
                return self._handle_text_generation(config)
                
        except Exception as e:
            self.logger.error(f"Error in execute_pipeline_step: {e}", exc_info=True)
            return {step_config.get("output", "error"): f"Pollinations ERROR: {e}"}
            
    def _handle_text_generation(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle text generation requests"""
        model = config.get("model", "openai")
        
        # Use messages if provided, otherwise use prompt
        messages = config.get("messages", [{"role": "user", "content": config["prompt"]}])
        
        payload = {
            "model": model,
            "messages": messages,
            "stream": config.get("stream", False)
        }
        
        response = requests.post(
            f"{self.text_base_url}/openai",
            headers={"Content-Type": "application/json"},
            json=payload,
            stream=config.get("stream", False)
        )
        
        if config.get("stream", False):
            return self._handle_streaming_response(response)
        return response.json()
        
    def _handle_image_generation(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle image generation requests"""
        prompt = config["prompt"]
        model = config.get("model", "flux")
        width = config.get("width", 1024)
        height = config.get("height", 1024)
        
        # Prepare URL parameters
        params = {
            "model": model,
            "width": width,
            "height": height,
            "seed": config.get("seed"),
            "nologo": config.get("nologo", False),
            "private": config.get("private", False),
            "enhance": config.get("enhance", False),
            "safe": config.get("safe", False)
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Make request
        response = requests.get(
            f"{self.image_base_url}/prompt/{prompt}",
            params=params,
            stream=True
        )
        
        if response.status_code == 200:
            # Save image to file
            import os
            from datetime import datetime
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"generated_image_{timestamp}.jpg"
            
            with open(filename, 'wb') as f:
                for chunk in response.iter_content(chunk_size=8192):
                    f.write(chunk)
            
            return {"image_path": os.path.abspath(filename)}
        else:
            response.raise_for_status()
            
    def _handle_audio_generation(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle audio generation requests"""
        text = config["text"]
        voice = config.get("voice", "alloy")
        
        # Prepare URL
        params = {
            "model": "openai-audio",
            "voice": voice
        }
        
        # Make request
        response = requests.get(
            f"{self.text_base_url}/{text}",
            params=params,
            stream=True
        )
        
        if response.status_code == 200:
            # Save audio to file
            import os
            from datetime import datetime
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"generated_audio_{timestamp}.mp3"
            
            with open(filename, 'wb') as f:
                for chunk in response.iter_content(chunk_size=8192):
                    f.write(chunk)
            
            return {"audio_path": os.path.abspath(filename)}
        else:
            response.raise_for_status()
            
    def _handle_multimodal_request(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle multimodal requests (text + image)"""
        model = config.get("model", "openai")
        prompt = config["prompt"]
        image_url = config["image"]  # Should be a URL
        
        # Prepare message with image
        messages = [{
            "role": "user",
            "content": [
                {"type": "text", "text": prompt},
                {
                    "type": "image_url",
                    "image_url": {"url": image_url}
                }
            ]
        }]
        
        payload = {
            "model": model,
            "messages": messages,
            "stream": config.get("stream", False)
        }
        
        response = requests.post(
            f"{self.text_base_url}/openai",
            headers={"Content-Type": "application/json"},
            json=payload,
            stream=config.get("stream", False)
        )
        
        if config.get("stream", False):
            return self._handle_streaming_response(response)
        return response.json()
        
    def _process_image(self, image_data: Any) -> str:
        """Convert image to base64 if needed"""
        if isinstance(image_data, str):  # URL
            return image_data
        elif isinstance(image_data, bytes):  # Raw bytes
            return f"data:image/jpeg;base64,{base64.b64encode(image_data).decode()}"
        else:
            raise ValueError("Unsupported image format")
            
    def _handle_streaming_response(self, response: requests.Response) -> Dict[str, Any]:
        """Handle streaming responses"""
        full_response = ""
        
        for chunk in response.iter_content(chunk_size=1024):
            if chunk:
                try:
                    chunk_text = chunk.decode('utf-8')
                    if chunk_text.strip() == '[DONE]':
                        break
                    full_response += chunk_text
                except Exception as e:
                    self.logger.error(f"Error processing stream chunk: {e}")
        
        return {"streamed_content": full_response}
        
    def chat_completion(self, model: str, messages: List[Dict[str, Any]]) -> str:
        """
        Generate a chat completion using the specified model
        
        Args:
            model: Model identifier to use for completion
            messages: List of message dicts with 'role' and 'content' keys
        
        Returns:
            str: Generated response text
        """
        payload = {
            "model": model,
            "messages": messages,
            "stream": False
        }
        
        response = requests.post(
            f"{self.text_base_url}/openai",
            headers={"Content-Type": "application/json"},
            json=payload
        )
        
        return response.json()["choices"][0]["message"]["content"]
        
    def text_completion(self, model: str, prompt: str) -> str:
        """
        Generate a text completion using the specified model
        
        Args:
            model: Model identifier to use for completion
            prompt: Text prompt to complete
        
        Returns:
            str: Generated completion text
        """
        return self.chat_completion(model, [{"role": "user", "content": prompt}])
        
    def get_available_models(self) -> List[str]:
        """
        Get list of available models by combining text, image, and audio models
        
        Returns:
            list: List of model identifiers that can be used with this plugin
        """
        try:
            # Get text/audio models
            text_models = self._get_text_models()
            # Get image models
            image_models = self._get_image_models()
            # Combine all models
            all_models = set(text_models + image_models + ["openai-audio"])
            return list(all_models)
        except Exception as e:
            self.logger.error(f"Error fetching models: {e}")
            # Fallback to hardcoded list if API calls fail
            return ["openai", "mistral", "flux", "openai-audio"]

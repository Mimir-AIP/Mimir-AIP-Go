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
import json
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
            models = response.json().get("models", [])
            return [{"name": m} for m in models]
        except Exception as e:
            self.logger.error(f"Error fetching text models: {e}")
            return [{"name": "openai"}, {"name": "mistral"}]
            
    def _get_image_models(self) -> List[str]:
        """Get available image models"""
        try:
            response = requests.get(f"{self.image_base_url}/models")
            response.raise_for_status()
            models = response.json()
            return [{"name": m} for m in models]
        except Exception as e:
            self.logger.error(f"Error fetching image models: {e}")
            return [{"name": "flux"}]
            
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
            elif output_type == "text":
                return self._handle_text_generation(config)
            elif output_type == "stream":
                return self._handle_streaming_generation(config)
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
        
    def _handle_streaming_generation(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle streaming text generation"""
        try:
            # Prepare request
            url = f"{self.text_base_url}/stream"
            headers = {"Content-Type": "application/json"}
            data = {
                "prompt": config["prompt"],
                "model": config.get("model", "openai"),
                "stream": True
            }
            
            # Make request
            response = requests.post(url, json=data, headers=headers, stream=True)
            response.raise_for_status()
            
            # Process streaming response
            content = ""
            for line in response.iter_lines():
                if line:
                    json_line = line.decode('utf-8')
                    if json_line.startswith("data: "):
                        json_data = json_line[6:]
                        if json_data == "[DONE]":
                            break
                        try:
                            data = json.loads(json_data)
                            if "choices" in data and len(data["choices"]) > 0:
                                delta = data["choices"][0].get("delta", {})
                                content += delta.get("content", "")
                        except json.JSONDecodeError:
                            self.logger.error(f"Failed to parse streaming line: {json_line}")
                            continue
            
            # Make sure we have some content
            if not content:
                raise ValueError("No content received from streaming response")
            
            return {
                "success": True,
                "streamed_content": json.dumps({
                    "choices": [{
                        "delta": {
                            "content": content
                        },
                        "finish_reason": "stop"
                    }]
                }),
                "model": config.get("model", "openai")
            }
        except Exception as e:
            self.logger.error(f"Error in streaming generation: {e}")
            return {
                "success": False,
                "error": str(e)
            }
            
    def _handle_image_generation(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Handle image generation requests"""
        try:
            # Prepare request
            url = f"{self.image_base_url}/prompt/{config["prompt"]}"
            headers = {"Content-Type": "application/json"}
            params = {
                "model": config.get("model", "flux"),
                "width": config.get("width", 512),
                "height": config.get("height", 512),
                "nologo": config.get("nologo", True),
                "private": config.get("private", True),
                "enhance": config.get("enhance", True),
                "safe": config.get("safe", True)
            }
            
            # Make request
            response = requests.get(url, params=params, headers=headers)
            response.raise_for_status()
            
            # Process response
            # The API returns the image file directly
            image_path = self.test_output_dir / "test_image.png"
            with open(image_path, "wb") as f:
                f.write(response.content)
            
            return {
                "success": True,
                "image_path": str(image_path)
            }
        except Exception as e:
            self.logger.error(f"Error in image generation: {e}")
            return {
                "success": False,
                "error": str(e)
            }
            
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
            
    def _handle_streaming_response(self, response: requests.Response):
        """Handle streaming responses"""
        try:
            content = ""
            for line in response.iter_lines():
                if line:
                    json_line = line.decode('utf-8')
                    if json_line.startswith("data: "):
                        json_data = json_line[6:]
                        if json_data == "[DONE]":
                            break
                        try:
                            data = json.loads(json_data)
                            if "choices" in data and len(data["choices"]) > 0:
                                delta = data["choices"][0].get("delta", {})
                                content += delta.get("content", "")
                        except json.JSONDecodeError:
                            self.logger.error(f"Failed to parse streaming line: {json_line}")
                            continue
            
            # Make sure we have some content
            if not content:
                raise ValueError("No content received from streaming response")
            
            return {
                "success": True,
                "streamed_content": json.dumps({
                    "choices": [{
                        "delta": {
                            "content": content
                        },
                        "finish_reason": "stop"
                    }]
                }),
                "model": "openai"
            }
        except Exception as e:
            self.logger.error(f"Error handling streaming response: {e}")
            return {
                "success": False,
                "error": str(e)
            }
        
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

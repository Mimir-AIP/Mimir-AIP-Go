"""
Integration tests for the Pollinations plugin
"""

import unittest
import os
import sys
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch
import base64
import json
import requests
from PIL import Image
from src.Plugins.AIModels.Pollinations.Pollinations import Pollinations


class TestPollinations(unittest.TestCase):
    """Integration tests for the Pollinations plugin"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.plugin = Pollinations()
        self.test_dir = Path(__file__).parent
        self.test_images_dir = self.test_dir / "test_images"
        self.test_output_dir = self.test_dir / "test_output"
        self.test_output_dir.mkdir(exist_ok=True)
        
        # Basic test configurations
        self.text_config = {
            "config": {
                "model": "openai",
                "prompt": "What is the capital of France?",
                "json": "true"
            }
        }
        
        self.image_config = {
            "config": {
                "output_type": "image",
                "prompt": "A beautiful sunset",
                "model": "flux",
                "width": 512,
                "height": 512,
                "nologo": True,
                "private": True,
                "enhance": True,
                "safe": True
            }
        }
        
        self.audio_config = {
            "config": {
                "output_type": "audio",
                "text": "Hello world",
                "model": "openai-audio",
                "voice": "alloy"
            }
        }
        
        self.multimodal_config = {
            "config": {
                "output_type": "multimodal",
                "prompt": "Describe this image",
                "image": "https://raw.githubusercontent.com/Pollinations/ai/main/examples/1000000000_1234567890.png",
                "model": "openai",
                "messages": [{"role": "user", "content": "Describe this image"}]
            }
        }
        
        self.streaming_config = {
            "config": {
                "prompt": "Tell me a short story",
                "model": "turbo",
                "stream": True
            }
        }
        
    def test_get_available_models(self):
        """Test getting available models from actual API endpoints"""
        models = self.plugin.get_available_models()
        self.assertIn("openai", models)
        self.assertIn("flux", models)
        self.assertIn("openai-audio", models)
        
        # Verify we got models from both endpoints
        text_models = self.plugin._get_text_models()
        self.assertIn("openai", [m["name"] for m in text_models])
        self.assertIn("mistral", [m["name"] for m in text_models])
        
        image_models = self.plugin._get_image_models()
        self.assertIn("flux", [m["name"] for m in image_models])
        
    def test_text_generation(self):
        """Test actual text generation with different models"""
        # Test with OpenAI model
        config = {
            "config": {
                "model": "openai",
                "prompt": "What is the capital of France?",
                "messages": [{"role": "user", "content": "What is the capital of France?"}]
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertIn("choices", result)
        self.assertIn("content", result["choices"][0]["message"])
        self.assertTrue(len(result["choices"][0]["message"]["content"]) > 0)
        
        # Test streaming
        streaming_config = {
            "config": {
                "prompt": "Tell me a short story",
                "model": "turbo",
                "stream": True
            }
        }
        result = self.plugin.execute_pipeline_step(streaming_config, {})
        self.assertIn("streamed_content", result)
        self.assertTrue(len(result["streamed_content"]) > 0)
        
    def test_image_generation(self):
        """Test actual image generation with different configurations"""
        # Test basic image generation
        result = self.plugin.execute_pipeline_step(self.image_config, {})
        self.assertIn("image_path", result)
        self.assertTrue(os.path.exists(result["image_path"]))
        
        # Verify image dimensions
        from PIL import Image
        with Image.open(result["image_path"]) as img:
            self.assertEqual(img.size, (512, 512))
        
        # Test with different model
        mistral_config = {
            "config": {
                "output_type": "image",
                "prompt": "A beautiful sunset",
                "model": "mistral",
                "width": 512,
                "height": 512
            }
        }
        result = self.plugin.execute_pipeline_step(mistral_config, {})
        self.assertIn("image_path", result)
        self.assertTrue(os.path.exists(result["image_path"]))
        
        # Test with all optional parameters
        custom_config = {
            "config": {
                "output_type": "image",
                "prompt": "A beautiful sunset",
                "model": "flux",
                "width": 512,
                "height": 512,
                "seed": 42,
                "nologo": True,
                "private": True,
                "enhance": True,
                "safe": True
            }
        }
        result = self.plugin.execute_pipeline_step(custom_config, {})
        self.assertIn("image_path", result)
        self.assertTrue(os.path.exists(result["image_path"]))
        
    def test_audio_generation(self):
        """Test actual audio generation with different voices"""
        # Test with default voice
        result = self.plugin.execute_pipeline_step(self.audio_config, {})
        self.assertIn("audio_path", result)
        self.assertTrue(os.path.exists(result["audio_path"]))
        
        # Verify audio file is valid
        import wave
        try:
            with wave.open(result["audio_path"], 'rb') as wav:
                self.assertGreater(wav.getnframes(), 0)
        except wave.Error:
            # If not WAV, check file size
            import os
            file_size = os.path.getsize(result["audio_path"])
            self.assertGreater(file_size, 0)
        
        # Test with different voice
        nova_config = {
            "config": {
                "output_type": "audio",
                "text": "Hello world",
                "voice": "nova",
                "private": True
            }
        }
        result = self.plugin.execute_pipeline_step(nova_config, {})
        self.assertIn("audio_path", result)
        self.assertTrue(os.path.exists(result["audio_path"]))
        
    def test_multimodal_request(self):
        """Test actual multimodal requests with real images"""
        # Test with Turbo model
        result = self.plugin.execute_pipeline_step(self.multimodal_config, {})
        self.assertIn("choices", result)
        self.assertIn("content", result["choices"][0]["message"])
        self.assertTrue(len(result["choices"][0]["message"]["content"]) > 0)
        
        # Test with different image
        different_image_config = {
            "config": {
                "output_type": "multimodal",
                "prompt": "Describe this image",
                "image": str(self.test_images_dir / "193_Dublin_Road__Antrim.jpg"),
                "model": "turbo",
                "private": True
            }
        }
        result = self.plugin.execute_pipeline_step(different_image_config, {})
        self.assertIn("choices", result)
        self.assertIn("content", result["choices"][0]["message"])
        self.assertTrue(len(result["choices"][0]["message"]["content"]) > 0)
        
    def test_streaming_response(self):
        """Test actual streaming responses with different models"""
        # Test with OpenAI model
        config = {
            "config": {
                "model": "openai",
                "prompt": "Tell me a short story",
                "messages": [{"role": "user", "content": "Tell me a short story"}],
                "stream": True
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertIn("streamed_content", result)
        self.assertTrue(len(result["streamed_content"]) > 0)
        
        # Test streaming response handling
        with patch('requests.post') as mock_post:
            # Mock SSE client
            mock_sse_client = MagicMock()
            mock_sse_client.events.return_value = [
                MagicMock(data="Hello"),
                MagicMock(data="World"),
                MagicMock(data="[DONE]")
            ]
            
            # Mock response
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_post.return_value = mock_response
            
            # Test with streaming
            streaming_config = {
                "config": {
                    "model": "openai",
                    "prompt": "Tell me a short story",
                    "messages": [{"role": "user", "content": "Tell me a short story"}],
                    "stream": True
                }
            }
        
        with patch('sseclient.SSEClient', return_value=mock_sse_client):
            result = self.plugin.execute_pipeline_step(streaming_config, {})
            self.assertIn("streamed_content", result)
            self.assertEqual(result["streamed_content"], "HelloWorld")

if __name__ == '__main__':
    unittest.main()

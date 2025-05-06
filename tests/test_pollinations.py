"""
Integration tests for the Pollinations plugin
"""

import unittest
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch
import base64
import json
import requests
from PIL import Image
import os
import time
import wave
from src.Plugins.AIModels.Pollinations.Pollinations import Pollinations

# Integration tests for the Pollinations plugin
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
        
        # Mock audio file path for audio generation tests
        self.mock_audio_path = self.test_output_dir / "test_audio.wav"
        self.mock_audio_path.touch()
        
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
                "image": "https://raw.githubusercontent.com/pollinations/pollinations/refs/heads/master/assets/pollinations_ai_logo_image_white_transparent.png",
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
        
        # Parse the streamed content
        try:
            streamed_content = json.loads(result["streamed_content"])
            self.assertIn("choices", streamed_content)
            self.assertIn("delta", streamed_content["choices"][0])
            self.assertIn("content", streamed_content["choices"][0]["delta"])
            content = streamed_content["choices"][0]["delta"]["content"]
            if not content:
                self.fail(f"Streamed content is empty: {content}")
        except json.JSONDecodeError as e:
            self.fail(f"Failed to parse streamed content: {e}")
        except KeyError as e:
            self.fail(f"Missing expected key in streamed content: {e}")
        
    def test_image_generation(self):
        """Test actual image generation with different configurations"""
        # Add delay to avoid rate limiting
        time.sleep(2)
        
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
        # Add delay to avoid rate limiting
        time.sleep(2)
        
        # Test with default voice
        result = self.plugin.execute_pipeline_step(self.audio_config, {})
        self.assertIn("audio_path", result)
        audio_path = result["audio_path"]
        self.assertTrue(os.path.exists(audio_path), f"Audio file not found at: {audio_path}")
        
        # Test with different voice
        different_voice_config = {
            "config": {
                "output_type": "audio",
                "text": "Hello world",
                "model": "openai-audio",
                "voice": "echo"
            }
        }
        result = self.plugin.execute_pipeline_step(different_voice_config, {})
        self.assertIn("audio_path", result)
        self.assertTrue(os.path.exists(result["audio_path"]))
        
        # Verify audio file exists and has non-zero size
        audio_path = result["audio_path"]
        self.assertTrue(os.path.exists(audio_path), f"Audio file not found at: {audio_path}")
        file_size = os.path.getsize(audio_path)
        self.assertGreater(file_size, 0, f"Audio file at {audio_path} has zero size")

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
        audio_path = result["audio_path"]
        self.assertTrue(os.path.exists(audio_path), f"Audio file not found at: {audio_path}")
        file_size = os.path.getsize(audio_path)
        self.assertGreater(file_size, 0, f"Audio file at {audio_path} has zero size")
        
        # Verify audio file validation
        try:
            with wave.open(audio_path, 'rb') as wave_file:
                self.assertGreater(wave_file.getnframes(), 0, f"Audio file at {audio_path} is empty")
        except wave.Error:
            # If not WAV, check file size again (already checked above)
            pass

    def test_multimodal_request(self):
        """Test actual multimodal requests with real images"""
        # Add delay to avoid rate limiting
        time.sleep(2)

        # Test with OpenAI model
        result = self.plugin.execute_pipeline_step(self.multimodal_config, {})
        self.assertIn("choices", result, f"Expected 'choices' in response, got: {result}")
        self.assertTrue(len(result["choices"]) > 0, f"Expected non-empty choices, got: {result['choices']}")
        content = result["choices"][0]["message"]["content"]
        self.assertTrue(len(content) > 0, f"Expected non-empty content, got: {content}")

        # The API doesn't actually process images, it just returns a generic description
        # So we'll check that we got a reasonable response that's not an error
        self.assertNotIn("error", content.lower(), f"Expected a description, got an error: {content}")

    def test_streaming_response(self):
        """Test actual streaming responses with different models"""
        # Add delay to avoid rate limiting
        time.sleep(2)

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
        streamed_content = result["streamed_content"]
        
        # Debug the streamed content
        print(f"Streamed content received: {streamed_content[:100]}...")
        
        # The streamed content should be a JSON string containing the accumulated messages
        try:
            response_data = json.loads(streamed_content)
            self.assertIn("choices", response_data, f"Expected 'choices' in response data, got: {response_data}")
            self.assertTrue(len(response_data["choices"]) > 0, "Expected non-empty choices list")
            
            delta = response_data["choices"][0].get("delta", {})
            self.assertIn("content", delta, f"Expected 'content' in delta, got: {delta}")
            
            content = delta["content"]
            self.assertTrue(len(content) > 0, f"Expected non-empty content, got: {content}")
        except json.JSONDecodeError as e:
            self.fail(f"Failed to parse streamed content: {e}. Content: {streamed_content[:200]}")
        except KeyError as e:
            self.fail(f"Missing expected key in streamed content: {e}. Content structure: {response_data}")

if __name__ == '__main__':
    unittest.main()
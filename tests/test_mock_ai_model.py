"""
Unit tests for the MockAIModel plugin

To run tests:
1. Ensure you're in the project root directory
2. Run: PYTHONPATH=<project_root> python tests/test_mock_ai_model.py
   (Replace <project_root> with the absolute path to the project directory)
"""

import os
import sys
import unittest

# Add src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.AIModels.MockAIModel.MockAIModel import MockAIModel

class TestMockAIModel(unittest.TestCase):
    """Test suite for MockAIModel"""

    def setUp(self):
        """Set up test fixture"""
        self.model = MockAIModel()
        self.context = {}

    def test_initialization(self):
        """Test model initialization"""
        self.assertEqual(self.model.name, "MockAIModel")
        self.assertIn("response_style", self.model.config)
        self.assertIn("mock-model-1", self.model.model_capabilities)

    def test_configuration(self):
        """Test model configuration"""
        self.model.configure(response_style="concise", error_rate=0.1)
        self.assertEqual(self.model.config["response_style"], "concise")
        self.assertEqual(self.model.config["error_rate"], 0.1)

    def test_text_completion(self):
        """Test text completion functionality"""
        response = self.model.text_completion("mock-model-1", "What is AI?")
        self.assertIsInstance(response, str)
        self.assertTrue(len(response) > 0)

    def test_chat_completion(self):
        """Test chat completion functionality"""
        messages = [
            {"role": "user", "content": "Hello"},
            {"role": "assistant", "content": "Hi there!"},
            {"role": "user", "content": "What's your name?"}
        ]
        response = self.model.chat_completion("mock-model-1", messages)
        self.assertIsInstance(response, str)
        self.assertTrue(len(response) > 0)

    def test_canned_responses(self):
        """Test canned responses"""
        response = self.model.text_completion("mock-model-1", "What is the capital of France?")
        self.assertEqual(response, "Paris")

    def test_error_simulation(self):
        """Test error simulation"""
        # Temporarily increase error rate to ensure we hit an error
        original_error_rate = self.model.config["error_rate"]
        self.model.configure(error_rate=1.0)

        # Execute with error-prone configuration
        result = self.model.execute_pipeline_step(
            {"config": {"prompt": "This should cause an error"}},
            self.context
        )

        # Restore original error rate
        self.model.configure(error_rate=original_error_rate)

        # Check that we got an error response
        self.assertIn("error", result)

    def test_context_persistence(self):
        """Test context persistence"""
        # Enable context persistence
        self.model.configure(context_persistence=True)

        # Execute first step
        step1 = {"config": {"prompt": "First message"}}
        result1 = self.model.execute_pipeline_step(step1, self.context)
        self.assertIn("mock_ai_output", result1)

        # Execute second step
        step2 = {"config": {"prompt": "Second message"}}
        result2 = self.model.execute_pipeline_step(step2, self.context)
        self.assertIn("mock_ai_output", result2)

        # Verify context history was updated
        self.assertGreater(len(self.model.context_history), 0)

    def test_metrics_collection(self):
        """Test metrics collection"""
        # Execute multiple steps
        for i in range(5):
            step = {"config": {"prompt": f"Message {i}"}}
            self.model.execute_pipeline_step(step, self.context)

        # Get metrics
        metrics = self.model.get_metrics()
        self.assertIn("call_count", metrics)
        self.assertIn("error_count", metrics)
        self.assertIn("average_response_length", metrics)

        # Reset metrics
        self.model.reset_metrics()
        metrics_after_reset = self.model.get_metrics()
        self.assertEqual(metrics_after_reset["call_count"], 0)
        self.assertEqual(metrics_after_reset["error_count"], 0)

if __name__ == "__main__":
    unittest.main()
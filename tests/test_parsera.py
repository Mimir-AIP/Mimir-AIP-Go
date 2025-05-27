import unittest
from src.Plugins.Input.Parsera.Parsera import Parsera

class TestParseraPlugin(unittest.TestCase):

    def setUp(self):
        self.plugin = Parsera()

    def test_llm_specs(self):
        """Test fetching LLM specifications from the Parsera API."""
        result = self.plugin.llm_specs()
        self.assertIsNotNone(result)
        self.assertIsInstance(result, list)
        self.assertGreater(len(result), 0)
        self.assertIn('name', result[0])
        self.assertIn('id', result[0])
        self.assertIn('provider', result[0])

        # Print the first model's name and provider
        print(f"LLM Specs - First model: {result[0]['name']} by {result[0]['provider']}")

    def test_execute_pipeline_step_get_specs(self):
        """Test executing a pipeline step to get full LLM specifications."""
        context = {}
        step_config = {"endpoint": "llm-specs"}

        result = self.plugin.execute_pipeline_step(step_config, context)

        self.assertIn("parsera_response", result)
        self.assertIsInstance(result["parsera_response"], list)
        self.assertGreater(len(result["parsera_response"]), 0)

        # Print the first model's name and provider from the response
        print(f"Pipeline Step (full specs) - First model: {result['parsera_response'][0]['name']} by {result['parsera_response'][0]['provider']}")

    def test_execute_pipeline_step_filter_models(self):
        """Test executing a pipeline step to filter models."""
        context = {}
        step_config = {
            "filter": {
                "provider": "anthropic",
                "min_context_window": 100000,
                "capabilities": ["function_calling"]
            }
        }

        result = self.plugin.execute_pipeline_step(step_config, context)

        self.assertIn("filtered_parsera_models", result)
        self.assertIsInstance(result["filtered_parsera_models"], list)
        self.assertGreater(len(result["filtered_parsera_models"]), 0)

        # Verify that all filtered models match the criteria
        for model in result["filtered_parsera_models"]:
            self.assertEqual(model["provider"], "anthropic")
            self.assertGreaterEqual(model["context_window"], 100000)
            self.assertIn("function_calling", model["capabilities"])

        # Print the number of filtered models and the first one's name
        print(f"Pipeline Step (filtered) - Found {len(result['filtered_parsera_models'])} models")
        print(f"First filtered model: {result['filtered_parsera_models'][0]['name']}")

    def test_filter_models(self):
        """Test filtering models based on various criteria."""
        # Get all models
        models = self.plugin.llm_specs()
        self.assertIsNotNone(models)

        # Test filtering by name (partial match)
        filtered = self.plugin.filter_models(models, name="Claude")
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertIn("claude", model["name"].lower())

        # Test filtering by provider
        filtered = self.plugin.filter_models(models, provider="anthropic")
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertEqual(model["provider"], "anthropic")

        # Test filtering by context window size
        filtered = self.plugin.filter_models(models, min_context_window=100000)
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertGreaterEqual(model["context_window"], 100000)

        # Test filtering by output tokens
        filtered = self.plugin.filter_models(models, min_output_tokens=10000)
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertGreaterEqual(model["max_output_tokens"], 10000)

        # Test filtering by capabilities
        filtered = self.plugin.filter_models(models, capabilities=["function_calling"])
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertIn("function_calling", model["capabilities"])

        # Test combining multiple criteria
        filtered = self.plugin.filter_models(
            models,
            provider="anthropic",
            min_context_window=100000,
            capabilities=["function_calling"]
        )
        self.assertGreater(len(filtered), 0)
        for model in filtered:
            self.assertEqual(model["provider"], "anthropic")
            self.assertGreaterEqual(model["context_window"], 100000)
            self.assertIn("function_calling", model["capabilities"])

if __name__ == '__main__':
    unittest.main()
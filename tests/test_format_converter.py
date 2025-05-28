import sys
import os
import unittest

# Add src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Data_Processing.FormatConverter.FormatConverter import FormatConverter

class TestFormatConverter(unittest.TestCase):
    def setUp(self):
        self.plugin = FormatConverter()
        self.test_context = {}

    def test_string_to_dict_conversion(self):
        # Test direct conversion
        test_string = '{"key": "value"}'
        expected = {"key": "value"}
        
        # Test with strict JSON
        result = self.plugin._convert_string_to_dict(test_string, strict_json=True)
        self.assertEqual(result, expected)
        
        # Test with Python literal fallback
        test_string = "{'key': 'value'}"
        result = self.plugin._convert_string_to_dict(test_string, strict_json=False)
        self.assertEqual(result, expected)

    def test_dict_to_string_conversion(self):
        # Test direct conversion
        test_dict = {"key": "value"}
        expected = '{"key": "value"}'
        
        result = self.plugin._convert_dict_to_string(test_dict)
        self.assertEqual(result, expected)

    def test_execute_pipeline_step_string_to_dict(self):
        # Test pipeline execution
        step_config = {
            "config": {
                "input_key": "test_input",
                "output_key": "test_output",
                "conversion_type": "string_to_dict",
                "strict_json": True
            }
        }
        
        self.test_context["test_input"] = '{"key": "value"}'
        expected = {"key": "value"}
        
        self.plugin.execute_pipeline_step(step_config, self.test_context)
        self.assertEqual(self.test_context["test_output"], expected)

    def test_execute_pipeline_step_dict_to_string(self):
        # Test pipeline execution
        step_config = {
            "config": {
                "input_key": "test_input",
                "output_key": "test_output",
                "conversion_type": "dict_to_string"
            }
        }
        
        self.test_context["test_input"] = {"key": "value"}
        expected = '{"key": "value"}'
        
        self.plugin.execute_pipeline_step(step_config, self.test_context)
        self.assertEqual(self.test_context["test_output"], expected)

    def test_invalid_conversion_type(self):
        step_config = {
            "config": {
                "input_key": "test_input",
                "output_key": "test_output",
                "conversion_type": "invalid_type"
            }
        }
        
        self.test_context["test_input"] = {"key": "value"}
        
        with self.assertRaises(ValueError):
            self.plugin.execute_pipeline_step(step_config, self.test_context)

    def test_missing_input_key(self):
        step_config = {
            "config": {
                "output_key": "test_output",
                "conversion_type": "string_to_dict"
            }
        }
        
        with self.assertRaises(ValueError):
            self.plugin.execute_pipeline_step(step_config, self.test_context)

    def test_missing_output_key(self):
        step_config = {
            "config": {
                "input_key": "test_input",
                "conversion_type": "string_to_dict"
            }
        }
        
        with self.assertRaises(ValueError):
            self.plugin.execute_pipeline_step(step_config, self.test_context)

if __name__ == "__main__":
    unittest.main()
import json
import ast
import logging
from Plugins.BasePlugin import BasePlugin

class FormatConverter(BasePlugin):
    """
    Plugin to convert data between various formats within the pipeline context.
    Supports string-to-dictionary (JSON/Python literal) and dictionary-to-string (JSON) conversions.
    """
    plugin_type = "Data_Processing"

    def _convert_string_to_dict(self, input_string: str, strict_json: bool = False) -> dict:
        """
        Converts a string to a dictionary.

        Args:
            input_string (str): The string to convert.
            strict_json (bool): If True, only JSON format is accepted. If False, also accepts Python literals.

        Returns:
            dict: The converted dictionary.

        Raises:
            ValueError: If the string cannot be converted to a dictionary.
        """
        try:
            return json.loads(input_string)
        except json.JSONDecodeError as e:
            if not strict_json:
                try:
                    return ast.literal_eval(input_string)
                except (ValueError, SyntaxError) as ast_e:
                    raise ValueError(f"Failed to parse string as JSON or Python literal: {e}, {ast_e}")
            else:
                raise ValueError(f"Failed to parse string as strict JSON: {e}")

    def _convert_dict_to_string(self, input_dict: dict) -> str:
        """
        Converts a dictionary to a JSON string.

        Args:
            input_dict (dict): The dictionary to convert.

        Returns:
            str: The JSON string representation of the dictionary.

        Raises:
            ValueError: If the dictionary cannot be serialized to JSON.
        """
        try:
            return json.dumps(input_dict)
        except TypeError as e:
            raise ValueError(f"Failed to serialize dictionary to string: {e}")

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """
        Executes the data conversion based on the specified configuration.

        Args:
            step_config (dict): Configuration for this pipeline step, including:
                - config (dict):
                    - input_key (str): The context key holding the data to convert.
                    - output_key (str): The context key where the converted data will be stored.
                    - conversion_type (str): Type of conversion ('string_to_dict' or 'dict_to_string').
                    - strict_json (bool, optional): For 'string_to_dict', if True, uses json.loads.
                                                    If False (default), tries json.loads, then falls back to ast.literal_eval.
            context (dict): The current pipeline context.

        Returns:
            dict: An empty dictionary, as the result is stored directly in the context.

        Raises:
            ValueError: If configuration is invalid, input data is missing/incorrect,
                        or conversion fails.
        """
        cfg = step_config.get("config", {})
        input_key = cfg.get("input_key")
        output_key = cfg.get("output_key")
        conversion_type = cfg.get("conversion_type")
        strict_json = cfg.get("strict_json", False)

        logger = logging.getLogger(__name__)
        logger.info(f"[FormatConverter] Attempting conversion: {conversion_type} from '{input_key}' to '{output_key}'")

        if not input_key or input_key not in context:
            raise ValueError(f"FormatConverter: Missing or invalid input_key: '{input_key}'")
        if not output_key:
            raise ValueError("FormatConverter: Missing 'output_key' in configuration.")
        if not conversion_type:
            raise ValueError("FormatConverter: Missing 'conversion_type' in configuration.")

        input_data = context.get(input_key)

        try:
            if conversion_type == "string_to_dict":
                if not isinstance(input_data, str):
                    raise ValueError(f"FormatConverter: Input for 'string_to_dict' must be a string, got {type(input_data)}")
                converted_data = self._convert_string_to_dict(input_data, strict_json)
                if not isinstance(converted_data, dict):
                    raise ValueError(f"FormatConverter: Converted data for 'string_to_dict' is not a dictionary, got {type(converted_data)}")

            elif conversion_type == "dict_to_string":
                if not isinstance(input_data, dict):
                    raise ValueError(f"FormatConverter: Input for 'dict_to_string' must be a dictionary, got {type(input_data)}")
                converted_data = self._convert_dict_to_string(input_data)

            else:
                raise ValueError(f"FormatConverter: Unknown conversion_type: '{conversion_type}'")

            context[output_key] = converted_data
            logger.info(f"[FormatConverter] Successfully converted data from '{input_key}' to '{output_key}'. Result type: {type(converted_data)}")
            return {}

        except Exception as e:
            logger.error(f"[FormatConverter] Conversion failed: {str(e)}")
            raise
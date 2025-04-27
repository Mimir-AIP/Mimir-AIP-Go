"""
ExtractDictKey module.

Extracts a subkey from a dictionary in the pipeline context and stores it under a new context key.

Config Options:
    input_key (str): Context key for the source dictionary.
    extract_key (str): Key to extract from the dictionary.
    output_key (str): Context key to store the extracted value.

Usage Example (YAML):
  plugin: Data_Processing.ExtractDictKey
  config:
    input_key: wh_report
    extract_key: items
    output_key: wh_items
"""
from Plugins.BasePlugin import BasePlugin

class ExtractDictKey(BasePlugin):
    """Plugin to extract a subkey from a dict in context and store under a new key."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute extraction of a subkey from a dict in context.

        Args:
            step_config (dict): Must include 'config' dict with:
                input_key (str): Context key for source dict.
                extract_key (str): Subkey to extract.
                output_key (str): Context key to store the extracted value.
            context (dict): Current pipeline context.

        Returns:
            dict: Updated context dictionary.
        """
        cfg = step_config.get("config", {})
        input_key = cfg.get("input_key")
        extract_key = cfg.get("extract_key")
        output_key = cfg.get("output_key")
        d = context.get(input_key, {})
        value = d.get(extract_key, None) if isinstance(d, dict) else None
        context[output_key] = value
        return context
"""
ExtractDictKey Plugin for Mimir-AIP

Extracts a subkey (e.g., list or value) from a dict in the pipeline context and stores it under a new context key.

Config options:
- input_key: context key for the dict
- extract_key: key to extract from the dict
- output_key: context key to store the extracted value

Example usage:
  config:
    input_key: wh_report
    extract_key: items
    output_key: wh_items

Author: Cascade AI
"""
from Plugins.BasePlugin import BasePlugin

class ExtractDictKey(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """Extract a subkey from a dict in context and store under a new key."""
        cfg = step_config.get("config", {})
        input_key = cfg.get("input_key")
        extract_key = cfg.get("extract_key")
        output_key = cfg.get("output_key")
        d = context.get(input_key, {})
        value = d.get(extract_key, None) if isinstance(d, dict) else None
        context[output_key] = value
        return context

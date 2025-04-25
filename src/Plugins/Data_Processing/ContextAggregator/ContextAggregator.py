"""
Plugin for aggregating values from the context into a list under a specified key.
"""

from Plugins.BasePlugin import BasePlugin

class ContextAggregator(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Aggregates a value from the context into a list under a specified key.
        step_config['config']:
          key: the name of the list in context to append to
          value: the key in context whose value to append
          required_keys: (optional) list of required keys for dicts being aggregated
        """
        config = step_config['config']
        key = config["key"]
        value_key = config["value"]
        required_keys = config.get("required_keys", [])
        value = context[value_key]
        # If required_keys is set and value is a dict, fill missing keys
        if required_keys and isinstance(value, dict):
            for k in required_keys:
                value.setdefault(k, "N/A")
        # Defensive patch: If value is a stringified list/dict, parse it recursively
        import ast
        def parse_if_str(val):
            if isinstance(val, str):
                try:
                    parsed = ast.literal_eval(val)
                    # Only accept if result is list/dict
                    if isinstance(parsed, (list, dict)):
                        return parsed
                except Exception:
                    pass
            return val
        value = parse_if_str(value)
        # Also patch aggregation: if context[key] exists and is string, parse it
        if key in context and isinstance(context[key], str):
            context[key] = parse_if_str(context[key])
        import logging
        logger = logging.getLogger(__name__)
        logger.info(f"[ContextAggregator] (Patched) Aggregating value of type: {type(value)}, sample: {str(value)[:300]}")
        if key not in context:
            context[key] = []
        context[key].append(value)
        logger.info(f"[ContextAggregator] {key} now has {len(context[key])} items. Sample: {str(context[key])[:300]}")
        return {key: context[key]}

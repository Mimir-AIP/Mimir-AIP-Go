"""
Plugin for aggregating values from the context into a list under a specified key.
"""

from Plugins.BasePlugin import BasePlugin

class ContextAggregator(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Aggregates a value from the context into a list under a specified key.
        step_config:
          key: the name of the list in context to append to
          value: the key in context whose value to append
          required_keys: (optional) list of required keys for dicts being aggregated
        """
        key = step_config["key"]
        value_key = step_config["value"]
        required_keys = step_config.get("required_keys", [])
        value = context[value_key]
        # If required_keys is set and value is a dict, fill missing keys
        if required_keys and isinstance(value, dict):
            for k in required_keys:
                value.setdefault(k, "N/A")
        if key not in context:
            context[key] = []
        context[key].append(value)
        return {key: context[key]}

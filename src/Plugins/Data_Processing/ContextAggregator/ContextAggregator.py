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
        """
        key = step_config["key"]
        value_key = step_config["value"]
        if key not in context:
            context[key] = []
        context[key].append(context[value_key])
        return {key: context[key]}

"""
Plugin for promoting a variable from a nested or previous context into the main context.
"""

from Plugins.BasePlugin import BasePlugin

class ContextPromoter(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Copies the value of a source key to a target key in the main context.
        step_config:
          source: the key to copy from (e.g., from previous/nested context)
          target: the key to copy to in the current context
        """
        source = step_config["source"]
        target = step_config["target"]
        value = context.get(source)
        if value is not None:
            context[target] = value
        return {target: context.get(target)}

"""
ContextSetter module.

Pipeline-agnostic plugin to set one or more key-value pairs in the pipeline context.

Config Example (YAML):
  plugin: Data_Processing.ContextSetter
  config:
    values:
      headline_text: "Some headline"
      foo: 42
  output: null

All key-value pairs in 'values' are set in the context.
Returns a dict of the set key-value pairs.
"""
from Plugins.BasePlugin import BasePlugin
import logging

class ContextSetter(BasePlugin):
    """Pipeline-agnostic plugin to set key-value pairs in the context."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """Set specified key-value pairs in the pipeline context.

        Args:
            step_config (dict): Step configuration with 'config' dict containing 'values' (dict of key-value pairs).
            context (dict): Current pipeline context.

        Returns:
            dict: The values that were set.
        """
        config = step_config.get('config', {})
        values = config.get('values', {})
        if not isinstance(values, dict):
            raise ValueError("ContextSetter: 'values' must be a dict of key-value pairs to set.")
        logger = logging.getLogger(__name__)
        logger.info(f"[ContextSetter] Setting values: {values}")
        for k, v in values.items():
            context[k] = v
            logger.info(f"[ContextSetter] Set {k}: {v}")
        return dict(values)
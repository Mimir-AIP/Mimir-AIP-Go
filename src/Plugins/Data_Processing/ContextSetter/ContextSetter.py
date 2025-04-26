"""
ContextSetter Plugin for Mimir-AIP

Pipeline-agnostic plugin to set one or more key-value pairs in the pipeline context.
Compatible with any pipeline runner that passes the full step_config and context to execute_pipeline_step.

Config Example (YAML):
  plugin: Data_Processing.ContextSetter
  config:
    values:
      headline_text: "Some headline"
      foo: 42
  output: null

- All key-value pairs in 'values' are set in the context.
- Returns a dict of the set key-value pairs (for context merging).
- Does not mutate context outside of the provided keys.
- Robust error handling and clear logging.
"""
from Plugins.BasePlugin import BasePlugin
import logging

class ContextSetter(BasePlugin):
    """Pipeline-agnostic plugin to set key-value pairs in the context."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
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

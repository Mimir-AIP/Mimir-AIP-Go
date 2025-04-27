"""
Delay plugin module.

Introduces a delay (sleep) in pipeline execution. Useful for rate-limiting or pacing between steps.
"""

import time
from Plugins.BasePlugin import BasePlugin

class Delay(BasePlugin):
    """Plugin to pause execution for a specified number of seconds.

    Attributes:
        plugin_type (str): Plugin type identifier.
        logger (logging.Logger): Logger for diagnostic messages.
    """
    plugin_type = "Data_Processing"

    def __init__(self):
        """Initialize Delay plugin with a default logger."""
        super().__init__()
        import logging
        self.logger = logging.getLogger(__name__)

    def execute_pipeline_step(self, step_config, context):
        """Sleep for a specified duration and return a context flag.

        Args:
            step_config (dict): Configuration for the step. Supports:
                - config (dict): Contains 'seconds' (int) and 'output' (str) keys.
            context (dict): Pipeline context.

        Returns:
            dict: Contains {output_key: True} indicating completion.
        """
        # Robustly support both legacy and new step_config structures
        if 'config' in step_config and isinstance(step_config['config'], dict):
            seconds = step_config['config'].get('seconds', 1)
            output_key = step_config.get('output', step_config['config'].get('output', 'delay_done'))
        else:
            seconds = step_config.get('seconds', 1)
            output_key = step_config.get('output', 'delay_done')
        self.logger.info(f"[Delay] Sleeping for {seconds} seconds...")
        time.sleep(seconds)
        return {output_key: True}
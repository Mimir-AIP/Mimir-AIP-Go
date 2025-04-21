"""
Plugin for introducing a delay (sleep) in pipeline execution.
"""

import time
from Plugins.BasePlugin import BasePlugin

class Delay(BasePlugin):
    plugin_type = "Data_Processing"

    def __init__(self):
        super().__init__()
        import logging
        self.logger = logging.getLogger(__name__)

    def execute_pipeline_step(self, step_config, context):
        """
        step_config:
          seconds: number of seconds to sleep (default: 1)
          output: name of the output context key (default: 'delay_done')
        """
        seconds = step_config.get("seconds", 1)
        self.logger.info(f"[Delay] Sleeping for {seconds} seconds...")
        time.sleep(seconds)
        return {step_config.get("output", "delay_done"): True}

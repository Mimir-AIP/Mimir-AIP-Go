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

"""
Data processing plugin to extract URL strings from a list of dicts.
Prefers 'link', falls back to 'FirstURL'.
"""

from Plugins.BasePlugin import BasePlugin
import logging

class ExtractURLStrings(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Expects:
        {
            "plugin": "ExtractURLStrings",
            "config": {
                "input": "context_key_or_list"
            },
            "output": "output_key"
        }
        """
        config = step_config.get("config", {})
        input_data = config.get("input")
        if isinstance(input_data, str) and input_data in context:
            data = context[input_data]
        else:
            data = input_data
        logger = logging.getLogger(__name__)
        logger.info(f"[ExtractURLStrings] Received input: {data}")
        url_list = []
        if isinstance(data, list):
            for item in data:
                if isinstance(item, dict):
                    if 'link' in item:
                        url_list.append(item['link'])
                    elif 'FirstURL' in item:
                        url_list.append(item['FirstURL'])
        logger.info(f"[ExtractURLStrings] Output url_list: {url_list}")
        return {step_config["output"]: url_list}

"""
Plugin for loading a variable from a JSON file into the context.
"""
import json
import os
import logging
from Plugins.BasePlugin import BasePlugin

class FileToContext(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Loads a variable from a JSON file into the context.
        step_config:
          filename: the file to read from (relative or absolute)
          variable: the context key to set
          output: the output key (optional, defaults to variable)
        """
        filename = step_config["filename"]
        variable = step_config["variable"]
        output = step_config.get("output", variable)
        if not os.path.exists(filename):
            raise FileNotFoundError(f"File '{filename}' does not exist.")
        with open(filename, "r") as f:
            value = json.load(f)
        logger = logging.getLogger(__name__)
        logger.info(f"[FileToContext] Loaded variable '{variable}' of type {type(value)} from file '{filename}'. Sample: {str(value)[:300]}")
        context[variable] = value
        return {output: value}

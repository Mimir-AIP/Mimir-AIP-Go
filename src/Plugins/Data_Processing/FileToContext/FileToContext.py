"""
FileToContext module.

Loads a variable from a JSON file into the pipeline context.

Config:
    filename (str): Path to JSON file.
    variable (str): Context key to assign loaded data.
    output (str, optional): Output context key (defaults to variable).

Returns:
    dict: {output_key: loaded_value}.
"""
import json
import os
import logging
from Plugins.BasePlugin import BasePlugin

class FileToContext(BasePlugin):
    """Plugin to load JSON data from a file into the pipeline context.

    Attributes:
        plugin_type (str): 'Data_Processing'.
    """
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Load JSON variable from file into context.

        Args:
            step_config (dict): Must include 'filename' (str) and 'variable' (str); optional 'output' (str).
            context (dict): Current pipeline context.

        Returns:
            dict: {output_key: loaded_value}.

        Raises:
            FileNotFoundError: If file does not exist.
            JSONDecodeError: If the file content is invalid JSON.
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
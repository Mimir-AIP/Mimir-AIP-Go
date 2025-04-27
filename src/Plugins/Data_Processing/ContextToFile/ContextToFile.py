"""
ContextToFile module.

Writes a context variable to a file in JSON format.

Config (step_config):
    variable (str): Context key to write.
    filename (str): File path to write.
    append (bool, optional): If True, append to JSON list (default False).

Returns:
    dict: Empty dict (no context modifications).
"""
import json
import os
import logging
from Plugins.BasePlugin import BasePlugin

class ContextToFile(BasePlugin):
    """Plugin to write a context variable to a file as JSON.

    Attributes:
        plugin_type (str): 'Data_Processing'.
    """
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Write a context variable to a file in JSON format.

        Args:
            step_config (dict): Step configuration including:
                variable (str): Context key to write.
                filename (str): File path for output.
                append (bool, optional): Append value to a JSON list.
            context (dict): Current pipeline context.

        Returns:
            dict: Empty dict (no context modifications).
        """
        variable = step_config["variable"]
        filename = step_config["filename"]
        value = context.get(variable)
        logger = logging.getLogger(__name__)
        logger.info(f"[ContextToFile] Called with variable='{variable}', filename='{filename}', value={repr(value)}")
        logger.info(f"[ContextToFile] Writing variable '{variable}' of type {type(value)}. Sample: {str(value)[:300]}")
        if value is None:
            logger.warning(f"[ContextToFile] Variable '{variable}' not found in context; skipping write.")
            return {}
        append = step_config.get("append", False)
        if append:
            # Append value to a JSON list in the file
            if os.path.exists(filename):
                with open(filename, "r") as f:
                    try:
                        data = json.load(f)
                    except Exception:
                        data = []
            else:
                data = []
            data.append(value)
            with open(filename, "w") as f:
                json.dump(data, f)
        else:
            # Write value as is
            with open(filename, "w") as f:
                json.dump(value, f)
        logger.info(f"[ContextToFile] Wrote variable '{variable}' to file '{filename}'")
        return {}
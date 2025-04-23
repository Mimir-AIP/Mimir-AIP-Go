"""
ImageToBase64: Data processing plugin to convert image files to base64-encoded strings for use in vision-language models or APIs.

Usage:
- Provide the image file path in the context or via step_config.
- The plugin will output a base64 string with the appropriate data URL prefix.
"""

import os
import base64
from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager
import logging

class ImageToBase64(BasePlugin):
    """
    Data processing plugin to convert image files to base64-encoded strings.
    """
    plugin_type = "Data_Processing"

    def __init__(self, plugin_manager=None, logger=None):
        self.plugin_manager = plugin_manager
        self.logger = logger if logger is not None else logging.getLogger(__name__)

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """
        Convert an image file to a base64-encoded string.
        Args:
            step_config (dict): Pipeline step configuration.
            context (dict): Pipeline context.
        Returns:
            dict: Updated context with base64 string under output_key.
        """
        input_key = step_config.get('input_image_path_key', 'image_path')
        output_key = step_config.get('output_key', 'image_base64')
        image_path = context.get(input_key) or step_config.get('image_path')
        if not image_path or not os.path.isfile(image_path):
            raise ValueError(f"Image file not found: {image_path}")
        with open(image_path, 'rb') as f:
            image_bytes = f.read()
        base64_str = base64.b64encode(image_bytes).decode('utf-8')
        # Default to jpeg, can be customized via step_config
        mime_type = step_config.get('mime_type', 'image/jpeg')
        data_url = f"data:{mime_type};base64,{base64_str}"
        context[output_key] = data_url
        return context

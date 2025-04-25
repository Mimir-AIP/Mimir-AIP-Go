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
        # Always extract config from step_config['config'] for robustness
        config = step_config.get('config', {})
        input_key = config.get('input_key') or config.get('input_image_path_key') or 'image_path'
        output_key = config.get('output_key', 'image_base64')
        image_path = context.get(input_key) or config.get('image_path')
        logger = self.logger if hasattr(self, 'logger') else logging.getLogger(__name__)
        logger.info(f"[ImageToBase64] Using input_key: '{input_key}', resolved image_path: '{image_path}'")
        try:
            if not image_path:
                logger.error(f"Image path is None or empty: {image_path}")
                context[output_key] = "No data available"
                return context
            file_exists = os.path.isfile(image_path)
            logger.info(f"[ImageToBase64] File exists: {file_exists}")
            if file_exists:
                file_size = os.path.getsize(image_path)
                logger.info(f"[ImageToBase64] File size: {file_size} bytes")
            else:
                logger.error(f"Image file not found: {image_path}")
                context[output_key] = "No data available"
                return context
            with open(image_path, 'rb') as f:
                image_bytes = f.read()
            base64_str = base64.b64encode(image_bytes).decode('utf-8')
            # Default to jpeg, can be customized via config
            mime_type = config.get('mime_type', 'image/jpeg')
            # Only store the raw base64 string in the context; the HTML template should add the data URL prefix
            context[output_key] = base64_str
            logger.info(f"[ImageToBase64] Successfully encoded image to base64 (raw, no prefix) and set context['{output_key}'].")
            return context
        except Exception as e:
            logger.error(f"ImageToBase64: Error encoding image: {e}")
            context[output_key] = "No data available"
            return context

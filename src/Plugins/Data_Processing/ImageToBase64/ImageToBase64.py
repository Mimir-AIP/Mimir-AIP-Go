"""
ImageToBase64 module.

Converts an image file to a base64-encoded data URL for embedding or API usage.

Config (step_config['config']):
    input_key (str, optional): Context key or image path (default 'image_path').
    image_path (str, optional): Direct image file path override.
    output_key (str, optional): Context key for base64 string (default 'image_base64').

Returns:
    dict: {output_key: base64_string or fallback message}.
"""

import os
import base64
from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager
import logging

class ImageToBase64(BasePlugin):
    """Plugin to convert an image file to a base64-encoded data URL.

    Attributes:
        plugin_type (str): 'Data_Processing'.
        plugin_manager (PluginManager): Optional manager instance.
        logger (logging.Logger): Logger for diagnostic messages.
    """
    plugin_type = "Data_Processing"

    def __init__(self, plugin_manager=None, logger=None):
        """Initialize ImageToBase64 plugin.

        Args:
            plugin_manager (PluginManager, optional): Manager for plugin discovery.
            logger (logging.Logger, optional): Logger for diagnostic messages.
        """
        self.plugin_manager = plugin_manager
        self.logger = logger if logger is not None else logging.getLogger(__name__)

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Convert an image file to base64 data URL and update context.

        Args:
            step_config (dict): Step configuration; must contain 'config' dict.
            context (dict): Current pipeline context.

        Returns:
            dict: Updated context with {output_key: base64_string or fallback}.
        """
        # Always extract config from step_config['config'] for robustness
        config = step_config.get('config', {})
        input_key = config.get('input_key') or config.get('input_image_path_key') or 'image_path'
        output_key = config.get('output_key', 'image_base64')
        print(f"[DEBUG][ImageToBase64] config: {config}")
        print(f"[DEBUG][ImageToBase64] Context keys BEFORE: {list(context.keys())}")
        image_path = context.get(input_key) or config.get('image_path')
        print(f"[DEBUG][ImageToBase64] Using input_key: '{input_key}', resolved image_path: '{image_path}'")
        if not image_path:
            print(f"[DEBUG][ImageToBase64] Image path is None or empty: {image_path}")
            context[output_key] = "No data available"
            print(f"[DEBUG][ImageToBase64] Context keys AFTER: {list(context.keys())}")
            return context
        try:
            print(f"[DEBUG][ImageToBase64] Checking existence of file: {os.path.abspath(image_path)}")
            if not os.path.exists(image_path):
                print(f"[DEBUG][ImageToBase64] File does not exist: {image_path}")
                context[output_key] = "No data available"
                print(f"[DEBUG][ImageToBase64] Context keys AFTER: {list(context.keys())}")
                return context
            print(f"[DEBUG][ImageToBase64] File exists: {os.path.exists(image_path)}")
            print(f"[DEBUG][ImageToBase64] File size: {os.path.getsize(image_path)} bytes")
            with open(image_path, "rb") as f:
                image_bytes = f.read()
            base64_str = f"data:image/jpeg;base64,{base64.b64encode(image_bytes).decode('utf-8')}"
            context[output_key] = base64_str
            print(f"[DEBUG][ImageToBase64] Setting {output_key} in context to: (base64 string, length={len(base64_str)})")
            print(f"[DEBUG][ImageToBase64] Context keys AFTER: {list(context.keys())}")
            if self.logger:
                self.logger.info(f"[ImageToBase64] Successfully encoded image to base64 (raw, no prefix) and set context['{output_key}'].")
            return context
        except Exception as e:
            if self.logger:
                self.logger.error(f"ImageToBase64: Error encoding image: {e}")
            context[output_key] = "No data available"
            print(f"[DEBUG][ImageToBase64] Exception occurred, context keys AFTER: {list(context.keys())}")
            return context
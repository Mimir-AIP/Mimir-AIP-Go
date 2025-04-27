"""
DrawBoundingBoxes module.

Provides DrawBoundingBoxes plugin to draw bounding boxes on images based on detection results.
"""
import os
from PIL import Image, ImageDraw
from Plugins.BasePlugin import BasePlugin

class DrawBoundingBoxes(BasePlugin):
    """
    Plugin to draw bounding boxes on an image given detection results.
    Expects a list of bounding boxes with normalized coordinates (0-1).
    """
    plugin_type = "Data_Processing"

    def __init__(self, plugin_manager=None, logger=None):
        """Initialize the DrawBoundingBoxes plugin.

        Args:
            plugin_manager (PluginManager, optional): Manager for plugin discovery and context.
            logger (logging.Logger, optional): Logger for diagnostic messages.
        """
        self.plugin_manager = plugin_manager
        self.logger = logger

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """
        Draw bounding boxes on an image and save the result.
        Args:
            step_config (dict): Should contain:
                - config: dictionary with configuration
            context (dict): Pipeline context.
        Returns:
            dict: Updated context with new keys.
        """
        config = step_config.get('config', {})
        input_image_path_key = config.get('input_image_path_key', 'image_path')
        input_boxes_key = config.get('input_boxes_key', 'boxes')
        output_path = config.get('output_path', 'output_with_boxes.jpg')
        color = config.get('color', 'red')
        width = config.get('width', 3)
        # Load configuration values
        image_path = context.get(input_image_path_key)
        boxes = context.get(input_boxes_key)
        # Fetch input from context
        if image_path is None or boxes is None:
            # Missing required context entries; skip drawing
            return context
        image = Image.open(image_path).convert("RGB")
        draw = ImageDraw.Draw(image)
        w, h = image.size
        for obj in boxes:
            x0 = int(obj['x_min'] * w)
            y0 = int(obj['y_min'] * h)
            x1 = int(obj['x_max'] * w)
            y1 = int(obj['y_max'] * h)
            draw.rectangle([x0, y0, x1, y1], outline=color, width=width)
        # Save the boxed image
        image.save(output_path)
        if self.logger:
            self.logger.info(f"Saved image with bounding boxes to {output_path}")
        output_key = config.get('output_image_path_key', 'output_image_path')
        # Insert output path into context under output_key
        context[output_key] = output_path
        return context

# Aliases for PluginManager compatibility
Drawboundingboxes = DrawBoundingBoxes
DrawboundingboxesPlugin = DrawBoundingBoxes
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
        print(f"[DEBUG][DrawBoundingBoxes] config: {config}")
        image_path = context.get(input_image_path_key)
        boxes = context.get(input_boxes_key)
        print(f"[DEBUG][DrawBoundingBoxes] image_path: {image_path}, boxes: {boxes}, output_path: {output_path}")
        if image_path is None or boxes is None:
            print(f"[DEBUG][DrawBoundingBoxes] Missing image_path or boxes. Skipping bounding box drawing.")
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
        image.save(output_path)
        print(f"[DEBUG][DrawBoundingBoxes] Saved boxed image to: {output_path}")
        if self.logger:
            self.logger.info(f"Saved image with bounding boxes to {output_path}")
        output_key = config.get('output_image_path_key', 'output_image_path')
        context[output_key] = output_path
        print(f"[DEBUG][DrawBoundingBoxes] Setting {output_key} in context to: {output_path}")
        return context

# Aliases for PluginManager compatibility
Drawboundingboxes = DrawBoundingBoxes
DrawboundingboxesPlugin = DrawBoundingBoxes

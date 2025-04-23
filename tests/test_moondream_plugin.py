import os
import sys
import pytest
import json
from unittest.mock import Mock, patch

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Data_Processing.Moondream.MoondreamPlugin import MoondreamPlugin
from Plugins.Data_Processing.ImageToBase64.ImageToBase64 import ImageToBase64
from Plugins.Data_Processing.DrawBoundingBoxes.DrawBoundingBoxes import DrawBoundingBoxes

TEST_IMAGE_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "test_images")
TEST_IMAGE = os.path.join(TEST_IMAGE_DIR, "9_A12_Clifton_Street.jpg")  # Use the vehicle-rich image for detection tests

@pytest.fixture
def moondream_plugin():
    # Use mock plugin manager and logger for fast and isolated tests
    return MoondreamPlugin(plugin_manager=Mock(), logger=Mock())

@pytest.fixture
def image_to_base64_plugin():
    return ImageToBase64(plugin_manager=Mock(), logger=Mock())

def test_moondream_plugin_init_only():
    plugin = MoondreamPlugin(plugin_manager=Mock(), logger=Mock())
    assert plugin is not None

def test_image_to_base64_only(image_to_base64_plugin):
    context = {"image_path": TEST_IMAGE}
    step_config = {"input_image_path_key": "image_path", "output_key": "image_base64"}
    context = image_to_base64_plugin.execute_pipeline_step(step_config, context)
    assert "image_base64" in context
    assert context["image_base64"].startswith("data:image/")

def test_image_to_base64_and_query(moondream_plugin, image_to_base64_plugin):
    """
    Test converting an image to base64 and querying Moondream API.
    """
    context = {"image_path": TEST_IMAGE}
    step_config_base64 = {
        "input_image_path_key": "image_path",
        "output_key": "image_base64"
    }
    context = image_to_base64_plugin.execute_pipeline_step(step_config_base64, context)
    assert "image_base64" in context
    base64_str = context["image_base64"]
    assert base64_str.startswith("data:image/")
    step_config_query = {
        "action": "query",
        "input_image_key": "image_base64",
        "question": "What is in this image?",
        "output_key": "moondream_answer"
    }
    orig_query_image = moondream_plugin.query_image
    def wrapped_query_image(image_bytes, question, stream=False, timeout=20):
        answer, raw = orig_query_image(image_bytes, question, stream=stream, timeout=timeout, return_raw=True)
        print("Moondream QUERY processed result:", answer)
        print("Moondream QUERY raw API response:", json.dumps(raw, indent=2))
        return answer
    moondream_plugin.query_image = wrapped_query_image
    context = moondream_plugin.execute_pipeline_step(step_config_query, context)
    assert "moondream_answer" in context
    answer = context["moondream_answer"]
    assert isinstance(answer, str)
    assert len(answer) > 0
    print("Moondream answer:", answer)
    moondream_plugin.query_image = orig_query_image

def test_image_to_base64_and_caption(moondream_plugin, image_to_base64_plugin):
    """
    Test converting an image to base64 and generating a caption with Moondream API.
    """
    context = {"image_path": TEST_IMAGE}
    step_config_base64 = {
        "input_image_path_key": "image_path",
        "output_key": "image_base64"
    }
    context = image_to_base64_plugin.execute_pipeline_step(step_config_base64, context)
    assert "image_base64" in context
    base64_str = context["image_base64"]
    assert base64_str.startswith("data:image/")
    step_config_caption = {
        "action": "caption",
        "input_image_key": "image_base64",
        "length": "normal",
        "output_key": "moondream_caption"
    }
    orig_caption_image = moondream_plugin.caption_image
    def wrapped_caption_image(image_bytes, length="normal", timeout=20):
        caption, raw = orig_caption_image(image_bytes, length=length, timeout=timeout, return_raw=True)
        print("Moondream CAPTION processed result:", caption)
        print("Moondream CAPTION raw API response:", json.dumps(raw, indent=2))
        return caption
    moondream_plugin.caption_image = wrapped_caption_image
    context = moondream_plugin.execute_pipeline_step(step_config_caption, context)
    assert "moondream_caption" in context
    caption = context["moondream_caption"]
    assert isinstance(caption, str)
    assert len(caption) > 0
    print("Moondream caption:", caption)
    moondream_plugin.caption_image = orig_caption_image

def test_image_to_base64_and_detect(moondream_plugin, image_to_base64_plugin):
    """
    Test converting an image to base64 and detecting objects with Moondream API.
    """
    context = {"image_path": TEST_IMAGE}
    step_config_base64 = {
        "input_image_path_key": "image_path",
        "output_key": "image_base64"
    }
    context = image_to_base64_plugin.execute_pipeline_step(step_config_base64, context)
    assert "image_base64" in context
    base64_str = context["image_base64"]
    assert base64_str.startswith("data:image/")
    step_config_detect = {
        "action": "detect",
        "input_image_key": "image_base64",
        "object": "car",
        "output_key": "moondream_detect"
    }
    orig_detect_objects = moondream_plugin.detect_objects
    def wrapped_detect_objects(image_bytes, object_name=None, timeout=20):
        objects, raw = orig_detect_objects(image_bytes, object_name=object_name, timeout=timeout, return_raw=True)
        print("Moondream DETECT processed result:", objects)
        print("Moondream DETECT raw API response:", json.dumps(raw, indent=2))
        return objects
    moondream_plugin.detect_objects = wrapped_detect_objects
    context = moondream_plugin.execute_pipeline_step(step_config_detect, context)
    assert "moondream_detect" in context
    detected = context["moondream_detect"]
    assert isinstance(detected, list)
    
    # After detection, use DrawBoundingBoxes plugin to draw and save the image
    draw_plugin = DrawBoundingBoxes(plugin_manager=Mock(), logger=Mock())
    step_config_draw = {
        "input_image_path_key": "image_path",
        "input_boxes_key": "moondream_detect",
        "output_path": "output_with_boxes.jpg",
        "output_image_path_key": "output_image_path",
        "color": "red",
        "width": 3
    }
    context = draw_plugin.execute_pipeline_step(step_config_draw, context)
    assert "output_image_path" in context
    print(f"Image with bounding boxes saved to: {context['output_image_path']}")
    moondream_plugin.detect_objects = orig_detect_objects

def test_image_to_base64_and_point(moondream_plugin, image_to_base64_plugin):
    """
    Test converting an image to base64 and locating object coordinates with Moondream API.
    """
    context = {"image_path": TEST_IMAGE}
    step_config_base64 = {
        "input_image_path_key": "image_path",
        "output_key": "image_base64"
    }
    context = image_to_base64_plugin.execute_pipeline_step(step_config_base64, context)
    assert "image_base64" in context
    base64_str = context["image_base64"]
    assert base64_str.startswith("data:image/")
    step_config_point = {
        "action": "point",
        "input_image_key": "image_base64",
        "object": "car",
        "output_key": "moondream_points"
    }
    orig_locate_object = moondream_plugin.locate_object
    def wrapped_locate_object(image_bytes, object_name=None, timeout=20):
        points, raw = orig_locate_object(image_bytes, object_name=object_name, timeout=timeout, return_raw=True)
        print("Moondream POINT processed result:", points)
        print("Moondream POINT raw API response:", json.dumps(raw, indent=2))
        return points
    moondream_plugin.locate_object = wrapped_locate_object
    context = moondream_plugin.execute_pipeline_step(step_config_point, context)
    assert "moondream_points" in context
    points = context["moondream_points"]
    assert isinstance(points, list)
    moondream_plugin.locate_object = orig_locate_object

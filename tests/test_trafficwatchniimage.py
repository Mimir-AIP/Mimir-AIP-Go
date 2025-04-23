import os
import sys
import pytest
import base64
from unittest.mock import Mock

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Input.TrafficWatchNIImage.TrafficWatchNIImage import TrafficWatchNIImage

@pytest.fixture
def trafficwatchni_plugin():
    return TrafficWatchNIImage(plugin_manager=Mock(), logger=Mock())

def test_trafficwatchniimage_b64_only(trafficwatchni_plugin):
    step_config = {
        "camera_id": "1",  # Use a valid camera_id for your environment
    }
    result = trafficwatchni_plugin.execute_pipeline_step(step_config, {})
    assert "traffic_image_b64" in result
    b64 = result["traffic_image_b64"]
    assert b64.startswith("data:image/")
    # Optionally decode and check bytes
    header, encoded = b64.split(',', 1)
    img_bytes = base64.b64decode(encoded)
    assert len(img_bytes) > 1000  # Should be a JPEG of reasonable size
    print("TrafficWatchNIImage b64 length:", len(b64))

def test_trafficwatchniimage_save_to_disk(trafficwatchni_plugin):
    step_config = {
        "camera_id": "1",  # Use a valid camera_id for your environment
        "save_to_disk": True,
        "output_dir": "test_traffic_images",
        "output": "traffic_image_path"
    }
    result = trafficwatchni_plugin.execute_pipeline_step(step_config, {})
    assert "traffic_image_b64" in result
    assert "traffic_image_path" in result
    b64 = result["traffic_image_b64"]
    path = result["traffic_image_path"]
    assert os.path.exists(path)
    print("TrafficWatchNIImage saved to:", path)

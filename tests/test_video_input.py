"""
Tests for the VideoInput plugin using real video processing
"""

import os
import sys
import pytest
import cv2
import numpy as np
import tempfile
import shutil
from datetime import datetime

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Input.VideoInput.VideoInput import VideoInput

def create_test_frames(output_dir, duration=3, fps=30, width=640, height=480):
    """Create a sequence of test frames with a moving object"""
    total_frames = duration * fps
    frame_paths = []
    
    for i in range(total_frames):
        # Create frame with white background
        frame = np.full((height, width, 3), 255, dtype=np.uint8)
        
        # Draw moving red square (BGR order)
        x = int((i / total_frames) * (width - 50))
        cv2.rectangle(frame, (x, height//2-25), (x+50, height//2+25), (0,0,255), -1)
        
        # Save frame
        frame_path = os.path.join(output_dir, f"frame_{i:06d}.png")
        cv2.imwrite(frame_path, frame)
        
        # Verify frame was saved with red pixels
        saved_frame = cv2.imread(frame_path)
        mask = cv2.inRange(saved_frame, (0,0,255), (0,0,255))
        assert cv2.countNonZero(mask) > 0, f"Frame {i} has no red pixels after saving"
        
        frame_paths.append(frame_path)
    
    return frame_paths

@pytest.fixture(scope="module")
def test_sequence():
    """Create a temporary sequence of test frames"""
    temp_dir = tempfile.mkdtemp()
    frame_paths = create_test_frames(temp_dir)
    
    # Return the frame pattern for the sequence
    yield os.path.join(temp_dir, "frame_*.png")
    
    # Cleanup
    shutil.rmtree(temp_dir)

@pytest.fixture(scope="module")
def output_dir():
    """Create a temporary directory for frame output"""
    temp_dir = tempfile.mkdtemp()
    yield temp_dir
    shutil.rmtree(temp_dir)

@pytest.fixture
def video_input_plugin():
    """Create VideoInput plugin instance"""
    return VideoInput()

def test_video_input_initialization(video_input_plugin):
    """Test VideoInput plugin initialization"""
    assert isinstance(video_input_plugin, VideoInput)
    assert video_input_plugin.plugin_type == "Input"
    assert '.mp4' in video_input_plugin.supported_formats
    assert '.png' in video_input_plugin.supported_formats
    assert video_input_plugin.default_output_format == 'jpg'

def test_sequence_metadata(video_input_plugin, test_sequence):
    """Test extracting sequence metadata"""
    step_config = {
        "config": {
            "video_path": test_sequence,
            "metadata_only": True
        },
        "output": "video_data"
    }
    
    result = video_input_plugin.execute_pipeline_step(step_config, {})
    assert "video_data" in result
    metadata = result["video_data"]["metadata"]
    
    assert metadata["fps"] == 30
    assert metadata["width"] == 640
    assert metadata["height"] == 480
    assert metadata["duration"] == 3
    assert metadata["frame_count"] == 90  # 3 seconds * 30 fps

def test_frame_extraction(video_input_plugin, test_sequence, output_dir):
    """Test extracting frames from sequence"""
    step_config = {
        "config": {
            "video_path": test_sequence,
            "frame_interval": 15,  # Extract every 15th frame (2 fps)
            "output_dir": output_dir,
            "output_format": "jpg"
        },
        "output": "video_data"
    }
    
    result = video_input_plugin.execute_pipeline_step(step_config, {})
    assert "video_data" in result
    data = result["video_data"]
    
    # Should have 6 frames (3 seconds * 2 fps)
    assert len(data["frames"]) == 6
    assert data["total_processed"] == 6
    
    # Verify frame properties
    for idx, frame_data in enumerate(data["frames"]):
        assert os.path.exists(frame_data["path"])
        assert frame_data["timestamp"] >= 0
        assert frame_data["timestamp"] <= 3  # Within sequence duration
        
        # Load frame and verify it's a valid image
        frame = cv2.imread(frame_data["path"])
        assert frame is not None
        assert frame.shape == (480, 640, 3)
        
        # Verify there's a red rectangle in the frame
        mask = cv2.inRange(frame, (0,0,255), (0,0,255))
        assert cv2.countNonZero(mask) > 0, f"No red pixels found in frame {idx}"

def test_frame_resizing(video_input_plugin, test_sequence, output_dir):
    """Test frame extraction with resizing"""
    target_size = (320, 240)
    
    step_config = {
        "config": {
            "video_path": test_sequence,
            "frame_interval": 30,  # 1 fps
            "output_dir": output_dir,
            "frame_size": target_size
        },
        "output": "video_data"
    }
    
    result = video_input_plugin.execute_pipeline_step(step_config, {})
    assert "video_data" in result
    data = result["video_data"]
    
    # Verify resized frames
    for idx, frame_data in enumerate(data["frames"]):
        frame = cv2.imread(frame_data["path"])
        assert frame.shape == (target_size[1], target_size[0], 3)
        
        # Verify content is maintained after resize
        mask = cv2.inRange(frame, (0,0,255), (0,0,255))
        assert cv2.countNonZero(mask) > 0, f"No red pixels found in resized frame {idx}"

def test_format_validation(video_input_plugin):
    """Test format validation"""
    with pytest.raises(ValueError) as exc_info:
        step_config = {
            "config": {
                "video_path": "test.invalid"
            },
            "output": "video_data"
        }
        video_input_plugin.execute_pipeline_step(step_config, {})
    assert "Unsupported format" in str(exc_info.value)

def test_error_handling(video_input_plugin):
    """Test error handling for invalid inputs"""
    # Test missing video path
    with pytest.raises(ValueError) as exc_info:
        step_config = {
            "config": {},
            "output": "video_data"
        }
        video_input_plugin.execute_pipeline_step(step_config, {})
    assert "video_path is required" in str(exc_info.value)
    
    # Test invalid path
    with pytest.raises(ValueError) as exc_info:
        step_config = {
            "config": {
                "video_path": "nonexistent.mp4"
            },
            "output": "video_data"
        }
        video_input_plugin.execute_pipeline_step(step_config, {})
    assert "Invalid video path" in str(exc_info.value)

def test_max_frames_limit(video_input_plugin, test_sequence, output_dir):
    """Test max_frames limitation"""
    max_frames = 2
    step_config = {
        "config": {
            "video_path": test_sequence,
            "frame_interval": 15,
            "output_dir": output_dir,
            "max_frames": max_frames
        },
        "output": "video_data"
    }
    
    result = video_input_plugin.execute_pipeline_step(step_config, {})
    data = result["video_data"]
    
    assert len(data["frames"]) == max_frames
    assert data["total_processed"] == max_frames

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
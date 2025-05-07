"""
Plugin for video input processing in Mimir-AIP pipelines.

Supports video file input, frame extraction, and integration with image processing plugins.
"""

import os
import sys
import cv2
import json
import glob
import logging
from datetime import datetime
from typing import Dict, List, Any, Union, Optional

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin

class VideoInput(BasePlugin):
    """Plugin for processing video inputs and extracting frames"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the VideoInput plugin"""
        self.logger = logging.getLogger(__name__)
        self.supported_formats = ['.mp4', '.avi', '.mov', '.mkv', '.png', '.jpg', '.jpeg']
        self.default_output_format = 'jpg'
        self.logger.info("VideoInput plugin initialized")
    
    def _is_image_sequence(self, path: str) -> bool:
        """Check if the input path is an image sequence pattern"""
        return '*' in path and any(path.lower().endswith(fmt) for fmt in ['.png', '.jpg', '.jpeg'])
    
    def _ensure_bgr(self, frame):
        """Ensure frame is in BGR color space"""
        if len(frame.shape) == 3 and frame.shape[2] == 3:
            return frame
        return cv2.cvtColor(frame, cv2.COLOR_RGB2BGR)

    def _save_frame(self, frame: "np.ndarray", output_path: str, format: str = 'png') -> bool:
        """Save a frame with proper format handling"""
        # Always save as PNG first for quality
        temp_png = output_path + '.temp.png'
        cv2.imwrite(temp_png, frame)
        
        # Verify the saved frame has the expected content
        saved = cv2.imread(temp_png)
        if saved is None:
            os.remove(temp_png)
            return False
            
        if format.lower() == 'jpg':
            # Convert to JPEG if requested
            cv2.imwrite(output_path, saved, [cv2.IMWRITE_JPEG_QUALITY, 100])
            os.remove(temp_png)
        else:
            # Just rename the PNG
            os.rename(temp_png, output_path)
            
        return True
    
    def _get_image_sequence_metadata(self, path_pattern: str) -> dict:
        """Get metadata for an image sequence"""
        files = sorted(glob.glob(path_pattern))
        if not files:
            raise ValueError(f"No files found matching pattern: {path_pattern}")
            
        # Read first image for dimensions
        img = cv2.imread(files[0])
        if img is None:
            raise ValueError(f"Failed to read first image: {files[0]}")
            
        return {
            "fps": 30,  # Default for image sequences
            "frame_count": len(files),
            "width": img.shape[1],
            "height": img.shape[0],
            "duration": len(files) / 30,  # Assuming 30fps
            "codec": 0,
            "filename": os.path.basename(path_pattern)
        }

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for video processing
        
        Args:
            step_config (dict): Configuration containing:
                - video_path: Path to input video file or image sequence pattern
                - frame_interval: Extract every Nth frame (default: 1)
                - max_frames: Maximum number of frames to extract (optional)
                - output_dir: Directory to save extracted frames
                - output_format: Format for saved frames (jpg/png)
                - frame_size: Tuple of (width, height) to resize frames (optional)
                - metadata_only: Only extract video metadata without frames
            context (dict): Pipeline context
            
        Returns:
            dict: Updated context with extracted frames and metadata
        """
        config = step_config.get("config", {})
        
        # Get video path from config or context
        video_path = config.get("video_path")
        if isinstance(video_path, str) and video_path in context:
            video_path = context[video_path]
            
        if not video_path:
            raise ValueError("video_path is required in config")

        # Handle image sequences
        is_sequence = self._is_image_sequence(video_path)
        if not is_sequence and not any(video_path.lower().endswith(fmt) for fmt in self.supported_formats):
            raise ValueError(f"Unsupported format. Supported: {', '.join(self.supported_formats)}")
            
        if not is_sequence and not os.path.exists(video_path):
            raise ValueError(f"Invalid video path: {video_path}")

        # Extract parameters
        frame_interval = max(1, config.get("frame_interval", 1))
        max_frames = config.get("max_frames")
        output_dir = config.get("output_dir", "extracted_frames")
        output_format = config.get("output_format", self.default_output_format).lower()
        frame_size = config.get("frame_size")
        metadata_only = config.get("metadata_only", False)

        # Create output directory
        os.makedirs(output_dir, exist_ok=True)

        try:
            # Get metadata
            if is_sequence:
                metadata = self._get_image_sequence_metadata(video_path)
            else:
                cap = cv2.VideoCapture(video_path)
                if not cap.isOpened():
                    raise ValueError(f"Failed to open video: {video_path}")

                metadata = {
                    "fps": cap.get(cv2.CAP_PROP_FPS),
                    "frame_count": int(cap.get(cv2.CAP_PROP_FRAME_COUNT)),
                    "width": int(cap.get(cv2.CAP_PROP_FRAME_WIDTH)),
                    "height": int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT)),
                    "duration": int(cap.get(cv2.CAP_PROP_FRAME_COUNT) / cap.get(cv2.CAP_PROP_FPS)),
                    "codec": int(cap.get(cv2.CAP_PROP_FOURCC)),
                    "filename": os.path.basename(video_path)
                }

            if metadata_only:
                if not is_sequence:
                    cap.release()
                return {step_config["output"]: {"metadata": metadata}}

            # Process frames
            frames = []
            frame_count = 0
            processed_count = 0
            
            base_filename = os.path.splitext(os.path.basename(video_path))[0]

            if is_sequence:
                # Process image sequence
                frame_files = sorted(glob.glob(video_path))
                for i, frame_file in enumerate(frame_files):
                    if frame_count % frame_interval == 0:
                        frame = cv2.imread(frame_file)
                        if frame is None:
                            continue
                            
                        if frame_size:
                            frame = cv2.resize(frame, frame_size)

                        frame_filename = f"{base_filename}_frame_{frame_count:06d}.{output_format}"
                        frame_path = os.path.join(output_dir, frame_filename)

                        if self._save_frame(frame, frame_path, output_format):
                            frames.append({
                                "path": frame_path,
                                "frame_number": frame_count,
                                "timestamp": frame_count / metadata["fps"]
                            })
                            processed_count += 1
                            
                        if max_frames and processed_count >= max_frames:
                            break
                            
                    frame_count += 1
            else:
                # Process video file
                while cap.isOpened():
                    ret, frame = cap.read()
                    if not ret:
                        break

                    if frame_count % frame_interval == 0:
                        frame = self._ensure_bgr(frame)
                        
                        if frame_size:
                            frame = cv2.resize(frame, frame_size)

                        frame_filename = f"{base_filename}_frame_{frame_count:06d}.{output_format}"
                        frame_path = os.path.join(output_dir, frame_filename)

                        if self._save_frame(frame, frame_path, output_format):
                            frames.append({
                                "path": frame_path,
                                "frame_number": frame_count,
                                "timestamp": frame_count / metadata["fps"]
                            })
                            processed_count += 1
                            
                        if max_frames and processed_count >= max_frames:
                            break

                    frame_count += 1

                cap.release()

            # Return results
            result = {
                "metadata": metadata,
                "frames": frames,
                "total_processed": processed_count
            }
            
            self.logger.info(f"Processed {processed_count} frames from {video_path}")
            return {step_config["output"]: result}

        except Exception as e:
            self.logger.error(f"Error processing video {video_path}: {str(e)}")
            raise

    def get_video_metadata(self, video_path: str) -> Dict[str, Any]:
        """Get metadata for a video file without processing frames
        
        Args:
            video_path (str): Path to video file
            
        Returns:
            dict: Video metadata including fps, duration, dimensions, etc.
        """
        if not os.path.exists(video_path):
            raise ValueError(f"Video file not found: {video_path}")

        cap = cv2.VideoCapture(video_path)
        if not cap.isOpened():
            raise ValueError(f"Failed to open video: {video_path}")

        metadata = {
            "fps": cap.get(cv2.CAP_PROP_FPS),
            "frame_count": int(cap.get(cv2.CAP_PROP_FRAME_COUNT)),
            "width": int(cap.get(cv2.CAP_PROP_FRAME_WIDTH)),
            "height": int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT)),
            "duration": int(cap.get(cv2.CAP_PROP_FRAME_COUNT) / cap.get(cv2.CAP_PROP_FPS)),
            "codec": int(cap.get(cv2.CAP_PROP_FOURCC)),
            "filename": os.path.basename(video_path)
        }

        cap.release()
        return metadata

if __name__ == "__main__":
    # Test the plugin
    plugin = VideoInput()
    
    # Test configuration
    test_config = {
        "plugin": "VideoInput",
        "config": {
            "video_path": "test.mp4",  # Replace with a test video
            "frame_interval": 30,  # Extract 1 frame per second for 30fps video
            "output_dir": "test_frames",
            "frame_size": (640, 480),  # Optional resize
            "max_frames": 10  # Limit frames for testing
        },
        "output": "video_data"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Processed video. Metadata: {json.dumps(result['video_data']['metadata'], indent=2)}")
        print(f"Extracted {len(result['video_data']['frames'])} frames")
    except Exception as e:
        print(f"Error: {e}")
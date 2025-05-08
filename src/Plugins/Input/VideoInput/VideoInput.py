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
import numpy as np
import psutil
from datetime import datetime
from typing import Dict, List, Any, Union, Optional, Tuple

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
        # Common video codecs and their FourCC codes
        self.supported_codecs = {
            'h264': cv2.VideoWriter_fourcc(*'H264'),
            'mp4v': cv2.VideoWriter_fourcc(*'MP4V'),
            'mjpg': cv2.VideoWriter_fourcc(*'MJPG'),
            'xvid': cv2.VideoWriter_fourcc(*'XVID')
        }
        self.logger.info("VideoInput plugin initialized")
    
    def _is_codec_supported(self, codec_fourcc: int) -> bool:
        """Check if the video codec is supported
        
        Args:
            codec_fourcc: FourCC code of the video codec
            
        Returns:
            bool: True if codec is supported, False otherwise
        """
        return any(codec == codec_fourcc for codec in self.supported_codecs.values())

    def _get_codec_name(self, codec_fourcc: int) -> str:
        """Get human-readable name for codec FourCC"""
        for name, code in self.supported_codecs.items():
            if code == codec_fourcc:
                return name
        return 'unknown'
    
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
        # Debug: Log a sample of pixel values from center of frame
        h, w = frame.shape[:2]
        center_y, center_x = h // 2, w // 2
        self.logger.debug(f"Frame shape before save: {frame.shape}")
        self.logger.debug(f"Sample pixel values: {frame[center_y, center_x]}")
        
        # Ensure frame is in BGR format for OpenCV
        frame = self._ensure_bgr(frame)
        self.logger.debug(f"Frame shape after BGR conversion: {frame.shape}")
        self.logger.debug(f"Sample pixel values after conversion: {frame[center_y, center_x]}")
        
        # Save directly to the target format
        success = False
        if format.lower() == 'jpg':
            success = cv2.imwrite(output_path, frame, [cv2.IMWRITE_JPEG_QUALITY, 100])
        else:
            success = cv2.imwrite(output_path, frame)
            
        if success:
            # Verify saved frame
            saved = cv2.imread(output_path)
            if saved is not None:
                self.logger.debug(f"Saved frame shape: {saved.shape}")
                self.logger.debug(f"Saved pixel values: {saved[center_y, center_x]}")
            
        return success

    def _get_image_sequence_metadata(self, path_pattern: str) -> dict:
        """Get metadata for an image sequence"""
        files = sorted(glob.glob(path_pattern))
        if not files:
            raise ValueError(f"No files found matching pattern: {path_pattern}")
            
        # Read first image for dimensions
        img = cv2.imread(files[0])
        if img is None:
            raise ValueError(f"Failed to read first image: {files[0]}")
            
        metadata = {
            "fps": 30,  # Default for image sequences
            "frame_count": len(files),
            "width": img.shape[1],
            "height": img.shape[0],
            "duration": len(files) / 30,  # Assuming 30fps
            "codec": 0,  # No codec for image sequences
            "codec_name": "image_sequence",
            "filename": os.path.basename(path_pattern)
        }
        return metadata

    def _verify_video_integrity(self, video_path: str) -> bool:
        """Verify video file can be read correctly"""
        try:
            cap = cv2.VideoCapture(video_path)
            if not cap.isOpened():
                return False
                
            # Try reading a few frames to verify
            for _ in range(5):
                ret, frame = cap.read()
                if not ret or frame is None:
                    cap.release()
                    return False
                    
            cap.release()
            return True
        except Exception:
            return False

    def _get_optimal_buffer_size(self, frame_size: tuple, fps: float) -> int:
        """Calculate optimal buffer size based on video properties
        
        Args:
            frame_size: (width, height) of frames
            fps: Frames per second
            
        Returns:
            int: Recommended buffer size in frames
        """
        # Calculate frame size in MB
        frame_mb = (frame_size[0] * frame_size[1] * 3) / (1024 * 1024)
        
        # Target ~1GB memory usage max
        max_buffer_mb = 1024
        
        # Calculate frames that fit in target memory
        frames_in_memory = int(max_buffer_mb / frame_mb)
        
        # Use smaller of calculated size or 1 second of video
        return min(frames_in_memory, int(fps))

    def _preprocess_frame(self, frame: "np.ndarray", frame_size: tuple = None) -> "np.ndarray":
        """Apply common preprocessing to a frame"""
        if frame is None:
            return None
            
        # Ensure BGR color space
        frame = self._ensure_bgr(frame)
        
        # Resize if needed
        if frame_size:
            frame = cv2.resize(frame, frame_size, interpolation=cv2.INTER_AREA)
            
        return frame

    def _validate_config(self, config: dict) -> None:
        """Validate pipeline configuration"""
        # Required parameters
        if not config.get("video_path"):
            raise ValueError("video_path is required in config")
            
        # Optional parameters validation
        frame_interval = config.get("frame_interval", 1)
        if frame_interval < 1:
            raise ValueError("frame_interval must be >= 1")
            
        if "frame_size" in config:
            size = config["frame_size"]
            if not isinstance(size, (tuple, list)) or len(size) != 2:
                raise ValueError("frame_size must be a tuple/list of (width, height)")
            if not all(isinstance(x, int) and x > 0 for x in size):
                raise ValueError("frame_size dimensions must be positive integers")
                
        if "max_frames" in config:
            max_frames = config["max_frames"]
            if not isinstance(max_frames, int) or max_frames < 1:
                raise ValueError("max_frames must be a positive integer")
                
        output_format = config.get("output_format", self.default_output_format)
        if output_format.lower() not in ["jpg", "png"]:
            raise ValueError("output_format must be 'jpg' or 'png'")

    def _check_memory_usage(self, frame_size: tuple, frame_count: int) -> None:
        """Check if processing will exceed memory limits
        
        Args:
            frame_size: (width, height) of frames
            frame_count: Number of frames to process
            
        Raises:
            ValueError: If estimated memory usage is too high
        """
        frame_mb = (frame_size[0] * frame_size[1] * 3) / (1024 * 1024)
        total_mb = frame_mb * frame_count
        
        # Warning if total memory usage might exceed 75% of system memory
        system_mb = psutil.virtual_memory().total / (1024 * 1024)
        if total_mb > system_mb * 0.75:
            self.logger.warning(
                f"Processing may require {total_mb:.1f}MB memory "
                f"({(total_mb/system_mb)*100:.1f}% of system memory)"
            )

    def _process_frame_batch(self, frames: List["np.ndarray"], base_filename: str, 
                            frame_numbers: List[int], metadata: dict,
                            output_dir: str, output_format: str,
                            frame_size: tuple = None) -> List[dict]:
        """Process a batch of frames in parallel
        
        Args:
            frames: List of frames to process
            base_filename: Base filename for output
            frame_numbers: List of frame numbers corresponding to frames
            metadata: Video metadata
            output_dir: Output directory
            output_format: Output format (jpg/png)
            frame_size: Optional frame resize dimensions
            
        Returns:
            List of frame metadata dictionaries
        """
        results = []
        for frame, frame_num in zip(frames, frame_numbers):
            if frame is None:
                continue
                
            frame = self._preprocess_frame(frame, frame_size)
            frame_filename = f"{base_filename}_frame_{frame_num:06d}.{output_format}"
            frame_path = os.path.join(output_dir, frame_filename)
            
            if self._save_frame(frame, frame_path, output_format):
                results.append({
                    "path": frame_path,
                    "frame_number": frame_num,
                    "timestamp": frame_num / metadata["fps"]
                })
                
        return results

    def _buffer_frames(self, cap: cv2.VideoCapture, buffer_size: int) -> List["np.ndarray"]:
        """Read a batch of frames into memory
        
        Args:
            cap: OpenCV VideoCapture object
            buffer_size: Number of frames to read
            
        Returns:
            List of frames read from video
        """
        frames = []
        for _ in range(buffer_size):
            ret, frame = cap.read()
            if not ret:
                break
            frames.append(frame)
        return frames

    def _track_progress(self, current: int, total: int, update_interval: int = 100) -> None:
        """Track processing progress
        
        Args:
            current: Current frame number
            total: Total frames to process
            update_interval: How often to log progress
        """
        if current % update_interval == 0:
            progress = (current / total) * 100
            self.logger.info(f"Processing progress: {progress:.1f}% ({current}/{total} frames)")

    def _recover_from_error(self, error: Exception, video_path: str, frame_count: int) -> Optional[cv2.VideoCapture]:
        """Attempt to recover from processing errors
        
        Args:
            error: The exception that occurred
            video_path: Path to the video being processed
            frame_count: Number of frames processed before error
            
        Returns:
            Optional[cv2.VideoCapture]: Recovered video capture object or None if recovery failed
            
        Raises:
            Exception: Re-raises the original error if recovery fails
        """
        self.logger.error(f"Error at frame {frame_count}: {str(error)}")
        
        if isinstance(error, cv2.error):
            self.logger.info("OpenCV error occurred, attempting to recover...")
            try:
                # Try reopening the video and skipping to last good frame
                cap = cv2.VideoCapture(video_path)
                cap.set(cv2.CAP_PROP_POS_FRAMES, frame_count)
                ret, _ = cap.read()
                if ret:
                    self.logger.info(f"Successfully recovered from frame {frame_count}")
                    return cap
                cap.release()
            except Exception as e:
                self.logger.error(f"Recovery failed: {str(e)}")
                
        raise error  # Re-raise if recovery failed

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for video processing"""
        config = step_config.get("config", {})
        
        # Validate configuration
        self._validate_config(config)
        
        # Get video path from config or context
        video_path = config.get("video_path")
        if isinstance(video_path, str) and video_path in context:
            video_path = context[video_path]

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

                # Verify codec support
                if not self._is_codec_supported(metadata["codec"]):
                    self.logger.warning(
                        f"Video codec {self._get_codec_name(metadata['codec'])} "
                        "may not be fully supported"
                    )

                # Check memory requirements
                self._check_memory_usage(
                    (metadata["width"], metadata["height"]),
                    metadata["frame_count"] // frame_interval
                )

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
                total_frames = len(frame_files)
                
                for i, frame_file in enumerate(frame_files):
                    if frame_count % frame_interval == 0:
                        try:
                            frame = cv2.imread(frame_file)
                            if frame is None:
                                continue

                            # Process frame
                            frame = self._preprocess_frame(frame, frame_size)
                            frame_filename = f"{base_filename}_frame_{frame_count:06d}.{output_format}"
                            frame_path = os.path.join(output_dir, frame_filename)

                            if self._save_frame(frame, frame_path, output_format):
                                frames.append({
                                    "path": frame_path,
                                    "frame_number": frame_count,
                                    "timestamp": frame_count / metadata["fps"]
                                })
                                processed_count += 1

                            self._track_progress(i + 1, total_frames)

                        except Exception as e:
                            self.logger.error(f"Error processing frame {frame_file}: {str(e)}")
                            continue

                    frame_count += 1
                    if max_frames and processed_count >= max_frames:
                        break

            else:
                # Process video file with buffering
                total_frames = metadata["frame_count"]
                buffer_size = self._get_optimal_buffer_size(
                    (metadata["width"], metadata["height"]),
                    metadata["fps"]
                )

                while True:
                    try:
                        # Read batch of frames
                        batch_frames = self._buffer_frames(cap, buffer_size)
                        if not batch_frames:
                            break

                        # Process frames in batch
                        batch_numbers = list(range(
                            frame_count,
                            frame_count + len(batch_frames)
                        ))
                        
                        frame_results = self._process_frame_batch(
                            batch_frames,
                            base_filename,
                            batch_numbers,
                            metadata,
                            output_dir,
                            output_format,
                            frame_size
                        )
                        
                        frames.extend(frame_results)
                        processed_count += len(frame_results)
                        frame_count += len(batch_frames)

                        self._track_progress(frame_count, total_frames)

                        if max_frames and processed_count >= max_frames:
                            break

                    except Exception as e:
                        # Try to recover from errors
                        cap = self._recover_from_error(e, video_path, frame_count)
                        if cap is None:
                            break

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
        """Get metadata for a video file or image sequence without processing frames
        
        Args:
            video_path (str): Path to video file or image sequence pattern
            
        Returns:
            dict: Video metadata including fps, duration, dimensions, etc.
        """
        if self._is_image_sequence(video_path):
            return self._get_image_sequence_metadata(video_path)

        if not os.path.exists(video_path):
            raise ValueError(f"Video file not found: {video_path}")
            
        if not any(video_path.lower().endswith(fmt) for fmt in self.supported_formats):
            raise ValueError(f"Unsupported format. Supported: {', '.join(self.supported_formats)}")

        cap = cv2.VideoCapture(video_path)
        if not cap.isOpened():
            raise ValueError(f"Failed to open video: {video_path}")

        codec_fourcc = int(cap.get(cv2.CAP_PROP_FOURCC))
        metadata = {
            "fps": cap.get(cv2.CAP_PROP_FPS),
            "frame_count": int(cap.get(cv2.CAP_PROP_FRAME_COUNT)),
            "width": int(cap.get(cv2.CAP_PROP_FRAME_WIDTH)),
            "height": int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT)),
            "duration": int(cap.get(cv2.CAP_PROP_FRAME_COUNT) / cap.get(cv2.CAP_PROP_FPS)),
            "codec": codec_fourcc,
            "codec_name": self._get_codec_name(codec_fourcc),
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
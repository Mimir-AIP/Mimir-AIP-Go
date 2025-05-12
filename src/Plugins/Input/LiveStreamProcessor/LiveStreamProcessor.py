"""
LiveStreamProcessor plugin for Mimir-AIP.

Handles live stream input processing including HLS (.m3u8) streams.
"""

import os
import logging
import requests
from typing import Dict, Any
from urllib.parse import urlparse

from Plugins.BasePlugin import BasePlugin

class LiveStreamProcessor(BasePlugin):
    """Plugin for processing live video streams in pipelines"""
    
    plugin_type = "Input"
    
    def __init__(self):
        super().__init__()
        self.supported_formats = ['.m3u8', '.mpd']
        self.logger = logging.getLogger(__name__)
        
    def _validate_stream(self, stream_url: str) -> bool:
        """Validate the stream URL and accessibility"""
        try:
            parsed = urlparse(stream_url)
            if not all([parsed.scheme, parsed.netloc]):
                return False
                
            if not any(stream_url.endswith(fmt) for fmt in self.supported_formats):
                return False
                
            # Check if stream is reachable
            resp = requests.head(stream_url, timeout=5)
            return resp.status_code == 200
            
        except Exception as e:
            self.logger.error(f"Stream validation failed: {e}")
            return False
            
    def _get_stream_metadata(self, stream_url: str) -> Dict[str, Any]:
        """Extract basic stream metadata"""
        return {
            'url': stream_url,
            'type': 'HLS' if stream_url.endswith('.m3u8') else 'DASH',
            'validated_at': self._current_timestamp()
        }
        
    def execute_pipeline_step(self, step_config: Dict, context: Dict) -> Dict:
        """
        Process live stream input for pipeline.
        
        Args:
            step_config: Dictionary containing 'config' and 'output' keys
            context: Pipeline context dictionary
            
        Returns:
            Dictionary with stream data to add to context
        """
        config = step_config.get('config', {})
        stream_url = config.get('stream_url')
        
        if not stream_url or not self._validate_stream(stream_url):
            raise ValueError(f"Invalid stream URL: {stream_url}")
            
        result = {
            'stream_url': stream_url,
            'metadata': self._get_stream_metadata(stream_url),
            'is_alive': True
        }
        
        # Optional frame capture
        if config.get('capture_frame', False):
            try:
                result['frame'] = self._capture_frame(stream_url)
            except Exception as e:
                self.logger.warning(f"Frame capture failed: {e}")
                
        return {step_config['output']: result}
        
    def _capture_frame(self, stream_url: str) -> str:
        """Capture single frame from stream (placeholder)"""
        # TODO Implementation will use OpenCV or similar
        return "base64_encoded_frame_data"
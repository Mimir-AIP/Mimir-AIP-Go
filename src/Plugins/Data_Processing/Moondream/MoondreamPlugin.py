"""
MoondreamPlugin: Data processing plugin for the Moondream.ai API.

Features:
- Visual Question Answering (/query)
- Image Captioning (/caption)
- Object Detection (/detect)
- Object Localization (/point)

API key must be provided in a `.env` file in the same directory with the variable `MOONDREAM_API_KEY`.
"""

import os
import base64
import requests
import re
from typing import List, Dict, Any, Tuple, Union
from dotenv import load_dotenv
from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager
import logging

class MoondreamPlugin(BasePlugin):
    """
    Data processing plugin for Moondream.ai API.
    Provides methods for visual Q&A, captioning, detection, and localization.
    """
    plugin_type = "Data_Processing"

    def __init__(self, plugin_manager=None, logger=None):
        """
        Initialize the MoondreamPlugin.
        Args:
            plugin_manager (PluginManager, optional): The plugin manager instance.
            logger (logging.Logger, optional): Logger instance.
        """
        self.plugin_manager = plugin_manager if plugin_manager is not None else PluginManager()
        self.logger = logger
        env_path = os.path.join(os.path.dirname(__file__), '.env')
        load_dotenv(env_path)
        self.api_key = os.getenv('MOONDREAM_API_KEY')
        if not self.api_key:
            raise ValueError("MOONDREAM_API_KEY not found in .env file.")

    def _headers(self) -> Dict[str, str]:
        return {
            "X-Moondream-Auth": self.api_key,
            "Content-Type": "application/json"
        }

    def _image_to_base64(self, image_bytes: bytes) -> str:
        base64_str = base64.b64encode(image_bytes).decode('utf-8')
        return f"data:image/jpeg;base64,{base64_str}"

    def query_image(self, image: bytes, question: str, stream: bool = False, timeout: int = 20, return_raw: bool = False) -> Union[str, Tuple[str, Dict]]:
        """
        Ask a natural language question about an image.
        Args:
            image (bytes): Image data in bytes.
            question (str): The question to ask about the image.
            stream (bool): Whether to stream the response.
            timeout (int): Timeout for the request in seconds.
            return_raw (bool): If True, return (result, raw_response_dict)
        Returns:
            str or (str, dict): The answer from Moondream, or (answer, raw response) if return_raw
        Raises:
            Exception: On network or API error.
        """
        url = f"https://api.moondream.ai/v1/query"
        payload = {
            "image_url": self._image_to_base64(image),
            "question": question,
            "stream": stream
        }
        if self.logger:
            self.logger.info(f"Sending /query request to Moondream API with question: {question}")
        try:
            resp = requests.post(url, headers=self._headers(), json=payload, timeout=timeout)
        except requests.Timeout:
            if self.logger:
                self.logger.error("Timeout occurred for /query request")
            raise Exception("Timeout occurred for /query request")
        if resp.status_code != 200:
            if self.logger:
                self.logger.error(f"Moondream /query error: {resp.status_code} {resp.text}")
            raise Exception(f"Moondream /query error: {resp.status_code} {resp.text}")
        if self.logger:
            self.logger.info(f"/query response: {resp.json()}")
        result = resp.json().get("answer", "")
        if return_raw:
            return result, resp.json()
        return result

    def caption_image(self, image: bytes, length: str = "normal", stream: bool = False, timeout: int = 20, return_raw: bool = False) -> Union[str, Tuple[str, Dict]]:
        """
        Generate a caption for an image.
        Args:
            image (bytes): Image data in bytes.
            length (str): Caption length ('normal' or 'long').
            stream (bool): Whether to stream the response.
            timeout (int): Timeout for the request in seconds.
            return_raw (bool): If True, return (caption, raw_response_dict)
        Returns:
            str or (str, dict): The caption from Moondream, or (caption, raw response) if return_raw
        Raises:
            Exception: On network or API error.
        """
        url = f"https://api.moondream.ai/v1/caption"
        payload = {
            "image_url": self._image_to_base64(image),
            "length": length,
            "stream": stream
        }
        if self.logger:
            self.logger.info(f"Sending /caption request to Moondream API with length: {length}")
        try:
            resp = requests.post(url, headers=self._headers(), json=payload, timeout=timeout)
        except requests.Timeout:
            if self.logger:
                self.logger.error("Timeout occurred for /caption request")
            raise Exception("Timeout occurred for /caption request")
        if resp.status_code != 200:
            if self.logger:
                self.logger.error(f"Moondream /caption error: {resp.status_code} {resp.text}")
            raise Exception(f"Moondream /caption error: {resp.status_code} {resp.text}")
        if self.logger:
            self.logger.info(f"/caption response: {resp.json()}")
        result = resp.json().get("caption", "")
        if return_raw:
            return result, resp.json()
        return result

    def detect_objects(self, image: bytes, object_name: str = None, stream: bool = False, timeout: int = 20, return_raw: bool = False) -> Union[List[Dict], Tuple[List[Dict], Dict]]:
        """
        Detect objects in an image.
        Args:
            image (bytes): Image data in bytes.
            object_name (str): Name of the object to detect.
            stream (bool): Whether to stream the response.
            timeout (int): Timeout for the request in seconds.
            return_raw (bool): If True, return (objects_list, raw_response_dict)
        Returns:
            list or (list, dict): List of detected objects, or (objects, raw response) if return_raw
        Raises:
            Exception: On network or API error.
        """
        url = f"https://api.moondream.ai/v1/detect"
        payload = {
            "image_url": self._image_to_base64(image),
            "object": object_name,
            "stream": stream
        }
        if self.logger:
            self.logger.info(f"Sending /detect request to Moondream API for object: {object_name}")
        try:
            resp = requests.post(url, headers=self._headers(), json=payload, timeout=timeout)
        except requests.Timeout:
            if self.logger:
                self.logger.error("Timeout occurred for /detect request")
            raise Exception("Timeout occurred for /detect request")
        if resp.status_code != 200:
            if self.logger:
                self.logger.error(f"Moondream /detect error: {resp.status_code} {resp.text}")
            raise Exception(f"Moondream /detect error: {resp.status_code} {resp.text}")
        if self.logger:
            self.logger.info(f"/detect response: {resp.json()}")
        result = resp.json().get("objects", [])
        if return_raw:
            return result, resp.json()
        return result

    def locate_object(self, image: bytes, object_name: str = None, timeout: int = 20, return_raw: bool = False) -> Union[list, Tuple[list, Dict]]:
        """
        Locate the coordinates of the specified object in the image using the Moondream API.
        Args:
            image (bytes): The image data
            object_name (str): The object to locate
            timeout (int): Timeout for the request
            return_raw (bool): If True, return (points_list, raw_response_dict)
        Returns:
            list or (list, dict): List of coordinates for the object, or (points, raw response) if return_raw
        """
        url = f"https://api.moondream.ai/v1/point"
        payload = {"image_url": self._image_to_base64(image)}
        if object_name:
            payload["object"] = object_name
        if self.logger:
            self.logger.info(f"Locating object '{object_name}' in image via Moondream API at {url}")
        try:
            resp = requests.post(url, headers=self._headers(), json=payload, timeout=timeout)
            resp.raise_for_status()
            data = resp.json()
            if self.logger:
                self.logger.info(f"Received object points: {data}")
            result = data.get("points", [])
            if return_raw:
                return result, data
            return result
        except requests.Timeout:
            if self.logger:
                self.logger.error("Moondream API request timed out.")
            raise
        except Exception as e:
            if self.logger:
                self.logger.error(f"Error locating object: {e}")
            raise

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """
        Execute a pipeline step for this plugin.
        Args:
            step_config (dict): Configuration for this step from the pipeline YAML
            context (dict): Current pipeline context with variables
        Returns:
            dict: Updated context with any new variables
        """
        action = step_config.get('action')
        image = context.get(step_config.get('input_image_key', 'image'))
        if image is None:
            raise ValueError("No image data found in context.")
        # Handle image input as bytes, base64 string, or file path
        if isinstance(image, bytes):
            image_bytes = image
        elif isinstance(image, str) and image.startswith('data:image/'):
            match = re.match(r'data:image/[^;]+;base64,(.*)', image)
            if not match:
                raise ValueError("Invalid base64 image data URL format.")
            image_bytes = base64.b64decode(match.group(1))
        elif isinstance(image, str):
            with open(image, 'rb') as f:
                image_bytes = f.read()
        else:
            raise ValueError("Unsupported image input type for MoondreamPlugin.")
        result = None
        if action == 'query':
            question = step_config.get('question')
            result = self.query_image(image_bytes, question)
        elif action == 'caption':
            length = step_config.get('length', 'normal')
            result = self.caption_image(image_bytes, length)
        elif action == 'detect':
            object_name = step_config.get('object')
            result = self.detect_objects(image_bytes, object_name)
        elif action == 'point':
            object_name = step_config.get('object')
            result = self.locate_object(image_bytes, object_name)
        else:
            raise ValueError(f"Unknown action: {action}")
        output_key = step_config.get('output_key', 'result')
        context[output_key] = result
        return context

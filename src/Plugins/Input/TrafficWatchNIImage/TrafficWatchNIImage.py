"""
Input plugin for fetching static images from TrafficWatchNI cameras by camera ID.
"""
import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../../..")))
import requests
import csv
import re
from datetime import datetime, timedelta
from Plugins.BasePlugin import BasePlugin

class TrafficWatchNIImage(BasePlugin):
    """
    Fetches a static image from TrafficWatchNI using the camera ID.
    Also provides a method to get the camera name, using a CSV cache.
    """
    plugin_type = "Input"
    CACHE_FILE = os.path.join(os.path.dirname(__file__), "camera_names_cache.csv")
    CACHE_EXPIRY_DAYS = 31

    def __init__(self, *args, logger=None, **kwargs):
        super().__init__(*args, **kwargs)
        import logging
        self.logger = logger or logging.getLogger(__name__)

    def execute_pipeline_step(self, step_config, context):
        """
        Downloads the image for the given camera_id from TrafficWatchNI and saves it locally.
        Args:
            step_config (dict): Should contain 'camera_id' and optionally 'output_dir' and 'output'.
            context (dict): Pipeline context (unused).
        Returns:
            dict: {output_key: image_path} if successful, else {output_key: None}
        """
        camera_id = step_config.get("camera_id")
        output_key = step_config.get("output", "traffic_image_path")
        output_dir = step_config.get("output_dir", "traffic_images")
        if camera_id is None:
            self.logger.error("No camera_id provided to TrafficWatchNIImage plugin.")
            return {output_key: None}
        url = f"https://cctv.trafficwatchni.com/{camera_id}.jpg"
        os.makedirs(output_dir, exist_ok=True)
        image_path = os.path.join(output_dir, f"{camera_id}.jpg")
        try:
            resp = requests.get(url, timeout=10)
            if resp.status_code == 200:
                with open(image_path, 'wb') as f:
                    f.write(resp.content)
                self.logger.info(f"Downloaded image for camera {camera_id} to {image_path}")
                return {output_key: image_path}
            else:
                self.logger.error(f"Failed to fetch image for camera {camera_id}: HTTP {resp.status_code}")
                return {output_key: None}
        except Exception as e:
            self.logger.error(f"Exception fetching image for camera {camera_id}: {e}")
            return {output_key: None}

    def get_camera_name(self, camera_id, update_cache=False):
        """
        Returns the camera name for a given camera_id, using a CSV cache (max age 1 month).
        If not cached or expired, fetches from the website and updates the cache.
        Args:
            camera_id (int or str): Camera ID
            update_cache (bool): Force update from website even if cache is fresh
        Returns:
            str or None: Camera name if found, else None
        """
        cache = self._load_cache()
        now = datetime.utcnow()
        str_id = str(camera_id)
        # Check cache
        if str_id in cache:
            name, last_updated = cache[str_id]
            age = now - last_updated
            if age < timedelta(days=self.CACHE_EXPIRY_DAYS) and not update_cache:
                self.logger.info(f"Camera name for {camera_id} found in cache: {name}")
                return name
        # Fetch from web
        name = self._fetch_camera_name_from_web(camera_id)
        if name:
            cache[str_id] = (name, now)
            self._save_cache(cache)
            self.logger.info(f"Fetched and cached camera name for {camera_id}: {name}")
        else:
            self.logger.error(f"Could not fetch camera name for {camera_id} from web.")
        return name

    def _fetch_camera_name_from_web(self, camera_id):
        """
        Fetches the camera name from the camera's web page.
        Returns:
            str or None: Camera name if found, else None
        """
        url = f"https://trafficwatchni.com/twni/cameras/static?id={camera_id}"
        try:
            resp = requests.get(url, timeout=10)
            if resp.status_code == 200:
                # Look for the camera name in the header tag
                match = re.search(r'<header[^>]*class="[^"]*h4[^"]*"[^>]*>(.*?)</header>', resp.text, re.IGNORECASE | re.DOTALL)
                if match:
                    return match.group(1).strip()
                else:
                    self.logger.error(f"Camera name not found in HTML for camera {camera_id}. Snippet: {resp.text[:200]}")
            elif resp.status_code == 404:
                self.logger.warning(f"Camera page for {camera_id} not found (404).")
            elif resp.status_code == 500:
                self.logger.error(f"Camera page for {camera_id} returned server error (500).")
            else:
                self.logger.error(f"Failed to fetch camera page for {camera_id}: HTTP {resp.status_code}")
        except Exception as e:
            self.logger.error(f"Exception fetching camera page for {camera_id}: {e}")
        return None

    def _load_cache(self):
        """
        Loads the camera name cache from CSV.
        Returns:
            dict: {camera_id: (camera_name, last_updated_datetime)}
        """
        cache = {}
        if not os.path.exists(self.CACHE_FILE):
            return cache
        try:
            with open(self.CACHE_FILE, newline='', encoding='utf-8') as csvfile:
                reader = csv.DictReader(csvfile)
                for row in reader:
                    cid = row['camera_id']
                    name = row['camera_name']
                    last_updated = datetime.fromisoformat(row['last_updated'])
                    cache[cid] = (name, last_updated)
        except Exception as e:
            self.logger.error(f"Error loading camera name cache: {e}")
        return cache

    def _save_cache(self, cache):
        """
        Saves the camera name cache to CSV.
        Args:
            cache (dict): {camera_id: (camera_name, last_updated_datetime)}
        """
        try:
            with open(self.CACHE_FILE, 'w', newline='', encoding='utf-8') as csvfile:
                fieldnames = ['camera_id', 'camera_name', 'last_updated']
                writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
                writer.writeheader()
                for cid, (name, last_updated) in cache.items():
                    writer.writerow({
                        'camera_id': cid,
                        'camera_name': name,
                        'last_updated': last_updated.isoformat()
                    })
        except Exception as e:
            self.logger.error(f"Error saving camera name cache: {e}")

if __name__ == "__main__":
    import time
    import logging
    logging.basicConfig(level=logging.INFO)
    plugin = TrafficWatchNIImage()
    for camera_id in range(787, 1001):
        name = plugin.get_camera_name(camera_id)
        if name:
            print(f"Camera {camera_id}: {name}")
        else:
            print(f"Camera {camera_id}: Name not found.")
        time.sleep(0.25)  # Basic rate limiting: 1 request per second
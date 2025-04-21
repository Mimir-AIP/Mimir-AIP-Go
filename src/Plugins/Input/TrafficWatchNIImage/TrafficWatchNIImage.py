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
    Also provides a method to get the camera name and image URL, using a CSV cache.
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
        name, image_url = self.get_camera_metadata(camera_id)
        if not image_url:
            self.logger.error(f"No image URL found for camera {camera_id}.")
            return {output_key: None}
        os.makedirs(output_dir, exist_ok=True)
        image_path = os.path.join(output_dir, f"{camera_id}.jpg")
        try:
            resp = requests.get(image_url, timeout=10)
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

    def get_camera_metadata(self, camera_id):
        """
        Gets the camera name and image URL, using the CSV cache if valid.
        Returns:
            tuple: (camera_name, image_url)
        """
        cache = self._load_cache()
        now = datetime.utcnow()
        if camera_id in cache:
            name, last_updated, image_url = cache[camera_id]
            # Only use cache if both name and image_url are valid and not expired
            if name and image_url and (now - last_updated).days < self.CACHE_EXPIRY_DAYS:
                return name, image_url
        # Fetch from web and update cache
        name, image_url = self._fetch_camera_metadata_from_web(camera_id)
        if name and image_url:
            cache[camera_id] = (name, now, image_url)
            self._save_cache(cache)
        return name, image_url

    def _fetch_camera_metadata_from_web(self, camera_id):
        url = f"https://trafficwatchni.com/twni/cameras/static?id={camera_id}"
        try:
            resp = requests.get(url, timeout=10)
            if resp.status_code == 200:
                html = resp.text
                # Camera name in header (robust regex: allow for any attribute order/whitespace)
                name_match = re.search(r'<header[^>]*class=["\']?[^"\'>]*h4[^"\'>]*["\']?[^>]*>(.*?)</header>', html, re.IGNORECASE | re.DOTALL)
                camera_name = name_match.group(1).strip() if name_match else None
                img_regex = r'<img[^>]*class=["\'][^"\'>]*cctvImage[^"\'>]*["\'][^>]*src=["\']([^"\']+)["\']'
                img_tags = re.findall(r'<img[^>]+>', html, re.IGNORECASE | re.DOTALL)
                cctv_img_tag = None
                for tag in img_tags:
                    if 'cctvImage' in tag:
                        cctv_img_tag = tag
                        break
                image_url = None
                if cctv_img_tag:
                    src_match = re.search(r'src=["\']([^"\']+)["\']', cctv_img_tag)
                    if src_match:
                        image_url = src_match.group(1)
                        if not image_url.startswith('http'):
                            if image_url.startswith('/'):
                                image_url = f"https://trafficwatchni.com{image_url}"
                            else:
                                image_url = f"https://trafficwatchni.com/twni/cameras/{image_url}"
                return camera_name, image_url
            else:
                return None, None
        except Exception as e:
            return None, None

    def _load_cache(self):
        """
        Loads the camera name cache from CSV.
        Returns:
            dict: {camera_id: (camera_name, last_updated_datetime, image_url)}
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
                    image_url = row.get('image_url')
                    cache[cid] = (name, last_updated, image_url)
        except Exception as e:
            self.logger.error(f"Error loading camera name cache: {e}")
        return cache

    def _save_cache(self, cache):
        """
        Saves the camera name cache to CSV.
        Args:
            cache (dict): {camera_id: (camera_name, last_updated_datetime, image_url)}
        """
        try:
            with open(self.CACHE_FILE, 'w', newline='', encoding='utf-8') as csvfile:
                fieldnames = ['camera_id', 'camera_name', 'last_updated', 'image_url']
                writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
                writer.writeheader()
                for cid, (name, last_updated, image_url) in cache.items():
                    writer.writerow({
                        'camera_id': cid,
                        'camera_name': name,
                        'last_updated': last_updated.isoformat(),
                        'image_url': image_url or ''
                    })
        except Exception as e:
            self.logger.error(f"Error saving camera name cache: {e}")

if __name__ == "__main__":
    import os
    import random
    import logging
    import time
    logging.basicConfig(level=logging.INFO)

    plugin = TrafficWatchNIImage()
    # --- Rebuild cache for all cameras ---
    print("Rebuilding cache for all cameras...")
    camera_ids = [str(i) for i in range(1, 1001)]  # Adjust range as needed
    for camera_id in camera_ids:
        name, image_url = plugin.get_camera_metadata(camera_id)
        if name and image_url:
            print(f"Cached {camera_id}: {name} | {image_url}")
        time.sleep(0.1)  # Rate limiting

    # --- Download random 10 images ---
    print("\nDownloading 10 random camera images...")
    valid_cameras = []
    for camera_id in camera_ids:
        name, image_url = plugin.get_camera_metadata(camera_id)
        if name and image_url:
            valid_cameras.append((camera_id, name, image_url))
    sample = random.sample(valid_cameras, min(10, len(valid_cameras)))
    os.makedirs("test_images", exist_ok=True)
    for camera_id, name, image_url in sample:
        filename = f"test_images/{camera_id}_{name.replace(' ', '_').replace('/', '_')}.jpg"
        try:
            resp = requests.get(image_url, timeout=10)
            if resp.status_code == 200:
                with open(filename, "wb") as f:
                    f.write(resp.content)
                print(f"Saved image for {camera_id}: {filename}")
            else:
                print(f"Failed to download image for {camera_id}: HTTP {resp.status_code}")
        except Exception as e:
            print(f"Error downloading image for {camera_id}: {e}")
"""
Plugin for making API requests to httpbin.org
"""

import requests

class APIPlugin:
    """
    Plugin for making API requests to httpbin.org
    """
    plugin_type = "Input"

    def __init__(self):
        """
        Initialize the plugin with the httpbin.org base URL
        """
        self.base_url = "https://httpbin.org"

    def get_bytes(self, n):
        """
        Get n random bytes from httpbin.org

        Args:
            n (int): Number of bytes to retrieve

        Returns:
            bytes: Random bytes from httpbin.org, or None if request fails
        """
        response = requests.get(f"{self.base_url}/bytes/{n}")
        if response.status_code == 200:
            return response.content
        return None

    def get_100_bytes(self):
        """
        Get 100 random bytes from httpbin.org

        Returns:
            bytes: 100 random bytes from httpbin.org, or None if request fails
        """
        return self.get_bytes(100)

if __name__ == "__main__":
    # Test the APIPlugin
    plugin = APIPlugin()
    print(plugin.get_100_bytes())
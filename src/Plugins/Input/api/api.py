import requests

class APIPlugin:
    plugin_type = "Input"

    def __init__(self):
        self.base_url = "https://httpbin.org"

    def get_bytes(self, n):
        response = requests.get(f"{self.base_url}/bytes/{n}")
        if response.status_code == 200:
            return response.content
        else:
            print(f"Error getting bytes: {response.status_code}")
            return None
    def get_100_bytes(self):
        return self.get_bytes(100)

if __name__ == "__main__":
    # Test the APIPlugin
    plugin = APIPlugin()
    print(plugin.get_100_bytes())
import requests
import json

class WebSearchPlugin:
    plugin_type = "Input"

    def __init__(self):
        self.base_url = "https://api.duckduckgo.com/"

    def search(self, query):
        """
        Searches the web using DuckDuckGo.
        """
        url = f"{self.base_url}?q={query}&format=json"
        response = requests.get(url)
        if response.status_code in [200, 202]:
            return response.json()
        else:
            print(f"Error: {response.status_code}")
            return None

    def get_search_results(self, query):
        """
        Returns the search results.
        """
        return self.search(query)

if __name__ == "__main__":
    # Test the plugin
    plugin = WebSearchPlugin()
    query = "test search query"
    results = plugin.get_search_results(query)
    if results:
        print(json.dumps(results, indent=2))
    else:
        print("No results found or an error occurred.")
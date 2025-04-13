"""
Plugin for scraping White House press pool reports from forth.news

Example usage:
    plugin = WhiteHousePressPool()
    result = plugin.execute_pipeline_step({
        "config": {
            "last_id": "123",  # Optional, fetch entries after this ID
            "max_entries": 10   # Optional, limit number of entries
        },
        "output": "press_pool"
    }, {})
"""

import os
import sys
import requests
import json
from datetime import datetime

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin


class WhiteHousePressPool(BasePlugin):
    """Plugin for scraping White House press pool reports from forth.news"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the scraper with API endpoint and request parameters"""
        self.url = 'https://www.forth.news/api/graphql'
        self.headers = self._get_headers()

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "WhiteHousePressPool",
            "config": {
                "last_id": "123",  # Optional, fetch entries after this ID
                "max_entries": 10   # Optional, limit number of entries
            },
            "output": "press_pool"
        }
        """
        config = step_config.get("config", {})
        
        # Get query parameters
        last_id = config.get("last_id")
        max_entries = config.get("max_entries")
        
        # Build payload
        payload = self._get_payload(last_id)
        
        # Fetch and parse data
        data = self.fetch_data(payload)
        
        # Limit entries if requested
        if max_entries and isinstance(max_entries, int):
            data["items"] = data["items"][:max_entries]
        
        return {step_config["output"]: data}

    def _get_headers(self):
        """Get headers for the API request"""
        return {
            'accept': '*/*',
            'accept-language': 'en-GB,en-US;q=0.9,en;q=0.8',
            'content-type': 'application/json',
            'origin': 'https://www.forth.news',
            'referer': 'https://www.forth.news/whpool',
            'user-agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
        }

    def _get_payload(self, last_id=None):
        """
        Get GraphQL query payload for the API request
        
        Args:
            last_id (str): Optional ID to fetch entries after
        """
        return {
            "operationName": "getList",
            "variables": {
                "shortName": "whpool",
                "last": last_id
            },
            "query": """
                query getList($shortName: String!, $last: ID) {
                    list(shortName: $shortName) {
                        id
                        shortName
                        entries(last: $last) {
                            id
                            title
                            pvwText
                            createdAt
                            __typename
                        }
                        __typename
                    }
                }
            """
        }

    def fetch_data(self, payload):
        """
        Fetch the latest press pool reports
        
        Args:
            payload (dict): GraphQL query payload
            
        Returns:
            dict: JSON feed containing press pool reports
            
        Raises:
            ValueError: If there's an error fetching or parsing the data
        """
        try:
            response = requests.post(self.url, headers=self.headers, json=payload)
            response.raise_for_status()
            
            data = response.json()
            
            if "errors" in data:
                raise ValueError(f"GraphQL error: {data['errors']}")
                
            return self._parse_data(data)
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Error fetching press pool data: {str(e)}")
        except (KeyError, json.JSONDecodeError) as e:
            raise ValueError(f"Error parsing press pool data: {str(e)}")

    def _parse_data(self, data):
        """
        Parse the JSON response into a JSON feed format
        
        Args:
            data (dict): Raw API response data
            
        Returns:
            dict: Formatted JSON feed
        """
        json_feed = {
            "version": "https://jsonfeed.org/version/1",
            "title": "Forth News - White House Pool",
            "home_page_url": "https://www.forth.news/whpool",
            "feed_url": "https://www.forth.news/api/graphql",
            "items": []
        }
        
        try:
            entries = data["data"]["list"]["entries"]
            for entry in entries:
                json_feed["items"].append({
                    "id": entry["id"],
                    "url": f"https://www.forth.news/whpool/{entry['id']}",
                    "title": entry["title"],
                    "content_text": entry["pvwText"],
                    "date_published": datetime.fromtimestamp(
                        int(entry["createdAt"]) / 1000
                    ).isoformat()
                })
                
            return json_feed
            
        except (KeyError, ValueError) as e:
            raise ValueError(f"Error parsing entry data: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = WhiteHousePressPool()
    
    # Test with basic configuration
    test_config = {
        "plugin": "WhiteHousePressPool",
        "config": {
            "max_entries": 3
        },
        "output": "feed"
    }
    
    # Test initial fetch
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        feed = result["feed"]
        
        print(f"Feed Title: {feed['title']}")
        print(f"Number of items: {len(feed['items'])}")
        
        # Print items
        for item in feed["items"]:
            print(f"\nTitle: {item['title']}")
            print(f"Date: {item['date_published']}")
            print(f"URL: {item['url']}")
            print(f"Content Preview: {item['content_text'][:100]}...")
            
    except ValueError as e:
        print(f"Error: {e}")
        feed = None
        
    # Test with last_id if we have items
    if feed and feed["items"]:
        try:
            test_config["config"]["last_id"] = feed["items"][0]["id"]
            result = plugin.execute_pipeline_step(test_config, {})
            feed = result["feed"]
            print(f"\nFetched {len(feed['items'])} new items after last_id")
        except ValueError as e:
            print(f"Error: {e}")
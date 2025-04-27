"""
Bloomberg News API plugin that fetches and formats data into RSSGuard-compatible JSON

Example usage:
    plugin = Bloomberg()
    result = plugin.execute_pipeline_step({
        "config": {
            "api_url": "https://feeds.bloomberg.com/news.json",
            "params": {
                "ageHours": 120,
                "token": "your_token",
                "tickers": "NTRS:US"
            }
        },
        "output": "bloomberg_feed"
    }, {})
"""

import requests
import json
import datetime
from Plugins.BasePlugin import BasePlugin
import logging

class Bloomberg(BasePlugin):
    """Bloomberg News API plugin that fetches and formats data into RSSGuard-compatible JSON"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the Bloomberg plugin"""
        self.base_url = "https://feeds.bloomberg.com/news.json"

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "Bloomberg",
            "config": {
                "api_url": "https://feeds.bloomberg.com/news.json",  # Optional
                "params": {
                    "ageHours": 120,
                    "token": "your_token",
                    "tickers": "NTRS:US"
                }
            },
            "output": "bloomberg_feed"
        }
        """
        config = step_config["config"]
        
        # Get API URL and params
        api_url = config.get("api_url", self.base_url)
        params = config.get("params", {})
        
        # Fetch and format data
        feed_data = self.get_feed(api_url, params)
        # Defensive patch: ensure output is always a native Python object, never a string
        import ast, logging
        logger = logging.getLogger(__name__)
        def parse_if_str(val):
        """parse_if_str: TODO add description."""
            if isinstance(val, str):
                try:
                    parsed = ast.literal_eval(val)
                    if isinstance(parsed, (list, dict)):
                        return parsed
                except Exception:
                    pass
            return val
        feed_data = parse_if_str(feed_data)
        logger.info(f"[Bloomberg:execute_pipeline_step] Returning type: {type(feed_data)}, sample: {str(feed_data)[:300]}")
        return {step_config["output"]: feed_data}

    def get_feed(self, api_url, params=None):
        """Fetches and formats the Bloomberg feed as a Python object, never as a string."""
        import requests
        import datetime
        import logging
        logger = logging.getLogger(__name__)
        response = requests.get(api_url, params=params)
        logger.info(f"[Bloomberg:get_feed] Response status: {response.status_code}")
        if response.status_code != 200:
            logger.error(f"[Bloomberg:get_feed] Failed to fetch feed: {response.text}")
            return {}
        try:
            data = response.json()
        except Exception:
            logger.error(f"[Bloomberg:get_feed] Response not JSON, attempting eval")
            import ast
            try:
                data = ast.literal_eval(response.text)
            except Exception:
                logger.error(f"[Bloomberg:get_feed] Could not parse response as JSON or Python object.")
                return {}
        # Format to RSSGuard-compatible JSON
        rssguard_data = {
            "version": 1,
            "title": data.get("title", "Bloomberg News Feed"),
            "link": data.get("link", "https://www.bloomberg.com/"),
            "description": data.get("description", "Bloomberg News"),
            "items": []
        }
        for item in data.get("items", []):
            rss_item = {
                "title": item.get("title", "No Title"),
                "link": item.get("link", ""),
                "guid": item.get("id", ""),
                "description": item.get("description", ""),
                "pubDate": item.get("pubDate", datetime.datetime.now().isoformat()),
                "author": item.get("author", "Bloomberg"),
                "categories": item.get("categories", []),
                "tickers": item.get("tickers", [])
            }
            rssguard_data["items"].append(rss_item)
        logger.info(f"[Bloomberg:get_feed] Returning type: {type(rssguard_data)}, sample: {str(rssguard_data)[:300]}")
        return rssguard_data


if __name__ == "__main__":
    # Test the plugin
    plugin = Bloomberg()
    
    # Test configuration
    test_config = {
        "plugin": "Bloomberg",
        "config": {
            "params": {
                "ageHours": 120,
                "token": "glassdoor:gd4bloomberg",
                "tickers": "NTRS:US"
            }
        },
        "output": "feed"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        feed = result["feed"]
        
        print(f"Feed Title: {feed['title']}")
        print(f"Number of items: {len(feed['items'])}")
        
        # Print first 3 items
        for item in feed["items"][:3]:
            print(f"\nTitle: {item['title']}")
            print(f"Link: {item['link']}")
            print(f"Author: {item['author']}")
            if item.get("tickers"):
                print(f"Tickers: {', '.join(item['tickers'])}")
                
    except ValueError as e:
        print(f"Error: {e}")
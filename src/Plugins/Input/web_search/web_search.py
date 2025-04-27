"""
Web search plugin using DuckDuckGo

Example usage:
    plugin = WebSearchPlugin()
    result = plugin.execute_pipeline_step({
        "config": {
            "query": "search query",
            "format": "json"  # or "text"
        },
        "output": "search_results"
    }, {})
"""

import requests
import re
import logging
from Plugins.BasePlugin import BasePlugin


class WebSearchPlugin(BasePlugin):
"""WebSearchPlugin: TODO add description."""
    plugin_type = "Input"

    def __init__(self):
        # WARNING: DuckDuckGo aggressively blocks automated scraping. This plugin is likely to fail.
    """__init__: TODO add description."""
        # Use a third-party search API or a headless browser for reliable results.
        self.base_url = "https://html.duckduckgo.com/html/"

    @staticmethod
    def extract_results_from_html(html):
        """
        Extract search result URLs and descriptions from DuckDuckGo HTML page using regex.
        Returns a list of dicts: {"url": ..., "description": ...}
        """
        results = []
        # Each result is in a <div class="result__body"> ... <a class="result__a" href="...">title</a> ... <a ...> ... </a> <div class="result__snippet">desc</div>
        pattern = re.compile(r'<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>[\s\S]*?<div class="result__snippet">(.*?)</div>', re.IGNORECASE)
        for match in pattern.finditer(html):
            url = match.group(1)
            # Remove HTML tags from description
            desc = re.sub(r'<.*?>', '', match.group(3)).strip()
            results.append({"url": url, "description": desc})
        return results

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin using DuckDuckGo HTML scraping
        WARNING: This approach is likely to fail due to bot detection and is not recommended for production use.
        """
        config = step_config.get("config", {})
        query = config.get("query")
        if not query:
            raise ValueError("No query provided for web search")
        if isinstance(query, list):
            query = ' '.join(query)
        params = {"q": query}
        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.5"
        }
        response = requests.get(self.base_url, params=params, headers=headers, timeout=10)
        if response.status_code != 200:
            raise RuntimeError(f"DuckDuckGo HTML returned status {response.status_code}")
        html = response.text
        results = self.extract_results_from_html(html)
        if config.get("extract_urls"):
            urls = [r["url"] for r in results]
            return {step_config.get("output", "search_results"): urls}
        return {step_config.get("output", "search_results"): results}


if __name__ == "__main__":
    # Test the plugin
    plugin = WebSearchPlugin()
    
    # Test single query
    test_config = {
        "plugin": "WebSearch",
        "config": {
            "query": "test search query",
            "format": "json"
        },
        "output": "results"
    }
    
    result = plugin.execute_pipeline_step(test_config, {})
    if result["results"]:
        print(result["results"])
    else:
        print("No results found or an error occurred.")
        
    # Test query from context
    test_context = {"my_query": "another test query"}
    test_config["config"]["query"] = "my_query"
    
    result = plugin.execute_pipeline_step(test_config, test_context)
    if result["results"]:
        print("\nResults from context query:")
        print(result["results"])
    else:
        print("No results found or an error occurred.")
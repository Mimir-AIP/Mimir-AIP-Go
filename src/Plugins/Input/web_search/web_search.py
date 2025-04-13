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
import json
from Plugins.BasePlugin import BasePlugin


class WebSearchPlugin(BasePlugin):
    plugin_type = "Input"

    def __init__(self):
        self.base_url = "https://api.duckduckgo.com/"

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "WebSearch",
            "config": {
                "query": "search query",
                "format": "json"  # or "text"
            },
            "output": "search_results"
        }
        
        If config.query is a variable from context, it will be evaluated.
        """
        config = step_config["config"]
        query = config["query"]
        
        # If query is a variable reference, evaluate it
        if isinstance(query, str) and query in context:
            query = context[query]
        elif isinstance(query, list):
            # Handle list of queries (as used in the POC pipeline)
            results = ""
            for q in query:
                if isinstance(q, str) and q in context:
                    q = context[q]
                results += self.search(q)
            return {step_config["output"]: results}
        
        result = self.search(query)
        return {step_config["output"]: result}

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
        print(json.dumps(result["results"], indent=2))
    else:
        print("No results found or an error occurred.")
        
    # Test query from context
    test_context = {"my_query": "another test query"}
    test_config["config"]["query"] = "my_query"
    
    result = plugin.execute_pipeline_step(test_config, test_context)
    if result["results"]:
        print("\nResults from context query:")
        print(json.dumps(result["results"], indent=2))
    else:
        print("No results found or an error occurred.")
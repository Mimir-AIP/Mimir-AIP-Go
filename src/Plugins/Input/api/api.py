"""
Plugin for making API requests to various endpoints

Example usage:
    plugin = ApiPlugin()
    result = plugin.execute_pipeline_step({
        "config": {
            "url": "https://httpbin.org/bytes/100",
            "method": "GET",
            "headers": {"Accept": "application/json"},
            "params": {"key": "value"},
            "data": {"field": "value"},
            "timeout": 30
        },
        "output": "api_response"
    }, {})
"""

import os
import sys
import json
import requests
from datetime import datetime

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin


class ApiPlugin(BasePlugin):
    """Plugin for making configurable API requests"""
    plugin_type = "Input"

    def __init__(self):
        """Initialize the plugin with default settings"""
        self.default_timeout = 30
        self.default_headers = {
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
        }

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "ApiPlugin",
            "config": {
                "url": "https://api.example.com/endpoint",
                "method": "GET",  # Optional, default: GET
                "headers": {},    # Optional, additional headers
                "params": {},     # Optional, URL parameters
                "data": {},      # Optional, request body for POST/PUT
                "timeout": 30    # Optional, request timeout in seconds
            },
            "output": "api_response"
        }
        """
        config = step_config.get("config", {})
        
        # Get request parameters
        url = config.get("url")
        if not url:
            raise ValueError("URL is required in config")
            
        method = config.get("method", "GET").upper()
        headers = {**self.default_headers, **(config.get("headers", {}))}
        params = config.get("params", {})
        data = config.get("data")
        timeout = config.get("timeout", self.default_timeout)
        
        # Make the request
        response = self.make_request(
            url=url,
            method=method,
            headers=headers,
            params=params,
            data=data,
            timeout=timeout
        )
        
        return {step_config["output"]: response}

    def make_request(self, url, method="GET", headers=None, params=None, data=None, timeout=None):
        """
        Make an HTTP request
        
        Args:
            url (str): Target URL
            method (str): HTTP method (GET, POST, PUT, DELETE)
            headers (dict): Request headers
            params (dict): URL parameters
            data (dict): Request body for POST/PUT
            timeout (int): Request timeout in seconds
            
        Returns:
            dict: Response data including status, headers, and content
            
        Raises:
            ValueError: If there's an error making the request
        """
        try:
            # Prepare request arguments
            kwargs = {
                "url": url,
                "headers": headers or {},
                "params": params or {},
                "timeout": timeout or self.default_timeout
            }
            
            # Add data for POST/PUT requests
            if data is not None and method in ["POST", "PUT"]:
                kwargs["json"] = data
            
            # Make the request
            response = requests.request(method, **kwargs)
            response.raise_for_status()
            
            # Try to parse as JSON
            try:
                content = response.json()
            except json.JSONDecodeError:
                content = response.text
                
            # Build response data
            return {
                "url": response.url,
                "status_code": response.status_code,
                "headers": dict(response.headers),
                "content": content,
                "timestamp": datetime.now().isoformat()
            }
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Request failed: {str(e)}")
        except Exception as e:
            raise ValueError(f"Error processing request: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = ApiPlugin()
    
    # Test configurations
    test_configs = [
        {
            "name": "GET bytes",
            "config": {
                "plugin": "ApiPlugin",
                "config": {
                    "url": "https://httpbin.org/bytes/100",
                    "method": "GET"
                },
                "output": "response"
            }
        },
        {
            "name": "POST data",
            "config": {
                "plugin": "ApiPlugin",
                "config": {
                    "url": "https://httpbin.org/post",
                    "method": "POST",
                    "data": {"test": "data"},
                    "headers": {"X-Test": "true"}
                },
                "output": "response"
            }
        },
        {
            "name": "GET with params",
            "config": {
                "plugin": "ApiPlugin",
                "config": {
                    "url": "https://httpbin.org/get",
                    "params": {"key": "value"}
                },
                "output": "response"
            }
        }
    ]
    
    # Run tests
    for test in test_configs:
        print(f"\nTesting: {test['name']}")
        try:
            result = plugin.execute_pipeline_step(test["config"], {})
            response = result["response"]
            
            print(f"Status: {response['status_code']}")
            print(f"URL: {response['url']}")
            print("Headers:")
            for key, value in response['headers'].items():
                print(f"  {key}: {value}")
            print("Content preview:")
            print(json.dumps(response['content'], indent=2)[:200] + "...")
            
        except ValueError as e:
            print(f"Error: {e}")
"""
Plugin for web scraping using BeautifulSoup4

Example usage:
    plugin = WebScraping()
    result = plugin.execute_pipeline_step({
        "config": {
            "url": "https://example.com",
            "selectors": {
                "title": "h1",
                "content": "article p",
                "links": "a.external"
            },
            "headers": {  # Optional
                "User-Agent": "Custom User Agent"
            }
        },
        "output": "scraped_data"
    }, {})
"""

import requests
from bs4 import BeautifulSoup
from Plugins.BasePlugin import BasePlugin


class WebScraping(BasePlugin):
    """Plugin for web scraping using BeautifulSoup4"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the WebScraping plugin"""
        self.default_headers = {
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
        }

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "WebScraping",
            "config": {
                "url": "https://example.com",
                "selectors": {
                    "title": "h1",
                    "content": "article p",
                    "links": "a.external"
                },
                "headers": {  # Optional
                    "User-Agent": "Custom User Agent"
                }
            },
            "output": "scraped_data"
        }
        
        If url is a variable from context, it will be evaluated.
        """
        config = step_config["config"]
        url = config["url"]
        
        # If url is a variable reference, evaluate it
        if isinstance(url, str) and url in context:
            url = context[url]
        
        # Get headers
        headers = {**self.default_headers, **(config.get("headers", {}))}
        
        # Scrape data
        data = self.scrape_url(
            url=url,
            selectors=config["selectors"],
            headers=headers
        )
        
        return {step_config["output"]: data}

    def scrape_url(self, url, selectors, headers=None):
        """
        Scrape data from a URL using BeautifulSoup4
        
        Args:
            url (str): URL to scrape
            selectors (dict): Dictionary mapping names to CSS selectors
            headers (dict): Optional request headers
            
        Returns:
            dict: Scraped data
        """
        if headers is None:
            headers = self.default_headers
            
        try:
            # Fetch page
            response = requests.get(url, headers=headers)
            response.raise_for_status()
            
            # Parse HTML
            soup = BeautifulSoup(response.text, 'html.parser')
            
            # Extract data using selectors
            data = {}
            for name, selector in selectors.items():
                elements = soup.select(selector)
                if len(elements) == 1:
                    data[name] = elements[0].get_text(strip=True)
                else:
                    data[name] = [el.get_text(strip=True) for el in elements]
                    
            return data
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Error scraping URL: {str(e)}")
        except Exception as e:
            raise ValueError(f"Error parsing HTML: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = WebScraping()
    
    # Test with direct URL
    test_config = {
        "plugin": "WebScraping",
        "config": {
            "url": "https://example.com",
            "selectors": {
                "title": "h1",
                "description": "p",
                "links": "a"
            }
        },
        "output": "data"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print("Scraped data:")
        for key, value in result["data"].items():
            if isinstance(value, list):
                print(f"\n{key}:")
                for item in value[:3]:  # Show first 3 items
                    print(f"  - {item}")
            else:
                print(f"\n{key}: {value}")
    except ValueError as e:
        print(f"Error: {e}")
        
    # Test with URL from context
    test_context = {
        "target_url": "https://news.ycombinator.com"
    }
    
    test_config["config"].update({
        "url": "target_url",
        "selectors": {
            "titles": ".title a",
            "scores": ".score",
            "authors": ".hnuser"
        }
    })
    
    try:
        result = plugin.execute_pipeline_step(test_config, test_context)
        print("\nHacker News Top Stories:")
        for i, title in enumerate(result["data"]["titles"][:5]):
            print(f"\n{i+1}. {title}")
            if i < len(result["data"].get("scores", [])):
                print(f"   Score: {result['data']['scores'][i]}")
            if i < len(result["data"].get("authors", [])):
                print(f"   Author: {result['data']['authors'][i]}")
    except ValueError as e:
        print(f"Error: {e}")
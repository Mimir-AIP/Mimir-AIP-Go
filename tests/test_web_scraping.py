import pytest
import os
from Plugins.Input.WebScraping.WebScraping import WebScraping

def test_web_scraping_initialization():
    """Test WebScraping plugin initialization"""
    plugin = WebScraping()
    assert isinstance(plugin, WebScraping)

def test_web_scraping_execute():
    """Test web scraping with sample URL"""
    plugin = WebScraping()
    
    # Test with sample web result
    config = {
        "url": "https://en.wikipedia.org/wiki/Paris",
        "selectors": {
            "title": "h1",
            "content": ".mw-parser-output > p"
        }
    }
    
    result = plugin.execute_pipeline_step({
        "config": config,
        "output": "scraped_results"
    }, {})
    
    assert "scraped_results" in result
    assert isinstance(result["scraped_results"], dict)
    assert len(result["scraped_results"]) > 0
    
    # Check structure of scraped results
    assert "title" in result["scraped_results"]
    assert "content" in result["scraped_results"]
    assert isinstance(result["scraped_results"]["content"], list)

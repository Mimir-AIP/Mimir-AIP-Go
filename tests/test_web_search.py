import pytest
import os
from Plugins.Input.web_search.web_search import WebSearchPlugin

def test_web_search_initialization():
    """Test WebSearch plugin initialization"""
    plugin = WebSearchPlugin()
    assert isinstance(plugin, WebSearchPlugin)

def test_web_search_execute():
    """Test executing web search with multiple queries"""
    plugin = WebSearchPlugin()
    
    # Test with multiple search queries
    config = {
        "query": "capital of France"
    }
    
    result = plugin.execute_pipeline_step({
        "config": config,
        "output": "search_results"
    }, {})
    
    assert "search_results" in result
    assert isinstance(result["search_results"], dict)
    assert len(result["search_results"]) > 0
    
    # Check structure of search results
    assert "Abstract" in result["search_results"]
    assert "AbstractSource" in result["search_results"]
    assert "AbstractText" in result["search_results"]

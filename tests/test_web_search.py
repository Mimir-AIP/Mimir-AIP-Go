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

def test_web_search_pipeline_queries():
    """Test executing web search with pipeline-style queries"""
    plugin = WebSearchPlugin()
    pipeline_queries = [
        "Biden Trump speech White House",
        "Biden attacks Trump",
        "Biden first speech since leaving office",
        "Trump Biden conflict",
        "Biden Trump rivalry"
    ]
    for query in pipeline_queries:
        config = {"query": query}
        result = plugin.execute_pipeline_step({"config": config, "output": "search_results"}, {})
        assert "search_results" in result
        assert isinstance(result["search_results"], dict)
        # Print results for inspection
        print(f"Query: {query}\nResults: {result['search_results']}")
        # No assertion on length, since DuckDuckGo API may return empty dict

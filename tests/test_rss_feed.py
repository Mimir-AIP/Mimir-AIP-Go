import pytest
import os
from Plugins.Input.rss_feed.rss_feed import RssFeed

def test_rss_feed_initialization():
    """Test RSSFeed plugin initialization"""
    plugin = RssFeed()
    assert isinstance(plugin, RssFeed)

def test_rss_feed_fetch():
    """Test fetching RSS feed data"""
    plugin = RssFeed()
    
    # Test with BBC News feed
    config = {
        "url": "http://feeds.bbci.co.uk/news/world/rss.xml",
        "feed_name": "BBC_News"
    }
    
    result = plugin.execute_pipeline_step({
        "config": config,
        "output": "feed_BBC_News"
    }, {})
    
    assert "feed_BBC_News" in result
    assert isinstance(result["feed_BBC_News"], list)
    assert len(result["feed_BBC_News"]) > 0
    
    # Check structure of a feed item
    first_item = result["feed_BBC_News"][0]
    assert "title" in first_item
    assert "description" in first_item
    assert "link" in first_item

import sys
import os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../src')))
import pytest
from Plugins.Input.archive_web_search.archive_web_search import ArchiveWebSearchPlugin

def test_archive_web_search_basic():
    plugin = ArchiveWebSearchPlugin()
    results = plugin.search("bbc news", max_results=3)
    assert isinstance(results, list)
    assert len(results) <= 3
    for result in results:
        assert "title" in result
        assert "url" in result
        print(result)

def test_archive_web_search_mediatype():
    plugin = ArchiveWebSearchPlugin()
    results = plugin.search("climate change", mediatype="texts", max_results=2)
    assert isinstance(results, list)
    for result in results:
        assert result["mediatype"] == "texts"
        print(result)

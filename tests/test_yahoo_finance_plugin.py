"""
Tests for the YahooFinance plugin
"""

import os
import sys
import pytest
from unittest.mock import Mock

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Input.YahooFinance.YahooFinance import YahooFinance

@pytest.fixture
def yahoo_finance_plugin():
    return YahooFinance()

def test_quote_data(yahoo_finance_plugin):
    """Test fetching quote data from Yahoo Finance"""
    step_config = {
        "config": {
            "symbol": "AAPL",
            "type": "quote"
        },
        "output": "quote_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "quote_data" in result
    quote = result["quote_data"]
    
    # Check essential quote fields
    assert "symbol" in quote
    assert quote["symbol"] == "AAPL"
    assert "regularMarketPrice" in quote
    assert "regularMarketVolume" in quote
    assert "timestamp" in quote
    print(f"AAPL quote price: {quote['regularMarketPrice']}")

def test_batch_quotes(yahoo_finance_plugin):
    """Test fetching multiple quotes at once"""
    step_config = {
        "config": {
            "symbol": ["AAPL", "MSFT", "GOOG"],
            "type": "quote"
        },
        "output": "quote_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "quote_data" in result
    quotes = result["quote_data"]
    
    # Check that we got all quotes
    assert isinstance(quotes, list)
    assert len(quotes) == 3
    symbols = {q["symbol"] for q in quotes}
    assert symbols == {"AAPL", "MSFT", "GOOG"}
    
    # Check quote fields
    for quote in quotes:
        assert "regularMarketPrice" in quote
        assert "regularMarketVolume" in quote
        assert "timestamp" in quote
        print(f"{quote['symbol']} price: {quote['regularMarketPrice']}")

def test_chart_data(yahoo_finance_plugin):
    """Test fetching historical chart data from Yahoo Finance"""
    step_config = {
        "config": {
            "symbol": "MSFT",
            "type": "chart",
            "interval": "1d",
            "range": "5d"
        },
        "output": "chart_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "chart_data" in result
    chart = result["chart_data"]
    
    # Check essential chart fields
    assert "meta" in chart
    assert chart["meta"]["symbol"] == "MSFT"
    assert "timestamp" in chart
    assert "indicators" in chart
    assert len(chart["timestamp"]) > 0
    print(f"MSFT data points: {len(chart['timestamp'])}")

def test_error_handling(yahoo_finance_plugin):
    """Test error handling for invalid inputs"""
    # Test missing symbol
    step_config = {
        "config": {
            "type": "quote"
        },
        "output": "quote_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "quote_data" in result
    assert isinstance(result["quote_data"], str)
    assert "Symbol is required" in result["quote_data"]
    
    # Test invalid symbol
    step_config = {
        "config": {
            "symbol": "INVALID_SYMBOL_123456",
            "type": "quote"
        },
        "output": "quote_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "quote_data" in result
    error_msg = result["quote_data"]
    assert "No quote data found" in error_msg or "Request failed" in error_msg
    
    # Test invalid data type
    step_config = {
        "config": {
            "symbol": "AAPL",
            "type": "invalid"
        },
        "output": "quote_data"
    }
    
    result = yahoo_finance_plugin.execute_pipeline_step(step_config, {})
    assert "quote_data" in result
    error_msg = result["quote_data"]
    assert "Unsupported data type" in error_msg
"""
Yahoo Finance API plugin for fetching financial data

This plugin provides access to Yahoo Finance data including:
- Real-time quotes (single or batch)
- Historical price data with customizable intervals
- Dividend and split event data

Example usage:
    plugin = YahooFinance()
    result = plugin.execute_pipeline_step({
        "config": {
            "symbol": "AAPL",  # Or ["AAPL", "MSFT", "GOOG"] for batch quotes
            "type": "quote",  # or "chart"
            "interval": "1d",  # 1d, 1wk, 1mo (for chart type)
            "range": "1mo"    # 1d, 5d, 1mo, 3mo, 6mo, 1y, 5y, max (for chart type)
        },
        "output": "stock_data"
    }, {})
"""

import os
import sys
import json
import logging
from datetime import datetime
import requests
from typing import Dict, Any, Optional

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin

class YahooFinance(BasePlugin):
    """Plugin for fetching financial data from Yahoo Finance"""
    
    plugin_type = "Input"
    
    def __init__(self):
        """Initialize the YahooFinance plugin with default settings"""
        self.logger = logging.getLogger(__name__)
        self.base_url = "https://query2.finance.yahoo.com"
        self.default_headers = {
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko)',
            'Accept': 'application/json',
            'Origin': 'https://finance.yahoo.com',
            'Referer': 'https://finance.yahoo.com/'
        }

    def get_quote(self, symbols: str | list) -> Dict[str, Any] | list:
        """Get real-time quote data for one or more symbols
        
        Args:
            symbols: String symbol or list of symbols (e.g. "AAPL" or ["AAPL", "MSFT"])
            
        Returns:
            dict or list: Quote data including price, volume, market cap etc.
        """
        if isinstance(symbols, str):
            symbols = [symbols]
        
        results = []
        for symbol in symbols:
            try:
                url = f"{self.base_url}/v8/finance/chart/{symbol}"
                params = {
                    "interval": "1d",
                    "range": "1d",
                    "includePrePost": True,
                    "useYfid": True,
                    "includeAdjustedClose": True
                }
                
                self.logger.debug(f"Fetching quote for {symbol}")
                response = requests.get(url, params=params, headers=self.default_headers)
                response.raise_for_status()
                
                data = response.json()
                if "chart" in data and "result" in data["chart"] and data["chart"]["result"]:
                    result = data["chart"]["result"][0]
                    meta = result["meta"]
                    quote = {
                        "symbol": meta["symbol"],
                        "regularMarketPrice": meta["regularMarketPrice"],
                        "regularMarketVolume": meta.get("regularMarketVolume", 0),
                        "regularMarketPreviousClose": meta.get("previousClose"),
                        "regularMarketOpen": meta.get("regularMarketOpen"),
                        "regularMarketDayHigh": meta.get("regularMarketDayHigh"),
                        "regularMarketDayLow": meta.get("regularMarketDayLow"),
                        "timestamp": meta.get("regularMarketTime")
                    }
                    results.append(quote)
                else:
                    self.logger.warning(f"No data found for {symbol}")
                
            except requests.exceptions.RequestException as e:
                self.logger.error(f"Request failed for symbol {symbol}: {str(e)}")
                if len(symbols) == 1:
                    raise ValueError(f"Request failed for symbol {symbol}: {str(e)}")
            except Exception as e:
                self.logger.error(f"Error getting quote for {symbol}: {str(e)}")
                if len(symbols) == 1:
                    raise ValueError(f"Error getting quote for {symbol}: {str(e)}")
        
        if not results:
            raise ValueError(f"No quote data found for any symbols: {symbols}")
            
        return results[0] if len(symbols) == 1 else results

    def get_chart_data(self, symbol: str, interval: str = "1d", range: str = "1mo") -> Dict[str, Any]:
        """Get historical price data for a symbol
        
        Args:
            symbol (str): Stock symbol (e.g. "AAPL")
            interval (str): Data interval - 1d, 1wk, 1mo
            range (str): Historical range - 1d, 5d, 1mo, 3mo, 6mo, 1y, 5y, max
            
        Returns:
            dict: Historical price data including OHLCV
        """
        try:
            url = f"{self.base_url}/v8/finance/chart/{symbol}"
            params = {
                "interval": interval,
                "range": range,
                "includePrePost": True,
                "events": "div,split",
                "useYfid": True,
                "includeAdjustedClose": True
            }
            
            self.logger.debug(f"Fetching chart data for {symbol}")
            response = requests.get(url, params=params, headers=self.default_headers)
            response.raise_for_status()
            
            data = response.json()
            if "chart" in data and "result" in data["chart"]:
                results = data["chart"]["result"]
                if results:
                    return results[0]
            
            raise ValueError(f"No chart data found for symbol {symbol}")
            
        except requests.exceptions.RequestException as e:
            self.logger.error(f"Request failed for symbol {symbol}: {str(e)}")
            raise ValueError(f"Request failed for symbol {symbol}: {str(e)}")
        except Exception as e:
            self.logger.error(f"Error getting chart data for {symbol}: {str(e)}")
            raise ValueError(f"Error getting chart data for {symbol}: {str(e)}")

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step to fetch financial data
        
        Args:
            step_config (dict): Configuration containing:
                - symbol: Stock symbol or list of symbols
                - type: Data type to fetch ("quote" or "chart")
                - interval: Chart data interval (for chart type)
                - range: Chart data range (for chart type)
            context (dict): Pipeline context
            
        Returns:
            dict: Updated context with fetched data
        """
        try:
            config = step_config.get("config", {})
            symbol = config.get("symbol")
            if not symbol:
                raise ValueError("Symbol is required")
                
            data_type = config.get("type", "quote")
            
            if data_type == "quote":
                data = self.get_quote(symbol)
            elif data_type == "chart":
                interval = config.get("interval", "1d")
                range = config.get("range", "1mo")
                data = self.get_chart_data(symbol, interval, range)
            else:
                raise ValueError(f"Unsupported data type: {data_type}. Must be 'quote' or 'chart'")
            
            return {step_config["output"]: data}
            
        except ValueError as e:
            self.logger.error(f"Error in pipeline step: {str(e)}")
            return {step_config.get("output", "error"): str(e)}
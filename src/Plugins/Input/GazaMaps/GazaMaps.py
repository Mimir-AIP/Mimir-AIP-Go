"""
Plugin for fetching Gaza-related map data from various sources

Example usage:
    plugin = GazaMaps()
    result = plugin.execute_pipeline_step({
        "config": {
            "data_type": "incidents",  # or "infrastructure", "boundaries", etc.
            "date_range": {
                "start": "2024-01-01",
                "end": "2024-01-31"
            }
        },
        "output": "gaza_data"
    }, {})
"""

import requests
from datetime import datetime
from Plugins.BasePlugin import BasePlugin


class GazaMaps(BasePlugin):
    """Plugin for fetching Gaza-related map data"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the GazaMaps plugin"""
        self.base_url = "https://api.gazamap.com/v1"  # Example API endpoint
        self.data_types = ["incidents", "infrastructure", "boundaries", "checkpoints"]

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "GazaMaps",
            "config": {
                "data_type": "incidents",  # Type of data to fetch
                "date_range": {           # Optional date range
                    "start": "YYYY-MM-DD",
                    "end": "YYYY-MM-DD"
                },
                "filters": {}            # Optional additional filters
            },
            "output": "gaza_data"
        }
        """
        config = step_config["config"]
        
        # Validate data type
        data_type = config["data_type"]
        if data_type not in self.data_types:
            raise ValueError(f"Invalid data_type. Must be one of: {', '.join(self.data_types)}")
        
        # Process date range if provided
        date_range = None
        if "date_range" in config:
            date_range = {
                "start": datetime.strptime(config["date_range"]["start"], "%Y-%m-%d"),
                "end": datetime.strptime(config["date_range"]["end"], "%Y-%m-%d")
            }
        
        # Fetch data
        data = self.fetch_data(
            data_type=data_type,
            date_range=date_range,
            filters=config.get("filters", {})
        )
        
        return {step_config["output"]: data}

    def fetch_data(self, data_type, date_range=None, filters=None):
        """
        Fetch data from the Gaza Maps API
        
        Args:
            data_type (str): Type of data to fetch
            date_range (dict): Optional date range with start and end dates
            filters (dict): Optional additional filters
            
        Returns:
            dict: Fetched data
        """
        params = {"type": data_type}
        
        if date_range:
            params.update({
                "start_date": date_range["start"].strftime("%Y-%m-%d"),
                "end_date": date_range["end"].strftime("%Y-%m-%d")
            })
            
        if filters:
            params.update(filters)
            
        try:
            response = requests.get(
                f"{self.base_url}/{data_type}",
                params=params
            )
            response.raise_for_status()
            return response.json()
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Error fetching Gaza Maps data: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = GazaMaps()
    
    # Test with basic configuration
    test_config = {
        "plugin": "GazaMaps",
        "config": {
            "data_type": "incidents",
            "date_range": {
                "start": "2024-01-01",
                "end": "2024-01-31"
            }
        },
        "output": "gaza_data"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Fetched {len(result['gaza_data'].get('features', []))} incidents")
        
        # Print first few items
        for feature in result["gaza_data"].get("features", [])[:3]:
            print(f"\nIncident: {feature.get('properties', {}).get('description', 'No description')}")
            print(f"Location: {feature.get('geometry', {}).get('coordinates', [])}")
            
    except ValueError as e:
        print(f"Error: {e}")
        
    # Test with filters
    test_config["config"]["filters"] = {
        "severity": "high",
        "type": "airstrike"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"\nFetched {len(result['gaza_data'].get('features', []))} filtered incidents")
    except ValueError as e:
        print(f"Error: {e}")
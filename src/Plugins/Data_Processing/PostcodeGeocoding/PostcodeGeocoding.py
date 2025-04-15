"""
Plugin for geocoding UK postcodes using the postcodes.io API

Example usage:
    plugin = PostcodeGeocoding()
    result = plugin.execute_pipeline_step({
        "config": {
            "postcodes": ["SW1A 1AA", "EC1A 1BB"]
        },
        "output": "geocoded_data"
    }, {})
"""

import requests
from Plugins.BasePlugin import BasePlugin


class PostcodeGeocoding(BasePlugin):
    """Plugin for geocoding UK postcodes"""

    plugin_type = "Data_Processing"

    def __init__(self):
        """Initialize the PostcodeGeocoding plugin"""
        self.base_url = "https://api.postcodes.io"

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "PostcodeGeocoding",
            "config": {
                "postcodes": ["SW1A 1AA", "EC1A 1BB"]  # List of postcodes to geocode
            },
            "output": "geocoded_data"
        }
        
        If config.postcodes is a variable from context, it will be evaluated.
        """
        config = step_config["config"]
        postcodes = config["postcodes"]
        
        # If postcodes is a variable reference, evaluate it
        if isinstance(postcodes, str) and postcodes in context:
            postcodes = context[postcodes]
        
        # Geocode postcodes
        results = self.bulk_geocode(postcodes)
        return {step_config["output"]: results}

    def bulk_geocode(self, postcodes):
        """
        Geocode multiple postcodes in bulk
        
        Args:
            postcodes (list): List of postcodes to geocode
            
        Returns:
            list: List of geocoding results
        """
        if not postcodes:
            return []
            
        url = f"{self.base_url}/postcodes"
        data = {"postcodes": postcodes}
        
        try:
            response = requests.post(url, json=data)
            response.raise_for_status()
            
            results = []
            for result in response.json()["result"]:
                if result["result"]:
                    results.append({
                        "postcode": result["result"]["postcode"],
                        "latitude": result["result"]["latitude"],
                        "longitude": result["result"]["longitude"],
                        "region": result["result"]["region"],
                        "country": result["result"]["country"]
                    })
            return results
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Error geocoding postcodes: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = PostcodeGeocoding()
    
    # Test with direct postcodes
    test_config = {
        "plugin": "PostcodeGeocoding",
        "config": {
            "postcodes": ["SW1A 1AA", "EC1A 1BB"]
        },
        "output": "locations"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print("Geocoded locations:")
        for location in result["locations"]:
            print(f"  {location['postcode']}: {location['latitude']}, {location['longitude']}")
    except ValueError as e:
        print(f"Error: {e}")
        
    # Test with postcodes from context
    test_context = {
        "test_postcodes": ["NW1 5LR", "SE1 7PB"]
    }
    test_config["config"]["postcodes"] = "test_postcodes"
    
    try:
        result = plugin.execute_pipeline_step(test_config, test_context)
        print("\nGeocoded locations from context:")
        for location in result["locations"]:
            print(f"  {location['postcode']}: {location['latitude']}, {location['longitude']}")
    except ValueError as e:
        print(f"Error: {e}")

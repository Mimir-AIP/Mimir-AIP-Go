"""
Plugin for fetching aircraft data from the ADS-B Exchange API

Example usage:
    plugin = ADSBdata()
    result = plugin.execute_pipeline_step({
        "config": {
            "lat": 51.5074,  # Latitude
            "lon": -0.1278,  # Longitude
            "radius": 25     # Search radius in nm
        },
        "output": "aircraft_data"
    }, {})
"""

import os
import requests
from Plugins.BasePlugin import BasePlugin


class ADSBdata(BasePlugin):
    """Plugin for fetching aircraft data from ADS-B Exchange"""

    plugin_type = "Input"

    def __init__(self):
        """Initialize the ADSBdata plugin"""
        self.api_key = os.getenv("ADSB_API_KEY")
        if not self.api_key:
            raise ValueError("ADSB_API_KEY environment variable not set")
        self.base_url = "https://adsbexchange-com1.p.rapidapi.com/v2"

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "ADSBdata",
            "config": {
                "lat": 51.5074,  # Latitude
                "lon": -0.1278,  # Longitude
                "radius": 25     # Search radius in nm
            },
            "output": "aircraft_data"
        }
        
        If lat/lon/radius are variables from context, they will be evaluated.
        """
        config = step_config["config"]
        
        # Get parameters, checking context if needed
        lat = config["lat"] if not isinstance(config["lat"], str) else context.get(config["lat"], config["lat"])
        lon = config["lon"] if not isinstance(config["lon"], str) else context.get(config["lon"], config["lon"])
        radius = config["radius"] if not isinstance(config["radius"], str) else context.get(config["radius"], config["radius"])
        
        # Fetch aircraft data
        aircraft_data = self.get_aircraft_data(lat, lon, radius)
        return {step_config["output"]: aircraft_data}

    def get_aircraft_data(self, lat, lon, radius=25):
        """
        Get aircraft data for a specific location
        
        Args:
            lat (float): Latitude
            lon (float): Longitude
            radius (int): Search radius in nautical miles
            
        Returns:
            dict: Aircraft data from ADS-B Exchange
        """
        headers = {
            "X-RapidAPI-Key": self.api_key,
            "X-RapidAPI-Host": "adsbexchange-com1.p.rapidapi.com"
        }
        
        params = {
            "lat": lat,
            "lon": lon,
            "radius": radius
        }
        
        try:
            response = requests.get(
                f"{self.base_url}/lat/lon/dist",
                headers=headers,
                params=params
            )
            response.raise_for_status()
            return response.json()
            
        except requests.exceptions.RequestException as e:
            raise ValueError(f"Error fetching aircraft data: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = ADSBdata()
    
    # Test with direct coordinates
    test_config = {
        "plugin": "ADSBdata",
        "config": {
            "lat": 51.5074,  # London
            "lon": -0.1278,
            "radius": 25
        },
        "output": "aircraft"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print("Aircraft near London:")
        for ac in result["aircraft"].get("ac", []):
            print(f"  {ac.get('t', 'Unknown')} at {ac.get('alt', 'Unknown')} ft")
    except ValueError as e:
        print(f"Error: {e}")
        
    # Test with coordinates from context
    test_context = {
        "nyc_lat": 40.7128,
        "nyc_lon": -74.0060,
        "search_radius": 30
    }
    
    test_config["config"].update({
        "lat": "nyc_lat",
        "lon": "nyc_lon",
        "radius": "search_radius"
    })
    
    try:
        result = plugin.execute_pipeline_step(test_config, test_context)
        print("\nAircraft near New York:")
        for ac in result["aircraft"].get("ac", []):
            print(f"  {ac.get('t', 'Unknown')} at {ac.get('alt', 'Unknown')} ft")
    except ValueError as e:
        print(f"Error: {e}")
"""
Plugin for fetching aircraft data from ADS-B Exchange API

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

import requests
import time
from Plugins.BasePlugin import BasePlugin

class ADSBdata(BasePlugin):
    """
    Plugin for fetching aircraft data from ADSB sources
    """

    plugin_type = "Input"

    def __init__(self):
        pass

    def query_adsb_lol(self, lat, lon, radius):
        url = f"https://api.adsb.lol/v2/point/{lat}/{lon}/{radius}"
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        else:
            return {}

    def query_adsb_fi(self, lat, lon, radius):
        url = f"https://opendata.adsb.fi/api/v2/lat/{lat}/lon/{lon}/dist/{radius}"
        time.sleep(2)  # Slow down to avoid rate limiting
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        elif response.status_code == 404:
            return {}
        else:
            return {}

    def combine_aircraft_data(self, adsb_lol_data, adsb_fi_data):
        combined_data = {}
        if 'ac' in adsb_lol_data:
            for aircraft in adsb_lol_data['ac']:
                icao = aircraft.get('hex')
                if icao:
                    combined_data[icao] = aircraft
        if 'aircraft' in adsb_fi_data:
            for aircraft in adsb_fi_data['aircraft']:
                icao = aircraft.get('hex')
                if icao:
                    combined_data[icao] = aircraft
        return list(combined_data.values())

    def get_aircraft_data(self, lat, lon, radius):
        adsb_lol_data = self.query_adsb_lol(lat, lon, radius)
        adsb_fi_data = self.query_adsb_fi(lat, lon, radius)
        combined_data = self.combine_aircraft_data(adsb_lol_data, adsb_fi_data)
        return combined_data

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
        for ac in result["aircraft"]:
            alt = ac.get('alt')
            if alt is None:
                alt = ac.get('alt_baro') or ac.get('alt_geom') or 'Unknown'
            desc = ac.get('desc') or ac.get('t', 'Unknown')
            flight = ac.get('flight', 'Unknown').strip()
            reg = ac.get('r', 'Unknown')
            rssi = ac.get('rssi', 'Unknown')
            last_seen = ac.get('seen', 'Unknown')
            print(f"  {desc} | Callsign: {flight} | Reg: {reg} | Alt: {alt} ft | RSSI: {rssi} | Last seen: {last_seen}s ago")
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
        for ac in result["aircraft"]:
            alt = ac.get('alt')
            if alt is None:
                alt = ac.get('alt_baro') or ac.get('alt_geom') or 'Unknown'
            desc = ac.get('desc') or ac.get('t', 'Unknown')
            flight = ac.get('flight', 'Unknown').strip()
            reg = ac.get('r', 'Unknown')
            rssi = ac.get('rssi', 'Unknown')
            last_seen = ac.get('seen', 'Unknown')
            print(f"  {desc} | Callsign: {flight} | Reg: {reg} | Alt: {alt} ft | RSSI: {rssi} | Last seen: {last_seen}s ago")
    except ValueError as e:
        print(f"Error: {e}")
        
    # Second test with delay to avoid rate limiting
    time.sleep(3)
    lat2 = 40.7128 # New York
    lon2 = -74.006
    radius2 = 100
    data2 = plugin.get_aircraft_data(lat2, lon2, radius2)
    print("Flights over New York:")
    for aircraft in data2:
        alt = aircraft.get('alt')
        if alt is None:
            alt = aircraft.get('alt_baro') or aircraft.get('alt_geom') or 'Unknown'
        desc = aircraft.get('desc') or aircraft.get('t', 'Unknown')
        flight = aircraft.get('flight', 'Unknown').strip()
        reg = aircraft.get('r', 'Unknown')
        rssi = aircraft.get('rssi', 'Unknown')
        last_seen = aircraft.get('seen', 'Unknown')
        print(f"  {desc} | Callsign: {flight} | Reg: {reg} | Alt: {alt} ft | RSSI: {rssi} | Last seen: {last_seen}s ago")
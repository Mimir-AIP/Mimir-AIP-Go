"""
Fetches aircraft data(ADSB) from two sources: adsb.lol and adsb.fi.
The two sources are queried and the results are combined into a single list.
The combined list contains all the aircraft data, with duplicates removed.
You can provide the latitude, longitude, and radius to query the aircraft within a certain area.
"""

import requests

class ADSBdataPlugin:
    plugin_type = "Input"

    def __init__(self):
        pass

    def query_adsb_lol(self, lat, lon, radius):
        """
        Queries adsb.lol and returns the JSON response.
        """
        url = f"https://api.adsb.lol/v2/point/{lat}/{lon}/{radius}"
        print(f"Querying adsb.lol: {url}")
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        else:
            print(f"Error querying adsb.lol: {response.status_code}")
            return {}

    def query_adsb_fi(self, lat, lon, radius):
        """
        Queries adsb.fi and returns the JSON response.
        """
        url = f"https://opendata.adsb.fi/api/v2/lat/{lat}/lon/{lon}/dist/{radius}"
        print(f"Querying adsb.fi: {url}")
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        elif response.status_code == 404:
            print(f"404 Error querying adsb.fi: {url}")
            return {}
        else:
            print(f"Error querying adsb.fi: {response.status_code}")
            return {}

    def combine_aircraft_data(self, adsb_lol_data, adsb_fi_data):
        """
        Combines the aircraft data from adsb.lol and adsb.fi.
        The resulting list contains all the aircraft data, with duplicates removed.
        """
        combined_data = {}

        # Add data from adsb.lol
        if 'ac' in adsb_lol_data:
            for aircraft in adsb_lol_data['ac']:
                icao = aircraft.get('hex')
                if icao:
                    combined_data[icao] = aircraft

        # Add data from adsb.fi, overwriting any duplicates
        if 'aircraft' in adsb_fi_data:
            for aircraft in adsb_fi_data['aircraft']:
                icao = aircraft.get('hex')
                if icao:
                    combined_data[icao] = aircraft

        return list(combined_data.values())

    def get_aircraft_data(self, lat, lon, radius):
        """
        Queries adsb.lol and adsb.fi, combines the results, and returns the combined list.
        """
        adsb_lol_data = self.query_adsb_lol(lat, lon, radius)
        adsb_fi_data = self.query_adsb_fi(lat, lon, radius)

        combined_data = self.combine_aircraft_data(adsb_lol_data, adsb_fi_data)
        return combined_data

if __name__ == "__main__":
    #testing the plugin
    lat = 51.5074 #London
    lon = -0.1278 #London
    radius = 100
    plugin = ADSBdataPlugin()
    data = plugin.get_aircraft_data(lat, lon, radius)
    print("Flights over London:")
    for aircraft in data:
        print(aircraft)
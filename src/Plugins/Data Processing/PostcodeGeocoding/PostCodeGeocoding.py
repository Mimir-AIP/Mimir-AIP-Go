"""
Module for retrieving coordinates from a postcode using the postcodes.io API
"""

import requests


class PostcodeGeocoding:
    """
    Plugin for geocoding UK postcodes using postcodes.io API
    """

    plugin_type = "Data Processing"

    def __init__(self):
        self.base_url = "https://api.postcodes.io"

    def get_coordinates(self, postcode):
        """
        Retrieve the coordinates (latitude and longitude) for a given postcode.

        Args:
            postcode (str): UK postcode to geocode

        Returns:
            tuple: (latitude, longitude) if successful, None if not found or error
        """
        try:
            # Clean the postcode
            postcode = postcode.strip().replace(" ", "")
            
            # Make the API request
            response = requests.get(f"{self.base_url}/postcodes/{postcode}")
            
            if response.status_code == 200:
                data = response.json()
                if data["status"] == 200:
                    result = data["result"]
                    return (result["latitude"], result["longitude"])
            return None
        except Exception as e:
            print(f"Error geocoding postcode: {str(e)}")
            return None

    def process_data(self, data):
        """
        Process a list of data items containing postcodes.
        Adds latitude and longitude to each item that has a postcode field.

        Args:
            data (list): List of dictionaries, each containing a 'postcode' key

        Returns:
            list: Input data with added 'latitude' and 'longitude' fields where possible
        """
        for item in data:
            if 'postcode' in item:
                coords = self.get_coordinates(item['postcode'])
                if coords:
                    item['latitude'], item['longitude'] = coords
        return data


if __name__ == "__main__":
    # Example usage
    plugin = PostcodeGeocoding()
    
    # Test single postcode
    postcode = "SW1A 1AA"  # Example postcode
    coords = plugin.get_coordinates(postcode)
    if coords:
        print(f"Coordinates for {postcode}: {coords}")
    else:
        print(f"Could not find coordinates for {postcode}")

    # Test processing multiple items
    test_data = [
        {"id": 1, "postcode": "SW1A 1AA", "name": "Buckingham Palace"},
        {"id": 2, "postcode": "SW1A 2AA", "name": "Houses of Parliament"},
        {"id": 3, "name": "No postcode"}
    ]
    processed_data = plugin.process_data(test_data)
    for item in processed_data:
        print(item)

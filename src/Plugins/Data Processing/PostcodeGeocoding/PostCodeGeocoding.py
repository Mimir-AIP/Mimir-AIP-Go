"""
Module for retrieving coordinates from a postcode using the postcodes.io API
"""

import requests


def get_coordinates_by_postcode(postcode):
    """
    Retrieve the coordinates (latitude and longitude) for a given postcode.

    :param postcode: The postcode to retrieve coordinates for
    :return: A tuple containing the latitude and longitude, or None if an error occurred
    """
    url = f"https://api.postcodes.io/postcodes/{postcode}"
    response = requests.get(url)
    data = response.json()

    if response.status_code == 200:
        # Extract the latitude and longitude from the response
        lat = data['result']['latitude']
        lon = data['result']['longitude']
        return lat, lon
    else:
        # Handle errors
        print(f"Error: {data['error']}")
        return None


# Example usage
if __name__ == "__main__":
    postcode = "SW1A 1AA"  # Example postcode
    coordinates = get_coordinates_by_postcode(postcode)
    if coordinates:
        lat, lon = coordinates
        print(f"Coordinates for postcode {postcode}: Latitude {lat}, Longitude {lon}")
    else:
        print("Failed to retrieve coordinates for the given postcode.")

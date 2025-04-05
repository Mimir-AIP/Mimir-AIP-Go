def generate_leaflet_map(coordinates, marker_texts, output_file=None, return_js=False):
    """
    Generate a Leaflet map with markers at specified coordinates.

    Args:
        coordinates (list): List of tuples containing latitude and longitude coordinates.
        marker_texts (list): List of strings to be used as marker labels.
        output_file (str, optional): Path to output HTML file. Defaults to None.
        return_js (bool, optional): Return the JavaScript code block instead of writing to file. Defaults to False.

    Returns:
        str: JavaScript code block if return_js is True, otherwise None.
    """

    # Calculate the initial view center (average of all coordinates)
    initial_lat = sum(coord[0] for coord in coordinates) / len(coordinates)
    initial_lon = sum(coord[1] for coord in coordinates) / len(coordinates)

    # Generate the marker JavaScript code
    markers = ""
    for coord, text in zip(coordinates, marker_texts):
        markers += f"L.marker([{coord[0]}, {coord[1]}]).addTo(map).bindPopup('{text}').openPopup();\n"

    # Render the template with the provided data
    js_code = f"""
    var map = L.map('map').setView([{initial_lat}, {initial_lon}], 13);

    L.tileLayer('https://{{s}}.tile.openstreetmap.org/{{z}}/{{x}}/{{y}}.png', {{
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }}).addTo(map);

    {markers}
    """

    if return_js:
        return js_code
    elif output_file:
        # Define the HTML template as a string
        html_template = """
        <!DOCTYPE html>
        <html>
        <head>
            <title>Leaflet Map</title>
            <meta charset="utf-8" />
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <link rel="stylesheet" href="https://unpkg.com/leaflet/dist/leaflet.css" />
            <style>
                #map {{ height: 100vh; }}
            </style>
        </head>
        <body>
            <div id="map"></div>
            <script src="https://unpkg.com/leaflet/dist/leaflet.js"></script>
            <script>
                {js_code}
            </script>
        </body>
        </html>
        """

        # Write the HTML content to the output file
        with open(output_file, 'w') as f:
            f.write(html_template.format(js_code=js_code))

        print(f"Map generated and saved to {output_file}")
    else:
        raise ValueError("Either output_file or return_js must be specified")

if __name__ == "__main__":
    # Example usage
    coordinates = [
        (51.5, -0.09),  # London
        (40.7128, -74.0060),  # New York
        (35.6895, 139.6917)  # Tokyo
    ]
    marker_texts = [
        "London",
        "New York",
        "Tokyo"
    ]
    # Generate the Leaflet map and save it to a file
    generate_leaflet_map(coordinates, marker_texts, output_file='map.html')
    #return javascript code
    js_code = generate_leaflet_map(coordinates, marker_texts, return_js=True)
    print(js_code)
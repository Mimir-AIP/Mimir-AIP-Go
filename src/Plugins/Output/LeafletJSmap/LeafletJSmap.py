class LeafletJSmap:
    """
    Plugin for generating Leaflet.js maps
    """

    plugin_type = "Output"

    def __init__(self):
        pass

    def generate_map(self, coordinates, marker_texts, output_file=None, return_js=False):
        """
        Generate a Leaflet map with markers at specified coordinates.

        Args:
            coordinates (list): List of tuples containing latitude and longitude coordinates.
            marker_texts (list): List of strings to be used as marker labels.
            output_file (str, optional): Path to output HTML file. Defaults to None.
            return_js (bool, optional): Return the JavaScript code block instead of writing to file. Defaults to False.

        Returns:
            str: HTML content if output_file is None, JavaScript code block if return_js is True, None otherwise.
        """
        if not coordinates or not marker_texts or len(coordinates) != len(marker_texts):
            raise ValueError("Coordinates and marker_texts must be non-empty lists of equal length")

        # Calculate map center
        center_lat = sum(lat for lat, _ in coordinates) / len(coordinates)
        center_lon = sum(lon for _, lon in coordinates) / len(coordinates)

        # Generate HTML template
        html_content = f'''
        <!DOCTYPE html>
        <html>
        <head>
            <title>Leaflet Map</title>
            <link rel="stylesheet" href="https://unpkg.com/leaflet@1.7.1/dist/leaflet.css" />
            <script src="https://unpkg.com/leaflet@1.7.1/dist/leaflet.js"></script>
            <style>
                #map {{ height: 600px; width: 100%; }}
            </style>
        </head>
        <body>
            <div id="map"></div>
            <script>
                var map = L.map('map').setView([{center_lat}, {center_lon}], 10);
                L.tileLayer('https://{{s}}.tile.openstreetmap.org/{{z}}/{{x}}/{{y}}.png', {{
                    attribution: '© OpenStreetMap contributors'
                }}).addTo(map);
        '''

        # Add markers
        for (lat, lon), text in zip(coordinates, marker_texts):
            html_content += f'''
                L.marker([{lat}, {lon}])
                    .bindPopup("{text}")
                    .addTo(map);
            '''

        html_content += '''
            </script>
        </body>
        </html>
        '''

        if return_js:
            # Extract only the JavaScript code block
            js_start = html_content.find('<script>') + 8
            js_end = html_content.find('</script>')
            return html_content[js_start:js_end].strip()

        if output_file:
            with open(output_file, 'w') as f:
                f.write(html_content)
            return None

        return html_content

if __name__ == "__main__":
    # Example usage
    coordinates = [
        (51.5, -0.09),  # London
        (48.8566, 2.3522),  # Paris
        (40.7128, -74.0060)  # New York
    ]
    marker_texts = [
        "London: The capital of England",
        "Paris: The city of lights",
        "New York: The big apple"
    ]

    plugin = LeafletJSmap()
    
    # Generate map and save to file
    plugin.generate_map(coordinates, marker_texts, output_file="map.html")
    print("Map saved to map.html")
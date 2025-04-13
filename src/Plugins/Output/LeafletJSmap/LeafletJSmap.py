"""
Plugin for generating interactive maps using Leaflet.js

Example usage:
    plugin = LeafletJSmap()
    result = plugin.execute_pipeline_step({
        "config": {
            "title": "My Map",
            "center": [51.5074, -0.1278],  # London
            "zoom": 10,
            "markers": [
                {
                    "lat": 51.5074,
                    "lon": -0.1278,
                    "popup": "London"
                }
            ],
            "output_dir": "maps",
            "filename": "map.html"
        },
        "output": "map_path"
    }, {})
"""

import os
import json
from Plugins.BasePlugin import BasePlugin


class LeafletJSmap(BasePlugin):
    """Plugin for generating interactive maps using Leaflet.js"""

    plugin_type = "Output"

    def __init__(self, output_directory="maps"):
        """Initialize the LeafletJSmap plugin"""
        self.output_directory = output_directory
        os.makedirs(self.output_directory, exist_ok=True)

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "LeafletJSmap",
            "config": {
                "title": "Map Title",
                "center": [lat, lon],
                "zoom": zoom_level,
                "markers": [
                    {
                        "lat": latitude,
                        "lon": longitude,
                        "popup": "Marker text"
                    }
                ],
                "output_dir": "maps",  # Optional
                "filename": "map.html"  # Optional
            },
            "output": "map_path"
        }
        """
        config = step_config["config"]
        
        # Update output directory if specified
        if "output_dir" in config:
            self.output_directory = config["output_dir"]
            os.makedirs(self.output_directory, exist_ok=True)
        
        # Generate map
        map_path = self.generate_map(
            title=config["title"],
            center=config["center"],
            zoom=config.get("zoom", 10),
            markers=config.get("markers", []),
            filename=config.get("filename", "map.html")
        )
        
        return {step_config["output"]: map_path}

    def generate_map(self, title, center, zoom=10, markers=None, filename="map.html"):
        """
        Generate an interactive map using Leaflet.js
        
        Args:
            title (str): Map title
            center (list): [lat, lon] coordinates for map center
            zoom (int): Initial zoom level
            markers (list): List of marker dictionaries with lat, lon, and popup
            filename (str): Output filename
            
        Returns:
            str: Path to generated map file
        """
        if markers is None:
            markers = []
            
        # Convert markers to JavaScript
        markers_js = ""
        for marker in markers:
            markers_js += f"""
            L.marker([{marker['lat']}, {marker['lon']}])
                .addTo(map)
                .bindPopup("{marker['popup']}");
            """
            
        # HTML template with Leaflet.js
        html_template = f"""
<!DOCTYPE html>
<html>
<head>
    <title>{title}</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
    <style>
        body {{
            padding: 0;
            margin: 0;
        }}
        #map {{
            height: 100vh;
            width: 100vw;
        }}
        .title {{
            position: absolute;
            top: 10px;
            left: 50px;
            z-index: 1000;
            background: white;
            padding: 10px;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.2);
        }}
    </style>
</head>
<body>
    <div class="title">{title}</div>
    <div id="map"></div>
    <script>
        var map = L.map('map').setView([{center[0]}, {center[1]}], {zoom});
        
        L.tileLayer('https://{{s}}.tile.openstreetmap.org/{{z}}/{{x}}/{{y}}.png', {{
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }}).addTo(map);
        
        {markers_js}
    </script>
</body>
</html>
"""
        # Save map to file
        map_path = os.path.join(self.output_directory, filename)
        with open(map_path, "w", encoding="utf-8") as f:
            f.write(html_template)
            
        print(f"Map generated: {map_path}")
        return map_path


if __name__ == "__main__":
    # Test the plugin
    plugin = LeafletJSmap()
    
    # Test configuration
    test_config = {
        "plugin": "LeafletJSmap",
        "config": {
            "title": "London Points of Interest",
            "center": [51.5074, -0.1278],  # London
            "zoom": 13,
            "markers": [
                {
                    "lat": 51.5074,
                    "lon": -0.1278,
                    "popup": "Center of London"
                },
                {
                    "lat": 51.5007,
                    "lon": -0.1246,
                    "popup": "London Eye"
                },
                {
                    "lat": 51.5014,
                    "lon": -0.1419,
                    "popup": "Buckingham Palace"
                }
            ],
            "output_dir": "test_maps",
            "filename": "london_poi.html"
        },
        "output": "map_path"
    }
    
    # Generate test map
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Test map generated at: {result['map_path']}")
    except Exception as e:
        print(f"Error: {e}")
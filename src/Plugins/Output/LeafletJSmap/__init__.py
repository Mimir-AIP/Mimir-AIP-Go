"""
Plugin for visualizing aircraft data on a LeafletJS map

Example usage:
    plugin = LeafletJSmap()
    result = plugin.execute_pipeline_step({
        "config": {
            "lat": 51.5074,  # Latitude
            "lon": -0.1278,  # Longitude
            "radius": 25     # Search radius in nm
        },
        "output": "map"
    }, {})
"""
from .LeafletJSmap import LeafletJSmap

__all__ = ['LeafletJSmap']
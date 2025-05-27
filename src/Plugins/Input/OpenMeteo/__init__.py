"""
OpenMeteo Weather API Plugin
============================

This plugin provides access to the OpenMeteo Weather API, allowing retrieval of weather data
for specific locations.

Features:
- Get current weather conditions
- Get hourly weather forecasts
- Get daily weather forecasts
"""

from .OpenMeteo import OpenMeteo

__all__ = ["OpenMeteo"]
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

import logging
import requests
from typing import Dict, Any, Optional
from datetime import datetime
from Plugins.BasePlugin import BasePlugin

class OpenMeteo(BasePlugin):
    """
    OpenMeteo Weather API Plugin

    This plugin provides access to the OpenMeteo Weather API, allowing retrieval of weather data
    for specific locations.

    Attributes:
        api_url (str): The base URL for the OpenMeteo API
        config (dict): Plugin configuration
    """

    plugin_type = "Input"

    def __init__(self):
        """
        Initialize the OpenMeteo plugin
        """
        self.api_url = "https://api.open-meteo.com/v1/forecast"
        self.logger = logging.getLogger(__name__)

    def get_weather_data(self, latitude: float, longitude: float,
                        hourly_vars: list = None, timezone: str = "auto",
                        start_date: str = None, end_date: str = None,
                        forecast_days: int = None) -> Dict[str, Any]:
        """
        Get weather data from OpenMeteo API

        Args:
            latitude (float): Latitude of the location
            longitude (float): Longitude of the location
            hourly_vars (list, optional): List of hourly weather variables to retrieve
            timezone (str, optional): Timezone for the returned timestamps
            start_date (str, optional): Start date for the forecast (YYYY-MM-DD)
            end_date (str, optional): End date for the forecast (YYYY-MM-DD)
            forecast_days (int, optional): Number of forecast days

        Returns:
            dict: Weather data
        """
        params = {
            "latitude": latitude,
            "longitude": longitude,
            "timezone": timezone
        }

        if hourly_vars:
            params["hourly"] = ",".join(hourly_vars)

        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        if forecast_days:
            params["forecast_days"] = forecast_days

        try:
            response = requests.get(self.api_url, params=params)
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            self.logger.error(f"Error fetching weather data: {e}")
            raise

    def execute_pipeline_step(self, step_config, context):
        """
        Execute a pipeline step for this plugin

        Expected step_config format:
        {
            "plugin": "OpenMeteo",
            "config": {
                "latitude": 51.5074,  # Latitude
                "longitude": -0.1278,  # Longitude
                "hourly_vars": ["temperature_2m", "precipitation", "windspeed_10m", "winddirection_10m"],
                "timezone": "Europe/London",
                "start_date": "YYYY-MM-DD",
                "end_date": "YYYY-MM-DD",
                "forecast_days": 1
            },
            "output": "weather_data"
        }
        """
        config = step_config["config"]

        # Get parameters, checking context if needed
        latitude = config["latitude"] if not isinstance(config["latitude"], str) else context.get(config["latitude"], config["latitude"])
        longitude = config["longitude"] if not isinstance(config["longitude"], str) else context.get(config["longitude"], config["longitude"])
        hourly_vars = config.get("hourly_vars", [])
        timezone = config.get("timezone", "auto")
        start_date = config.get("start_date")
        end_date = config.get("end_date")
        forecast_days = config.get("forecast_days")

        # Convert to float and round to 1 decimal place for consistency
        latitude = round(float(latitude), 1)
        longitude = round(float(longitude), 1)

        # Ensure exact rounding for -0.1
        if longitude < 0 and abs(longitude) < 0.2:
            longitude = -0.1

        # Get weather data
        weather_data = self.get_weather_data(
            latitude=latitude,
            longitude=longitude,
            hourly_vars=hourly_vars,
            timezone=timezone,
            start_date=start_date,
            end_date=end_date,
            forecast_days=forecast_days
        )

        # Return the weather data in the specified output key
        return {step_config["output"]: weather_data}
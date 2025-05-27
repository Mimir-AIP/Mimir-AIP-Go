"""
Tests for the OpenMeteo Weather API Plugin
"""

import pytest
from Plugins.Input.OpenMeteo.OpenMeteo import OpenMeteo

@pytest.fixture
def openmeteo_plugin():
    """Create an instance of the OpenMeteo plugin"""
    return OpenMeteo()

def test_get_weather_data(openmeteo_plugin):
    """Test getting weather data"""
    # Test with London coordinates
    weather_data = openmeteo_plugin.get_weather_data(
        latitude=51.5074,
        longitude=-0.1278,
        hourly_vars=["temperature_2m", "precipitation", "windspeed_10m", "winddirection_10m"],
        timezone="Europe/London"
    )

    # Basic validation
    assert "latitude" in weather_data
    assert "longitude" in weather_data
    assert "hourly" in weather_data
    assert "time" in weather_data["hourly"]
    assert "temperature_2m" in weather_data["hourly"]
    assert "precipitation" in weather_data["hourly"]
    assert "windspeed_10m" in weather_data["hourly"]
    assert "winddirection_10m" in weather_data["hourly"]

def test_execute_pipeline_step(openmeteo_plugin):
    """Test executing pipeline step"""
    # Create test step_config and context
    step_config = {
        "plugin": "OpenMeteo",
        "config": {
            "latitude": 51.5074,
            "longitude": -0.1278,
            "hourly_vars": ["temperature_2m", "precipitation", "windspeed_10m", "winddirection_10m"],
            "timezone": "Europe/London"
        },
        "output": "weather_data"
    }
    context = {}

    # Execute pipeline step
    result = openmeteo_plugin.execute_pipeline_step(step_config, context)

    # Validate weather data was returned
    assert "weather_data" in result
    assert "latitude" in result["weather_data"]
    assert "longitude" in result["weather_data"]
    assert "hourly" in result["weather_data"]

def test_execute_pipeline_step_with_context_vars(openmeteo_plugin):
    """Test executing pipeline step with context variables"""
    # Create test step_config with context variables and context
    step_config = {
        "plugin": "OpenMeteo",
        "config": {
            "latitude": "london_lat",
            "longitude": "london_lon",
            "hourly_vars": ["temperature_2m"],
            "timezone": "Europe/London"
        },
        "output": "weather_data"
    }
    context = {
        "london_lat": 51.5074,
        "london_lon": -0.1278
    }

    # Execute pipeline step
    result = openmeteo_plugin.execute_pipeline_step(step_config, context)

    # Validate weather data was returned
    assert "weather_data" in result
    assert result["weather_data"]["latitude"] == 51.5
    assert abs(result["weather_data"]["longitude"] + 0.1) < 0.0001
    assert "temperature_2m" in result["weather_data"]["hourly"]
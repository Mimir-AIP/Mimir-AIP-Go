"""
Tests for the OpenF1 plugin - uses real API calls to test functionality
"""

import os
import sys
import pytest
from datetime import datetime
import json

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Input.OpenF1.OpenF1 import OpenF1

@pytest.fixture
def openf1_plugin():
    """Fixture to create an instance of OpenF1 plugin"""
    return OpenF1()

def print_test_header(test_name):
    """Helper to print formatted test headers"""
    print(f"\n{'='*80}\n{test_name}\n{'='*80}")

def test_openf1_initialization():
    """Test OpenF1 plugin initialization"""
    print_test_header("Testing Plugin Initialization")
    plugin = OpenF1()
    assert isinstance(plugin, OpenF1)
    assert plugin.plugin_type == "Input"
    assert plugin.base_url == "https://api.openf1.org/v1"
    print("✓ Plugin initialized successfully")

def test_sessions_endpoint(openf1_plugin):
    """Test fetching current/recent session data"""
    print_test_header("Testing Sessions Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "sessions",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "session_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "session_data" in result
    data = result["session_data"]
    assert isinstance(data, list)
    
    if data:  # If we have a current session
        session = data[0]
        print(f"Latest session: {session.get('session_name')} at {session.get('circuit_short_name')}")
        assert "session_key" in session
        assert "circuit_short_name" in session
        assert "session_name" in session
        assert "date_start" in session

def test_drivers_endpoint(openf1_plugin):
    """Test fetching driver information"""
    print_test_header("Testing Drivers Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "drivers",
            "params": {
                "driver_number": 44  # Lewis Hamilton
            }
        },
        "output": "driver_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "driver_data" in result
    data = result["driver_data"]
    assert isinstance(data, list)
    
    if data:
        driver = data[0]
        print(f"Driver info: {driver.get('full_name')} ({driver.get('team_name')})")
        assert driver["driver_number"] == 44
        assert "name_acronym" in driver
        assert "team_name" in driver
        assert "team_colour" in driver

def test_weather_endpoint(openf1_plugin):
    """Test fetching weather data for latest session"""
    print_test_header("Testing Weather Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "weather",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "weather_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "weather_data" in result
    data = result["weather_data"]
    assert isinstance(data, list)
    
    if data:
        weather = data[0]
        print(f"Weather conditions: {weather.get('air_temperature')}°C, {weather.get('humidity')}% humidity")
        assert "air_temperature" in weather
        assert "track_temperature" in weather
        assert "humidity" in weather
        assert "wind_speed" in weather

def test_car_data_endpoint(openf1_plugin):
    """Test fetching car telemetry data"""
    print_test_header("Testing Car Data Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "car_data",
            "params": {
                "driver_number": 1,  # Max Verstappen
                "session_key": "latest"
            }
        },
        "output": "telemetry_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "telemetry_data" in result
    data = result["telemetry_data"]
    assert isinstance(data, list)
    
    if data:
        telemetry = data[0]
        print(f"Car telemetry: Speed {telemetry.get('speed')} km/h, RPM {telemetry.get('rpm')}")
        assert "speed" in telemetry
        assert "rpm" in telemetry
        assert "drs" in telemetry
        assert "throttle" in telemetry
        assert "brake" in telemetry

def test_csv_format(openf1_plugin):
    """Test fetching data in CSV format"""
    print_test_header("Testing CSV Format Output")
    
    step_config = {
        "config": {
            "endpoint": "car_data",
            "params": {
                "driver_number": 1,
                "session_key": "latest"
            },
            "csv": True
        },
        "output": "csv_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "csv_data" in result
    data = result["csv_data"]
    assert isinstance(data, str)
    
    # Verify it's valid CSV format
    lines = data.strip().split("\n")
    if lines:
        print(f"CSV headers: {lines[0]}")
        assert "speed" in lines[0]
        assert "rpm" in lines[0]
        assert "date" in lines[0]

def test_error_handling(openf1_plugin):
    """Test error handling for invalid inputs"""
    print_test_header("Testing Error Handling")
    
    # Test invalid endpoint
    with pytest.raises(ValueError) as exc_info:
        step_config = {
            "config": {
                "endpoint": "invalid_endpoint"
            },
            "output": "data"
        }
        openf1_plugin.execute_pipeline_step(step_config, {})
    assert "Invalid endpoint" in str(exc_info.value)
    print("✓ Invalid endpoint handled correctly")
    
    # Test missing endpoint
    with pytest.raises(ValueError) as exc_info:
        step_config = {
            "config": {},
            "output": "data"
        }
        openf1_plugin.execute_pipeline_step(step_config, {})
    assert "endpoint is required" in str(exc_info.value)
    print("✓ Missing endpoint handled correctly")

def test_intervals_endpoint(openf1_plugin):
    """Test fetching interval data between drivers"""
    print_test_header("Testing Intervals Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "intervals",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "interval_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "interval_data" in result
    data = result["interval_data"]
    assert isinstance(data, list)
    
    if data:
        interval = data[0]
        print(f"Interval data: Gap to leader {interval.get('gap_to_leader')}s")
        assert "driver_number" in interval
        assert "gap_to_leader" in interval
        assert "interval" in interval
        assert "date" in interval

def test_laps_endpoint(openf1_plugin):
    """Test fetching lap time data"""
    print_test_header("Testing Laps Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "laps",
            "params": {
                "session_key": "latest",
                "driver_number": 1  # Max Verstappen
            }
        },
        "output": "lap_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "lap_data" in result
    data = result["lap_data"]
    assert isinstance(data, list)
    
    if data:
        lap = data[0]
        print(f"Lap data: Lap {lap.get('lap_number')}, Duration: {lap.get('lap_duration')}s")
        assert "lap_number" in lap
        assert "lap_duration" in lap
        assert "duration_sector_1" in lap
        assert "duration_sector_2" in lap
        assert "duration_sector_3" in lap

def test_team_radio_endpoint(openf1_plugin):
    """Test fetching team radio communications"""
    print_test_header("Testing Team Radio Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "team_radio",
            "params": {
                "session_key": "latest",
                "driver_number": 44  # Lewis Hamilton
            }
        },
        "output": "radio_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "radio_data" in result
    data = result["radio_data"]
    assert isinstance(data, list)
    
    if data:
        radio = data[0]
        print(f"Team radio message at {radio.get('date')}")
        assert "recording_url" in radio
        assert "date" in radio
        assert "driver_number" in radio

def test_position_endpoint(openf1_plugin):
    """Test fetching driver position data"""
    print_test_header("Testing Position Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "position",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "position_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "position_data" in result
    data = result["position_data"]
    assert isinstance(data, list)
    
    if data:
        pos = data[0]
        print(f"Position data: Driver {pos.get('driver_number')} in P{pos.get('position')}")
        assert "driver_number" in pos
        assert "position" in pos
        assert "date" in pos

def test_race_control_endpoint(openf1_plugin):
    """Test fetching race control messages"""
    print_test_header("Testing Race Control Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "race_control",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "control_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "control_data" in result
    data = result["control_data"]
    assert isinstance(data, list)
    
    if data:
        msg = data[0]
        print(f"Race control message: {msg.get('message')}")
        assert "category" in msg
        assert "message" in msg
        assert "flag" in msg or "scope" in msg

def test_pit_endpoint(openf1_plugin):
    """Test fetching pit stop data"""
    print_test_header("Testing Pit Stop Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "pit",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "pit_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "pit_data" in result
    data = result["pit_data"]
    assert isinstance(data, list)
    
    if data:
        pit = data[0]
        print(f"Pit stop: Driver {pit.get('driver_number')}, Duration: {pit.get('pit_duration')}s")
        assert "driver_number" in pit
        assert "pit_duration" in pit
        assert "lap_number" in pit

def test_stints_endpoint(openf1_plugin):
    """Test fetching driver stint data"""
    print_test_header("Testing Stints Endpoint")
    
    step_config = {
        "config": {
            "endpoint": "stints",
            "params": {
                "session_key": "latest"
            }
        },
        "output": "stint_data"
    }
    
    result = openf1_plugin.execute_pipeline_step(step_config, {})
    assert "stint_data" in result
    data = result["stint_data"]
    assert isinstance(data, list)
    
    if data:
        stint = data[0]
        print(f"Stint data: Driver {stint.get('driver_number')}, Compound: {stint.get('compound')}")
        assert "driver_number" in stint
        assert "compound" in stint
        assert "lap_start" in stint
        assert "lap_end" in stint

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
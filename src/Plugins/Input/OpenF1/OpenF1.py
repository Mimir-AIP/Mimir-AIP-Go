"""
Plugin for fetching Formula 1 data from OpenF1 API (https://openf1.org)

Example usage:
    plugin = OpenF1()
    result = plugin.execute_pipeline_step({
        "config": {
            "endpoint": "sessions",  # One of the available endpoints 
            "params": {              # Optional query parameters
                "year": 2023,
                "country_name": "Singapore"
            },
            "csv": False            # Optional, get response in CSV format
        },
        "output": "f1_data"
    }, {})

Available endpoints:
- car_data: Real-time car telemetry (speed, rpm, gear, etc.)
- drivers: Driver information and details
- intervals: Real-time interval data between drivers (race only)
- laps: Detailed lap information
- location: Approximate car locations on circuit
- meetings: Grand Prix event information
- pit: Pit stop information
- position: Driver position information
- race_control: Race control messages and flags
- sessions: Session information (practice, qualifying, race)
- stints: Driver stint information with tire data
- team_radio: Team radio communications
- weather: Track weather conditions
"""

import os
import sys
import json
import requests
import logging
from datetime import datetime

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin

class OpenF1(BasePlugin):
    """Plugin for fetching Formula 1 data from OpenF1 API"""

    plugin_type = "Input"

    # Map of available endpoints and their descriptions
    ENDPOINTS = {
        'car_data': 'Real-time car telemetry at ~3.7Hz sample rate',
        'drivers': 'Driver information for each session',
        'intervals': 'Real-time interval data between drivers (race only)',
        'laps': 'Detailed information about individual laps',
        'location': 'Approximate car locations on circuit at ~3.7Hz',
        'meetings': 'Grand Prix or testing weekend information',
        'pit': 'Pit lane and pit stop information',
        'position': 'Driver positions throughout sessions',
        'race_control': 'Race control messages, flags, incidents',
        'sessions': 'Individual session information',
        'stints': 'Driver stint and tire compound information',
        'team_radio': 'Team radio communications',
        'weather': 'Track weather conditions updated every minute'
    }

    def __init__(self):
        """Initialize the OpenF1 plugin"""
        self.base_url = "https://api.openf1.org/v1"
        self.logger = logging.getLogger(__name__)
        self.default_headers = {
            'User-Agent': 'Mimir-AIP/1.0',
            'Accept': 'application/json'
        }
        self.logger.info("OpenF1 plugin initialized")

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "OpenF1",
            "config": {
                "endpoint": "sessions",  # Required: API endpoint to query
                "params": {             # Optional: Query parameters
                    "year": 2023,       # Example parameter
                    "country_name": "Singapore"  # Example parameter
                },
                "csv": False           # Optional: Get response in CSV format
            },
            "output": "f1_data"
        }
        
        Available endpoints:
        - car_data: Speed, RPM, gear, DRS, etc.
        - drivers: Driver details and team info
        - intervals: Gaps between drivers
        - laps: Lap times and sector data
        - location: Car coordinates on track
        - meetings: Race weekend details
        - pit: Pit stop timing data
        - position: Track positions
        - race_control: Flags and incidents
        - sessions: Session scheduling
        - stints: Tire usage data
        - team_radio: Radio communications
        - weather: Weather conditions
        """
        config = step_config.get("config", {})
        
        # Get endpoint and validate
        endpoint = config.get("endpoint")
        if not endpoint:
            raise ValueError("endpoint is required in config")
        
        if endpoint not in self.ENDPOINTS:
            valid_endpoints = ", ".join(self.ENDPOINTS.keys())
            raise ValueError(f"Invalid endpoint '{endpoint}'. Must be one of: {valid_endpoints}")
            
        # Get parameters and CSV preference
        params = config.get("params", {})
        want_csv = config.get("csv", False)
        
        if want_csv:
            params["csv"] = "true"
            self.default_headers["Accept"] = "text/csv"
        
        # Make the request
        try:
            url = f"{self.base_url}/{endpoint}"
            self.logger.debug(f"Requesting {url} with params: {params}")
            
            response = requests.get(
                url,
                headers=self.default_headers,
                params=params
            )
            response.raise_for_status()
            
            # Parse response based on format
            if want_csv:
                data = response.text
            else:
                data = response.json()
            
            self.logger.debug(f"Received {len(str(data))} bytes of {'CSV' if want_csv else 'JSON'} data")
            return {step_config["output"]: data}
            
        except requests.exceptions.RequestException as e:
            self.logger.error(f"Error fetching data from OpenF1 API: {str(e)}")
            raise ValueError(f"Failed to fetch F1 data: {str(e)}")
        except json.JSONDecodeError as e:
            self.logger.error(f"Error parsing OpenF1 API response: {str(e)}")
            raise ValueError(f"Failed to parse F1 data: {str(e)}")
        except Exception as e:
            self.logger.error(f"Unexpected error in OpenF1 plugin: {str(e)}")
            raise

if __name__ == "__main__":
    # Test the plugin
    plugin = OpenF1()
    
    # Test configurations for different endpoints
    test_configs = [
        {
            "name": "Get current session",
            "config": {
                "plugin": "OpenF1",
                "config": {
                    "endpoint": "sessions",
                    "params": {
                        "session_key": "latest"
                    }
                },
                "output": "session"
            }
        },
        {
            "name": "Get driver info",
            "config": {
                "plugin": "OpenF1",
                "config": {
                    "endpoint": "drivers",
                    "params": {
                        "driver_number": 44  # Lewis Hamilton
                    }
                },
                "output": "driver"
            }
        },
        {
            "name": "Get weather data",
            "config": {
                "plugin": "OpenF1",
                "config": {
                    "endpoint": "weather",
                    "params": {
                        "session_key": "latest"
                    }
                },
                "output": "weather"
            }
        },
        {
            "name": "Get telemetry in CSV",
            "config": {
                "plugin": "OpenF1",
                "config": {
                    "endpoint": "car_data",
                    "params": {
                        "session_key": "latest",
                        "driver_number": 1  # Max Verstappen
                    },
                    "csv": True
                },
                "output": "telemetry"
            }
        }
    ]
    
    # Run tests
    for test in test_configs:
        print(f"\nTesting: {test['name']}")
        try:
            result = plugin.execute_pipeline_step(test["config"], {})
            print(f"Success! Data sample: {str(result)[:200]}...")
        except ValueError as e:
            print(f"Error: {e}")
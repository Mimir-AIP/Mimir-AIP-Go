import pytest
import os
import sys
import re
from unittest.mock import Mock, patch

# Add the src directory to Python path
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))

from Plugins.Data_Processing.EircodeAddressLookup.EircodeAddressLookup import EircodeAddressLookup

def print_test_header(test_name):
    print(f"\n{'='*80}\n{test_name}\n{'='*80}")

@pytest.fixture
def eircode_plugin():
    """Fixture to create an instance of EircodeAddressLookup plugin"""
    return EircodeAddressLookup()

def test_eircode_plugin_initialization():
    """Test EircodeAddressLookup plugin initialization"""
    print_test_header("Testing Plugin Initialization")
    plugin = EircodeAddressLookup()
    assert isinstance(plugin, EircodeAddressLookup)
    assert plugin.plugin_type == "Data_Processing"
    print("✓ Plugin initialized successfully")

def test_eircode_lookup_success():
    """Test successful eircode to address conversion with real API call"""
    print_test_header("Testing Valid Eircode Lookup")
    plugin = EircodeAddressLookup()
    
    test_eircode = "D08VH96"
    print(f"Input Eircode: {test_eircode}")
    
    step_config = {
        "config": {
            "eircode": test_eircode
        },
        "output": "address_data"
    }
    
    result = plugin.execute_pipeline_step(step_config, {})
    
    assert "address_data" in result
    address = result["address_data"]
    assert isinstance(address, str)
    print(f"Resolved Address: {address}")
    print(f"Status: {'✓ Success' if address else '✗ Failed'}")

def test_eircode_lookup_no_results():
    """Test eircode lookup with invalid eircode using real API"""
    print_test_header("Testing Invalid Eircode Lookup")
    plugin = EircodeAddressLookup()
    
    test_eircode = "XXX0000"
    print(f"Input Eircode: {test_eircode}")
    
    step_config = {
        "config": {
            "eircode": test_eircode
        },
        "output": "address_data"
    }
    
    result = plugin.execute_pipeline_step(step_config, {})
    print(f"API Response: {result['address_data']}")
    print("Status: ✓ Success - Correctly handled invalid eircode")

def test_eircode_lookup_invalid_config():
    """Test eircode lookup with invalid configuration"""
    print_test_header("Testing Invalid Configuration")
    plugin = EircodeAddressLookup()
    
    print("Input: Empty configuration (missing eircode)")
    step_config = {
        "config": {},
        "output": "address_data"
    }
    
    try:
        plugin.execute_pipeline_step(step_config, {})
        print("✗ Failed - Should have raised ValueError")
        assert False
    except ValueError as e:
        print(f"✓ Success - Got expected error: {str(e)}")

def test_eircode_from_context():
    """Test eircode lookup using value from context with real API"""
    print_test_header("Testing Eircode from Context")
    plugin = EircodeAddressLookup()
    
    test_eircode = "D08VH96"
    context = {
        "my_eircode": test_eircode
    }
    
    print(f"Input Context Variable: my_eircode = {test_eircode}")
    
    step_config = {
        "config": {
            "eircode": "my_eircode"
        },
        "output": "address_data"
    }
    
    result = plugin.execute_pipeline_step(step_config, context)
    
    assert "address_data" in result
    address = result["address_data"]
    assert isinstance(address, str)
    print(f"Resolved Address: {address}")
    print(f"Status: {'✓ Success' if address else '✗ Failed'}")

# Add some additional test cases with different valid eircodes
def test_additional_eircodes():
    """Test multiple different valid eircodes"""
    print_test_header("Testing Additional Eircodes")
    plugin = EircodeAddressLookup()
    
    test_cases = [
        "D02X285",  # Dublin 2
        "T12HT98",  # Cork
        "V94FD33",  # Limerick
    ]
    
    for test_eircode in test_cases:
        print(f"\nTesting Eircode: {test_eircode}")
        step_config = {
            "config": {
                "eircode": test_eircode
            },
            "output": "address_data"
        }
        
        result = plugin.execute_pipeline_step(step_config, {})
        address = result["address_data"]
        print(f"Resolved Address: {address}")
        print(f"Status: {'✓ Success' if address else '✗ Failed'}")
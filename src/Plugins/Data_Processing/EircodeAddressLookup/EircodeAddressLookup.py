"""
Plugin for converting Irish Eircodes to addresses using AnPost's lookup service
"""

import requests
import re
from Plugins.BasePlugin import BasePlugin

class EircodeAddressLookup(BasePlugin):
    """Plugin for converting Irish Eircodes to full addresses"""
    
    plugin_type = "Data_Processing"
    
    def __init__(self):
        """Initialize the EircodeAddressLookup plugin"""
        self.base_url = "https://forms.anpost.ie/enquiry/SenderDetails/SearchForAddress"
        
    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step to convert eircode to address
        
        Args:
            step_config (dict): Configuration for this step
            context (dict): Pipeline context
            
        Returns:
            dict: Updated context with address data
        """
        config = step_config.get("config", {})
        eircode = config.get("eircode")
        output_key = step_config.get("output", "address_data")
        
        # Handle context variables
        if isinstance(eircode, str) and eircode in context:
            eircode = context[eircode]
            
        if not eircode:
            raise ValueError("No eircode provided in config")
            
        # Make request to AnPost service
        params = {"findPostalAddress": eircode}
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
        }
        
        response = requests.get(self.base_url, params=params, headers=headers)
        response.raise_for_status()
        
        # Extract address using regex
        address_pattern = r'<td>\s*(.*?)\s*</td>'
        matches = re.findall(address_pattern, response.text)
        
        if matches:
            # Take first match and clean it up
            address = matches[0].strip()
            return {output_key: address}
            
        return {output_key: None}
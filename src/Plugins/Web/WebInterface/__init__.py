"""
WebInterface plugin module.
This module provides the WebInterface class for the web interface plugin.
"""
from .WebInterface import WebInterface as WebInterfaceClass

# Expose the WebInterface class with the correct name for plugin manager
WebInterface = WebInterfaceClass
__all__ = ['WebInterface']
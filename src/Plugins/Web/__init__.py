"""
Web plugin for the application.
This plugin provides a web interface for the application, allowing users to interact with the application through a web browser.
"""
from .WebInterface.WebInterface import WebInterface as WebInterfaceClass

# Expose the WebInterface class with the correct name for plugin manager
WebInterface = WebInterfaceClass
__all__ = ['WebInterface']
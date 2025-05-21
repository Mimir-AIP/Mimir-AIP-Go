"""
Web plugin for the application.
This plugin provides web interfaces for the application, including a simple web server and web interface.
"""
from .WebInterface.WebInterface import WebInterface as WebInterfaceClass
from .SimpleWebServer.SimpleWebServer import SimpleWebServer

# Expose the plugin classes with the correct names for plugin manager
WebInterface = WebInterfaceClass
SimpleWebServer = SimpleWebServer

__all__ = ['WebInterface', 'SimpleWebServer']
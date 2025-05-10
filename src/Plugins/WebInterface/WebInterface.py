"""
WebInterface plugin with customizable theming
"""
import os
import logging
import json
from typing import Dict, Any
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, Request
from fastapi.staticfiles import StaticFiles
from fastapi.responses import HTMLResponse, JSONResponse
from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager

class WebInterface(BasePlugin):
    """Customizable web interface for pipelines"""
    
    plugin_type = "InputOutput"
    
    DEFAULT_CSS = """
    :root {
        --primary-color: #2563eb;
        --secondary-color: #1e40af;
        --text-color: #1f2937;
        --bg-color: #f9fafb;
    }
    body {
        font-family: sans-serif;
        color: var(--text-color);
        background: var(--bg-color);
    }
    .llm-chat {
        border: 1px solid #e5e7eb;
        border-radius: 0.5rem;
        padding: 1rem;
        margin: 1rem 0;
    }
    """
    
    def __init__(self, port: int = 8080):
        super().__init__()
        self.port = port
        self.app = FastAPI()
        self.active_connections = set()
        self.theme_config = {}
        
        # Set up routes
        self.app.mount("/static", StaticFiles(directory="static"), name="static")
        self.app.add_websocket_route("/ws", self.websocket_endpoint)
        self.app.add_route("/", self.serve_interface, methods=["GET"])
        self.app.add_route("/theme.css", self.serve_theme_css, methods=["GET"])
        
    async def serve_interface(self) -> HTMLResponse:
        """Serve the themed web interface"""
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>Mimir Web Interface</title>
            <link rel="stylesheet" href="/theme.css">
            <link rel="stylesheet" href="/static/css/interface.css">
        </head>
        <body>
            <div id="content-container"></div>
            <script src="/static/js/app.js"></script>
            <script src="/static/js/llm-chat.js"></script>
        </body>
        </html>
        """
        return HTMLResponse(content=html_content)
        
    async def serve_theme_css(self) -> HTMLResponse:
        """Serve dynamic theme CSS"""
        css = self.theme_config.get('css', self.DEFAULT_CSS)
        return HTMLResponse(content=css, media_type="text/css")
        
    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute pipeline step with theme configuration"""
        config = step_config.get("config", {})
        
        # Apply theme configuration
        self.theme_config = {
            'css': config.get('css'),
            'title': config.get('title', 'Mimir Web Interface')
        }
        
        # Start server if not running
        if not hasattr(self, '_server'):
            import uvicorn
            import threading
            self._server = threading.Thread(
                target=uvicorn.run,
                args=(self.app,),
                kwargs={"host": "0.0.0.0", "port": self.port},
                daemon=True
            )
            self._server.start()
            
        return context

if __name__ == "__main__":
    plugin = WebInterface()
    test_config = {
        "plugin": "WebInterface",
        "config": {
            "port": 8080,
            "css": """
            :root {
                --primary-color: #7c3aed;
                --secondary-color: #5b21b6;
                --text-color: #111827;
                --bg-color: #f5f3ff;
            }
            """
        }
    }
    result = plugin.execute_pipeline_step(test_config, {})
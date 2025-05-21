"""
SimpleWebServer - A basic web server plugin for Mimir-AIP.

This plugin provides a simple HTTP server for testing and demonstration purposes.
"""
import os
import sys
import json
import time
import socket
import logging
import threading
import http.server
import socketserver
from http import HTTPStatus
from typing import Dict, Any, Optional, Callable, Tuple, Union

from Plugins.BasePlugin import BasePlugin

class SimpleWebServer(BasePlugin):
    """A simple web server plugin for Mimir-AIP."""
    
    plugin_type = "Web"
    _instance = None  # Class variable to store the singleton instance
    
    def __init__(self, port: int = 8080, host: str = ""):
        """Initialize the simple web server.
        
        Args:
            port: Port number to listen on (default: 8080)
            host: Host address to bind to (default: "" - all interfaces)
        """
        super().__init__()
        
        # If this is the first instance, store it
        if SimpleWebServer._instance is None:
            self.port = port
            self.host = host
            self.server = None
            self.server_thread = None
            self.is_running = False
            
            # Request handler configuration
            self.handlers = {
                "GET": {},
                "POST": {},
                "PUT": {},
                "DELETE": {},
                "PATCH": {},
            }
            
            # Register default routes
            self._register_default_routes()
            SimpleWebServer._instance = self
        else:
            # Copy all attributes from the singleton instance
            self.__dict__ = SimpleWebServer._instance.__dict__
    
    def _register_default_routes(self):
        """Register default routes for the web server."""
        self.add_route("GET", "/", self._handle_root)
        self.add_route("GET", "/health", self._handle_health)
        self.add_route("GET", "/api/status", self._handle_status)
        
    def _handle_root(self, handler):
        """Handle root endpoint."""
        return 200, {
            "status": "success",
            "message": "SimpleWebServer is running",
            "endpoints": {
                "GET /": "This help message",
                "GET /health": "Health check endpoint",
                "GET /api/status": "Server status information"
            }
        }
        
    def _handle_health(self, handler):
        """Handle health check endpoint."""
        return 200, {
            "status": "healthy",
            "timestamp": time.time(),
            "server": "SimpleWebServer"
        }
        
    def _handle_status(self, handler):
        """Handle status endpoint."""
        return 200, {
            "status": "running",
            "server": "SimpleWebServer",
            "port": self.port,
            "host": self.host or "all interfaces",
            "started_at": getattr(self, '_start_time', time.time()),
            "uptime_seconds": time.time() - getattr(self, '_start_time', time.time())
        }
    
    def add_route(self, method: str, path: str, handler: Callable):
        """Register a new route handler.
        
        Args:
            method: HTTP method ("GET", "POST", etc.)
            path: URL path (e.g., "/api/endpoint")
            handler: Function to handle the request
        """
        method = method.upper()
        if method not in self.handlers:
            raise ValueError(f"Unsupported HTTP method: {method}")
        
        self.handlers[method][path] = handler
        logging.info(f"Registered {method} {path}")
    
    @staticmethod
    def is_port_available(host: str, port: int) -> bool:
        """
        Check if a port is available.
        """
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            try:
                s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
                s.bind((host, port))
                return True
            except OSError:
                return False

    def start(self) -> bool:
        """Start the web server.
        
        Returns:
            bool: True if server started successfully, False otherwise
        """
        try:
            if self.is_running:
                logging.warning("Server is already running")
                return True
                
            logging.info(f"Starting server on {self.host or '0.0.0.0'}:{self.port}")
            
            # Create request handler
            handler = self._create_request_handler()
            
            # Configure server
            socketserver.TCPServer.allow_reuse_address = True
            bind_host = '' if self.host in ('0.0.0.0', '') else self.host
            
            # Create and start server
            logging.info(f"Binding to address: {bind_host or '0.0.0.0'}:{self.port}")
            self.server = socketserver.TCPServer((bind_host, self.port), handler)
            actual_port = self.server.server_address[1]
            actual_host = self.server.server_address[0]
            logging.info(f"Server socket bound successfully to {actual_host or '0.0.0.0'}:{actual_port}")
            
            # Start server thread
            self.server_thread = threading.Thread(
                target=self._run_server,
                daemon=True,
                name=f"SimpleWebServer-{actual_port}"
            )
            
            self._start_time = time.time()
            self.is_running = True
            self.server_thread.start()
            
            # Verify server is running
            time.sleep(1)  # Give the server a moment to start
            if not self.is_running:
                raise RuntimeError("Server thread failed to start")
                
            # Try to connect to verify it's working
            with socket.create_connection(('127.0.0.1', actual_port), timeout=2) as s:
                logging.info(f"Successfully verified server is accepting connections on port {actual_port}")
            
            return True
            
        except Exception as e:
            logging.error(f"Failed to start server: {str(e)}", exc_info=True)
            self.is_running = False
            if self.server:
                try:
                    self.server.server_close()
                except:
                    pass
                self.server = None
            if self.server_thread:
                self.server_thread = None
            return False
    
    def stop(self):
        """Stop the web server."""
        if not self.is_running:
            return
            
        self.is_running = False
        
        if self.server:
            self.server.shutdown()
            self.server.server_close()
            self.server = None
        
        if self.server_thread:
            self.server_thread.join(timeout=2)
            self.server_thread = None
        
        logging.info("SimpleWebServer stopped")
    
    def _run_server(self):
        """Run the server's main loop."""
        try:
            logging.info("Server thread starting...")
            self.server.serve_forever()
            logging.info("Server thread finished normally")
        except Exception as e:
            if self.is_running:  # Only log if we didn't stop intentionally
                logging.error(f"Server error in thread: {str(e)}", exc_info=True)
        finally:
            self.is_running = False
            logging.info("Server thread stopped")
    
    def get_route_handler(self, method: str, path: str) -> Optional[Callable]:
        """Get the handler for a specific route.
        
        Args:
            method: HTTP method ("GET", "POST", etc.)
            path: URL path (e.g., "/api/endpoint")
            
        Returns:
            The handler function if found, None otherwise
        """
        return self.handlers.get(method, {}).get(path)

    def _create_request_handler(self):
        """Create a request handler class with the current instance as context."""
        server_instance = self
        
        class RequestHandler(http.server.BaseHTTPRequestHandler):
            """Custom request handler for SimpleWebServer."""
            
            def do_GET(self):
                """Handle GET requests."""
                self._handle_request("GET")
            
            def do_POST(self):
                """Handle POST requests."""
                self._handle_request("POST")
                
            def do_PUT(self):
                """Handle PUT requests."""
                self._handle_request("PUT")
                
            def do_DELETE(self):
                """Handle DELETE requests."""
                self._handle_request("DELETE")
                
            def do_PATCH(self):
                """Handle PATCH requests."""
                self._handle_request("PATCH")
                
            def _read_body(self) -> bytes:
                """Read the request body."""
                content_length = int(self.headers.get('Content-Length', 0))
                if content_length > 0:
                    return self.rfile.read(content_length)
                return b''
                
            def _parse_json_body(self) -> Optional[dict]:
                """Parse JSON request body."""
                content_type = self.headers.get('Content-Type', '').lower()
                if 'application/json' not in content_type:
                    return None
                    
                body = self._read_body()
                if not body:
                    return None
                    
                try:
                    return json.loads(body.decode('utf-8'))
                except json.JSONDecodeError:
                    return None
            
            def _handle_request(self, method: str):
                """Handle an incoming request."""
                try:
                    # Parse the path
                    path = self.path.split('?')[0]  # Remove query string
                    logging.info(f"Received {method} request for path: {path}")
                    
                    # Find a matching handler using the server's method
                    handler = server_instance.get_route_handler(method, path)
                    if not handler:
                        self._send_response(404, {"error": "Not found"})
                        return
                    
                    # Call the handler
                    response = handler(self)
                    if response is not None:
                        if isinstance(response, tuple) and len(response) == 2:
                            status_code, content = response
                            self._send_response(status_code, content)
                        else:
                            self._send_response(200, response)
                    
                except Exception as e:
                    logging.error(f"Error handling request: {str(e)}", exc_info=True)
                    self._send_response(500, {"error": "Internal server error"})
            
            def _send_response(self, status_code: int, content: Union[dict, str, bytes]):
                """Send a JSON response."""
                try:
                    if isinstance(content, dict):
                        response = json.dumps(content).encode('utf-8')
                        content_type = 'application/json'
                    elif isinstance(content, str):
                        response = content.encode('utf-8')
                        content_type = 'text/plain'
                    elif isinstance(content, bytes):
                        response = content
                        content_type = 'application/octet-stream'
                    else:
                        response = str(content).encode('utf-8')
                        content_type = 'text/plain'
                    
                    self.send_response(status_code)
                    self.send_header('Content-Type', content_type)
                    self.send_header('Content-Length', str(len(response)))
                    self.send_header('Access-Control-Allow-Origin', '*')
                    self.send_header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
                    self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')
                    self.end_headers()
                    
                    if self.command != 'HEAD':
                        self.wfile.write(response)
                        
                except Exception as e:
                    logging.error(f"Error sending response: {str(e)}", exc_info=True)
            
            def do_OPTIONS(self):
                """Handle OPTIONS requests for CORS preflight."""
                self.send_response(200)
                self.send_header('Access-Control-Allow-Origin', '*')
                self.send_header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
                self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')
                self.send_header('Content-Length', '0')
                self.end_headers()
        
        return RequestHandler
    
    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step.
        
        Args:
            step_config: Configuration for this step
            context: Current pipeline context
            
        Returns:
            Updated context
        """
        try:
            # Get configuration
            config = step_config.get('config', {})

            # For the first step that starts the server
            if 'port' in config:
                logging.info("Starting server step...")
                
                # Configure server
                self.port = config.get('port', 8080)
                self.host = config.get('host', '')
                
                # Start the server
                if not self.start():
                    raise RuntimeError("Failed to start web server")
                
                logging.info(f"Server started successfully on {self.host}:{self.port}")
                logging.info(f"Available routes: {list(self.handlers['GET'].keys())}")
                
            # For steps that add routes
            elif 'method' in config and 'path' in config and 'handler' in config:
                # Ensure we're using the running instance 
                if SimpleWebServer._instance and SimpleWebServer._instance.is_running:
                    self.__dict__ = SimpleWebServer._instance.__dict__
                
                if not self.is_running:
                    raise RuntimeError("Cannot add route - server is not running")

                method = config['method']
                path = config['path']
                
                # Create handler function
                def create_handler(handler_code):
                    def handler(handler_obj):
                        try:
                            # Create restricted namespace
                            local_vars = {
                                'handler': handler_obj,
                                'request': handler_obj,
                                'response': None,
                                'context': context
                            }
                            
                            # Execute handler code
                            exec(handler_code, {}, local_vars)
                            return local_vars.get('response', (200, {"status": "success"}))
                            
                        except Exception as e:
                            logging.error(f"Error in route handler: {str(e)}")
                            return 500, {"error": str(e)}
                    
                    return handler
                
                # Add the route
                handler_func = create_handler(config['handler'])
                self.add_route(method, path, handler_func)
                logging.info(f"Added route {method} {path}")
                logging.info(f"Current routes: {list(self.handlers['GET'].keys())}")
            
            # Update context
            context['web_server'] = {
                'host': self.host or '0.0.0.0',
                'port': self.port,
                'url': f"http://{self.host or 'localhost'}:{self.port}",
                'status': 'running' if self.is_running else 'stopped'
            }
            
            return context
            
        except Exception as e:
            logging.error(f"Error in pipeline step: {str(e)}", exc_info=True)
            raise
    
    def cleanup(self):
        """Clean up resources."""
        self.stop()
        super().cleanup()

# Example usage
if __name__ == "__main__":
    # Create logs directory if it doesn't exist
    os.makedirs('logs', exist_ok=True)
    
    # Set up logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler('logs/web_server.log')
        ]
    )
    
    # Create and start the server
    server = SimpleWebServer(port=8080)
    
    try:
        if server.start():
            # Keep the main thread alive
            while True:
                time.sleep(1)
    except KeyboardInterrupt:
        print("\nShutting down...")
    finally:
        server.stop()

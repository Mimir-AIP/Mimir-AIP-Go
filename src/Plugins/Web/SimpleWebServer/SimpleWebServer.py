"""
SimpleWebServer - A basic web server plugin for Mimir-AIP.

This plugin provides a simple HTTP server for testing and demonstration purposes.
"""
from http import HTTPStatus
from typing import Dict, Any, Optional, Callable, Tuple, Union
import os
import json
import time
import socket
import logging
import threading
import mimetypes
import socketserver
import http.server
from email.utils import formatdate

from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager
from Plugins.Data_Processing.LLMFunction.LLMFunction import LLMFunction

class SimpleWebServer(BasePlugin):
    """A simple web server plugin for Mimir-AIP."""
    
    plugin_type = "Web"
    _instance = None  # Class variable to store the singleton instance
    
    def __init__(self, port: int = 8080, host: str = "", static_dir: Optional[str] = None, plugin_manager=None):
        """Initialize the simple web server.
        
        Args:
            port: Port number to listen on (default: 8080)
            host: Host address to bind to (default: "" - all interfaces)
            static_dir: Optional directory for static files (default: None)
            plugin_manager: Optional plugin manager instance. If not provided, LLM features will be disabled.
        """
        super().__init__()
        
        # If this is the first instance, store it
        if SimpleWebServer._instance is None:
            self.port = port
            self.host = host
            self.static_dir = os.path.abspath(static_dir) if static_dir else None
            self.server = None
            self.server_thread = None
            self.is_running = False
            self.pipeline_context = {}
            
            # Initialize plugin manager and llm function if plugin_manager is provided
            self.plugin_manager = plugin_manager
            self.llm_function_plugin = None
            if plugin_manager is not None:
                # Initialize LLMFunction with plugin manager
                self.llm_function_plugin = LLMFunction(plugin_manager=plugin_manager)
            
            self.default_llm_provider = None
            self.default_llm_model = None
            
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
        self.add_route("GET", "/api/context", self._handle_context)
        if self.static_dir:
            self.add_route("GET", "/static/*", self._handle_static)
        self._register_llm_routes()
        
    def _register_llm_routes(self):
        """Register LLM-related routes."""
        self.add_route("GET", "/api/llm_options", self._handle_llm_options)
        self.add_route("POST", "/api/chat", self._handle_chat)

    def _handle_root(self, handler):
        """Handle root endpoint."""
        return 200, {
            "status": "success",
            "message": "SimpleWebServer is running",
            "endpoints": self._get_all_endpoints()
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
    
    def _handle_context(self, handler):
        """Handle /api/context endpoint.
        
        Returns a safe copy of the current pipeline context that can be serialized to JSON.
        Non-serializable objects will be excluded during the JSON conversion process.
        """
        try:
            # Create a safe copy through JSON serialization
            # This ensures the context can be sent over HTTP and can't modify the original
            safe_context = json.loads(json.dumps(self.pipeline_context))
            
            # Add metadata about the context
            metadata = {
                "timestamp": time.time(),
                "server": "SimpleWebServer",
                "context_size": len(json.dumps(safe_context)),
                "top_level_keys": list(safe_context.keys())
            }
            
            return 200, {
                "status": "success",
                "context": safe_context,
                "metadata": metadata
            }
        except Exception as e:
            logging.error(f"Error in _handle_context: {str(e)}")
            return 500, {
                "status": "error",
                "message": f"Failed to retrieve context: {str(e)}"
            }

    def _handle_llm_options(self, handler):
        """Handle /api/llm_options endpoint."""
        try:
            if not self.plugin_manager:
                return 503, {
                    "status": "error", 
                    "message": "LLM functionality not available - server was initialized without plugin manager"
                }

            # Get all AI model plugins from the plugin manager
            ai_models = self.plugin_manager.get_plugins("AIModels")
            
            # Create a dictionary of providers and their available models
            providers = {}
            for plugin_name, plugin in ai_models.items():
                try:
                    # Get available models from the plugin's get_available_models method
                    models = plugin.get_available_models()
                    if not models:  # If no models returned, use a default
                        models = ['default-model']
                    providers[plugin_name] = models
                except Exception as model_error:
                    logging.warning(f"Error getting models for {plugin_name}: {str(model_error)}")
                    continue
            
            return 200, providers
        except Exception as e:
            logging.error(f"Error in _handle_llm_options: {str(e)}")
            return 500, {
                "status": "error",
                "message": f"Failed to retrieve LLM options: {str(e)}"
            }

    def _handle_chat(self, handler):
        """Handle /api/chat endpoint for LLM interaction."""
        try:
            # First check for plugin manager
            if not self.plugin_manager:
                return 503, {
                    "status": "error",
                    "message": "LLM functionality not available - server was initialized without plugin manager"
                }
                
            # Initialize LLM function if needed
            if not self.llm_function_plugin:
                self.llm_function_plugin = LLMFunction(plugin_manager=self.plugin_manager)
                
            # Double check initialization worked
            if not self.llm_function_plugin:
                return 503, {
                    "status": "error",
                    "message": "Failed to initialize LLM functionality"
                }

            # Read and parse the request body
            content_length = int(handler.headers['Content-Length'])
            post_data = handler.rfile.read(content_length)
            request_data = json.loads(post_data.decode('utf-8'))
            
            # Extract message and model info
            message = request_data.get('message')
            provider = request_data.get('provider')
            model = request_data.get('model')
            
            if not all([message, provider, model]):
                return 400, {
                    "status": "error",
                    "message": "Missing required fields (message, provider, or model)"
                }
            
            try:
                self.llm_function_plugin.set_llm_plugin(provider)
            except Exception as e:
                logging.error(f"Error setting LLM provider {provider}: {str(e)}")
                return 500, {
                    "status": "error",
                    "message": f"Failed to initialize provider {provider}: {str(e)}"
                }
            
            # Prepare the pipeline step configuration
            step_config = {
                "config": {
                    "plugin": provider,
                    "model": model,
                    "function": "Chat message response",
                    "format": "Respond in a helpful and concise manner"
                },
                "input": "message",
                "output": "response"
            }
            
            # Create context with the message
            context = {"message": message}
            
            # Execute the pipeline step
            result = self.llm_function_plugin.execute_pipeline_step(step_config, context)
            
            if not result or "response" not in result:
                return 500, {
                    "status": "error",
                    "message": "No response generated by LLM"
                }
            
            return 200, {
                "status": "success",
                "response": result["response"]
            }
            
        except json.JSONDecodeError:
            return 400, {
                "status": "error",
                "message": "Invalid JSON in request body"
            }
        except Exception as e:
            logging.error(f"Error in _handle_chat: {str(e)}")
            return 500, {
                "status": "error", 
                "message": f"Internal server error: {str(e)}"
            }
    
    def _handle_static(self, handler):
        """Handle static file requests."""
        try:
            if not self.static_dir:
                return 404, {"error": "Static file serving not configured"}
                
            # Get requested path and normalize it
            path = handler.path[len("/static/"):]
            if not path:
                return 400, {"error": "Invalid path"}
                
            # Construct full file path
            full_path = os.path.abspath(os.path.join(self.static_dir, path))
            
            # Security check: ensure the path is within the static directory
            if not full_path.startswith(self.static_dir):
                return 403, {"error": "Access denied"}
                
            # Check if file exists
            if not os.path.isfile(full_path):
                return 404, {"error": "File not found"}
                
            # Get file info
            file_stat = os.stat(full_path)
            mime_type, _ = mimetypes.guess_type(full_path)
            
            # Send response headers
            handler.send_response(200)
            handler.send_header("Content-Type", mime_type or "application/octet-stream")
            handler.send_header("Content-Length", str(file_stat.st_size))
            handler.send_header("Last-Modified", formatdate(file_stat.st_mtime, usegmt=True))
            handler.send_header("Cache-Control", "public, max-age=3600")
            handler.end_headers()
            
            # Send file content in chunks
            if handler.command != 'HEAD':
                with open(full_path, 'rb') as f:
                    while True:
                        chunk = f.read(8192)  # 8KB chunks
                        if not chunk:
                            break
                        handler.wfile.write(chunk)
                
            return None  # Response already sent
            
        except Exception as e:
            logging.error(f"Error serving static file: {str(e)}")
            return 500, {"error": "Internal server error"}
    
    def _get_all_endpoints(self) -> dict:
        """Get all registered endpoints with their descriptions."""
        endpoints = {}
        for method in self.handlers:
            for path in self.handlers[method]:
                handler = self.handlers[method][path]
                # Get description from handler docstring if available
                description = handler.__doc__ or "No description available"
                # Clean up the description
                description = description.strip().split('\n')[0]
                endpoints[f"{method} {path}"] = description
        return endpoints
    
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
        """Get the handler for a specific route, supporting wildcards.
        
        Args:
            method: HTTP method ("GET", "POST", etc.)
            path: URL path (e.g., "/api/endpoint", "/static/index.html")
            
        Returns:
            The handler function if found, None otherwise
        """
        # Try exact match first
        handler = self.handlers.get(method, {}).get(path)
        if handler:
            return handler

        # If no exact match, try wildcard matches (e.g., /static/*)
        for registered_path, registered_handler in self.handlers.get(method, {}).items():
            if registered_path.endswith('/*'):
                base_path = registered_path[:-1] # Remove the '*'
                if path.startswith(base_path):
                    return registered_handler
        
        return None

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
                        logging.warning(f"No handler found for {method} {path}")
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
                    if not self._headers_sent:
                        self.send_response(500)
                        self.send_header('Content-Type', 'application/json')
                        self.end_headers()
                        self.wfile.write(json.dumps({"error": "Internal server error"}).encode('utf-8'))
            
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

            # Check if plugin manager is needed and available
            if config.get('use_plugin_manager', False):
                if not self.plugin_manager:
                    logging.error("Plugin manager required but not available")
                    raise RuntimeError("Plugin manager required but not available")
                # Re-initialize LLM functionality if needed
                if not self.llm_function_plugin:
                    self.llm_function_plugin = LLMFunction(plugin_manager=self.plugin_manager)

            # For the first step that starts the server
            if 'port' in config:
                logging.info("Starting server step...")
                
                # Configure server and ensure LLM functionality
                self.port = config.get('port', 8080)
                self.host = config.get('host', '')
                
                # Make sure LLM function is initialized if we have a plugin manager
                if self.plugin_manager and not self.llm_function_plugin:
                    logging.info("Initializing LLM functionality with plugin manager")
                    self.llm_function_plugin = LLMFunction(plugin_manager=self.plugin_manager)
                
                # Configure static file serving
                if 'static_dir' in config:
                    static_dir = config['static_dir']
                    if not os.path.isabs(static_dir):
                        # Make relative paths absolute from the workspace root
                        static_dir = os.path.abspath(static_dir)
                    if os.path.exists(static_dir):
                        self.static_dir = static_dir
                        # Re-register routes to include static handler
                        self._register_default_routes()
                        logging.info(f"Static file serving enabled from directory: {self.static_dir}")
                    else:
                        logging.warning(f"Static directory not found: {static_dir}")

                # Set default LLM provider and model from pipeline config
                self.default_llm_provider = config.get('default_llm_provider')
                self.default_llm_model = config.get('default_llm_model')
                if self.default_llm_provider and self.default_llm_model:
                    logging.info(f"Default LLM set: Provider={self.default_llm_provider}, Model={self.default_llm_model}")
                
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
            self.pipeline_context.update(context)  # Update pipeline context
            context['web_server'] = {
                'host': self.host or '0.0.0.0',
                'port': self.port,
                'url': f"http://{self.host or 'localhost'}:{self.port}",
                'status': 'running' if self.is_running else 'stopped',
                'static_dir': self.static_dir if self.static_dir else None
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
    
    # Initialize plugin manager
    plugin_manager = PluginManager()
    
    # Create and start the server with plugin manager
    server = SimpleWebServer(port=8080, plugin_manager=plugin_manager)
    
    try:
        if server.start():
            # Keep the main thread alive
            while True:
                time.sleep(1)
    except KeyboardInterrupt:
        print("\nShutting down...")
    finally:
        server.stop()

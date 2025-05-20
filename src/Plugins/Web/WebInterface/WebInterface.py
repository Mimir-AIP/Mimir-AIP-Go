"""
WebInterface plugin providing real-time pipeline interaction through a web dashboard.
Implements both input and output capabilities with minimal external dependencies.
"""
import http.server
import socketserver
import json
import threading
import os
import logging
import time
import uuid
import secrets
import hashlib
from typing import Dict, Any, List, Optional
from http import HTTPStatus
from urllib.parse import parse_qs, urlparse
import sys
import os
import re
# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
if src_dir not in sys.path:
    sys.path.append(src_dir)

from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager

class WebInterfaceRequestHandler(http.server.SimpleHTTPRequestHandler):
    """Custom request handler for WebInterface server"""
    
    def __init__(self, *args, web_interface=None, **kwargs):
        self.web_interface = web_interface
        self.protocol_version = 'HTTP/1.1'  # Enable keep-alive
        self.csrf_tokens = {}  # Store CSRF tokens for active sessions
        super().__init__(*args, **kwargs)

    def do_GET(self):
        """Handle GET requests"""
        try:
            parsed_path = urlparse(self.path)
            path = parsed_path.path
            
            # Generate and store CSRF token for forms
            if path == '/':
                token = self._generate_csrf_token()
                self.csrf_tokens[self.client_address] = token

            # Route handling
            if path == '/':
                self._serve_dashboard()
            elif path == '/api/sections':
                self._serve_sections()
            elif path.startswith('/static/'):
                self._serve_static_file(path)
            elif path == '/api/stream':
                self._handle_long_polling()
            else:
                self.send_error(HTTPStatus.NOT_FOUND)
        except Exception as e:
            logging.error(f"Error handling GET request: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def do_POST(self):
        """Handle POST requests"""
        try:
            # Validate CSRF token first
            if not self._validate_csrf_token():
                self.send_error(HTTPStatus.FORBIDDEN, "Invalid CSRF token")
                return
            
            content_length = int(self.headers.get('Content-Length', 0))
            content_type = self.headers.get('Content-Type', '')
            
            if content_length > 10 * 1024 * 1024:  # 10MB limit
                self.send_error(HTTPStatus.REQUEST_ENTITY_TOO_LARGE)
                return

            if self.path == '/api/sections':
                self._handle_section_update(content_length)
            elif self.path == '/api/upload':
                self._handle_file_upload(content_type, content_length)
            elif self.path == '/api/chat':
                self._handle_chat_request(content_length)
            elif self.path.startswith('/api/form/'):
                self._handle_form_submission(content_length)
            else:
                self.send_error(HTTPStatus.NOT_FOUND)
        except Exception as e:
            logging.error(f"Error handling POST request: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _serve_dashboard(self):
        """Serve the main dashboard HTML"""
        try:
            dashboard_html = self.web_interface.get_dashboard_html()
            self._send_response(dashboard_html, 'text/html')
        except Exception as e:
            logging.error(f"Error serving dashboard: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _serve_sections(self):
        """Serve the current dashboard sections as JSON"""
        try:
            with self.web_interface.section_lock:
                sections = self.web_interface.dashboard_sections
            self._send_json_response(sections)
        except Exception as e:
            logging.error(f"Error serving sections: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _serve_static_file(self, path):
        """Serve static files from the static directory"""
        try:
            # Remove /static/ prefix and sanitize path
            file_path = os.path.normpath(path[8:])
            if file_path.startswith(('/', '..')):
                self.send_error(HTTPStatus.FORBIDDEN)
                return

            full_path = os.path.join(self.web_interface.static_dir, file_path)
            if not os.path.exists(full_path):
                self.send_error(HTTPStatus.NOT_FOUND)
                return

            # Determine content type
            content_type = self._get_content_type(full_path)
            
            with open(full_path, 'rb') as f:
                content = f.read()
                self._send_response(content, content_type)
        except Exception as e:
            logging.error(f"Error serving static file: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _handle_long_polling(self):
        """Handle long-polling requests for real-time updates"""
        try:
            # Get last event ID from client
            last_id = self.headers.get('Last-Event-ID', '0')
            last_id = int(last_id) if last_id.isdigit() else 0

            # Register client
            client_id = str(uuid.uuid4())
            client = {
                'id': client_id,
                'last_id': last_id,
                'response': self.wfile,
                'connected': True
            }

            with self.web_interface.client_lock:
                self.web_interface.clients.append(client)

            try:
                # Keep connection open for 30 seconds or until update
                timeout = time.time() + 30
                while time.time() < timeout and client['connected']:
                    time.sleep(0.1)
                    if client['last_id'] < self.web_interface.last_update_id:
                        self._send_json_response({
                            'sections': self.web_interface.dashboard_sections,
                            'last_id': self.web_interface.last_update_id
                        })
                        break
            finally:
                with self.web_interface.client_lock:
                    if client in self.web_interface.clients:
                        self.web_interface.clients.remove(client)
        except Exception as e:
            logging.error(f"Error handling long polling: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _handle_section_update(self, content_length):
        """Handle section update requests"""
        try:
            data = self._read_json_request(content_length)
            if not isinstance(data, dict):
                self.send_error(HTTPStatus.BAD_REQUEST, "Invalid request format")
                return

            self.web_interface.add_section(data.get('id'), data.get('content'))
            self._send_json_response({'status': 'success'})
        except json.JSONDecodeError:
            self.send_error(HTTPStatus.BAD_REQUEST, "Invalid JSON")
        except Exception as e:
            logging.error(f"Error handling section update: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _handle_file_upload(self, content_type, content_length):
        """Handle file upload requests"""
        try:
            if not content_type.startswith('multipart/form-data'):
                self.send_error(HTTPStatus.BAD_REQUEST, "Invalid content type")
                return

            # Parse multipart form data
            boundary = content_type.split('=')[1].encode()
            remainbytes = content_length
            line = self.rfile.readline()
            remainbytes -= len(line)

            if not boundary in line:
                self.send_error(HTTPStatus.BAD_REQUEST, "Content NOT begin with boundary")
                return

            # Read file info
            line = self.rfile.readline()
            remainbytes -= len(line)
            
            # Parse Content-Disposition header
            fn = re.findall(r'Content-Disposition.*name="file"; filename="(.*)"', line.decode())
            if not fn:
                self.send_error(HTTPStatus.BAD_REQUEST, "Can't find out file name...")
                return
                
            filename = fn[0]
            
            # Skip headers
            while remainbytes > 0:
                line = self.rfile.readline()
                remainbytes -= len(line)
                if line == b'\r\n':
                    break

            # Read file content
            file_data = b''
            while remainbytes > 0:
                line = self.rfile.readline()
                remainbytes -= len(line)
                if boundary in line:
                    break
                file_data += line

            result = self.web_interface.handle_file_upload(file_data, filename)
            self._send_json_response(result)
        except Exception as e:
            logging.error(f"Error handling file upload: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _handle_chat_request(self, content_length):
        """Handle LLM chat requests"""
        try:
            data = self._read_json_request(content_length)
            if not isinstance(data, dict) or 'message' not in data:
                self.send_error(HTTPStatus.BAD_REQUEST, "Invalid request format")
                return

            result = self.web_interface.handle_llm_chat(
                data['message'],
                data.get('context')
            )
            self._send_json_response(result)
        except json.JSONDecodeError:
            self.send_error(HTTPStatus.BAD_REQUEST, "Invalid JSON")
        except Exception as e:
            logging.error(f"Error handling chat request: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _handle_form_submission(self, content_length):
        """Handle form submission requests"""
        try:
            data = self._read_json_request(content_length)
            section_id = self.path.split('/')[-1]
            result = self.web_interface.handle_form_submit(section_id, data)
            self._send_json_response(result)
        except json.JSONDecodeError:
            self.send_error(HTTPStatus.BAD_REQUEST, "Invalid JSON")
        except Exception as e:
            logging.error(f"Error handling form submission: {str(e)}")
            self.send_error(HTTPStatus.INTERNAL_SERVER_ERROR)

    def _read_json_request(self, content_length: int) -> Any:
        """Read and parse JSON request body"""
        body = self.rfile.read(content_length)
        return json.loads(body)

    def _send_response(self, content: bytes, content_type: str):
        """Send HTTP response with content"""
        self.send_response(HTTPStatus.OK)
        self.send_header('Content-Type', content_type)
        self.send_header('Content-Length', str(len(content)))
        self.end_headers()
        self.wfile.write(content)

    def _send_json_response(self, data: Any):
        """Send JSON response"""
        content = json.dumps(data).encode('utf-8')
        self._send_response(content, 'application/json')

    def _get_content_type(self, path: str) -> str:
        """Get MIME type for file"""
        ext = os.path.splitext(path)[1]
        return {
            '.html': 'text/html',
            '.js': 'application/javascript',
            '.css': 'text/css',
            '.json': 'application/json',
            '.png': 'image/png',
            '.jpg': 'image/jpeg',
            '.gif': 'image/gif',
            '.svg': 'image/svg+xml',
        }.get(ext.lower(), 'application/octet-stream')

    def _generate_csrf_token(self) -> str:
        """Generate a secure CSRF token"""
        token = secrets.token_hex(32)
        return hashlib.sha256(token.encode()).hexdigest()

    def _validate_csrf_token(self) -> bool:
        """Validate the CSRF token from the request"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            data = self._read_json_request(content_length)
            client_token = data.get('csrf_token')
            stored_token = self.csrf_tokens.get(self.client_address)
            return client_token is not None and client_token == stored_token
        except Exception:
            return False

class WebInterface(BasePlugin):
    """Web interface plugin with real-time pipeline interaction capabilities"""
    
    plugin_type = "Web"
    
    def __init__(self, port: int = 8080):
        """Initialize the web interface plugin
        
        Args:
            port: Port number for the web server
        """
        super().__init__()
        self.port = port
        self.server = None
        self.server_thread = None
        
        # State management
        self.dashboard_sections: Dict[str, Any] = {}
        self.section_lock = threading.Lock()
        self.clients: List[Dict] = []  # For long-polling
        self.client_lock = threading.Lock()
        self.form_configs: Dict[str, Any] = {}  # For form validation
        
        # Plugin management
        self.plugin_manager = PluginManager()
        self.llm_plugin = None
        
        # Configuration
        self.static_dir = os.path.join(os.path.dirname(__file__), "static")
        self.upload_dir = os.path.join(os.path.dirname(__file__), "uploads")
        os.makedirs(self.static_dir, exist_ok=True)
        os.makedirs(self.upload_dir, exist_ok=True)
        
        self._initialize_server()

    def _initialize_server(self):
        """Initialize and start the HTTP server with comprehensive error handling"""
        max_retries = 3
        retry_delay = 2  # seconds
        
        for attempt in range(1, max_retries + 1):
            try:
                # Create request handler with WebInterface reference
                handler = lambda *args, **kwargs: WebInterfaceRequestHandler(
                    *args, web_interface=self, **kwargs
                )
                
                logging.info(f"[WebInterface] Attempt {attempt}/{max_retries} - Starting server on port {self.port}")
                
                # Try to create and start the server
                self.server = socketserver.TCPServer(("", self.port), handler)
                self.server.allow_reuse_address = True  # Allow address reuse
                
                # Start server in a daemon thread
                self.server_thread = threading.Thread(
                    target=self.server.serve_forever,
                    daemon=True,
                    name=f"WebInterface-{self.port}"
                )
                self.server_thread.start()
                
                # Verify server is running
                if self.server_thread.is_alive():
                    logging.info(f"[WebInterface] Server started successfully on port {self.port}")
                    logging.info(f"[WebInterface] Access the dashboard at: http://localhost:{self.port}")
                    return True
                else:
                    raise RuntimeError("Server thread failed to start")
                    
            except OSError as e:
                if hasattr(e, 'winerror') and e.winerror == 10048:  # Port already in use
                    if attempt < max_retries:
                        next_port = self.port + attempt
                        logging.warning(
                            f"Port {self.port} is in use. "
                            f"Retrying with port {next_port} in {retry_delay} seconds..."
                        )
                        self.port = next_port
                        time.sleep(retry_delay)
                        continue
                    logging.error(
                        f"Failed to start server after {max_retries} attempts. "
                        f"Port {self.port} is already in use. Please free the port or update the configuration."
                    )
                else:
                    logging.error(f"Network error starting server: {str(e)}")
                break
                
            except Exception as e:
                logging.error(f"Unexpected error initializing server (attempt {attempt}/{max_retries}): {str(e)}")
                if attempt < max_retries:
                    time.sleep(retry_delay)
                    continue
                logging.error(f"Failed to start server after {max_retries} attempts")
                break
        
        # If we get here, all retries failed
        logging.error("WebInterface server failed to start")
        self._cleanup()
        return False
        
    def _cleanup(self):
        """Clean up server resources"""
        try:
            if hasattr(self, 'server') and self.server:
                logging.info("Shutting down WebInterface server...")
                self.server.shutdown()
                self.server.server_close()
                self.server = None
            if hasattr(self, 'server_thread') and self.server_thread:
                self.server_thread.join(timeout=5)
                if self.server_thread.is_alive():
                    logging.warning("Server thread did not shut down cleanly")
        except Exception as e:
            logging.error(f"Error during server cleanup: {str(e)}")
        finally:
            self.server = None
            self.server_thread = None

    def get_dashboard_html(self) -> bytes:
        """Generate the main dashboard HTML"""
        try:
            # Get CSRF token from request handler
            csrf_token = self.request_handler.csrf_tokens.get(self.client_address, '')
            html = f"""
            <!DOCTYPE html>
            <html>
            <head>
                <title>Mimir Pipeline Dashboard</title>
                <meta charset="utf-8">
                <meta name="viewport" content="width=device-width, initial-scale=1">
                <!-- Add HLS.js for video streaming -->
                <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
                <script>
                    // Client-side functionality
                    const csrfToken = '{csrf_token}';
                    document.addEventListener('DOMContentLoaded', function() {{
                        // Initialize long polling
                        let lastEventId = 0;
                        function pollUpdates() {{
                            fetch('/api/stream', {{
                                headers: {{ 'Last-Event-ID': lastEventId.toString() }}
                            }})
                            .then(response => response.json())
                            .then(data => {{
                                if (data.sections) {{
                                    updateDashboard(data.sections);
                                    lastEventId = data.last_id;
                                }}
                                pollUpdates();
                            }})
                            .catch(() => setTimeout(pollUpdates, 5000));
                        }}
                        pollUpdates();
                    }});

                    // Handle chat messages
                    async function sendChatMessage(sectionId) {{
                        const input = document.querySelector(`#${{sectionId}} input`);
                        const messagesDiv = document.querySelector(`#${{sectionId}} .chat-messages`);
                        const message = input.value.trim();
                        
                        if (!message) return;
                        
                        // Add user message
                        messagesDiv.innerHTML += `<div class="user-message">${{message}}</div>`;
                        input.value = '';
                        
                        try {{
                            const response = await fetch('/api/chat', {{
                                method: 'POST',
                                headers: {{
                                    'Content-Type': 'application/json',
                                    'X-CSRF-Token': csrfToken
                                }},
                                body: JSON.stringify({{
                                    message,
                                    csrf_token: csrfToken
                                }})
                            }});
                            
                            const result = await response.json();
                            if (result.status === 'success') {{
                                messagesDiv.innerHTML += `<div class="bot-message">${{result.response}}</div>`;
                            }} else {{
                                messagesDiv.innerHTML += `<div class="error-message">${{result.message}}</div>`;
                            }}
                        }} catch (e) {{
                            messagesDiv.innerHTML += `<div class="error-message">Error: ${{e.message}}</div>`;
                        }}
                        
                        messagesDiv.scrollTop = messagesDiv.scrollHeight;
                    }}

                    // Initialize video player for HLS streams
                    function initVideoPlayer(sectionId, streamUrl) {{
                        const video = document.querySelector(`#${{sectionId}} video`);
                        if (Hls.isSupported()) {{
                            const hls = new Hls();
                            hls.loadSource(streamUrl);
                            hls.attachMedia(video);
                        }} else if (video.canPlayType('application/vnd.apple.mpegurl')) {{
                            video.src = streamUrl;
                        }}
                    }}

                    // Handle file uploads with progress
                    function handleFileUpload(form) {{
                        const formData = new FormData(form);
                        const progressBar = form.querySelector('.progress-bar');
                        
                        fetch('/api/upload', {{
                            method: 'POST',
                            headers: {
                                'X-CSRF-Token': csrfToken
                            },
                            body: formData
                        }})
                        .then(response => response.json())
                        .then(result => {{
                            if (result.status === 'success') {{
                                form.reset();
                                alert('File uploaded successfully');
                            }} else {{
                                throw new Error(result.message);
                            }}
                        }})
                        .catch(error => {{
                            alert('Upload failed: ' + error.message);
                        }});
                        
                        return false; // Prevent form submission
                    }}

                    // Handle form submissions
                    async function handleFormSubmit(form, sectionId) {{
                        const formData = new FormData(form);
                        const data = Object.fromEntries(formData.entries());
                        
                        try {{
                            const response = await fetch(`/api/form/${{sectionId}}`, {{
                                method: 'POST',
                                headers: {
                                    'Content-Type': 'application/json',
                                    'X-CSRF-Token': csrfToken
                                },
                                body: JSON.stringify({{
                                    ...data,
                                    csrf_token: csrfToken
                                }})
                            }});
                            
                            const result = await response.json();
                            if (result.status === 'success') {{
                                alert(result.message);
                            }} else {{
                                alert(result.message);
                            }}
                        }} catch (e) {{
                            alert('Error: ' + e.message);
                        }}
                        
                        return false; // Prevent form submission
                    }}

                    function updateDashboard(sections) {{
                        // Update dashboard sections
                        const dashboard = document.querySelector('.dashboard');
                        dashboard.innerHTML = '';
                        Object.entries(sections).forEach(([id, content]) => {{
                            const section = document.createElement('div');
                            section.className = 'section';
                            section.innerHTML = `
                                <h2>${{content.heading || id}}</h2>
                                <div class="section-content">
                                    ${{content.content || ''}}
                                </div>
                            `;
                            dashboard.appendChild(section);
                        }});
                    }}
                </script>
                <style>
                    body {{
                        font-family: system-ui, -apple-system, sans-serif;
                        margin: 0;
                        padding: 20px;
                        background: #f5f5f5;
                    }}
                    .dashboard {{
                        max-width: 1200px;
                        margin: 0 auto;
                    }}
                    .section {{
                        background: white;
                        border-radius: 8px;
                        box-shadow: 0 2px 4px rgba(0,0,0,0.1);
                        margin-bottom: 20px;
                        padding: 20px;
                    }}
                    .section h2 {{
                        margin-top: 0;
                        color: #333;
                    }}
                    .upload-section {{
                        border: 2px dashed #ccc;
                        padding: 20px;
                        text-align: center;
                        margin-bottom: 20px;
                    }}
                    .chat-section {{
                        border-top: 1px solid #eee;
                        margin-top: 20px;
                        padding-top: 20px;
                    }}
                    .chat-messages {{
                        max-height: 300px;
                        overflow-y: auto;
                        margin-bottom: 10px;
                    }}
                    .video-player {{
                        width: 100%;
                        max-width: 800px;
                        margin: 0 auto;
                    }}
                    .user-message, .bot-message {{
                        padding: 8px 12px;
                        margin: 4px 0;
                        border-radius: 8px;
                    }}
                    .user-message {{
                        background: #e3f2fd;
                        margin-left: 20%;
                    }}
                    .bot-message {{
                        background: #f5f5f5;
                        margin-right: 20%;
                    }}
                    .error-message {{
                        background: #ffebee;
                        color: #c62828;
                        padding: 8px 12px;
                        margin: 4px 0;
                        border-radius: 8px;
                    }}
                    .progress-bar {{
                        height: 4px;
                        background: #e0e0e0;
                        margin-top: 8px;
                    }}
                    .progress-bar .fill {{
                        height: 100%;
                        background: #2196f3;
                        width: 0%;
                        transition: width 0.3s;
                    }}
                </style>
            </head>
            <body>
                <div class="dashboard">
                    <!-- Sections will be dynamically added here -->
                </div>
            </body>
            </html>
            """
            return html.encode('utf-8')
        except Exception as e:
            logging.error(f"Error generating dashboard HTML: {str(e)}")
            raise

    def add_section(self, section_id: str, content: Dict[str, Any]):
        """Add or update a dashboard section"""
        try:
            with self.section_lock:
                self.dashboard_sections[section_id] = content
                self.notify_clients()
        except Exception as e:
            logging.error(f"Error adding dashboard section: {str(e)}")
            raise

    def remove_section(self, section_id: str):
        """Remove a dashboard section"""
        try:
            with self.section_lock:
                if section_id in self.dashboard_sections:
                    del self.dashboard_sections[section_id]
                    self.notify_clients()
        except Exception as e:
            logging.error(f"Error removing dashboard section: {str(e)}")
            raise

    def notify_clients(self):
        """Notify all long-polling clients of updates"""
        try:
            with self.client_lock:
                self.last_update_id = int(time.time())
                for client in self.clients:
                    client['connected'] = False
        except Exception as e:
            logging.error(f"Error notifying clients: {str(e)}")

    def handle_file_upload(self, file_data: bytes, filename: str) -> Dict[str, Any]:
        """Handle file upload and processing"""
        try:
            # Sanitize filename
            filename = os.path.basename(filename)
            if not filename:
                return {'status': 'error', 'message': 'Invalid filename'}

            # Ensure upload directory exists
            os.makedirs(self.upload_dir, exist_ok=True)

            # Write file
            file_path = os.path.join(self.upload_dir, filename)
            with open(file_path, 'wb') as f:
                f.write(file_data)

            return {'status': 'success', 'filename': filename}
        except Exception as e:
            logging.error(f"Error handling file upload: {str(e)}")
            return {'status': 'error', 'message': str(e)}

    def handle_llm_chat(self, message: str, context: Optional[Dict] = None) -> Dict[str, Any]:
        """Handle LLM chat requests"""
        try:
            if not self.llm_plugin:
                # Try to get an LLM plugin
                ai_plugins = self.plugin_manager.get_plugins("AIModels")
                if ai_plugins:
                    self.llm_plugin = next(iter(ai_plugins.values()))
                else:
                    return {'status': 'error', 'message': 'No LLM plugin available'}

            # Process message through LLM plugin
            response = self.llm_plugin.process_message(message, context or {})
            return {'status': 'success', 'response': response}
        except Exception as e:
            logging.error(f"Error handling LLM chat: {str(e)}")
            return {'status': 'error', 'message': str(e)}

    def handle_form_submit(self, section_id: str, form_data: Dict[str, Any]) -> Dict[str, Any]:
        """Handle form submission from web interface"""
        try:
            form_config = self.form_configs.get(section_id)
            if not form_config:
                return {'status': 'error', 'message': 'Invalid form ID'}

            # Validate required fields
            for field in form_config['fields']:
                if field.get('required') and not form_data.get(field['name']):
                    return {
                        'status': 'error',
                        'message': f"Field {field.get('label', field['name'])} is required"
                    }

            # Update pipeline context
            context_var = form_config['context_var']
            self.pipeline_context[context_var] = form_data

            return {'status': 'success', 'message': 'Data submitted successfully'}
        except Exception as e:
            logging.error(f"Error handling form submission: {str(e)}")
            return {'status': 'error', 'message': str(e)}

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a pipeline step for this plugin"""
        try:
            operation = step_config.get('operation')
            config = step_config.get('config', {})

            if operation == 'section_add':
                self.add_section(
                    config.get('id', str(uuid.uuid4())),
                    config.get('content', {})
                )
            elif operation == 'section_remove':
                self.remove_section(config.get('id'))
            elif operation == 'chat_interface':
                section_id = config.get('id', 'chat_' + str(uuid.uuid4()))
                self.add_section(section_id, {
                    'heading': config.get('heading', 'Chat Interface'),
                    'content': self._generate_chat_interface_html(section_id)
                })
            elif operation == 'file_upload':
                section_id = config.get('id', 'upload_' + str(uuid.uuid4()))
                self.add_section(section_id, {
                    'heading': config.get('heading', 'File Upload'),
                    'content': self._generate_upload_interface_html(section_id)
                })
            elif operation == 'video_stream':
                section_id = config.get('id', 'video_' + str(uuid.uuid4()))
                stream_url = config.get('stream_url')
                if not stream_url:
                    raise ValueError("stream_url is required for video_stream operation")
                self.add_section(section_id, {
                    'heading': config.get('heading', 'Video Stream'),
                    'content': self._generate_video_player_html(section_id, stream_url)
                })
            elif operation == 'form_input':
                section_id = config.get('id', 'form_' + str(uuid.uuid4()))
                form_fields = config.get('fields', [])
                if not form_fields:
                    raise ValueError("fields is required for form_input operation")
                self.add_section(section_id, {
                    'heading': config.get('heading', 'Data Input'),
                    'content': self._generate_form_interface_html(section_id, form_fields)
                })
                # Store form configuration for later validation
                with self.section_lock:
                    self.form_configs[section_id] = {
                        'fields': form_fields,
                        'context_var': config.get('context_var', 'form_data')
                    }

            return context
        except Exception as e:
            logging.error(f"Pipeline step execution failed: {str(e)}")
            raise

    def _generate_chat_interface_html(self, section_id: str) -> str:
        """Generate HTML for chat interface"""
        return f"""
        <div class="chat-section" id="{section_id}">
            <div class="chat-messages"></div>
            <div class="chat-input">
                <input type="text" placeholder="Type your message...">
                <button onclick="sendChatMessage('{section_id}')">Send</button>
            </div>
        </div>
        """

    def _generate_upload_interface_html(self, section_id: str) -> str:
        """Generate HTML for file upload interface"""
        return f"""
        <div class="upload-section" id="{section_id}">
            <form action="/api/upload" method="post" enctype="multipart/form-data" onsubmit="return handleFileUpload(this)">
                <input type="file" name="file" accept=".csv,.json">
                <button type="submit">Upload</button>
                <div class="progress-bar"><div class="fill"></div></div>
            </form>
        </div>
        """

    def _generate_video_player_html(self, section_id: str, stream_url: str) -> str:
        """Generate HTML for video player interface"""
        return f"""
        <div class="video-section" id="{section_id}">
            <div class="video-player">
                <video id="video-{section_id}" controls></video>
            </div>
            <script>
                document.addEventListener('DOMContentLoaded', function() {{
                    initVideoPlayer("{section_id}", "{stream_url}");
                }});
            </script>
        </div>
        """

    def _generate_form_interface_html(self, section_id: str, fields: List[Dict]) -> str:
        """Generate HTML for form interface"""
        field_html = []
        for field in fields:
            field_type = field.get('type', 'text')
            field_id = f"{section_id}_{field.get('name')}"
            field_html.append(f"""
                <div class="form-field">
                    <label for="{field_id}">{field.get('label', field.get('name'))}</label>
                    <input type="{field_type}" 
                           id="{field_id}" 
                           name="{field.get('name')}"
                           {' required' if field.get('required') else ''}
                           {f' pattern="{field.get("pattern")}"' if field.get('pattern') else ''}
                           {f' placeholder="{field.get("placeholder")}"' if field.get("placeholder") else ''}>
                </div>
            """)
        return f"""
        <div class="form-section" id="{section_id}">
            <form onsubmit="return handleFormSubmit(this, '{section_id}')">
                {''.join(field_html)}
                <button type="submit">Submit</button>
            </form>
        </div>
        """


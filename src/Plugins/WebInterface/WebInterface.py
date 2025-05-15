"""
WebInterface with LLMFunction-compatible processing
"""
import os
import logging
import json
from typing import Dict, Any, List
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, Request, File, UploadFile
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from fastapi.responses import HTMLResponse, JSONResponse
from ..BasePlugin import BasePlugin
from ..PluginManager import PluginManager
import threading
import time
import uuid

class WebInterface(BasePlugin):
    """Web interface with direct LLM processing"""
    
    plugin_type = "InputOutput"
    
    def __init__(self, port: int = 8080):
        super().__init__()
        self.port = port
        self.app = FastAPI()
        self.active_connections = set()
        self.llm_plugin = None
        self.plugin_manager = PluginManager()
        self.dashboard_sections = {}  # section_id -> section_data
        self.section_lock = threading.Lock()
        self.pipeline_state = None
        self.pipeline_visualizer_enabled = False
        
        # Initialize routes
        self._setup_routes()

        # Create module-level instance
        web_interface = WebInterface()
        app = web_interface.app
        self.default_css = """
            .dashboard-section {
                 margin: 20px 0;
                 padding: 15px;
                 background: #fff;
                 border: 1px solid #e1e4e8;
                 border-radius: 6px;
                 box-shadow: 0 1px 5px rgba(27,31,35,0.07);
             }
             /* Pipeline visualization styles */
             .pipeline-node {
                 font-family: monospace;
                 white-space: pre;
                 margin: 5px 0;
             }
             .pipeline-line {
                 line-height: 1.5;
             }
             .pipeline-node.highlight .pipeline-line {
                 background-color: #f0f7ff;
                 font-weight: bold;
             }
             .section-header h2 {
                 font-size: 1.5em;
                 color: #34495e;
                 margin: 0 0 10px 0;
                 padding-bottom: 5px;
                 border-bottom: 1px solid #eee;
             }
             .section-content {
                 padding: 10px;
             }
             table {
                 border-collapse: collapse;
                 width: 100%;
                 margin: 10px 0;
             }
             th, td {
                 border: 1px solid #e1e4e8;
                 padding: 8px 12px;
                 text-align: left;
             }
             th {
                 background: #f6f8fa;
             }
             code, pre {
                 font-family: monospace;
                 background: #f6f8fa;
                 padding: 2px 4px;
                 border-radius: 4px;
             }
            """
        
        # Set up routes
        self.app.mount("/static", StaticFiles(directory="static"), name="static")
        self.app.add_websocket_route("/ws", self.websocket_endpoint)
        self.pipeline_handlers = {
            'pipeline_input': self.handle_pipeline_input,
            'pipeline_output': self.handle_pipeline_output,
            'pipeline_status': self.handle_pipeline_status
        }
        self.app.add_route("/", self.serve_interface, methods=["GET"])
        self.app.add_route("/llm-query", self.handle_llm_query, methods=["POST"])
        self.app.add_route("/update-dashboard", self.handle_dashboard_update, methods=["POST"])
        self.app.add_route("/toggle-pipeline-visualizer", self.toggle_visualizer, methods=["POST"])
        self.app.add_route("/upload", self.handle_file_upload, methods=["POST"])
        self.app.add_route("/download/{filename}", self.handle_file_download, methods=["GET"])

        # Create uploads directory if it doesn't exist
        self.upload_dir = "uploads"
        os.makedirs(self.upload_dir, exist_ok=True)

    async def handle_dashboard_update(self, request: Request):
        """Handle dashboard section updates from pipelines"""
        try:
            updates = await request.json()
            if not isinstance(updates, list):
                return JSONResponse(
                    {"error": "Invalid request format", "code": "invalid_format"},
                    status_code=400
                )
                
            with self.section_lock:
                for update in updates:
                    if not isinstance(update, dict) or 'id' not in update:
                        continue
                        
                    section_id = update['id']
                    if update.get('action') == 'remove':
                        self.dashboard_sections.pop(section_id, None)
                    else:  # add/update
                        if 'data' not in update:
                            continue
                        self.dashboard_sections[section_id] = update['data']
            
            await self._broadcast_sections()
            return JSONResponse({"status": "success"})
        except json.JSONDecodeError:
            logging.error("Dashboard update failed: Invalid JSON payload")
            return JSONResponse(
                {"error": "Invalid JSON payload", "code": "invalid_json"},
                status_code=400
            )
        except Exception as e:
            logging.error(f"Dashboard update error: {str(e)}", exc_info=True)
            return JSONResponse(
                {
                    "error": "Internal server error",
                    "code": "server_error",
                    "details": str(e)
                },
                status_code=500
            )

    async def handle_pipeline_input(self, data: Dict):
        """Handle input data from web interface to pipeline"""
        try:
            # Forward to pipeline with standardized format
            pipeline_message = {
                'type': 'web_input',
                'data': data.get('data'),
                'timestamp': time.time()
            }
            # TODO: Implement actual pipeline forwarding
            return {'status': 'received', 'message_id': str(uuid.uuid4())}
        except Exception as e:
            logging.error(f"Pipeline input error: {str(e)}", exc_info=True)
            return {'error': str(e)}

    async def handle_pipeline_output(self, data: Dict):
        """Handle output data from pipeline to web interface"""
        try:
            # Broadcast to all connected clients
            await self._broadcast({
                'type': 'pipeline_output',
                'data': data.get('data'),
                'timestamp': time.time()
            })
            return {'status': 'broadcasted'}
        except Exception as e:
            logging.error(f"Pipeline output error: {str(e)}", exc_info=True)
            return {'error': str(e)}

    async def handle_pipeline_status(self, data: Dict):
        """Handle pipeline status updates"""
        try:
            await self._broadcast({
                'type': 'pipeline_status',
                'status': data.get('status'),
                'message': data.get('message'),
                'timestamp': time.time()
            })
            return {'status': 'updated'}
        except Exception as e:
            logging.error(f"Pipeline status error: {str(e)}", exc_info=True)
            return {'error': str(e)}

    async def _broadcast_sections(self):
        """Send current dashboard state to all clients"""
        message = {
            'type': 'dashboard_update',
            'sections': list(self.dashboard_sections.values()),
            'timestamp': time.time()
        }
        await self._broadcast(message)

    async def update_pipeline_visualization(self, tree: Dict, highlight_path: List[int] = None):
        """Update the pipeline visualization for all connected clients"""
        if not self.pipeline_visualizer_enabled:
            return
            
        message = {
            'type': 'pipeline_update',
            'tree': tree,
            'highlight_path': highlight_path,
            'timestamp': time.time()
        }
        await self._broadcast(message)

    async def handle_llm_query(self, request: Request):
        """Handle LLM query using same pattern as LLMFunction"""
        try:
            data = await request.json()
            if not isinstance(data, dict) or 'prompt' not in data:
                return JSONResponse(
                    {"error": "Missing required field: prompt", "code": "missing_field"},
                    status_code=400
                )
                
            config = data.get('config', {})
            plugin_name = config.get("plugin", "MockAIModel")
            
            try:
                self.llm_plugin = self.plugin_manager.get_plugin("AIModels", plugin_name)
                if not self.llm_plugin:
                    return JSONResponse(
                        {
                            "error": f"LLM plugin {plugin_name} not found",
                            "code": "plugin_not_found"
                        },
                        status_code=404
                    )

                messages = [{
                    "role": "user",
                    "content": f"{config.get('function', '')}\n\n{config.get('format', '')}\n\n{data['prompt']}"
                }]

                response = self.llm_plugin.chat_completion(
                    model=config.get("model", "gpt-3.5-turbo"),
                    messages=messages
                )

                return JSONResponse({
                    "response": response['content'] if isinstance(response, dict) else response
                })
                
            except Exception as e:
                logging.error(f"LLM processing error: {str(e)}", exc_info=True)
                return JSONResponse(
                    {
                        "error": "LLM processing failed",
                        "code": "llm_error",
                        "details": str(e)
                    },
                    status_code=503
                )
                
        except json.JSONDecodeError:
            logging.error("LLM query failed: Invalid JSON payload")
            return JSONResponse(
                {"error": "Invalid JSON payload", "code": "invalid_json"},
                status_code=400
            )
        except Exception as e:
            logging.error(f"LLM query error: {str(e)}", exc_info=True)
            return JSONResponse(
                {
                    "error": "Internal server error",
                    "code": "server_error",
                    "details": str(e)
                },
                status_code=500
            )

    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute step with same interface as LLMFunction"""
        try:
            config = step_config["config"]
            
            # Same direct LLM processing
            plugin_name = config.get("plugin", "OpenAI")
            self.llm_plugin = self.plugin_manager.get_plugin("AIModels", plugin_name)
            if not self.llm_plugin:
                raise ValueError(f"LLM plugin {plugin_name} not found")

            messages = [{
                "role": "user",
                "content": f"{config.get('function', '')}\n\n{config.get('format', '')}\n\n{step_config.get('input', '')}"
            }]

            response = self.llm_plugin.chat_completion(
                model=config.get("model", "gpt-3.5-turbo"),
                messages=messages
            )

            result = response['content'] if isinstance(response, dict) else response
            context[step_config["output"]] = result
            
            # Start web interface if not running
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
                
            return {step_config["output"]: result}
            
        except Exception as e:
            logging.error(f"Pipeline step error: {e}")
            return {step_config.get("output", "llm_result"): f"Error: {str(e)}"}

    async def toggle_visualizer(self, request: Request):
        """Enable/disable the pipeline visualizer"""
        try:
            data = await request.json()
            if not isinstance(data, dict) or 'enabled' not in data:
                return JSONResponse(
                    {"error": "Missing required field: enabled", "code": "missing_field"},
                    status_code=400
                )
                
            self.pipeline_visualizer_enabled = data.get('enabled', False)
            return JSONResponse({"status": "success"})
            
        except json.JSONDecodeError:
            logging.error("Toggle visualizer failed: Invalid JSON payload")
            return JSONResponse(
                {"error": "Invalid JSON payload", "code": "invalid_json"},
                status_code=400
            )
        except Exception as e:
            logging.error(f"Toggle visualizer error: {str(e)}", exc_info=True)
            return JSONResponse(
                {
                    "error": "Internal server error",
                    "code": "server_error",
                    "details": str(e)
                },
                status_code=500
            )

    async def websocket_endpoint(self, websocket: WebSocket):
        """Handle WebSocket connections with pipeline message routing"""
        await websocket.accept()
        self.active_connections.add(websocket)
        try:
            while True:
                data = await websocket.receive_json()
                if not isinstance(data, dict) or 'type' not in data:
                    continue

                handler = self.pipeline_handlers.get(data['type'])
                if handler:
                    response = await handler(data)
                    await websocket.send_json(response)
                else:
                    await self.handle_stream_message(data)

        except WebSocketDisconnect:
            self.active_connections.remove(websocket)
        except Exception as e:
            logging.error(f"WebSocket error: {str(e)}", exc_info=True)
            await websocket.send_json({
                'error': 'Internal server error',
                'details': str(e)
            })
            self.active_connections.remove(websocket)

    async def handle_stream_message(self, message: Dict):
        """Handle stream-related messages from pipelines"""
        if message['type'] == 'stream_init':
            await self._init_stream_player(
                message['player_id'],
                message['stream_url'],
                message.get('config', {})
            )
        elif message['type'] == 'stream_stop':
            await self._stop_stream_player(message['player_id'])

    async def _init_stream_player(self, player_id: str, stream_url: str, config: Dict):
        """Initialize a new stream player instance"""
        await self._broadcast({
            'type': 'stream_init',
            'player_id': player_id,
            'stream_url': stream_url,
            'config': config
        })

    async def _stop_stream_player(self, player_id: str):
        """Stop and cleanup a stream player"""
        await self._broadcast({
            'type': 'stream_stop',
            'player_id': player_id
        })

    async def serve_interface(self, request: Request):
        """Serve the main web interface with default CSS styling"""
        return HTMLResponse(f"""
        <!DOCTYPE html>
        <html>
            <head>
                <title>Mimir AIP Web Interface</title>
                <meta name="viewport" content="width=device-width, initial-scale=1">
                <style>
                /* Base Responsive Styles */
                :root {{
                    --spacing: 1rem;
                    --border-radius: 6px;
                }}

                /* Visualizer Controls */
                .visualizer-wrapper {{
                    position: relative;
                    margin: var(--spacing) 0;
                }}
                
                .toggle-visualizer {{
                    background: #f6f8fa;
                    border: 1px solid #e1e4e8;
                    border-radius: var(--border-radius);
                    padding: 8px 12px;
                    cursor: pointer;
                    margin-bottom: 8px;
                }}
                
                .toggle-visualizer:hover {{
                    background: #e1e4e8;
                }}
                
                body {{
                    margin: 0;
                    padding: 0;
                    min-height: 100vh;
                    display: grid;
                    grid-template-rows: auto 1fr;
                }}
                
                #app-container {{
                    display: grid;
                    grid-template-columns: 1fr;
                    gap: var(--spacing);
                    padding: var(--spacing);
                    max-width: 100%;
                }}
                
                #content-container {{
                    display: grid;
                    gap: var(--spacing);
                    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
                }}
                
                /* Preserve existing styles */
                {self.default_css}
                
                /* Responsive adjustments */
                @media (max-width: 768px) {{
                    .dashboard-section {{
                        margin: 10px 0;
                        padding: 10px;
                    }}
                }}
                </style>
            </head>
            <body>
                <div id="app-container">
                    <div id="content-container"></div>
                    <div class="visualizer-wrapper">
                        <button class="toggle-visualizer" onclick="toggleVisualizer()">Toggle Pipeline View</button>
                        <div id="pipeline-visualizer-container" style="display:none;"></div>
                    </div>
                    <div id="stream-players-container"></div>
                </div>
                <script src="/static/js/app.js"></script>
                <script src="/static/js/pipeline-visualizer.js"></script>
                <script src="/static/js/stream-player.js"></script>
                <script>
                    // Toggle visualizer visibility
                    function toggleVisualizer() {{
                        const container = document.getElementById('pipeline-visualizer-container');
                        if (container.style.display === 'none') {{
                            container.style.display = 'block';
                            pipelineVisualizer.renderLast();
                        }} else {{
                            container.style.display = 'none';
                        }}
                    }}

                    // Initialize pipeline visualizer
                    const pipelineVisualizer = new PipelineVisualizer('pipeline-visualizer-container');
                    
                    // Function to update pipeline visualization
                    function updatePipeline(tree, highlightPath) {{
                        document.getElementById('pipeline-visualizer-container').style.display = 'block';
                        pipelineVisualizer.render(tree, highlightPath);
                    }}

                    // Stream player instances
                    const streamPlayers = {{}};

                    // Handle stream messages
                    function handleStreamMessage(message) {{
                        if (message.type === 'stream_init') {{
                            const container = document.getElementById('stream-players-container');
                            const playerDiv = document.createElement('div');
                            playerDiv.id = `stream-player-${{message.player_id}}`;
                            container.appendChild(playerDiv);
                            
                            streamPlayers[message.player_id] = new StreamPlayer({{
                                targetElementId: playerDiv.id,
                                ...message.config
                            }});
                            streamPlayers[message.player_id].loadStream(message.stream_url);
                        }}
                        else if (message.type === 'stream_stop') {{
                            if (streamPlayers[message.player_id]) {{
                                streamPlayers[message.player_id].destroy();
                                delete streamPlayers[message.player_id];
                            }}
                        }}
                    }}
                </script>
            </body>
        </html>
        """)

    async def handle_file_upload(self, file: UploadFile = File(...)):
        """Handle file upload to server"""
        try:
            if not file.filename:
                return JSONResponse(
                    {"error": "No filename provided", "code": "missing_filename"},
                    status_code=400
                )
                
            if '/' in file.filename or '\\' in file.filename:
                return JSONResponse(
                    {"error": "Invalid filename", "code": "invalid_filename"},
                    status_code=400
                )
            
            file_path = os.path.join(self.upload_dir, file.filename)
            
            try:
                # Write file in chunks to handle large files
                with open(file_path, "wb") as f:
                    while content := await file.read(1024 * 1024):  # 1MB chunks
                        f.write(content)
                
                return JSONResponse({
                    "status": "success",
                    "filename": file.filename,
                    "size": os.path.getsize(file_path)
                })
                
            except IOError as e:
                logging.error(f"File write error: {str(e)}", exc_info=True)
                return JSONResponse(
                    {
                        "error": "Failed to save file",
                        "code": "file_write_error",
                        "details": str(e)
                    },
                    status_code=500
                )
                
        except Exception as e:
            logging.error(f"File upload error: {str(e)}", exc_info=True)
            return JSONResponse(
                {
                    "error": "Internal server error",
                    "code": "server_error",
                    "details": str(e)
                },
                status_code=500
            )

    async def handle_file_download(self, filename: str):
        """Handle file download from server"""
        try:
            if not filename:
                return JSONResponse(
                    {"error": "No filename provided", "code": "missing_filename"},
                    status_code=400
                )
                
            if '/' in filename or '\\' in filename:
                return JSONResponse(
                    {"error": "Invalid filename", "code": "invalid_filename"},
                    status_code=400
                )
            
            file_path = os.path.join(self.upload_dir, filename)
            
            if not os.path.exists(file_path):
                return JSONResponse(
                    {"error": "File not found", "code": "file_not_found"},
                    status_code=404
                )
                
            try:
                return FileResponse(
                    file_path,
                    filename=filename,
                    media_type="application/octet-stream"
                )
            except IOError as e:
                logging.error(f"File read error: {str(e)}", exc_info=True)
                return JSONResponse(
                    {
                        "error": "Failed to read file",
                        "code": "file_read_error",
                        "details": str(e)
                    },
                    status_code=500
                )
                
        except Exception as e:
            logging.error(f"File download error: {str(e)}", exc_info=True)
            return JSONResponse(
                {
                    "error": "Internal server error",
                    "code": "server_error",
                    "details": str(e)
                },
                status_code=500
            )
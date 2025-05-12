"""
WebInterface with LLMFunction-compatible processing
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
        self.app.add_route("/", self.serve_interface, methods=["GET"])
        self.app.add_route("/llm-query", self.handle_llm_query, methods=["POST"])
        self.app.add_route("/update-dashboard", self.handle_dashboard_update, methods=["POST"])
        self.app.add_route("/toggle-pipeline-visualizer", self.toggle_visualizer, methods=["POST"])

    async def handle_dashboard_update(self, request: Request):
        """Handle dashboard section updates from pipelines"""
        try:
            updates = await request.json()
            with self.section_lock:
                for update in updates:
                    section_id = update['id']
                    if update.get('action') == 'remove':
                        self.dashboard_sections.pop(section_id, None)
                    else:  # add/update
                        self.dashboard_sections[section_id] = update['data']
            
            await self._broadcast_sections()
            return JSONResponse({"status": "success"})
        except Exception as e:
            logging.error(f"Dashboard update error: {e}")
            return JSONResponse({"error": str(e)}, status_code=500)

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
            config = data.get('config', {})
            
            # Same LLM plugin selection as LLMFunction
            plugin_name = config.get("plugin", "OpenAI")
            self.llm_plugin = self.plugin_manager.get_plugin("AIModels", plugin_name)
            if not self.llm_plugin:
                raise ValueError(f"LLM plugin {plugin_name} not found")

            # Format messages like LLMFunction
            messages = [{
                "role": "user",
                "content": f"{config.get('function', '')}\n\n{config.get('format', '')}\n\n{data['prompt']}"
            }]

            response = self.llm_plugin.chat_completion(
                model=config.get("model", "gpt-3.5-turbo"),
                messages=messages
            )

            # Return consistent response format
            return JSONResponse({
                "response": response['content'] if isinstance(response, dict) else response
            })
            
        except Exception as e:
            logging.error(f"LLM query error: {e}")
            return JSONResponse(
                {"error": str(e)}, 
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
            self.pipeline_visualizer_enabled = data.get('enabled', False)
            return JSONResponse({"status": "success"})
        except Exception as e:
            logging.error(f"Toggle visualizer error: {e}")
            return JSONResponse({"error": str(e)}, status_code=500)

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
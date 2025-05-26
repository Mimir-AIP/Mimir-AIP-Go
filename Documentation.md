# Mimir-AIP Documentation

## Table of Contents
- [Project Overview](#project-overview)
- [Getting Started](#getting-started)
- [Core Concepts](#core-concepts)
- [Configuration](#configuration)
- [Plugin Development](#plugin-development)
- [Testing & Debugging](#testing--debugging)
- [Advanced Features](#advanced-features)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Project Overview
Mimir-AIP is a modular, plugin-driven pipeline framework for automating data processing, AI/LLM tasks, and report generation. It is designed for extensibility, robust error handling, and flexible testing.

## Getting Started

### Prerequisites
- Python 3.8 or higher
- pip package manager
- Virtual environment (recommended)

### Installation
1. Clone the repository:
   ```bash
   git clone <repo_url>
   cd Mimir-AIP
   ```

2. Set up virtual environment (recommended):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

3. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

### Project Structure
```
Mimir-AIP/
├── src/
│   ├── Plugins/           # All plugin code
│   │   ├── AIModels/     # AI/LLM integration plugins
│   │   ├── Input/        # Data input plugins
│   │   ├── Output/       # Output formatting plugins
│   │   └── Data_Processing/ # Data transformation plugins
│   ├── pipelines/        # Pipeline YAML definitions
│   ├── config.yaml       # Main configuration file
│   └── main.py          # Pipeline runner
├── tests/               # Unit and integration tests
├── reports/            # Generated reports
└── docs/              # Additional documentation
```

## Core Concepts

### Pipeline Architecture
- Pipelines are declarative and defined in YAML
- Each pipeline is a sequence of steps
- Steps are executed in order, with context passing between them
- Plugins handle the actual processing in each step

### Context System

The ContextManager provides centralized state management with thread-safe operations.

#### Features
- **Thread Safety**: All operations protected by threading.Lock()
- **Version Control**: Take and restore snapshots of context state
- **Conflict Resolution**:
  - `overwrite`: Replace existing values (default)
  - `keep`: Preserve existing values
  - `merge`: Combine dictionaries recursively
- **Performance**: Optimized for high throughput (tested to 1000 ops in <100ms)
- **Error Handling**: Validates inputs and logs errors

#### API Reference
```python
get_context(key=None) -> Any
set_context(key, value, overwrite=True) -> bool
merge_context(new_context, conflict_strategy='overwrite') -> Dict
snapshot_context() -> int
restore_context(snapshot_id) -> bool
clear_context() -> None
```

#### Example Usage
```python
# Basic operations
cm.set_context('user', {'name': 'Alice'})
user = cm.get_context('user')

# Merge dictionaries
cm.merge_context({'user': {'age': 30}}, conflict_strategy='merge')

# Version control
snap_id = cm.snapshot_context()
cm.restore_context(snap_id)
```

### Plugin System
- Plugins are auto-discovered from the Plugins directory
- Each plugin type (Input, Output, etc.) has its own subdirectory
- Plugins must implement the BasePlugin interface
- Configuration is passed via step_config['config']

## Configuration

### Environment Variables
The following environment variables are supported across different plugins:

#### Core Settings
- `LOG_LEVEL`: Logging level (DEBUG, INFO, WARNING, ERROR)
- `OUTPUT_DIR`: Directory for generated files
- `CACHE_DIR`: Directory for caching responses

#### AI Model Integration
- `OPENROUTER_API_KEY`: API key for OpenRouter integration
- `AZURE_OPENAI_API_KEY`: API key for Azure OpenAI services
- `AZURE_OPENAI_ENDPOINT`: Endpoint URL for Azure OpenAI
- `MOONDREAM_API_KEY`: API key for Moondream AI services

#### Database Integration
- `DB_HOST`: Database host address
- `DB_PORT`: Database port number
- `DB_NAME`: Database name
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password

#### Web Interface
- `WEBINTERFACE_PORT`: Port for the web interface (default: 8080)
- `WEBINTERFACE_HOST`: Host for the web interface (default: localhost)
- `WEBINTERFACE_UPLOAD_DIR`: Custom directory for file uploads
- `WEBINTERFACE_SESSION_TIMEOUT`: Session timeout in seconds (default: 3600)

Create a `.env` file based on `.env.template` in the root directory to configure these variables.

### Dependencies

Core dependencies are listed in `requirements.txt`. Some plugins have additional requirements:

#### Database Plugins
```
psycopg2-binary>=2.9.9  # PostgreSQL
mysql-connector-python>=8.2.0  # MySQL
pymongo>=4.6.0  # MongoDB
```

#### Video Processing
```
opencv-python>=4.8.0
```

#### API Integration
```
requests>=2.25.0
```

Install plugin-specific dependencies when using those features:
```bash
pip install -r src/Plugins/Input/Database/requirements.txt  # For database support
```

### Pipeline Configuration
Example pipeline step:
```yaml
- name: "Generate Summary"
  plugin: "LLMFunction"
  config:
    prompt: "Summarize: {text}"
    model: "openrouter/mistral-7b"
    mock_response: {"summary": "Test summary"}
  output: "summary_result"
```

## Plugin Development

### Creating New Plugins
1. Create a new directory under appropriate plugin type
2. Implement BasePlugin interface
3. Add configuration validation
4. Document inputs and outputs
5. Add error handling and logging

Example plugin structure:
```python
from Plugins.BasePlugin import BasePlugin

class MyPlugin(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        config = step_config['config']
        # Validate config
        # Process data
        return {step_config['output']: result}
```

### Best Practices
- Use clear error messages
- Document all config options
- Implement proper error handling
- Use typing hints
- Write unit tests
- Follow PEP standards

## Testing & Debugging

### Test Mode
Enable test mode in config.yaml:
```yaml
settings:
  test_mode: true
```

Features:
- Uses mock responses instead of API calls
- Cleans up test outputs automatically
- Validates pipeline configuration
- Faster execution for testing

### Logging
- Check src/mimir.log for detailed logs
- Log levels: DEBUG, INFO, WARNING, ERROR
- Each plugin should log appropriate information
- Use structured logging for better parsing

### Adding New Tests

When adding new tests, follow these guidelines:

1. **Test File Location**: Place test files in the `tests/` directory.
2. **Test File Naming**: Name test files with the pattern `test_*.py`.
3. **Test Class Naming**: Use the pattern `Test<PluginName>` for test classes.
4. **Test Methods**: Name test methods with the prefix `test_`.
5. **Test Setup**: Use the `setUp` method for test fixture setup.
6. **Test Teardown**: Use the `tearDown` method for cleanup.
7. **Mocking**: Use mock objects for external dependencies.
8. **Assertions**: Use Python's `unittest` framework for assertions.
9. **Running Tests**: Use the command:
   ```bash
   PYTHONPATH=<project_root> python tests/test_<plugin_name>.py
   ```
   (Replace `<project_root>` with the absolute path to the project directory)
10. **Python Path**: Ensure you add the `src` directory to the Python path:
    ```python
    import sys
    import os
    sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "src"))
    ```
11. **Plugin Import**: Import plugins using the full path:
    ```python
    from Plugins.<PluginType>.<PluginName>.<PluginName> import <PluginClass>
    ```

## Advanced Features

### Interactive Web Interface
The WebInterface plugin provides a real-time, interactive web dashboard for pipeline interaction:

- Input Features:
  - File uploads (CSV, JSON)
  - Form-based data input
  - Real-time LLM chat interface
  - Context integration

- Output Features:
  - Dynamic content sections
  - Video streaming (HLS)
  - Real-time updates
  - Interactive elements

See [Web Interface Documentation](Docs/Web_Interface.md) for detailed usage.

### Video Processing
The VideoInput plugin supports:
- Frame extraction
- Metadata handling
- Format conversion
- Integration with image processing

### Live Stream Processing
The LiveStreamProcessor plugin supports:
- HLS (.m3u8) and DASH (.mpd) streams
- Stream validation and health checks
- Frame capture from live streams
- Integration with WebInterface player

#### WebInterface Player Features:
- Uses hls.js for HLS playback
- Native fallback for Safari
- Configurable player controls
- Multiple concurrent streams
- Error handling and recovery

#### Example Pipeline:
```yaml
- plugin: "Input/LiveStreamProcessor/LiveStreamProcessor"
  config:
    stream_url: "http://example.com/stream.m3u8"
    capture_frame: true  # Optional frame capture
  output: "stream_data"

- plugin: "WebInterface/WebInterface"
  config:
    stream_displays:
      - player_id: "stream1"
        source: "stream_data.stream_url"
        config:
          autoplay: true
          controls: true
          width: "800px"
```

### Report Generation
- Multiple output formats (HTML, JSON, etc.)
- Customizable templates
- Dynamic content generation
- Interactive elements

### LLM Integration
- Support for multiple providers
- Prompt management
- Response parsing
- Error handling

### Plugin Dependencies

Each plugin type may have specific dependencies. These are documented in plugin-specific `requirements.txt` files. Always check the plugin's directory for additional requirements before use.

Common plugin dependencies:
- Database plugins: PostgreSQL, MySQL, or MongoDB drivers
- Video processing: OpenCV
- API plugins: requests library
- LLM plugins: Various AI model SDKs
- WebInterface plugin:
  - `websockets>=12.0`: WebSocket support for real-time updates
  - `httpx>=0.27.0`: Modern HTTP client for async operations
  - `Pillow>=11.2.1`: Image handling for file uploads
  - `aiofiles>=23.2.1`: Async file operations

### Generated Files

The following file types may be generated during pipeline execution:
- `generated_*.mp3`: Audio output files
- `generated_*.jpg/png`: Image output files
- `output_*.jpg/png`: Processed image files
- `**/reports/*.html`: HTML reports
- `map.html`: Map visualizations
- Debug logs and temporary files

These files are automatically managed in test mode and are excluded from version control.

## Troubleshooting

### Common Issues
- **API Errors**: Check environment variables and API keys
- **Plugin Loading**: Verify plugin directory structure
- **Pipeline Errors**: Validate YAML syntax and config
- **Context Errors**: Check step output names and dependencies

### Debug Tools
- Enable debug logging
- Use test mode for isolation
- Check plugin-specific logs
- Validate configuration files

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Write tests
4. Implement changes
5. Update documentation
6. Submit pull request

### Code Style
- Follow PEP 8
- Use type hints
- Write clear docstrings
- Keep functions focused
- Comment complex logic

### Testing Requirements
- Write unit tests for new features
- Update existing tests as needed
- Include integration tests
- Verify in test mode

#### WebInterface Testing
The WebInterface plugin includes comprehensive testing suites:

1. Unit Tests (test_webinterface.py):
   - File upload functionality
   - WebSocket connections
   - Section management
   - Form validation
   - Request handling

2. Integration Tests (test_webinterface_integration.py):
   - Complete LLM chat flows
   - Real-time pipeline updates
   - WebSocket communication
   - Long-polling updates
   - Context synchronization

3. Test Requirements:
   ```
   pytest>=8.2.0
   pytest-asyncio>=0.23.7
   httpx>=0.27.0
   websockets>=12.0
   ```

4. Running Tests:
   ```bash
   pytest tests/test_webinterface.py tests/test_webinterface_integration.py -v

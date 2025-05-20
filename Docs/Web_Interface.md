# Web Interface Plugin

The WebInterface plugin provides a real-time, interactive web dashboard for Mimir-AIP pipelines. It serves as both an input and output interface, allowing users to interact with running pipelines through a modern web UI.

## Features

### 1. Real-Time Dashboard
- Dynamic section-based interface
- Automatic updates via long-polling
- Responsive layout
- Error handling and notifications
- Pipeline status visualization

### 2. Input Capabilities

#### File Upload
```yaml
- name: Add File Upload
  plugin: Web.WebInterface.WebInterface
  operation: file_upload
  config:
    id: file_upload_section
    heading: Upload Data Files
```
- Supports CSV and JSON files
- Progress bar indication
- Automatic context integration
- File validation

#### Form Input
```yaml
- name: Add User Input Form
  plugin: Web.WebInterface.WebInterface
  operation: form_input
  config:
    id: user_form
    heading: User Information
    fields:
      - name: username
        label: Username
        type: text
        required: true
      - name: age
        label: Age
        type: number
        pattern: "[0-9]*"
    context_var: user_data
```
- Multiple input types (text, number, date, etc.)
- Required field validation
- Custom patterns and validation
- Automatic context updates

#### LLM Chat Interface
```yaml
- name: Add Chat Interface
  plugin: Web.WebInterface.WebInterface
  operation: chat_interface
  config:
    id: chat_section
    heading: AI Assistant
```
- Real-time chat interaction
- Integration with LLM plugins
- Context-aware responses
- Error handling and retry logic

### 3. Output Capabilities

#### Dynamic Content Sections
```yaml
- name: Add Results Section
  plugin: Web.WebInterface.WebInterface
  operation: section_add
  config:
    id: results_section
    content:
      heading: Analysis Results
      content: ${analysis_results}
```
- Text, HTML, and JavaScript support
- Real-time updates
- Structured content layout
- Custom styling options

#### Video Streaming
```yaml
- name: Add Video Stream
  plugin: Web.WebInterface.WebInterface
  operation: video_stream
  config:
    id: video_section
    heading: Live Feed
    stream_url: https://example.com/stream.m3u8
```
- HLS streaming support
- Built-in video controls
- Adaptive quality
- Cross-browser compatibility

### 4. Interactive Features

#### Real-Time Updates
- Long-polling for instant updates
- WebSocket fallback option
- Automatic reconnection
- Status indicators

#### Error Handling
- User-friendly error messages
- Automatic retry logic
- Network error recovery
- Context validation

## Operations

### section_add
Adds or updates a section in the dashboard.

```yaml
operation: section_add
config:
  id: "unique_section_id"        # Required: Unique identifier for the section
  content:
    heading: "Section Title"     # Optional: Section heading
    content: "Section content"   # Required: Content to display (text, HTML, or template)
  position: "top"               # Optional: Position in dashboard (top, bottom)
```

**Response Format:**
```json
{
  "status": "success",
  "section_id": "unique_section_id"
}
```

### section_remove
Removes a section from the dashboard.

```yaml
operation: section_remove
config:
  id: "section_id_to_remove"    # Required: ID of section to remove
```

**Response Format:**
```json
{
  "status": "success",
  "removed_id": "section_id_to_remove"
}
```

### chat_interface
Creates an interactive chat interface with LLM integration.

```yaml
operation: chat_interface
config:
  id: "chat_section"            # Required: Unique identifier for the chat
  heading: "AI Assistant"       # Optional: Chat interface heading
  llm_plugin: "OpenAI"         # Optional: Specific LLM plugin to use
  context_var: "chat_history"   # Optional: Context variable for chat history
  options:
    max_history: 10            # Optional: Maximum messages to retain
    stream_response: true      # Optional: Enable streaming responses
```

**Response Format:**
```json
{
  "status": "success",
  "section_id": "chat_section",
  "event": "message",
  "data": {
    "role": "assistant",
    "content": "Response content"
  }
}
```

### file_upload
Creates a file upload interface with progress tracking.

```yaml
operation: file_upload
config:
  id: "upload_section"          # Required: Unique identifier for upload section
  heading: "Upload Files"       # Optional: Section heading
  allowed_types: [".csv", ".json"] # Optional: Allowed file extensions
  max_size: 10485760           # Optional: Maximum file size in bytes (10MB)
  context_var: "uploaded_file"  # Optional: Context variable for upload info
```

**Response Format:**
```json
{
  "status": "success",
  "filename": "example.csv",
  "path": "/upload/example.csv",
  "size": 1024,
  "mime_type": "text/csv"
}
```

### video_stream
Creates a video player for HLS streams.

```yaml
operation: video_stream
config:
  id: "video_section"           # Required: Unique identifier for video section
  heading: "Live Stream"        # Optional: Section heading
  stream_url: "https://..."    # Required: HLS stream URL
  autoplay: true               # Optional: Start playing automatically
  controls: true               # Optional: Show player controls
  quality_levels: ["auto", "high", "medium", "low"]  # Optional: Available qualities
```

**Response Format:**
```json
{
  "status": "success",
  "section_id": "video_section",
  "stream_status": "playing"
}
```

### form_input
Creates an interactive form with validation.

```yaml
operation: form_input
config:
  id: "form_section"           # Required: Unique identifier for form
  heading: "Input Form"        # Optional: Form heading
  fields:                      # Required: Array of form fields
    - name: "field_name"       # Required: Field identifier
      label: "Field Label"     # Optional: Display label
      type: "text"            # Required: Field type
      required: true          # Optional: Field is required
      pattern: "[A-Za-z]+"    # Optional: Validation pattern
      options:                # Optional: For select/radio fields
        - value: "option1"
          label: "Option 1"
  context_var: "form_data"    # Optional: Context variable for form data
```

**Response Format:**
```json
{
  "status": "success",
  "form_id": "form_section",
  "data": {
    "field_name": "submitted value"
  }
}
```

## Configuration

### Environment Variables
```bash
# Server Configuration
WEBINTERFACE_PORT=8080          # Server port (default: 8080)
WEBINTERFACE_HOST=0.0.0.0       # Bind address (default: localhost)
WEBINTERFACE_UPLOAD_DIR=/path   # Upload directory (default: plugin_dir/uploads)
WEBINTERFACE_MAX_UPLOAD=10485760 # Max upload size in bytes (default: 10MB)

# Security Settings
WEBINTERFACE_ALLOWED_ORIGINS=*   # CORS origins (default: *)
WEBINTERFACE_SESSION_TIMEOUT=3600 # Session timeout in seconds (default: 1h)
WEBINTERFACE_RATE_LIMIT=100      # Requests per minute (default: 100)

# Feature Flags
WEBINTERFACE_ENABLE_WEBSOCKET=true    # Enable WebSocket (default: true)
WEBINTERFACE_ENABLE_FILE_SCAN=false   # Enable malware scan (default: false)
WEBINTERFACE_DEBUG_MODE=false         # Enable debug mode (default: false)
```

### Plugin Initialization
```yaml
plugins:
  enabled:
    - Web.WebInterface.WebInterface

settings:
  webinterface_port: 8080  # Optional, defaults to 8080
```

### Server Configuration
```yaml
settings:
  webinterface:
    port: 8080                 # Server port
    host: "0.0.0.0"           # Bind address
    upload_dir: "./uploads"    # Upload directory
    client_options:
      long_polling: true       # Enable long polling
      websocket: true         # Enable WebSocket
      update_interval: 1000   # Update interval (ms)
    security:
      allowed_origins: ["*"]   # CORS settings
      max_upload_size: 10485760 # Max upload size
      rate_limit: 100         # Requests per minute
    logging:
      level: "info"           # Logging level
      file: "webinterface.log" # Log file
```

### Resource Management
```yaml
settings:
  webinterface:
    cleanup:
      session_timeout: 3600    # Session timeout (s)
      upload_ttl: 3600        # Upload file TTL (s)
      max_sections: 50        # Max sections per client
      max_history: 100        # Max chat history
    performance:
      cache_ttl: 300          # Cache lifetime (s)
      max_connections: 1000   # Max concurrent clients
      batch_updates: true     # Batch section updates
```

## Pipeline Integration
```yaml
steps:
  - name: Initialize Web Interface
    plugin: Web.WebInterface.WebInterface
    operation: section_add
    config:
      id: welcome_section
      content:
        heading: Welcome
        content: Pipeline started and ready for interaction.
```

## Context System Integration

### File Uploads
Uploaded files are automatically integrated into the pipeline context:
```yaml
context:
  uploaded_file:
    path: "<upload_dir>/<filename>"  # Absolute path to uploaded file
    filename: "example.csv"          # Original filename
    mime_type: "text/csv"           # Detected MIME type
    size: 1024                      # File size in bytes
    timestamp: "2025-05-19T10:00:00Z"  # Upload timestamp
```

### Form Data
Form submissions are stored in the context using the specified `context_var`:
```yaml
context:
  user_data:  # From context_var in form config
    username: "example_user"
    age: 25
    # ... other form fields ...
```

### Chat Interactions
Chat messages and responses are stored in the context:
```yaml
context:
  chat_history:
    section_id:  # Chat section ID
      messages:
        - role: "user"
          content: "Hello"
          timestamp: "2025-05-19T10:00:00Z"
        - role: "assistant"
          content: "Hi! How can I help?"
          timestamp: "2025-05-19T10:00:01Z"
```

### Video Streams
Stream information is maintained in the context:
```yaml
context:
  video_streams:
    section_id:  # Video section ID
      url: "https://example.com/stream.m3u8"
      status: "playing"  # playing, paused, stopped
      quality: "auto"    # Current quality setting
      timestamp: "2025-05-19T10:00:00Z"
```

## Section Types

1. **Basic Section**
   - Simple text/HTML content
   - Static information display
   - Basic formatting

2. **Form Section**
   - Input fields
   - Validation rules
   - Submit handlers
   - Context integration

3. **File Upload Section**
   - File selection
   - Upload progress
   - Format validation
   - Context storage

4. **Chat Section**
   - Message display
   - Input field
   - Send button
   - Auto-scroll

5. **Video Section**
   - Video player
   - Stream controls
   - Quality settings
   - Error recovery

## Best Practices

1. **Section Management**
   - Use meaningful section IDs
   - Group related content
   - Remove unused sections
   - Limit total sections

2. **Form Design**
   - Clear labels
   - Appropriate validation
   - Error feedback
   - Context naming

3. **Error Handling**
   - Validate inputs
   - Provide feedback
   - Implement retries
   - Log issues

4. **Performance**
   - Optimize updates
   - Clean up resources
   - Monitor connections
   - Cache when possible

## Error Handling

### Response Format
All operations return responses in a consistent format:
```json
{
  "status": "success|error",
  "message": "Human readable message",
  "data": {},      // Optional: Operation-specific data
  "error": {       // Only present if status is "error"
    "code": "ERROR_CODE",
    "message": "Detailed error message",
    "details": {}  // Optional: Error-specific details
  }
}
```

### Common Error Codes
- `INVALID_CONFIG`: Missing or invalid configuration
- `SECTION_NOT_FOUND`: Referenced section doesn't exist
- `VALIDATION_ERROR`: Form/file validation failed
- `CONTEXT_ERROR`: Context variable issues
- `PLUGIN_ERROR`: Plugin-specific errors
- `NETWORK_ERROR`: Communication issues
- `SERVER_ERROR`: Internal server errors

### Error Recovery
1. **Automatic Retries**
   - Network errors: 3 retries with exponential backoff
   - WebSocket disconnects: Automatic reconnection
   - Failed updates: Retry with fresh context

2. **Manual Recovery**
   - Reload section: `section_add` with same ID
   - Clear form: Reset button or new load
   - Restart upload: Fresh file selection
   - Reconnect stream: Player reload

## Security

### File Upload Security
1. **Validation**
   - File size limits (default: 10MB)
   - MIME type verification
   - Extension whitelist
   - Malware scanning (if configured)

2. **Storage**
   - Secure temp directory
   - Randomized filenames
   - Automatic cleanup
   - Permission restrictions

### Input Sanitization
1. **Form Data**
   - HTML escaping
   - SQL injection prevention
   - XSS protection
   - Format validation

2. **Chat Input**
   - Content filtering
   - Length limits
   - Rate limiting
   - Pattern matching

### Access Control
1. **Server**
   - Port binding restrictions
   - Host whitelist
   - CORS policies
   - Rate limiting

2. **Operations**
   - Section permissions
   - Upload restrictions
   - Stream access control
   - Form submission rules

### Session Management
1. **Client Sessions**
   - Unique session IDs
   - Timeout configuration
   - Activity tracking
   - Clean termination

2. **Resource Cleanup**
   - Automatic file deletion
   - Session expiration
   - Context clearing
   - Connection cleanup

## Performance Optimization

### Client-Side Optimization
1. **Update Batching**
   ```javascript
   // Configure batching in client options
   settings:
     webinterface:
       client_options:
         batch_updates: true
         batch_interval: 100    // ms
         max_batch_size: 10
   ```

2. **Resource Loading**
   - Lazy section loading
   - Image/video optimization
   - CSS/JS minification
   - Browser caching

3. **WebSocket Usage**
   ```yaml
   settings:
     webinterface:
       websocket:
         enabled: true
         compression: true
         heartbeat_interval: 30  # seconds
   ```

### Server-Side Optimization
1. **Memory Management**
   ```yaml
   settings:
     webinterface:
       memory:
         max_sections_per_client: 50
         section_cache_size: 100
         file_buffer_size: 8192
         cleanup_interval: 300  # seconds
   ```

2. **Database Integration**
   ```yaml
   settings:
     webinterface:
       storage:
         type: "sqlite"              # or "postgres", "mysql"
         connection: "webinterface.db"
         cache_enabled: true
         pool_size: 10
   ```

3. **Load Balancing**
   ```yaml
   settings:
     webinterface:
       load_balancing:
         enabled: true
         max_clients: 1000
         max_requests_per_client: 100
         throttle_threshold: 0.8
   ```

## Concurrency Management

### Section Locking
```python
# Example of section locking in custom code
async with section_lock:
    await update_section(section_id, content)
```

### Pipeline Synchronization
1. **Context Updates**
   ```yaml
   settings:
     webinterface:
       sync:
         context_lock_timeout: 5    # seconds
         max_concurrent_updates: 10
         retry_attempts: 3
   ```

2. **File Operations**
   ```yaml
   settings:
     webinterface:
       file_handling:
         max_concurrent_uploads: 5
         chunk_size: 1048576       # 1MB
         temp_cleanup_interval: 300 # seconds
   ```

### Client Connection Management
1. **Connection Pooling**
   ```yaml
   settings:
     webinterface:
       connections:
         pool_size: 1000
         keepalive: 60             # seconds
         timeout: 30               # seconds
         max_pending: 100
   ```

2. **Event Queue Management**
   ```yaml
   settings:
     webinterface:
       queue:
         max_size: 1000
         overflow_strategy: "drop"  # or "block"
         processing_threads: 4
   ```

### Error Recovery
1. **Automatic Recovery**
   - Transaction rollback
   - Connection reset
   - Cache invalidation
   - State reconciliation

2. **Manual Intervention**
   ```bash
   # Reset section state
   curl -X POST http://localhost:8080/api/admin/reset-section/{id}
   
   # Clear connection pool
   curl -X POST http://localhost:8080/api/admin/clear-connections
   
   # Force sync context
   curl -X POST http://localhost:8080/api/admin/sync-context
   ```

## Example Pipeline

```yaml
name: Interactive Data Analysis
description: Pipeline with web interface for data input and analysis

steps:
  - name: Setup Interface
    plugin: Web.WebInterface.WebInterface
    operation: form_input
    config:
      id: data_input
      heading: Data Parameters
      fields:
        - name: dataset
          label: Dataset Name
          type: text
          required: true
        - name: analysis_type
          label: Analysis Type
          type: select
          options:
            - value: basic
              label: Basic Statistics
            - value: advanced
              label: Advanced Analysis
      context_var: analysis_params

  - name: Add File Upload
    plugin: Web.WebInterface.WebInterface
    operation: file_upload
    config:
      id: file_section
      heading: Upload Dataset

  - name: Process Data
    plugin: Data_Processing.Analyzer
    config:
      input: ${uploaded_file}
      params: ${analysis_params}
    output: analysis_results

  - name: Display Results
    plugin: Web.WebInterface.WebInterface
    operation: section_add
    config:
      id: results
      content:
        heading: Analysis Results
        content: ${analysis_results}
```

## Troubleshooting

1. **Connection Problems**
   - Check port availability
   - Verify network access
   - Check for firewalls
   - Review proxy settings

2. **Update Issues**
   - Verify section IDs
   - Check context variables
   - Monitor client connections
   - Review update triggers

3. **Form Submission Errors**
   - Validate input formats
   - Check required fields
   - Review context updates
   - Monitor submission handlers

4. **File Upload Problems**
   - Check file size limits
   - Verify supported formats
   - Review upload directory
   - Monitor upload progress

## Debugging

### Debug Mode
Enable debug mode in configuration:
```yaml
settings:
  webinterface:
    debug_mode: true
    logging:
      level: "debug"
      console: true
```

### Available Debug Tools
1. **Browser DevTools**
   - Network tab: Monitor requests
   - Console: View client logs
   - Elements: Inspect sections
   - Application: Check WebSocket

2. **Server Logging**
   - Request/response details
   - WebSocket messages
   - Context changes
   - Error stack traces

3. **Status Endpoints**
   - `/api/status`: Server health
   - `/api/sections`: Active sections
   - `/api/clients`: Connected clients
   - `/api/debug`: Debug information

### Common Debug Steps
1. **Connection Issues**
   ```bash
   # Check server status
   curl http://localhost:8080/api/status
   
   # Test WebSocket
   wscat -c ws://localhost:8080/ws
   
   # Monitor log file
   tail -f webinterface.log
   ```

2. **Section Problems**
   ```bash
   # List sections
   curl http://localhost:8080/api/sections
   
   # Test section update
   curl -X POST http://localhost:8080/api/section/{id} \
        -H "Content-Type: application/json" \
        -d '{"content": "test"}'
   ```

3. **Context Issues**
   ```bash
   # View context
   curl http://localhost:8080/api/debug/context
   
   # Clear context
   curl -X POST http://localhost:8080/api/debug/clear-context
   ```

## Testing

### Unit Tests
Located in `tests/test_webinterface.py`:
```python
def test_section_management():
    """Test section CRUD operations"""

def test_file_upload():
    """Test file upload handling"""

def test_form_validation():
    """Test form input validation"""

def test_chat_interface():
    """Test chat functionality"""
```

### Integration Tests
Located in `tests/test_webinterface_integration.py`:
```python
def test_complete_pipeline():
    """Test full pipeline integration"""

def test_realtime_updates():
    """Test WebSocket updates"""

def test_error_handling():
    """Test error recovery"""
```

### Test Configuration
```yaml
# test_config.yaml
webinterface:
  test_mode: true
  mock_responses: true
  test_port: 8081
```

### Running Tests
```bash
# Run all tests
pytest tests/test_webinterface*.py -v

# Run specific test
pytest tests/test_webinterface.py::test_section_management -v

# Run with coverage
pytest tests/test_webinterface*.py --cov=src/Plugins/Web
```

### UI Testing
```bash
# Install test dependencies
npm install --save-dev playwright

# Run UI tests
npx playwright test tests/ui/webinterface.spec.ts
```

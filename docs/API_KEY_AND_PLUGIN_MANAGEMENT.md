# API Key and Plugin Management Design

## Overview

This document describes the design for managing LLM provider API keys and custom plugins in the Mimir AIP unified Docker deployment.

## 1. Database Schema

### API Keys Table
```sql
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,  -- openai, anthropic, ollama, etc.
    name TEXT NOT NULL,  -- User-friendly name
    key_value TEXT NOT NULL,  -- Encrypted API key
    endpoint_url TEXT,  -- Custom endpoint (for Ollama, local models)
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    metadata TEXT  -- JSON: model defaults, rate limits, etc.
);
```

### Plugins Table
```sql
CREATE TABLE IF NOT EXISTS plugins (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,  -- input, output, ai, data_processing, etc.
    version TEXT NOT NULL,
    file_path TEXT NOT NULL,  -- Path to .so/.dll file
    description TEXT,
    author TEXT,
    is_enabled BOOLEAN NOT NULL DEFAULT 1,
    is_builtin BOOLEAN NOT NULL DEFAULT 0,
    config TEXT,  -- JSON: plugin-specific configuration
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 2. Backend API Endpoints

### API Key Management
- `GET /api/v1/settings/api-keys` - List all API keys (encrypted values NOT returned)
- `POST /api/v1/settings/api-keys` - Create new API key
- `PUT /api/v1/settings/api-keys/:id` - Update API key
- `DELETE /api/v1/settings/api-keys/:id` - Delete API key
- `POST /api/v1/settings/api-keys/:id/test` - Test API key validity

### Plugin Management
- `GET /api/v1/settings/plugins` - List all plugins
- `POST /api/v1/settings/plugins/upload` - Upload plugin file (.so/.dll)
- `PUT /api/v1/settings/plugins/:id` - Update plugin (enable/disable, config)
- `DELETE /api/v1/settings/plugins/:id` - Delete plugin (user-uploaded only)
- `POST /api/v1/settings/plugins/:id/reload` - Reload plugin without restart

## 3. Security Considerations

### API Key Encryption
- Use AES-256-GCM encryption for API keys at rest
- Encryption key stored in environment variable: `MIMIR_ENCRYPTION_KEY`
- Keys decrypted only when needed for API calls
- NEVER return decrypted keys in API responses

### Plugin Security
- Validate plugin signatures before loading
- Sandboxed execution (if possible with Go plugins)
- Only load plugins from `/app/plugins` directory
- Built-in plugins cannot be deleted
- Log all plugin operations for audit

## 4. Frontend UI Components

### Settings Page Structure
```
/settings
  /api-keys
    - List of API keys (grouped by provider)
    - Add new key button
    - Edit/delete actions
    - Test key button
  /plugins
    - List of installed plugins
    - Upload plugin button
    - Enable/disable toggle
    - Configure plugin dialog
    - Delete button (user-uploaded only)
```

### API Key Form Fields
- Provider (dropdown): OpenAI, Anthropic, Ollama, Custom
- Name (text): User-friendly identifier
- API Key (password field): The actual key
- Endpoint URL (text, optional): For custom providers
- Default Model (dropdown): Preferred model for this key
- Rate Limit (number, optional): Requests per minute

### Plugin Upload Flow
1. User selects .so/.dll file
2. Frontend uploads to `/api/v1/settings/plugins/upload`
3. Backend validates and extracts metadata
4. Plugin added to database and loaded
5. User can configure plugin-specific settings

## 5. Unified Docker Integration

### Environment Variables
```yaml
environment:
  # Encryption key for API keys (MUST be set in production)
  - MIMIR_ENCRYPTION_KEY=change-me-32-byte-key-for-aes
  
  # Plugin directory (already mounted)
  - MIMIR_PLUGIN_DIR=/app/plugins
  
  # Database path (already set)
  - MIMIR_DB_PATH=/app/data/mimir.db
```

### Volume Mounts (Already Configured)
```yaml
volumes:
  - mimir_plugins:/app/plugins  # For user-uploaded plugins
  - mimir_data:/app/data        # For database and encryption keys
```

### Plugin Loading on Startup
1. Scan `/app/plugins` directory for .so/.dll files
2. Query `plugins` table for enabled plugins
3. Load only enabled plugins into registry
4. Log any plugins that fail to load

## 6. Usage Workflow

### User Adds OpenAI API Key
1. Navigate to `/settings/api-keys`
2. Click "Add API Key"
3. Select "OpenAI" as provider
4. Enter name: "My OpenAI Key"
5. Paste API key: `sk-...`
6. Select default model: "GPT-4"
7. Click "Save"
8. Key encrypted and stored in database
9. Agent chat can now use this key

### Agent Uses API Key
1. User selects model in chat (e.g., "GPT-4")
2. Backend queries `api_keys` table for active OpenAI key
3. Decrypt key using encryption key
4. Make API call to OpenAI with decrypted key
5. Update `last_used_at` timestamp
6. Return response to user

### User Uploads Custom Plugin
1. Navigate to `/settings/plugins`
2. Click "Upload Plugin"
3. Select compiled .so file (e.g., `my_plugin.so`)
4. Backend validates plugin:
   - Check file signature
   - Extract metadata (name, version, type, description)
   - Verify plugin implements required interface
5. Save to `/app/plugins/my_plugin.so`
6. Insert into `plugins` table
7. Load plugin into registry
8. Plugin now available in pipeline builder

## 7. Implementation Status

### ✅ Completed
- Database schema added to `persistence.go`
- Unified Docker already has `/app/plugins` volume
- Unified Docker already has database persistence

### ⏳ To Implement
- Backend API handlers for key/plugin management
- Frontend Settings page with tabs
- API key form and list components
- Plugin upload and management UI
- Encryption key management (environment variable)
- Plugin validation and loading logic

## 8. Example: Using Custom Ollama Instance

### Scenario
User wants to use a local Ollama instance running on their network.

### Steps
1. Add API key with:
   - Provider: "ollama"
   - Name: "Local Ollama"
   - Endpoint URL: "http://192.168.1.100:11434"
   - API Key: (blank, Ollama doesn't require auth by default)
2. In agent chat, select model: "llama3"
3. Backend routes request to custom endpoint
4. Response returned from local Ollama

### Configuration
```json
{
  "id": "key_12345",
  "provider": "ollama",
  "name": "Local Ollama",
  "endpoint_url": "http://192.168.1.100:11434",
  "is_active": true,
  "metadata": {
    "available_models": ["llama3", "mistral", "codellama"],
    "default_model": "llama3",
    "timeout_seconds": 120
  }
}
```

## 9. Security Best Practices

### Encryption Key Management
```bash
# Generate strong encryption key
openssl rand -base64 32

# Set in docker-compose.unified.yml
environment:
  - MIMIR_ENCRYPTION_KEY=<generated-key>
```

### Plugin Signature Verification
```go
// Before loading plugin, verify signature
func verifyPluginSignature(filePath string) error {
    // Read plugin file
    // Check SHA256 hash against known hashes
    // Or verify GPG signature
    // Return error if invalid
}
```

### Access Control
- Require authentication for `/api/v1/settings/*` endpoints
- Only admin users can manage API keys and plugins
- Log all key creation/deletion events
- Audit plugin uploads

## 10. Testing the System

### Test API Key Creation
```bash
# Create OpenAI key
curl -X POST http://localhost:8080/api/v1/settings/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "name": "Test Key",
    "key_value": "sk-test123",
    "metadata": "{\"default_model\":\"gpt-4\"}"
  }'

# List keys
curl http://localhost:8080/api/v1/settings/api-keys

# Test key
curl -X POST http://localhost:8080/api/v1/settings/api-keys/key_12345/test
```

### Test Plugin Upload
```bash
# Upload plugin
curl -X POST http://localhost:8080/api/v1/settings/plugins/upload \
  -F "file=@my_plugin.so" \
  -F "name=My Custom Plugin" \
  -F "type=data_processing" \
  -F "description=Does cool stuff"

# List plugins
curl http://localhost:8080/api/v1/settings/plugins

# Enable/disable plugin
curl -X PUT http://localhost:8080/api/v1/settings/plugins/plugin_123 \
  -H "Content-Type: application/json" \
  -d '{"is_enabled": false}'
```

## 11. Next Steps

1. Implement backend handlers for API key management
2. Implement backend handlers for plugin management
3. Create frontend `/settings` page
4. Add API key form and list components
5. Add plugin upload UI
6. Integrate with agent chat for LLM provider selection
7. Test with real OpenAI/Anthropic keys
8. Document plugin development guide
9. Add plugin examples to `/examples` directory
10. Update deployment docs with encryption key setup

## 12. Plugin Development Guide (Future)

Users who want to create custom plugins will need:
- Go SDK with plugin interface definitions
- Build instructions for compiling plugins
- Example plugins as templates
- Testing framework for plugins
- Documentation on available hooks and APIs

This will be documented in `/docs/PLUGIN_DEVELOPMENT_GUIDE.md` (already exists).

# Mimir AIP Plugin Development Guide

This guide provides comprehensive information for developing plugins for the Mimir AIP platform.

## Table of Contents

1. [Plugin Architecture](#plugin-architecture)
2. [Plugin Types](#plugin-types)
3. [Creating Your First Plugin](#creating-your-first-plugin)
4. [Plugin Interface](#plugin-interface)
5. [Configuration](#configuration)
6. [Context Management](#context-management)
7. [Error Handling](#error-handling)
8. [Testing](#testing)
9. [Best Practices](#best-practices)
10. [Deployment](#deployment)

## Plugin Architecture

Mimir AIP plugins are self-contained modules that implement specific functionality. Each plugin must implement the `BasePlugin` interface and follow the plugin development patterns.

### Key Components

- **Plugin Interface**: Standardized contract for all plugins
- **Plugin Registry**: Manages plugin discovery and instantiation
- **Context System**: Shared data between pipeline steps
- **Configuration**: Type-safe configuration management
- **Logging**: Structured logging with context

## Plugin Types

### 1. Input Plugins
Collect data from external sources.

**Examples:**
- `Input.api` - HTTP API calls
- `Input.rss` - RSS feed parsing
- `Input.file` - File system operations

### 2. Data Processing Plugins
Transform, filter, and enrich data.

**Examples:**
- `Data_Processing.transform` - Data transformation operations
- `Data_Processing.filter` - Data filtering and validation
- `Data_Processing.aggregate` - Data aggregation and grouping

### 3. AI Models Plugins
Integrate with AI/ML models and services.

**Examples:**
- `AIModels.openai` - OpenAI API integration
- `AIModels.anthropic` - Anthropic Claude integration
- `AIModels.huggingface` - Hugging Face models

### 4. Output Plugins
Generate final outputs and reports.

**Examples:**
- `Output.json` - JSON file generation
- `Output.html` - HTML report generation
- `Output.database` - Database storage

## Creating Your First Plugin

### Step 1: Set Up Development Environment

```bash
# Create plugin directory
mkdir -p plugins/my_plugin

# Copy template
cp plugin_template.go plugins/my_plugin/my_plugin.go
```

### Step 2: Implement the Plugin

```go
package main

import (
    "context"
    "fmt"

    "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

type MyPlugin struct {
    name    string
    version string
}

func NewMyPlugin() *MyPlugin {
    return &MyPlugin{
        name:    "MyPlugin",
        version: "1.0.0",
    }
}

func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // Plugin implementation
    return pipelines.PluginContext{
        stepConfig.Output: "result",
    }, nil
}

func (p *MyPlugin) GetPluginType() string {
    return "Data_Processing" // or "Input", "AIModels", "Output"
}

func (p *MyPlugin) GetPluginName() string {
    return "my_plugin"
}

func (p *MyPlugin) ValidateConfig(config map[string]interface{}) error {
    // Configuration validation
    return nil
}
```

### Step 3: Register the Plugin

```go
// In your plugin's init() function or main registration
func init() {
    // Plugin will be auto-registered by the plugin registry
    // based on the GetPluginType() and GetPluginName() methods
}
```

### Step 4: Test the Plugin

```go
// Create a test pipeline
pipelines:
  - name: "Test Pipeline"
    steps:
      - name: "Test My Plugin"
        plugin: "Data_Processing.my_plugin"
        config:
          param1: "value1"
        output: "result"
```

## Plugin Interface

### BasePlugin Interface

```go
type BasePlugin interface {
    // ExecuteStep executes a single pipeline step
    ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext PluginContext) (PluginContext, error)

    // GetPluginType returns the plugin type (Input, Data_Processing, AIModels, Output)
    GetPluginType() string

    // GetPluginName returns the plugin name
    GetPluginName() string

    // ValidateConfig validates the plugin configuration
    ValidateConfig(config map[string]interface{}) error
}
```

### StepConfig Structure

```go
type StepConfig struct {
    Name   string                 // Step name
    Plugin string                 // Plugin reference (e.g., "Input.api")
    Config map[string]interface{} // Plugin-specific configuration
    Output string                 // Output key for context
}
```

### PluginContext

```go
type PluginContext map[string]interface{}
```

Context is a key-value store that persists data between pipeline steps.

## Configuration

### Configuration Best Practices

1. **Validate all inputs** in `ValidateConfig()`
2. **Provide sensible defaults** for optional parameters
3. **Use strong typing** where possible
4. **Document all configuration options**

### Example Configuration Validation

```go
func (p *MyPlugin) ValidateConfig(config map[string]interface{}) error {
    // Required parameters
    if config["api_key"] == nil {
        return fmt.Errorf("api_key is required")
    }

    // Optional parameters with defaults
    if config["timeout"] == nil {
        config["timeout"] = 30
    }

    // Type validation
    if timeout, ok := config["timeout"].(float64); ok {
        if timeout <= 0 {
            return fmt.Errorf("timeout must be positive")
        }
    }

    return nil
}
```

## Context Management

### Reading from Context

```go
func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // Read from previous step
    if previousData, exists := globalContext["previous_step_output"]; exists {
        // Process previousData
    }

    return pipelines.PluginContext{
        stepConfig.Output: "processed_result",
    }, nil
}
```

### Writing to Context

```go
func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    result := map[string]interface{}{
        "data": "processed_data",
        "timestamp": time.Now(),
        "plugin_version": p.version,
    }

    return pipelines.PluginContext{
        stepConfig.Output: result,
    }, nil
}
```

## Error Handling

### Error Handling Best Practices

1. **Wrap errors** with context
2. **Use structured error messages**
3. **Handle partial failures** gracefully
4. **Log errors** with appropriate levels

### Example Error Handling

```go
func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // Validate configuration
    if err := p.ValidateConfig(stepConfig.Config); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }

    // Attempt operation
    result, err := p.performOperation(ctx, stepConfig.Config)
    if err != nil {
        // Log the error
        utils.GetLogger().Error("Operation failed",
            fmt.Errorf("operation failed: %w", err),
            utils.String("plugin", p.GetPluginName()),
            utils.String("step", stepConfig.Name))

        return nil, fmt.Errorf("operation failed: %w", err)
    }

    return pipelines.PluginContext{
        stepConfig.Output: result,
    }, nil
}
```

## Testing

### Unit Testing

```go
func TestMyPlugin_ExecuteStep(t *testing.T) {
    plugin := NewMyPlugin()

    stepConfig := pipelines.StepConfig{
        Name:   "test_step",
        Plugin: "Data_Processing.my_plugin",
        Config: map[string]interface{}{
            "param1": "value1",
        },
        Output: "result",
    }

    context := pipelines.PluginContext{}
    result, err := plugin.ExecuteStep(context.Background(), stepConfig, context)

    assert.NoError(t, err)
    assert.Contains(t, result, "result")
}
```

### Integration Testing

```go
func TestMyPlugin_EndToEnd(t *testing.T) {
    // Test with actual pipeline execution
    pipeline := &PipelineConfig{
        Name: "Test Pipeline",
        Steps: []pipelines.StepConfig{
            {
                Name:   "My Step",
                Plugin: "Data_Processing.my_plugin",
                Config: map[string]interface{}{
                    "input": "test_data",
                },
                Output: "result",
            },
        },
    }

    result, err := ExecutePipeline(context.Background(), pipeline)
    assert.NoError(t, err)
    assert.True(t, result.Success)
}
```

## Best Practices

### 1. Plugin Design

- **Single Responsibility**: Each plugin should do one thing well
- **Idempotent Operations**: Plugins should be safe to run multiple times
- **Resource Management**: Properly clean up resources (files, connections, etc.)
- **Thread Safety**: Ensure plugins work correctly in concurrent environments

### 2. Configuration

- **Validate Early**: Validate configuration in `ValidateConfig()`
- **Document Parameters**: Comment all configuration options
- **Use Environment Variables**: For sensitive data like API keys
- **Provide Defaults**: Sensible defaults for optional parameters

### 3. Error Handling

- **Descriptive Messages**: Clear, actionable error messages
- **Context Preservation**: Include relevant context in errors
- **Recovery Strategies**: Handle transient failures gracefully
- **Logging**: Appropriate log levels for different error types

### 4. Performance

- **Efficient Processing**: Minimize memory usage and processing time
- **Streaming**: For large data sets, consider streaming processing
- **Caching**: Cache expensive operations when appropriate
- **Timeouts**: Implement appropriate timeouts for external operations

### 5. Documentation

- **README**: Include setup and usage instructions
- **Configuration**: Document all configuration parameters
- **Examples**: Provide working examples
- **Troubleshooting**: Common issues and solutions

## Deployment

### 1. Plugin Packaging

```bash
# Create plugin package
go mod init github.com/your-org/mimir-plugin-myplugin
go mod tidy

# Build plugin
go build -o myplugin.so -buildmode=plugin .
```

### 2. Plugin Directory Structure

```
plugins/
├── input/
│   ├── api/
│   │   ├── api.go
│   │   └── README.md
│   └── rss/
│       ├── rss.go
│       └── README.md
├── data_processing/
│   └── transform/
│       ├── transform.go
│       └── README.md
└── output/
    └── json/
        ├── json.go
        └── README.md
```

### 3. Configuration

```yaml
# config.yaml
plugins:
  directories:
    - "./plugins"
  auto_discovery: true
  timeout: 60
  max_concurrency: 10
```

### 4. Health Checks

```go
func (p *MyPlugin) HealthCheck() error {
    // Implement health check logic
    return nil
}
```

## Advanced Topics

### 1. Asynchronous Operations

```go
func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // For long-running operations, use goroutines
    resultChan := make(chan interface{}, 1)
    errorChan := make(chan error, 1)

    go func() {
        // Long-running operation
        result, err := p.longRunningOperation(ctx, stepConfig.Config)
        if err != nil {
            errorChan <- err
        } else {
            resultChan <- result
        }
    }()

    select {
    case result := <-resultChan:
        return pipelines.PluginContext{stepConfig.Output: result}, nil
    case err := <-errorChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 4. Plugin Testing and CI Integration

- Write unit and integration tests for all plugin logic.
- Use Go’s `testing` package and assert libraries.
- Example:

```go
func TestMyPlugin_ExecuteStep(t *testing.T) {
    plugin := NewMyPlugin()
    stepConfig := pipelines.StepConfig{
        Name:   "test_step",
        Plugin: "Data_Processing.my_plugin",
        Config: map[string]interface{}{"param1": "value1"},
        Output: "result",
    }
    context := pipelines.PluginContext{}
    result, err := plugin.ExecuteStep(context.Background(), stepConfig, context)
    assert.NoError(t, err)
    assert.Contains(t, result, "result")
}
```

- Integrate plugin tests into CI pipelines (GitHub Actions, etc.)
- Ensure all plugins pass tests before merging or releasing.

### 2. Streaming Data

```go
func (p *MyPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // For large datasets, process in chunks
    return p.processStreaming(ctx, stepConfig, globalContext)
}

func (p *MyPlugin) processStreaming(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
    // Implement streaming logic
    return pipelines.PluginContext{stepConfig.Output: "streaming_result"}, nil
}
```

### 3. Plugin Dependencies

```go
type MyPlugin struct {
    dependencies []string // List of required plugins
}

func (p *MyPlugin) GetDependencies() []string {
    return p.dependencies
}
```

## Troubleshooting

### Common Issues

1. **Plugin Not Found**
   - Check plugin directory structure
   - Verify `GetPluginType()` and `GetPluginName()` return correct values
   - Ensure plugin is properly compiled

2. **Configuration Errors**
   - Check `ValidateConfig()` implementation
   - Verify configuration parameter types
   - Review error messages for specific issues

3. **Context Issues**
   - Check input/output key names
   - Verify data types in context
   - Ensure proper context cleanup

4. **Performance Issues**
   - Implement timeouts for external operations
   - Use efficient data structures
   - Consider streaming for large datasets

### Debugging

```go
// Enable debug logging
logger := utils.GetLogger()
logger.SetLevel(utils.DEBUG)

// Add debug fields to operations
utils.GetLogger().Debug("Processing data",
    utils.String("plugin", p.GetPluginName()),
    utils.String("step", stepConfig.Name),
    utils.Int("data_size", len(data)))
```

## Contributing

1. Follow the plugin development guidelines
2. Include comprehensive tests
3. Update documentation
4. Follow the established code style
5. Add examples and usage documentation

## Support

For plugin development support:
- Check the examples in the `examples/` directory
- Review the template in `plugin_template.go`
- Consult the main documentation in `README.md`
- Review existing plugins for patterns and best practices

---

## Data Ingestion Plugins for Auto-Training

### Overview

As of the data ingestion refactor, Mimir-AIP supports **custom data ingestion plugins** that integrate seamlessly with the auto-training system. Any business can create a proprietary data format plugin that will automatically work with ML training, monitoring, and ontology-driven analytics.

### How It Works

1. **Create an Input Plugin** following the standard plugin interface
2. **Output the Required Format** (rows + columns structure)
3. **Register Your Plugin** with the plugin registry
4. **Use It Immediately** via the `/api/v1/auto-train-with-data` endpoint

The system automatically:
- Converts your plugin output to `UnifiedDataset`
- Infers column types (numeric, datetime, string, boolean)
- Computes statistics (min, max, mean, etc.)
- Detects time-series structure
- Creates monitoring jobs for time-series data
- Maps data to ontology properties

### Required Output Format

Your Input plugin **must** return data in this format:

```go
result := map[string]any{
    "columns": []string{"column1", "column2", "column3"},
    "rows": []map[string]any{
        {"column1": value1, "column2": value2, "column3": value3},
        {"column1": value4, "column2": value5, "column3": value6},
        // ... more rows
    },
    "row_count": 100,
    "column_count": 3,
    // Optional metadata
    "source_info": map[string]any{
        "custom_field": "custom_value",
    },
}
```

**Key Requirements:**
- `columns`: Array of column name strings
- `rows`: Array of maps (each map = one row of data)
- Row keys must match column names
- Values can be: `string`, `float64`, `int`, `bool`, `time.Time`, or `nil`

### Example: SAP ERP Plugin

Here's a complete example of a proprietary data format plugin:

```go
package Input

import (
    "context"
    "fmt"
    "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// SAPERPPlugin reads data from SAP ERP systems
type SAPERPPlugin struct {
    name    string
    version string
}

func NewSAPERPPlugin() *SAPERPPlugin {
    return &SAPERPPlugin{
        name:    "SAPERPPlugin",
        version: "1.0.0",
    }
}

func (p *SAPERPPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
    config := stepConfig.Config
    
    // Extract your custom configuration
    connectionString, ok := config["connection_string"].(string)
    if !ok {
        return nil, fmt.Errorf("connection_string is required")
    }
    
    tableName, ok := config["table"].(string)
    if !ok {
        return nil, fmt.Errorf("table is required")
    }
    
    // Connect to SAP ERP (your proprietary logic)
    sapClient := connectToSAP(connectionString)
    defer sapClient.Close()
    
    // Query data (your proprietary logic)
    sapRecords, err := sapClient.Query(tableName)
    if err != nil {
        return nil, fmt.Errorf("SAP query failed: %w", err)
    }
    
    // Convert to required format
    columns := []string{"order_id", "customer", "amount", "date"}
    rows := make([]map[string]any, 0, len(sapRecords))
    
    for _, record := range sapRecords {
        rows = append(rows, map[string]any{
            "order_id": record.OrderID,
            "customer": record.Customer,
            "amount":   float64(record.Amount),
            "date":     record.Date,
        })
    }
    
    // Return in the required format
    result := map[string]any{
        "columns":      columns,
        "rows":         rows,
        "row_count":    len(rows),
        "column_count": len(columns),
        "source_info": map[string]any{
            "sap_table":      tableName,
            "sap_connection": connectionString,
        },
    }
    
    context := pipelines.NewPluginContext()
    context.Set(stepConfig.Output, result)
    return context, nil
}

func (p *SAPERPPlugin) GetPluginType() string {
    return "Input"
}

func (p *SAPERPPlugin) GetPluginName() string {
    return "sap_erp"
}

func (p *SAPERPPlugin) ValidateConfig(config map[string]any) error {
    if _, ok := config["connection_string"].(string); !ok {
        return fmt.Errorf("connection_string is required")
    }
    if _, ok := config["table"].(string); !ok {
        return fmt.Errorf("table is required")
    }
    return nil
}

// Your proprietary connection logic
func connectToSAP(connectionString string) *SAPClient {
    // ... your SAP connection code
    return &SAPClient{}
}
```

### Registering Your Plugin

**Option 1: Programmatic Registration** (in `server.go`)

```go
func (s *Server) registerDefaultPlugins() {
    // ... existing plugins ...
    
    // Register your custom plugin
    sapPlugin := Input.NewSAPERPPlugin()
    if err := s.registry.RegisterPlugin(sapPlugin); err != nil {
        log.Printf("Failed to register SAP plugin: %v", err)
    }
}
```

**Option 2: Dynamic Loading** (place in plugins directory)

```go
// Mimir can auto-discover plugins in ./plugins/ directory
// Just compile your plugin as a .so file (Go plugin)
```

### Using Your Plugin

**Via API:**

```bash
curl -X POST http://localhost:8080/api/v1/auto-train-with-data \
  -H "Content-Type: application/json" \
  -d '{
    "ontology_id": "sales-analytics",
    "data_source": {
      "type": "plugin",
      "plugin_name": "sap_erp",
      "plugin_config": {
        "connection_string": "sap://server:port/client",
        "table": "VBAK",
        "filters": {
          "date_from": "2024-01-01",
          "date_to": "2024-12-31"
        }
      }
    },
    "enable_regression": true,
    "enable_classification": true,
    "enable_monitoring": true
  }'
```

**Response:**

```json
{
  "ontology_id": "sales-analytics",
  "models_created": 3,
  "monitoring_jobs_created": 1,
  "trained_models": [
    {
      "model_id": "auto_sales-analytics_amount_1234567890",
      "target_property": "amount",
      "model_type": "regression",
      "r2_score": 0.87,
      "sample_count": 1500
    }
  ],
  "monitoring_setup": {
    "job_id": "mon-job-uuid",
    "metrics_count": 3,
    "rules_created": ["threshold_amount", "threshold_orders"]
  }
}
```

### What Happens Automatically

1. **Data Extraction**: Your plugin executes with the provided config
2. **Validation**: System validates the output format (rows + columns)
3. **Type Inference**: Automatically detects numeric, datetime, string types
4. **Statistics**: Computes min, max, mean, sum for numeric columns
5. **Time-Series Detection**: Identifies datetime columns and time-based patterns
6. **Ontology Mapping**: Maps columns to ontology properties (if ontology exists)
7. **ML Training**: Trains regression/classification models on numeric targets
8. **Monitoring**: Creates monitoring jobs for time-series metrics
9. **Alerting**: Sets up threshold/anomaly detection rules

### Advanced Features

#### Type Hints

If automatic type inference doesn't work correctly, users can provide hints:

```json
{
  "data_source": {
    "type": "plugin",
    "plugin_name": "sap_erp",
    "plugin_config": {...},
    "options": {
      "type_hints": {
        "order_date": "datetime",
        "amount": "numeric",
        "status": "string"
      }
    }
  }
}
```

#### Column Selection

Users can select specific columns to analyze:

```json
{
  "data_source": {
    "type": "plugin",
    "plugin_name": "sap_erp",
    "plugin_config": {...},
    "options": {
      "selected_columns": ["amount", "quantity", "order_date"]
    }
  }
}
```

#### Row Limiting

For large datasets, users can limit rows:

```json
{
  "data_source": {
    "type": "plugin",
    "plugin_name": "sap_erp",
    "plugin_config": {...},
    "options": {
      "limit": 10000
    }
  }
}
```

### Testing Your Plugin

**1. Unit Test:**

```go
func TestSAPERPPlugin(t *testing.T) {
    plugin := NewSAPERPPlugin()
    
    config := pipelines.StepConfig{
        Config: map[string]any{
            "connection_string": "test://localhost",
            "table": "TEST_TABLE",
        },
        Output: "sap_data",
    }
    
    ctx := context.Background()
    result, err := plugin.ExecuteStep(ctx, config, pipelines.NewPluginContext())
    
    assert.NoError(t, err)
    
    data, ok := result.Get("sap_data")
    assert.True(t, ok)
    
    dataMap := data.(map[string]any)
    assert.Contains(t, dataMap, "rows")
    assert.Contains(t, dataMap, "columns")
}
```

**2. Integration Test:**

```bash
# Start server
./mimir-aip-server

# Test your plugin
curl -X POST http://localhost:8080/api/v1/auto-train-with-data \
  -d '{"ontology_id": "test", "data_source": {"type": "plugin", "plugin_name": "sap_erp", ...}}'
```

### Plugin Output Examples

**Valid Output:**

```go
✅ map[string]any{
    "columns": []string{"date", "revenue", "orders"},
    "rows": []map[string]any{
        {"date": "2024-01-01", "revenue": 15000.0, "orders": 120},
        {"date": "2024-01-02", "revenue": 18000.0, "orders": 145},
    },
    "row_count": 2,
    "column_count": 3,
}
```

**Invalid Output (will fail):**

```go
❌ []string{"data", "without", "structure"}
❌ map[string]any{"missing": "rows_field"}
❌ map[string]any{"rows": "not_an_array"}
```

### Error Handling

The system provides helpful error messages:

```json
{
  "error": "plugin execution failed: SAP connection timeout"
}
```

```json
{
  "error": "plugin output validation failed: missing 'rows' field"
}
```

```json
{
  "error": "plugin not found: Input.sap_erp - did you register it?"
}
```

### Best Practices

1. **Always validate configuration** in `ValidateConfig()`
2. **Return consistent column names** across queries
3. **Handle connection errors** gracefully
4. **Use numeric types** for values that should be analyzed (float64, int)
5. **Use string types** for categorical data
6. **Use time.Time or RFC3339 strings** for datetime columns
7. **Include metadata** in `source_info` for debugging
8. **Test with small datasets** first
9. **Document your plugin config** clearly
10. **Handle authentication** securely (use environment variables)

### Real-World Use Cases

**Manufacturing ERP:**
```bash
# Extract production data from proprietary MES system
"plugin_name": "mes_connector"
"plugin_config": {"line": "Assembly-1", "shift": "morning"}
```

**Financial Systems:**
```bash
# Pull transaction data from internal financial system
"plugin_name": "fintech_db"
"plugin_config": {"account": "12345", "start_date": "2024-01-01"}
```

**Healthcare EMR:**
```bash
# Extract patient data from EMR system
"plugin_name": "emr_hl7"
"plugin_config": {"facility": "hospital-1", "department": "cardiology"}
```

**IoT Platforms:**
```bash
# Get sensor data from proprietary IoT platform
"plugin_name": "iot_gateway"
"plugin_config": {"device_id": "sensor-001", "metrics": ["temp", "humidity"]}
```

### FAQ

**Q: Do I need to modify Mimir source code?**
A: No! Just create an Input plugin following the interface.

**Q: Can I use external libraries?**
A: Yes! Import any Go libraries you need.

**Q: What if my data format is complex/nested?**
A: Flatten it in your plugin before returning. The system expects tabular rows.

**Q: Can I return millions of rows?**
A: Yes, but consider implementing pagination in your plugin and letting users use the `limit` option.

**Q: How do I handle authentication?**
A: Pass credentials via `plugin_config` or read from environment variables.

**Q: Will my plugin work with pipelines?**
A: Yes! You can use it in pipeline YAML files or via the data adapter system.

**Q: Can I create a plugin without Go knowledge?**
A: The plugin must be written in Go, but you can wrap external CLI tools or APIs.

### Summary

✅ **Create Plugin**: Implement Input plugin interface  
✅ **Output Format**: Return `{columns: [...], rows: [...]}`  
✅ **Register**: Add to plugin registry  
✅ **Use**: Call `/auto-train-with-data` with `type: "plugin"`  
✅ **Benefit**: Automatic ML training, monitoring, type inference, statistics  

Your proprietary data format is now a **first-class citizen** in Mimir-AIP! 🎉
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
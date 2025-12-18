package ml

import (
	"context"
	"fmt"
	"log"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// DataAdapter is the interface that all data source adapters must implement
// This allows for plugin-based extensibility - anyone can create a custom adapter
type DataAdapter interface {
	// GetName returns the unique name of this adapter (e.g., "csv", "excel", "custom_erp")
	GetName() string

	// GetDescription returns a human-readable description
	GetDescription() string

	// Supports returns true if this adapter can handle the given source config
	Supports(config DataSourceConfig) bool

	// Extract converts the source data into a UnifiedDataset
	Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error)

	// ValidateConfig validates the source configuration before extraction
	ValidateConfig(config DataSourceConfig) error
}

// DataAdapterRegistry manages all registered data adapters
// This is the plugin system for data ingestion
type DataAdapterRegistry struct {
	adapters map[string]DataAdapter
}

var globalAdapterRegistry *DataAdapterRegistry
var globalPluginRegistry *pipelines.PluginRegistry

// InitializeGlobalAdapterRegistry initializes the global adapter registry with the plugin registry
// This should be called once during server startup, after plugins are registered
func InitializeGlobalAdapterRegistry(pluginRegistry *pipelines.PluginRegistry) {
	if globalAdapterRegistry != nil {
		log.Println("⚠️  Global adapter registry already initialized")
		return
	}

	globalPluginRegistry = pluginRegistry
	globalAdapterRegistry = NewDataAdapterRegistry()

	// Register built-in adapters
	registerBuiltInAdapters(globalAdapterRegistry, pluginRegistry)
}

// GetGlobalAdapterRegistry returns the global adapter registry (singleton)
func GetGlobalAdapterRegistry() *DataAdapterRegistry {
	if globalAdapterRegistry == nil {
		log.Println("⚠️  Global adapter registry not initialized! Call InitializeGlobalAdapterRegistry first")
		// Return empty registry as fallback
		return NewDataAdapterRegistry()
	}
	return globalAdapterRegistry
}

// NewDataAdapterRegistry creates a new adapter registry
func NewDataAdapterRegistry() *DataAdapterRegistry {
	return &DataAdapterRegistry{
		adapters: make(map[string]DataAdapter),
	}
}

// RegisterAdapter registers a new data adapter
// This is how custom plugins can add their own data sources
func (r *DataAdapterRegistry) RegisterAdapter(adapter DataAdapter) error {
	name := adapter.GetName()
	if name == "" {
		return fmt.Errorf("adapter name cannot be empty")
	}

	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter already registered: %s", name)
	}

	r.adapters[name] = adapter
	log.Printf("✅ Registered data adapter: %s - %s", name, adapter.GetDescription())
	return nil
}

// GetAdapter retrieves an adapter by name
func (r *DataAdapterRegistry) GetAdapter(name string) (DataAdapter, error) {
	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter not found: %s", name)
	}
	return adapter, nil
}

// FindAdapter finds the appropriate adapter for a given source config
// It iterates through all registered adapters and returns the first one that supports the config
func (r *DataAdapterRegistry) FindAdapter(config DataSourceConfig) (DataAdapter, error) {
	// First try explicit type match
	if config.Type != "" {
		if adapter, exists := r.adapters[config.Type]; exists && adapter.Supports(config) {
			return adapter, nil
		}
	}

	// Fall back to checking all adapters
	for _, adapter := range r.adapters {
		if adapter.Supports(config) {
			return adapter, nil
		}
	}

	return nil, fmt.Errorf("no adapter found for data source type: %s", config.Type)
}

// ListAdapters returns all registered adapters
func (r *DataAdapterRegistry) ListAdapters() []DataAdapter {
	adapters := make([]DataAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}

// ExtractData is the main entry point for data extraction
// It automatically finds the right adapter and extracts the data
func (r *DataAdapterRegistry) ExtractData(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	// Find appropriate adapter
	adapter, err := r.FindAdapter(config)
	if err != nil {
		return nil, err
	}

	// Validate configuration
	if err := adapter.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Extract data
	log.Printf("🔄 Extracting data using adapter: %s", adapter.GetName())
	dataset, err := adapter.Extract(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Validate result
	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	log.Printf("✅ Extracted dataset: %d rows, %d columns from %s",
		dataset.RowCount, dataset.ColumnCount, dataset.Source)

	return dataset, nil
}

// BaseDataAdapter provides common functionality for adapters
// Custom adapters can embed this to get default implementations
type BaseDataAdapter struct {
	name        string
	description string
}

func NewBaseDataAdapter(name, description string) BaseDataAdapter {
	return BaseDataAdapter{
		name:        name,
		description: description,
	}
}

func (b BaseDataAdapter) GetName() string {
	return b.name
}

func (b BaseDataAdapter) GetDescription() string {
	return b.description
}

// PipelineDataAdapter extracts data from pipeline execution results
type PipelineDataAdapter struct {
	BaseDataAdapter
	storage interface{} // PersistenceBackend for querying pipeline results
}

func NewPipelineDataAdapter(storage interface{}) *PipelineDataAdapter {
	return &PipelineDataAdapter{
		BaseDataAdapter: NewBaseDataAdapter("pipeline", "Extract data from pipeline execution outputs"),
		storage:         storage,
	}
}

func (p *PipelineDataAdapter) Supports(config DataSourceConfig) bool {
	return config.Type == "pipeline" && config.PipelineID != ""
}

func (p *PipelineDataAdapter) ValidateConfig(config DataSourceConfig) error {
	if config.PipelineID == "" {
		return fmt.Errorf("pipeline_id is required")
	}
	if config.OutputKey == "" {
		return fmt.Errorf("output_key is required (which pipeline output to use)")
	}
	return nil
}

func (p *PipelineDataAdapter) Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	// This would execute the pipeline and extract the specified output
	// For now, return a placeholder implementation
	return nil, fmt.Errorf("pipeline adapter not fully implemented yet - execute pipeline '%s' and extract output key '%s'",
		config.PipelineID, config.OutputKey)
}

// PluginDataAdapter allows custom plugins to provide data
// This is the ultimate extensibility - businesses can write their own adapters
type PluginDataAdapter struct {
	BaseDataAdapter
	pluginRegistry *pipelines.PluginRegistry
}

func NewPluginDataAdapter(registry *pipelines.PluginRegistry) *PluginDataAdapter {
	return &PluginDataAdapter{
		BaseDataAdapter: NewBaseDataAdapter("plugin", "Extract data from custom input plugins"),
		pluginRegistry:  registry,
	}
}

func (p *PluginDataAdapter) Supports(config DataSourceConfig) bool {
	return config.Type == "plugin" || config.PluginName != ""
}

func (p *PluginDataAdapter) ValidateConfig(config DataSourceConfig) error {
	if config.PluginName == "" {
		return fmt.Errorf("plugin_name is required")
	}
	return nil
}

func (p *PluginDataAdapter) Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	// Get the plugin from registry
	pluginType := "Input" // Input plugins handle data ingestion
	plugin, err := p.pluginRegistry.GetPlugin(pluginType, config.PluginName)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: Input.%s - %w", config.PluginName, err)
	}

	// Execute plugin with config
	stepConfig := pipelines.StepConfig{
		Name:   "data_extraction",
		Plugin: fmt.Sprintf("%s.%s", pluginType, config.PluginName),
		Config: config.PluginConfig,
		Output: "extracted_data",
	}

	pluginContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, pluginContext)
	if err != nil {
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	// Convert plugin output to UnifiedDataset
	outputData, _ := result.Get("extracted_data")
	return p.convertPluginOutputToDataset(outputData, config.PluginName)
}

func (p *PluginDataAdapter) convertPluginOutputToDataset(pluginOutput interface{}, pluginName string) (*UnifiedDataset, error) {
	// Convert plugin output (which should have rows/columns structure) to UnifiedDataset
	outputMap, ok := pluginOutput.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("plugin output is not a map")
	}

	// Extract rows
	rowsInterface, hasRows := outputMap["rows"]
	if !hasRows {
		return nil, fmt.Errorf("plugin output missing 'rows' field")
	}

	rows, ok := rowsInterface.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("plugin output 'rows' field is not a []map[string]interface{}")
	}

	// Extract or infer columns
	var columnNames []string
	if colsInterface, hasCols := outputMap["columns"]; hasCols {
		if cols, ok := colsInterface.([]string); ok {
			columnNames = cols
		}
	}

	// If no columns specified, infer from first row
	if len(columnNames) == 0 && len(rows) > 0 {
		for colName := range rows[0] {
			columnNames = append(columnNames, colName)
		}
	}

	// Build dataset
	dataset := NewUnifiedDataset(fmt.Sprintf("plugin:%s", pluginName))
	dataset.Rows = rows
	dataset.RowCount = len(rows)
	dataset.ColumnCount = len(columnNames)

	// Build column metadata
	dataset.Columns = make([]ColumnMetadata, len(columnNames))
	for i, colName := range columnNames {
		dataset.Columns[i] = ColumnMetadata{
			Name:     colName,
			Index:    i,
			DataType: "string", // Default, would need type inference
		}
	}

	// Store original plugin output in metadata
	dataset.SourceInfo["plugin_output"] = outputMap
	dataset.SourceInfo["plugin_name"] = pluginName

	return dataset, nil
}

// registerBuiltInAdapters registers all built-in adapters
// This should be called after the plugin registry has been populated with Input plugins
func registerBuiltInAdapters(adapterRegistry *DataAdapterRegistry, pluginRegistry *pipelines.PluginRegistry) {
	// Register CSV adapter (uses Input.csv plugin)
	if err := adapterRegistry.RegisterAdapter(NewCSVDataAdapter(pluginRegistry)); err != nil {
		log.Printf("⚠️  Failed to register CSV adapter: %v", err)
	}

	// Register JSON adapter (native implementation, no plugin dependency)
	if err := adapterRegistry.RegisterAdapter(NewJSONDataAdapter()); err != nil {
		log.Printf("⚠️  Failed to register JSON adapter: %v", err)
	}

	// Register Excel adapter (uses Input.excel plugin)
	if err := adapterRegistry.RegisterAdapter(NewExcelDataAdapter(pluginRegistry)); err != nil {
		log.Printf("⚠️  Failed to register Excel adapter: %v", err)
	}

	// Register plugin adapter (allows custom Input plugins to be used as data sources)
	if err := adapterRegistry.RegisterAdapter(NewPluginDataAdapter(pluginRegistry)); err != nil {
		log.Printf("⚠️  Failed to register Plugin adapter: %v", err)
	}

	log.Println("📦 Built-in data adapters registration complete (CSV, JSON, Excel, Plugin)")
}

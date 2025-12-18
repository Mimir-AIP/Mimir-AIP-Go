// Custom ERP Plugin Example
// This example demonstrates how to create a data ingestion plugin for proprietary formats
// that seamlessly integrates with Mimir-AIP's auto-training system

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// CustomERPPlugin demonstrates how to create a plugin for proprietary data formats
// This example shows integration with a fictional ERP system
type CustomERPPlugin struct {
	name    string
	version string
}

func NewCustomERPPlugin() *CustomERPPlugin {
	return &CustomERPPlugin{
		name:    "CustomERPPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads data from your proprietary ERP system
func (p *CustomERPPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Extract your custom configuration
	apiEndpoint, ok := config["api_endpoint"].(string)
	if !ok || apiEndpoint == "" {
		return nil, fmt.Errorf("api_endpoint is required in config")
	}

	apiKey, ok := config["api_key"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key is required in config")
	}

	reportType, ok := config["report_type"].(string)
	if !ok {
		reportType = "sales" // default
	}

	// Validate config
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Connect to your proprietary ERP system
	// In a real implementation, you would:
	// 1. Make HTTP requests to your ERP API
	// 2. Authenticate with your credentials
	// 3. Parse the proprietary response format
	// 4. Transform it to the required format below

	// Example: Simulated ERP data extraction
	erpData, err := p.fetchDataFromERP(apiEndpoint, apiKey, reportType)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ERP data: %w", err)
	}

	// IMPORTANT: Convert your proprietary format to the required format
	// Your plugin MUST return this structure for auto-training to work
	result := map[string]any{
		// Column names (required)
		"columns": erpData.Columns,

		// Data rows (required) - each row is a map with column names as keys
		"rows": erpData.Rows,

		// Row and column counts (required)
		"row_count":    len(erpData.Rows),
		"column_count": len(erpData.Columns),

		// Optional metadata (helps with debugging)
		"source_info": map[string]any{
			"erp_endpoint":  apiEndpoint,
			"report_type":   reportType,
			"extracted_at":  time.Now().Format(time.RFC3339),
			"erp_version":   "2.5.1",
			"record_source": "production_database",
		},
	}

	// Return the result
	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// ERPData represents data from your ERP system
type ERPData struct {
	Columns []string
	Rows    []map[string]any
}

// fetchDataFromERP simulates fetching data from a proprietary ERP system
// Replace this with your actual ERP integration logic
func (p *CustomERPPlugin) fetchDataFromERP(endpoint, apiKey, reportType string) (*ERPData, error) {
	// In a real implementation, you would:
	// 1. Make authenticated HTTP requests
	// 2. Parse XML/JSON/Binary responses
	// 3. Handle pagination
	// 4. Transform proprietary formats

	// Example: Simulated ERP response
	// This would come from your actual ERP API
	switch reportType {
	case "sales":
		return &ERPData{
			Columns: []string{"date", "region", "revenue", "units_sold", "product_category"},
			Rows: []map[string]any{
				{
					"date":             "2024-01-01",
					"region":           "North",
					"revenue":          125000.50,
					"units_sold":       1250.0,
					"product_category": "Electronics",
				},
				{
					"date":             "2024-01-02",
					"region":           "South",
					"revenue":          98000.25,
					"units_sold":       890.0,
					"product_category": "Electronics",
				},
				{
					"date":             "2024-01-03",
					"region":           "East",
					"revenue":          156000.75,
					"units_sold":       1680.0,
					"product_category": "Home Goods",
				},
			},
		}, nil

	case "inventory":
		return &ERPData{
			Columns: []string{"product_id", "warehouse", "quantity", "reorder_point", "last_restocked"},
			Rows: []map[string]any{
				{
					"product_id":     "PROD-001",
					"warehouse":      "WH-NORTH",
					"quantity":       450.0,
					"reorder_point":  100.0,
					"last_restocked": "2024-01-15",
				},
				{
					"product_id":     "PROD-002",
					"warehouse":      "WH-SOUTH",
					"quantity":       89.0,
					"reorder_point":  150.0,
					"last_restocked": "2024-01-10",
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}
}

// GetPluginType returns the plugin type (must be "Input" for data ingestion)
func (p *CustomERPPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the unique name for your plugin
func (p *CustomERPPlugin) GetPluginName() string {
	return "custom_erp"
}

// ValidateConfig validates the plugin configuration
func (p *CustomERPPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["api_endpoint"].(string); !ok {
		return fmt.Errorf("api_endpoint is required and must be a string")
	}

	if _, ok := config["api_key"].(string); !ok {
		return fmt.Errorf("api_key is required and must be a string")
	}

	// Validate report type if provided
	if reportType, ok := config["report_type"].(string); ok {
		validTypes := []string{"sales", "inventory", "customers", "orders"}
		valid := false
		for _, vt := range validTypes {
			if reportType == vt {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("report_type must be one of: sales, inventory, customers, orders")
		}
	}

	return nil
}

// Example usage in your application
//
// Register the plugin:
//   erpPlugin := examples.NewCustomERPPlugin()
//   registry.RegisterPlugin(erpPlugin)
//
// Use via API:
//   curl -X POST http://localhost:8080/api/v1/auto-train-with-data \
//     -d '{
//       "ontology_id": "sales-analytics",
//       "data_source": {
//         "type": "plugin",
//         "plugin_name": "custom_erp",
//         "plugin_config": {
//           "api_endpoint": "https://erp.company.com/api",
//           "api_key": "your-secret-key",
//           "report_type": "sales"
//         }
//       },
//       "enable_regression": true,
//       "enable_monitoring": true
//     }'
//
// The system will automatically:
// - Extract your ERP data using this plugin
// - Infer column types (revenue = numeric, date = datetime, region = string)
// - Compute statistics (min/max/mean for revenue and units_sold)
// - Detect time-series patterns (daily sales data)
// - Train ML models to predict revenue
// - Create monitoring jobs to track sales metrics
// - Set up alerts for anomalies

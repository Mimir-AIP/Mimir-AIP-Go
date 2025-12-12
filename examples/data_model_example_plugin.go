// Data Model Example Plugin
// This example demonstrates the new generalized data model with JSON, Binary, and Time Series data

package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// DataModelExamplePlugin demonstrates the new data model capabilities
type DataModelExamplePlugin struct {
	name    string
	version string
}

// NewDataModelExamplePlugin creates a new example plugin instance
func NewDataModelExamplePlugin() *DataModelExamplePlugin {
	return &DataModelExamplePlugin{
		name:    "DataModelExamplePlugin",
		version: "1.0.0",
	}
}

// ExecuteStep demonstrates various data types and operations
func (p *DataModelExamplePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("configuration validation failed: %w", err)
	}

	// Get operation type
	operation, _ := config["operation"].(string)

	// Create result context
	result := pipelines.NewPluginContext()

	switch operation {
	case "create_json_data":
		return p.createJSONDataExample(result)
	case "create_binary_data":
		return p.createBinaryDataExample(result)
	case "create_time_series":
		return p.createTimeSeriesExample(result)
	case "create_image_data":
		return p.createImageDataExample(result)
	case "process_mixed_data":
		return p.processMixedDataExample(globalContext, result)
	case "serialize_context":
		return p.serializeContextExample(globalContext, result)
	default:
		return pipelines.NewPluginContext(), fmt.Errorf("unsupported operation: %s", operation)
	}
}

// createJSONDataExample demonstrates JSON data handling
func (p *DataModelExamplePlugin) createJSONDataExample(result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Create structured JSON data
	userData := pipelines.NewJSONData(map[string]any{
		"user_id": 12345,
		"name":    "John Doe",
		"email":   "john.doe@example.com",
		"preferences": map[string]any{
			"theme":         "dark",
			"language":      "en",
			"notifications": true,
		},
		"tags":       []string{"premium", "active", "verified"},
		"created_at": time.Now().Format(time.RFC3339),
	})

	// Create product data
	productData := pipelines.NewJSONData(map[string]any{
		"product_id": 789,
		"name":       "Advanced Analytics Platform",
		"category":   "Software",
		"price":      299.99,
		"features":   []string{"AI", "ML", "Real-time", "Scalable"},
		"metadata": map[string]any{
			"version": "2.1.0",
			"license": "Enterprise",
		},
	})

	result.SetTyped("user_profile", userData)
	result.SetTyped("product_info", productData)

	fmt.Println("Created JSON data examples")
	return result, nil
}

// createBinaryDataExample demonstrates binary data handling
func (p *DataModelExamplePlugin) createBinaryDataExample(result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Create sample binary data (could be any binary content)
	sampleData := make([]byte, 1024)
	for i := range sampleData {
		sampleData[i] = byte(rand.Intn(256))
	}

	binaryData := pipelines.NewBinaryData(sampleData, "application/octet-stream")

	// Create CSV data as binary
	csvContent := "id,name,value,timestamp\n1,metric1,42.5,2024-01-01T10:00:00Z\n2,metric2,38.7,2024-01-01T10:05:00Z\n"
	csvData := pipelines.NewBinaryData([]byte(csvContent), "text/csv")

	result.SetTyped("random_binary", binaryData)
	result.SetTyped("csv_data", csvData)

	fmt.Println("Created binary data examples")
	return result, nil
}

// createTimeSeriesExample demonstrates time series data
func (p *DataModelExamplePlugin) createTimeSeriesExample(result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	tsData := pipelines.NewTimeSeriesData()

	// Add metadata
	tsData.Metadata = map[string]any{
		"metric_name": "cpu_usage",
		"unit":        "percentage",
		"server":      "web-server-01",
		"interval":    "5s",
	}

	// Generate sample time series data
	baseTime := time.Now().Add(-time.Hour)
	for i := 0; i < 60; i++ { // 5 minutes of data (5-second intervals)
		timestamp := baseTime.Add(time.Duration(i*5) * time.Second)
		value := 40.0 + rand.Float64()*40.0 // Random value between 40-80

		tsData.AddPoint(timestamp, value, map[string]string{
			"data_center": "us-east-1",
			"instance_id": "i-1234567890abcdef0",
		})
	}

	result.SetTyped("cpu_metrics", tsData)

	fmt.Println("Created time series data example")
	return result, nil
}

// createImageDataExample demonstrates image data handling
func (p *DataModelExamplePlugin) createImageDataExample(result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Create a simple image programmatically
	width, height := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Encode to PNG
	var buffer bytes.Buffer
	png.Encode(&buffer, img)

	imageData := pipelines.NewImageData(buffer.Bytes(), "image/png", "png", width, height)

	result.SetTyped("generated_image", imageData)

	fmt.Println("Created image data example")
	return result, nil
}

// processMixedDataExample demonstrates processing multiple data types
func (p *DataModelExamplePlugin) processMixedDataExample(input *pipelines.PluginContext, result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Process JSON data
	if userData, exists := input.GetTyped("user_profile"); exists {
		if jsonData, ok := userData.(*pipelines.JSONData); ok {
			// Extract and transform user data
			if name, exists := jsonData.Content["name"]; exists {
				processedName := fmt.Sprintf("PROCESSED_%s", name)
				jsonData.Content["processed_name"] = processedName
			}
			result.SetTyped("processed_user", jsonData)
		}
	}

	// Process time series data
	if tsData, exists := input.GetTyped("cpu_metrics"); exists {
		if timeSeries, ok := tsData.(*pipelines.TimeSeriesData); ok {
			// Calculate statistics
			if len(timeSeries.Points) > 0 {
				var sum float64
				min := timeSeries.Points[0].Value.(float64)
				max := min

				for _, point := range timeSeries.Points {
					if value, ok := point.Value.(float64); ok {
						sum += value
						if value < min {
							min = value
						}
						if value > max {
							max = value
						}
					}
				}

				avg := sum / float64(len(timeSeries.Points))

				// Create statistics JSON
				stats := pipelines.NewJSONData(map[string]any{
					"count":   len(timeSeries.Points),
					"average": avg,
					"minimum": min,
					"maximum": max,
					"metric":  timeSeries.Metadata["metric_name"],
					"server":  timeSeries.Metadata["server"],
				})

				result.SetTyped("cpu_statistics", stats)
			}
		}
	}

	// Process binary data
	if binData, exists := input.GetTyped("csv_data"); exists {
		if binary, ok := binData.(*pipelines.BinaryData); ok {
			// Convert CSV to JSON structure
			csvContent := string(binary.Content)

			// Simple CSV parsing (in real implementation, use a proper CSV library)
			lines := []string{}
			currentLine := ""
			for _, char := range csvContent {
				if char == '\n' {
					lines = append(lines, currentLine)
					currentLine = ""
				} else {
					currentLine += string(char)
				}
			}
			if currentLine != "" {
				lines = append(lines, currentLine)
			}

			// Convert to JSON structure
			var records []map[string]any
			if len(lines) > 1 {
				headers := []string{}
				// Parse header
				for i, char := range lines[0] {
					if char == ',' || i == len(lines[0])-1 {
						if i == len(lines[0])-1 {
							headers = append(headers, lines[0])
						}
						break
					}
				}

				// For simplicity, create a basic structure
				records = append(records, map[string]any{
					"total_lines": len(lines),
					"headers":     []string{"id", "name", "value", "timestamp"},
					"sample_data": lines[0],
				})
			}

			jsonRecords := pipelines.NewJSONData(map[string]any{
				"records":    records,
				"total_size": len(csvContent),
			})

			result.SetTyped("parsed_csv", jsonRecords)
		}
	}

	fmt.Println("Processed mixed data types")
	return result, nil
}

// serializeContextExample demonstrates context serialization
func (p *DataModelExamplePlugin) serializeContextExample(input *pipelines.PluginContext, result *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Serialize the input context
	serialized, err := pipelines.ContextSerializerInstance.SerializeContext(input)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to serialize context: %w", err)
	}

	// Store serialized data as binary
	serializedData := pipelines.NewBinaryData(serialized, "application/json")
	result.SetTyped("serialized_context", serializedData)

	// Deserialize back to verify
	deserialized, err := pipelines.ContextSerializerInstance.DeserializeContext(serialized)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to deserialize context: %w", err)
	}

	// Store the size of deserialized context as metadata
	result.SetMetadata("deserialized_size", deserialized.Size())

	fmt.Printf("Serialized context size: %d bytes\n", len(serialized))
	return result, nil
}

// GetPluginType returns the plugin type
func (p *DataModelExamplePlugin) GetPluginType() string {
	return "Data_Processing"
}

// GetPluginName returns the plugin name
func (p *DataModelExamplePlugin) GetPluginName() string {
	return "data_model_example"
}

// ValidateConfig validates the plugin configuration
func (p *DataModelExamplePlugin) ValidateConfig(config map[string]any) error {
	operation, exists := config["operation"]
	if !exists {
		return fmt.Errorf("operation is required")
	}

	if op, ok := operation.(string); !ok || op == "" {
		return fmt.Errorf("operation must be a non-empty string")
	}

	return nil
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "Data Model Demonstration Pipeline"
//     steps:
//       - name: "Create JSON Data"
//         plugin: "Data_Processing.data_model_example"
//         config:
//           operation: "create_json_data"
//         output: "json_result"
//       - name: "Create Time Series"
//         plugin: "Data_Processing.data_model_example"
//         config:
//           operation: "create_time_series"
//         output: "ts_result"
//       - name: "Create Binary Data"
//         plugin: "Data_Processing.data_model_example"
//         config:
//           operation: "create_binary_data"
//         output: "binary_result"
//       - name: "Process Mixed Data"
//         plugin: "Data_Processing.data_model_example"
//         config:
//           operation: "process_mixed_data"
//           input: ["json_result", "ts_result", "binary_result"]
//         output: "processed_result"
//       - name: "Serialize Context"
//         plugin: "Data_Processing.data_model_example"
//         config:
//           operation: "serialize_context"
//           input: "processed_result"
//         output: "serialized_result"

func main() {
	fmt.Println("Data Model Example Plugin")
	fmt.Println("This is an example of how to use the new data model with various data types")
	fmt.Println("To use this plugin, you would register it with the Mimir AIP framework")
	fmt.Println("See the comments in the code for usage examples")

	// Example usage demonstration
	plugin := NewDataModelExamplePlugin()
	fmt.Printf("Plugin: %s v%s\n", plugin.name, plugin.version)
}

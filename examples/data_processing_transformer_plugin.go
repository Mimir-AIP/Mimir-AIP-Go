// Data Processing Transformer Plugin Example
// This example shows how to create a Data_Processing plugin for Mimir AIP

package main

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// TransformerPlugin implements data transformation operations
type TransformerPlugin struct {
	name    string
	version string
}

// NewTransformerPlugin creates a new transformer plugin instance
func NewTransformerPlugin() *TransformerPlugin {
	return &TransformerPlugin{
		name:    "TransformerPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep performs data transformation operations
func (p *TransformerPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Get input data
	inputData := p.getInputData(config, globalContext)

	// Get operation
	operation, _ := config["operation"].(string)

	// Perform transformation
	result, err := p.transformData(inputData, operation, config)
	if err != nil {
		return nil, fmt.Errorf("data transformation failed: %w", err)
	}

	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// GetPluginType returns the plugin type
func (p *TransformerPlugin) GetPluginType() string {
	return "Data_Processing"
}

// GetPluginName returns the plugin name
func (p *TransformerPlugin) GetPluginName() string {
	return "transform"
}

// ValidateConfig validates the plugin configuration
func (p *TransformerPlugin) ValidateConfig(config map[string]any) error {
	operation, exists := config["operation"]
	if !exists {
		return fmt.Errorf("operation is required")
	}

	if op, ok := operation.(string); !ok || op == "" {
		return fmt.Errorf("operation must be a non-empty string")
	}

	return nil
}

// getInputData extracts input data from context or config
func (p *TransformerPlugin) getInputData(config map[string]any, context *pipelines.PluginContext) any {
	// Check if input is specified in config
	if inputKey, ok := config["input"].(string); ok {
		if data, exists := context.Get(inputKey); exists {
			return data
		}
	}

	// Check for direct data in config
	if data, exists := config["data"]; exists {
		return data
	}

	// Default to empty string
	return ""
}

// transformData performs the actual transformation
func (p *TransformerPlugin) transformData(data any, operation string, config map[string]any) (any, error) {
	switch operation {
	case "uppercase":
		return p.transformUppercase(data)
	case "lowercase":
		return p.transformLowercase(data)
	case "extract_numbers":
		return p.extractNumbers(data)
	case "remove_duplicates":
		return p.removeDuplicates(data)
	case "filter":
		return p.filterData(data, config)
	case "extract_pattern":
		return p.extractPattern(data, config)
	case "split":
		return p.splitData(data, config)
	case "join":
		return p.joinData(data, config)
	case "count":
		return p.countData(data)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// transformUppercase converts strings to uppercase
func (p *TransformerPlugin) transformUppercase(data any) (any, error) {
	if str, ok := data.(string); ok {
		return strings.ToUpper(str), nil
	}
	return data, nil
}

// transformLowercase converts strings to lowercase
func (p *TransformerPlugin) transformLowercase(data any) (any, error) {
	if str, ok := data.(string); ok {
		return strings.ToLower(str), nil
	}
	return data, nil
}

// extractNumbers extracts all numbers from a string
func (p *TransformerPlugin) extractNumbers(data any) (any, error) {
	if str, ok := data.(string); ok {
		re := regexp.MustCompile(`\d+`)
		matches := re.FindAllString(str, -1)

		numbers := make([]int, len(matches))
		for i, match := range matches {
			if num, err := strconv.Atoi(match); err == nil {
				numbers[i] = num
			}
		}
		return numbers, nil
	}
	return data, nil
}

// removeDuplicates removes duplicate items from arrays/slices
func (p *TransformerPlugin) removeDuplicates(data any) (any, error) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return data, nil
	}

	seen := make(map[any]bool)
	result := reflect.MakeSlice(v.Type(), 0, v.Len())

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if !seen[item] {
			seen[item] = true
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface(), nil
}

// filterData filters data based on criteria
func (p *TransformerPlugin) filterData(data any, config map[string]any) (any, error) {
	// Simple filtering example - can be extended
	if str, ok := data.(string); ok {
		if pattern, ok := config["pattern"].(string); ok {
			return strings.Contains(str, pattern), nil
		}
	}
	return data, nil
}

// extractPattern extracts data matching a regex pattern
func (p *TransformerPlugin) extractPattern(data any, config map[string]any) (any, error) {
	pattern, ok := config["pattern"].(string)
	if !ok {
		return data, fmt.Errorf("pattern is required for extract_pattern operation")
	}

	if str, ok := data.(string); ok {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}

		matches := re.FindAllString(str, -1)
		return matches, nil
	}

	return data, nil
}

// splitData splits strings into arrays
func (p *TransformerPlugin) splitData(data any, config map[string]any) (any, error) {
	separator := ","
	if sep, ok := config["separator"].(string); ok {
		separator = sep
	}

	if str, ok := data.(string); ok {
		return strings.Split(str, separator), nil
	}

	return data, nil
}

// joinData joins arrays into strings
func (p *TransformerPlugin) joinData(data any, config map[string]any) (any, error) {
	separator := ","
	if sep, ok := config["separator"].(string); ok {
		separator = sep
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		var parts []string
		for i := 0; i < v.Len(); i++ {
			parts = append(parts, fmt.Sprintf("%v", v.Index(i).Interface()))
		}
		return strings.Join(parts, separator), nil
	}

	return data, nil
}

// countData counts items in arrays or characters in strings
func (p *TransformerPlugin) countData(data any) (any, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return v.Len(), nil
	}

	if str, ok := data.(string); ok {
		return len(str), nil
	}

	return 1, nil
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "Data Transformation Pipeline"
//     steps:
//       - name: "Fetch Data"
//         plugin: "Input.api"
//         config:
//           url: "https://api.example.com/data"
//         output: "raw_data"
//       - name: "Extract Titles"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "extract_pattern"
//           pattern: "<title>(.*?)</title>"
//           input: "raw_data"
//         output: "titles"
//       - name: "Convert to Uppercase"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "uppercase"
//           input: "titles"
//         output: "upper_titles"
//       - name: "Count Items"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "count"
//           input: "upper_titles"
//         output: "item_count"

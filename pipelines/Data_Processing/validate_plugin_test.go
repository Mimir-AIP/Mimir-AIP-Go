package Data_Processing

import (
	"context"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlugin_ExecuteStep_RequiredFields tests validation of required fields
func TestValidatePlugin_ExecuteStep_RequiredFields(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice", "email": "alice@example.com"}, // Valid
		{"id": "2", "name": "Bob"},                                 // Missing email
		{"id": "3", "email": "charlie@example.com"},                // Missing name
		{"id": "4", "name": "David", "email": "david@example.com"}, // Valid
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-required",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"required": []any{"id", "name", "email"},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	// Check stats
	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 4, stats["total"])
	assert.Equal(t, 2, stats["valid"])
	assert.Equal(t, 2, stats["invalid"])

	// Check valid records
	validRecords, ok := resultMap["valid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, validRecords, 2)
	assert.Equal(t, "Alice", validRecords[0]["name"])
	assert.Equal(t, "David", validRecords[1]["name"])

	// Check invalid records
	invalidRecords, ok := resultMap["invalid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, invalidRecords, 2)

	// Check error details
	errors, ok := resultMap["errors"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, errors, 2)

	// Verify error messages contain field names
	errorMessages := errors[0]["errors"].([]string)
	assert.Contains(t, errorMessages[0], "missing required field")
}

// TestValidatePlugin_ExecuteStep_TypeValidation tests type validation rules
func TestValidatePlugin_ExecuteStep_TypeValidation(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice", "age": 30, "active": true},      // Valid
		{"id": "2", "name": "Bob", "age": "thirty", "active": false}, // Invalid age type
		{"id": "3", "name": 123, "age": 25, "active": true},          // Invalid name type
		{"id": "4", "name": "David", "age": 35, "active": "yes"},     // Invalid active type
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-types",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"types": map[string]any{
					"name":   "string",
					"age":    "number",
					"active": "boolean",
				},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 4, stats["total"])
	assert.Equal(t, 1, stats["valid"])
	assert.Equal(t, 3, stats["invalid"])

	// Check valid record
	validRecords, ok := resultMap["valid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, validRecords, 1)
	assert.Equal(t, "Alice", validRecords[0]["name"])
}

// TestValidatePlugin_ExecuteStep_RangeValidation tests numeric range validation
func TestValidatePlugin_ExecuteStep_RangeValidation(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice", "age": 25, "score": 85},    // Valid
		{"id": "2", "name": "Bob", "age": 17, "score": 90},      // Age too low
		{"id": "3", "name": "Charlie", "age": 35, "score": 105}, // Score too high
		{"id": "4", "name": "David", "age": 101, "score": 75},   // Age too high
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-ranges",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"ranges": map[string]any{
					"age": map[string]any{
						"min": float64(18),
						"max": float64(100),
					},
					"score": map[string]any{
						"min": float64(0),
						"max": float64(100),
					},
				},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 4, stats["total"])
	assert.Equal(t, 1, stats["valid"])   // Only Alice
	assert.Equal(t, 3, stats["invalid"]) // Bob, Charlie, David

	// Check error details
	errors, ok := resultMap["errors"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, errors, 3)
}

// TestValidatePlugin_ExecuteStep_CombinedRules tests combined required, type, and range validation
func TestValidatePlugin_ExecuteStep_CombinedRules(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice", "age": 25, "email": "alice@example.com"},  // Valid
		{"id": "2", "name": "Bob", "age": "thirty"},                            // Missing email, wrong age type
		{"id": "3", "email": "charlie@example.com"},                            // Missing name and age
		{"id": "4", "name": "David", "age": 150, "email": "david@example.com"}, // Age out of range
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-all",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"required": []any{"id", "name", "age", "email"},
				"types": map[string]any{
					"name":  "string",
					"age":   "number",
					"email": "string",
				},
				"ranges": map[string]any{
					"age": map[string]any{
						"min": float64(0),
						"max": float64(120),
					},
				},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 4, stats["total"])
	assert.Equal(t, 1, stats["valid"])   // Only Alice
	assert.Equal(t, 3, stats["invalid"]) // All others

	// Check invalid records
	invalidRecords, ok := resultMap["invalid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, invalidRecords, 3)

	// Check errors contain all types of validation failures
	errors, ok := resultMap["errors"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, errors, 3)

	// Check that the last record (David) has age range error
	davidErrors := errors[2]["errors"].([]string)
	hasRangeError := false
	for _, errMsg := range davidErrors {
		if containsStr(errMsg, "max") {
			hasRangeError = true
			break
		}
	}
	assert.True(t, hasRangeError, "Should have range error for age")
}

// Helper function to check string contains
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestValidatePlugin_ExecuteStep_EmptyInput tests validation with empty input
func TestValidatePlugin_ExecuteStep_EmptyInput(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-empty",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"required": []any{"name"},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0, stats["total"])
	assert.Equal(t, 0, stats["valid"])
	assert.Equal(t, 0, stats["invalid"])

	// All arrays should be empty
	validRecords, ok := resultMap["valid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, validRecords, 0)

	invalidRecords, ok := resultMap["invalid_records"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, invalidRecords, 0)
}

// TestValidatePlugin_ExecuteStep_NoRules tests validation with empty rules (should pass all)
func TestValidatePlugin_ExecuteStep_NoRules(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
		{"id": "3"}, // No name, but no required rule
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-no-rules",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{}, // Empty rules
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, stats["total"])
	assert.Equal(t, 3, stats["valid"]) // All valid with no rules
	assert.Equal(t, 0, stats["invalid"])
}

// TestValidatePlugin_ExecuteStep_MapInputFormat tests validation with map-shaped input (from previous plugin)
func TestValidatePlugin_ExecuteStep_MapInputFormat(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	// Input in the format produced by other plugins (e.g., CSV plugin)
	inputData := map[string]any{
		"row_count":    3,
		"column_count": 2,
		"columns":      []string{"name", "age"},
		"rows": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob"}, // Missing age
			map[string]any{"name": "Charlie", "age": 25},
		},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("csv_output", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-csv-output",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "csv_output",
			"rules": map[string]any{
				"required": []any{"name", "age"},
				"types": map[string]any{
					"age": "number",
				},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, stats["total"])
	assert.Equal(t, 2, stats["valid"])
	assert.Equal(t, 1, stats["invalid"])
}

// TestValidatePlugin_ExecuteStep_MissingInputKey tests error on missing input key
func TestValidatePlugin_ExecuteStep_MissingInputKey(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	globalContext := pipelines.NewPluginContext()
	// Don't set any data

	stepConfig := pipelines.StepConfig{
		Name:   "validate-missing",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "nonexistent_key",
			"rules": map[string]any{
				"required": []any{"name"},
			},
		},
		Output: "result",
	}

	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input key not found")
}

// TestValidatePlugin_ExecuteStep_InvalidNumberType tests handling of non-numeric values for range checks
func TestValidatePlugin_ExecuteStep_InvalidNumberType(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "age": "not-a-number"},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("input_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-invalid-number",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "input_data",
			"rules": map[string]any{
				"ranges": map[string]any{
					"age": map[string]any{
						"min": float64(0),
						"max": float64(100),
					},
				},
			},
		},
		Output: "validated_data",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	validatedData, exists := result.Get("validated_data")
	require.True(t, exists)

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	// Should be invalid due to non-numeric value
	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0, stats["valid"])
	assert.Equal(t, 1, stats["invalid"])

	// Check error message
	errors, ok := resultMap["errors"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, errors, 1)

	errorMessages := errors[0]["errors"].([]string)
	assert.Contains(t, errorMessages[0], "not numeric for range check")
}

// TestValidatePlugin_ValidateConfig_MissingInput tests validation without input
func TestValidatePlugin_ValidateConfig_MissingInput(t *testing.T) {
	plugin := NewValidatePlugin()

	err := plugin.ValidateConfig(map[string]any{
		"rules": map[string]any{
			"required": []any{"name"},
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config 'input' is required")
}

// TestValidatePlugin_ValidateConfig_InvalidInputType tests validation with wrong input type
func TestValidatePlugin_ValidateConfig_InvalidInputType(t *testing.T) {
	plugin := NewValidatePlugin()

	err := plugin.ValidateConfig(map[string]any{
		"input": 123, // Should be string
		"rules": map[string]any{},
	})

	assert.Error(t, err)
}

// TestValidatePlugin_ValidateConfig_Valid tests valid configuration
func TestValidatePlugin_ValidateConfig_Valid(t *testing.T) {
	plugin := NewValidatePlugin()

	err := plugin.ValidateConfig(map[string]any{
		"input": "data",
		"rules": map[string]any{
			"required": []any{"name", "email"},
			"types": map[string]any{
				"age": "number",
			},
		},
	})

	assert.NoError(t, err)
}

// TestValidatePlugin_ValidateConfig_ValidMinimal tests valid configuration with minimal fields
func TestValidatePlugin_ValidateConfig_ValidMinimal(t *testing.T) {
	plugin := NewValidatePlugin()

	// Only input is required, rules are optional
	err := plugin.ValidateConfig(map[string]any{
		"input": "data",
	})

	assert.NoError(t, err)
}

// TestValidatePlugin_GetPluginType tests plugin type
func TestValidatePlugin_GetPluginType(t *testing.T) {
	plugin := NewValidatePlugin()
	assert.Equal(t, "Data_Processing", plugin.GetPluginType())
}

// TestValidatePlugin_GetPluginName tests plugin name
func TestValidatePlugin_GetPluginName(t *testing.T) {
	plugin := NewValidatePlugin()
	assert.Equal(t, "validate", plugin.GetPluginName())
}

// TestValidatePlugin_GetInputSchema tests schema definition
func TestValidatePlugin_GetInputSchema(t *testing.T) {
	plugin := NewValidatePlugin()
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	// Check required fields
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "input")

	// Check properties exist
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "input")
	assert.Contains(t, properties, "rules")
}

// TestValidatePlugin_ExecuteStep_OutputKeyGeneration tests automatic output key generation
func TestValidatePlugin_ExecuteStep_OutputKeyGeneration(t *testing.T) {
	plugin := NewValidatePlugin()
	ctx := context.Background()

	inputData := []map[string]any{
		{"id": "1", "name": "Alice"},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("my_data", inputData)

	stepConfig := pipelines.StepConfig{
		Name:   "validate-data",
		Plugin: "Data_Processing.validate",
		Config: map[string]any{
			"input": "my_data",
			"rules": map[string]any{
				"required": []any{"name"},
			},
		},
		// Output not specified - should default to "my_data_validated"
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that result was stored with auto-generated key
	validatedData, exists := result.Get("my_data_validated")
	require.True(t, exists, "Should auto-generate output key as input_key + '_validated'")

	resultMap, ok := validatedData.(map[string]any)
	require.True(t, ok)

	stats, ok := resultMap["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, stats["valid"])
}

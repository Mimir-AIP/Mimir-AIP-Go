package Input

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVPlugin_ExecuteStep_Success(t *testing.T) {
	// Create temp CSV file
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	csvContent := "name,age,city\nAlice,30,NYC\nBob,25,LA\nCharlie,35,Chicago"
	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	// Create plugin
	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   csvFile,
			"has_headers": true,
			"delimiter":   ",",
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	// Execute
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check result data
	data, exists := result.Get("csv_data")
	require.True(t, exists, "csv_data should exist in result")

	csvData, ok := data.(map[string]any)
	require.True(t, ok, "result should be a map")

	assert.Equal(t, csvFile, csvData["file_path"])
	assert.Equal(t, true, csvData["has_headers"])
	assert.Equal(t, ",", csvData["delimiter"])
	assert.Equal(t, 3, csvData["row_count"])
	assert.Equal(t, 3, csvData["column_count"])

	// Check columns
	columns, ok := csvData["columns"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"name", "age", "city"}, columns)
}

func TestCSVPlugin_ExecuteStep_NoHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	csvContent := "Alice,30,NYC\nBob,25,LA"
	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   csvFile,
			"has_headers": false,
			"delimiter":   ",",
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	data, _ := result.Get("csv_data")
	csvData := data.(map[string]any)

	columns, _ := csvData["columns"].([]string)
	assert.Equal(t, []string{"column_1", "column_2", "column_3"}, columns)
	assert.Equal(t, 2, csvData["row_count"])
}

func TestCSVPlugin_ExecuteStep_CustomDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	csvContent := "name;age\nAlice;30\nBob;25"
	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   csvFile,
			"has_headers": true,
			"delimiter":   ";",
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	data, _ := result.Get("csv_data")
	csvData := data.(map[string]any)
	assert.Equal(t, 2, csvData["row_count"])
}

func TestCSVPlugin_ExecuteStep_FileNotFound(t *testing.T) {
	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   "/non/existent/file.csv",
			"has_headers": true,
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")
}

func TestCSVPlugin_ExecuteStep_MissingFilePath(t *testing.T) {
	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"has_headers": true,
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_path is required")
}

func TestCSVPlugin_ExecuteStep_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "empty.csv")
	err := os.WriteFile(csvFile, []byte(""), 0644)
	require.NoError(t, err)

	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   csvFile,
			"has_headers": true,
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	data, _ := result.Get("csv_data")
	csvData := data.(map[string]any)
	assert.Equal(t, 0, csvData["row_count"])
	assert.Equal(t, 0, csvData["column_count"])
}

func TestCSVPlugin_ValidateConfig(t *testing.T) {
	plugin := NewCSVPlugin()

	// Valid config
	err := plugin.ValidateConfig(map[string]any{
		"file_path": "/path/to/file.csv",
	})
	assert.NoError(t, err)

	// Missing file_path
	err = plugin.ValidateConfig(map[string]any{
		"has_headers": true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_path is required")

	// Invalid delimiter (too long)
	err = plugin.ValidateConfig(map[string]any{
		"file_path": "/path/to/file.csv",
		"delimiter": "||",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delimiter must be a single character")
}

func TestCSVPlugin_GetPluginType(t *testing.T) {
	plugin := NewCSVPlugin()
	assert.Equal(t, "Input", plugin.GetPluginType())
}

func TestCSVPlugin_GetPluginName(t *testing.T) {
	plugin := NewCSVPlugin()
	assert.Equal(t, "csv", plugin.GetPluginName())
}

func TestCSVPlugin_GetInputSchema(t *testing.T) {
	plugin := NewCSVPlugin()
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "file_path")
	assert.Contains(t, properties, "has_headers")
	assert.Contains(t, properties, "delimiter")

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "file_path")
}

func TestCSVPlugin_parseCSV_DataTypes(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	csvContent := "string_col,int_col,float_col,bool_col\nhello,42,3.14,true\nworld,-5,0.0,false"
	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	plugin := NewCSVPlugin()
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   csvFile,
			"has_headers": true,
		},
		Output: "csv_data",
	}
	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)

	data, _ := result.Get("csv_data")
	csvData := data.(map[string]any)
	rows, _ := csvData["rows"].([]map[string]any)

	require.Len(t, rows, 2)

	// Check first row data types
	firstRow := rows[0]
	assert.Equal(t, "hello", firstRow["string_col"])
	assert.Equal(t, float64(42), firstRow["int_col"]) // JSON numbers become float64
	assert.Equal(t, float64(3.14), firstRow["float_col"])
	assert.Equal(t, true, firstRow["bool_col"])
}

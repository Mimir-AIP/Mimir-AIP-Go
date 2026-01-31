package Output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExcelPlugin tests Excel plugin creation
func TestNewExcelPlugin(t *testing.T) {
	plugin := NewExcelPlugin()
	require.NotNil(t, plugin, "Plugin should not be nil")
	assert.Equal(t, "Output", plugin.GetPluginType())
	assert.Equal(t, "excel", plugin.GetPluginName())
}

// TestExcelPlugin_ValidateConfig tests config validation
func TestExcelPlugin_ValidateConfig(t *testing.T) {
	plugin := NewExcelPlugin()

	// Valid config
	err := plugin.ValidateConfig(map[string]interface{}{
		"input": "test_data",
	})
	assert.NoError(t, err, "Should validate with input")

	// Missing input
	err = plugin.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err, "Should error without input")

	// Empty input
	err = plugin.ValidateConfig(map[string]interface{}{
		"input": "",
	})
	assert.Error(t, err, "Should error with empty input")
}

// TestExcelPlugin_ExecuteStep_WithMapArray tests execution with map array
func TestExcelPlugin_ExecuteStep_WithMapArray(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_output.xlsx")

	// Create test data
	data := []map[string]interface{}{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
		{"name": "Charlie", "age": 35, "city": "Chicago"},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("test_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input":        "test_data",
			"file_path":    filePath,
			"sheet_name":   "People",
			"with_headers": true,
		},
		Output: "excel_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should execute without error")
	require.NotNil(t, result, "Should return result")

	// Verify result
	excelResult, exists := result.Get("excel_result")
	assert.True(t, exists, "Should have excel_result")

	resultMap, ok := excelResult.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	assert.True(t, resultMap["success"].(bool), "Should be successful")
	assert.Equal(t, filePath, resultMap["file_path"])
	assert.Equal(t, "People", resultMap["sheet_name"])
	assert.Equal(t, 4, resultMap["rows_written"], "Should have header + 3 data rows")

	// Verify file was created
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestExcelPlugin_ExecuteStep_WithAnyArray tests execution with any array
func TestExcelPlugin_ExecuteStep_WithAnyArray(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_any.xlsx")

	data := []interface{}{
		map[string]interface{}{"col1": "A", "col2": 1},
		map[string]interface{}{"col1": "B", "col2": 2},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("test_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input":     "test_data",
			"file_path": filePath,
		},
		Output: "excel_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify file was created
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestExcelPlugin_ExecuteStep_WithMap tests execution with single map
func TestExcelPlugin_ExecuteStep_WithMap(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_single.xlsx")

	data := map[string]interface{}{
		"name":  "Test",
		"value": 100,
		"date":  time.Now(),
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("test_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input":     "test_data",
			"file_path": filePath,
		},
		Output: "excel_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestExcelPlugin_ExecuteStep_DefaultPath tests default file path
func TestExcelPlugin_ExecuteStep_DefaultPath(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	data := []map[string]interface{}{
		{"test": "data"},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("test_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input": "test_data",
			// No file_path - should use default
		},
		Output: "excel_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result has default file path
	excelResult, _ := result.Get("excel_result")
	resultMap := excelResult.(map[string]interface{})
	assert.NotEmpty(t, resultMap["file_path"], "Should have default file path")
	assert.Contains(t, resultMap["file_path"], "output_", "Should use default naming")
	assert.Contains(t, resultMap["file_path"], ".xlsx", "Should have xlsx extension")
}

// TestExcelPlugin_ExecuteStep_MissingInput tests missing input handling
func TestExcelPlugin_ExecuteStep_MissingInput(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	globalContext := pipelines.NewPluginContext()
	// Don't set any data

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input": "missing_data",
		},
	}

	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.Error(t, err, "Should error when input not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestExcelPlugin_ExecuteStep_InvalidData tests invalid data handling
func TestExcelPlugin_ExecuteStep_InvalidData(t *testing.T) {
	plugin := NewExcelPlugin()
	ctx := context.Background()

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("invalid_data", 12345) // Unsupported type

	stepConfig := pipelines.StepConfig{
		Name:   "excel_output",
		Plugin: "Output.excel",
		Config: map[string]interface{}{
			"input": "invalid_data",
		},
	}

	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.Error(t, err, "Should error with unsupported data type")
}

// TestExcelPlugin_formatValue tests value formatting
func TestExcelPlugin_formatValue(t *testing.T) {
	plugin := NewExcelPlugin()

	tests := []struct {
		input    interface{}
		expected string
	}{
		{float64(3.14159), "3.14"},
		{float32(2.5), "2.50"},
		{42, "42"},
		{int64(100), "100"},
		{true, "true"},
		{false, "false"},
		{time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), "2024-01-15 10:30:00"},
		{nil, ""},
		{"string value", "string value"},
	}

	for _, test := range tests {
		result := plugin.formatValue(test.input)
		assert.Equal(t, test.expected, result, "Formatting %v should give %s", test.input, test.expected)
	}
}

// TestExcelPlugin_GetInputSchema tests input schema
func TestExcelPlugin_GetInputSchema(t *testing.T) {
	plugin := NewExcelPlugin()
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])
	assert.NotNil(t, schema["properties"])
}

// TestNewPDFPlugin tests PDF plugin creation
func TestNewPDFPlugin(t *testing.T) {
	plugin := NewPDFPlugin()
	require.NotNil(t, plugin, "Plugin should not be nil")
	assert.Equal(t, "Output", plugin.GetPluginType())
	assert.Equal(t, "pdf", plugin.GetPluginName())
}

// TestPDFPlugin_ValidateConfig tests config validation
func TestPDFPlugin_ValidateConfig(t *testing.T) {
	plugin := NewPDFPlugin()

	// Valid config
	err := plugin.ValidateConfig(map[string]interface{}{
		"input": "test_data",
	})
	assert.NoError(t, err, "Should validate with input")

	// Missing input
	err = plugin.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err, "Should error without input")
}

// TestPDFPlugin_ExecuteStep_WithMapArray tests PDF execution with map array
func TestPDFPlugin_ExecuteStep_WithMapArray(t *testing.T) {
	plugin := NewPDFPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_output.pdf")

	data := []map[string]interface{}{
		{"product": "Widget", "price": 19.99, "quantity": 100},
		{"product": "Gadget", "price": 29.99, "quantity": 50},
		{"product": "Tool", "price": 49.99, "quantity": 25},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("report_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "pdf_output",
		Plugin: "Output.pdf",
		Config: map[string]interface{}{
			"input":       "report_data",
			"file_path":   filePath,
			"title":       "Product Report",
			"page_size":   "A4",
			"orientation": "P",
		},
		Output: "pdf_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should execute without error")
	require.NotNil(t, result, "Should return result")

	// Verify result
	pdfResult, exists := result.Get("pdf_result")
	assert.True(t, exists, "Should have pdf_result")

	resultMap, ok := pdfResult.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	assert.True(t, resultMap["success"].(bool), "Should be successful")
	assert.Equal(t, filePath, resultMap["file_path"])
	assert.Equal(t, "Product Report", resultMap["title"])
	assert.Equal(t, 3, resultMap["rows_written"])

	// Verify file was created
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestPDFPlugin_ExecuteStep_WithAnyArray tests PDF execution with any array
func TestPDFPlugin_ExecuteStep_WithAnyArray(t *testing.T) {
	plugin := NewPDFPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_items.pdf")

	data := []interface{}{
		"Item 1",
		"Item 2",
		"Item 3",
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("items", data)

	stepConfig := pipelines.StepConfig{
		Name:   "pdf_output",
		Plugin: "Output.pdf",
		Config: map[string]interface{}{
			"input":     "items",
			"file_path": filePath,
			"title":     "Item List",
		},
		Output: "pdf_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestPDFPlugin_ExecuteStep_WithString tests PDF execution with string
func TestPDFPlugin_ExecuteStep_WithString(t *testing.T) {
	plugin := NewPDFPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_text.pdf")

	data := "This is a test report.\nIt contains multiple lines of text."

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("text_content", data)

	stepConfig := pipelines.StepConfig{
		Name:   "pdf_output",
		Plugin: "Output.pdf",
		Config: map[string]interface{}{
			"input":     "text_content",
			"file_path": filePath,
			"title":     "Text Report",
		},
		Output: "pdf_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist")
}

// TestPDFPlugin_ExecuteStep_DefaultPath tests default PDF path
func TestPDFPlugin_ExecuteStep_DefaultPath(t *testing.T) {
	plugin := NewPDFPlugin()
	ctx := context.Background()

	data := []map[string]interface{}{
		{"test": "data"},
	}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("test_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "pdf_output",
		Plugin: "Output.pdf",
		Config: map[string]interface{}{
			"input": "test_data",
		},
		Output: "pdf_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	pdfResult, _ := result.Get("pdf_result")
	resultMap := pdfResult.(map[string]interface{})
	assert.NotEmpty(t, resultMap["file_path"])
	assert.Contains(t, resultMap["file_path"], "report_")
	assert.Contains(t, resultMap["file_path"], ".pdf")
}

// TestPDFPlugin_ExecuteStep_EmptyData tests empty data handling
func TestPDFPlugin_ExecuteStep_EmptyData(t *testing.T) {
	plugin := NewPDFPlugin()
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_empty.pdf")

	data := []map[string]interface{}{}

	globalContext := pipelines.NewPluginContext()
	globalContext.Set("empty_data", data)

	stepConfig := pipelines.StepConfig{
		Name:   "pdf_output",
		Plugin: "Output.pdf",
		Config: map[string]interface{}{
			"input":     "empty_data",
			"file_path": filePath,
		},
		Output: "pdf_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)

	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist even with empty data")
}

// TestPDFPlugin_truncateText tests text truncation
func TestPDFPlugin_truncateText(t *testing.T) {
	plugin := NewPDFPlugin()

	// Short text should not be truncated
	short := "Short text"
	assert.Equal(t, short, plugin.truncateText(short, 100))

	// Long text should be truncated
	long := "This is a very long text that should be truncated because it exceeds the limit"
	truncated := plugin.truncateText(long, 50)
	assert.Less(t, len(truncated), len(long))
	assert.Contains(t, truncated, "...")
}

// TestPDFPlugin_GetInputSchema tests input schema
func TestPDFPlugin_GetInputSchema(t *testing.T) {
	plugin := NewPDFPlugin()
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])
	assert.NotNil(t, schema["properties"])
}

// TestCellName tests Excel cell name helper
func TestCellName(t *testing.T) {
	tests := []struct {
		col      int
		row      int
		expected string
	}{
		{1, 1, "A1"},
		{2, 1, "B1"},
		{26, 1, "Z1"},
		{27, 1, "AA1"},
		{28, 1, "AB1"},
		{1, 10, "A10"},
		{1, 100, "A100"},
	}

	for _, test := range tests {
		result := cellName(test.col, test.row)
		assert.Equal(t, test.expected, result, "cellName(%d, %d) should give %s", test.col, test.row, test.expected)
	}
}

// Benchmark tests

func BenchmarkExcelPlugin_ExecuteStep(b *testing.B) {
	plugin := NewExcelPlugin()
	ctx := context.Background()
	tmpDir := b.TempDir()

	data := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = map[string]interface{}{
			"id":     i,
			"name":   fmt.Sprintf("Name %d", i),
			"value":  float64(i) * 1.5,
			"active": i%2 == 0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalContext := pipelines.NewPluginContext()
		globalContext.Set("bench_data", data)

		stepConfig := pipelines.StepConfig{
			Name:   "excel_output",
			Plugin: "Output.excel",
			Config: map[string]interface{}{
				"input":     "bench_data",
				"file_path": filepath.Join(tmpDir, fmt.Sprintf("bench_%d.xlsx", i)),
			},
			Output: "result",
		}

		_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

func BenchmarkPDFPlugin_ExecuteStep(b *testing.B) {
	plugin := NewPDFPlugin()
	ctx := context.Background()
	tmpDir := b.TempDir()

	data := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		data[i] = map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("Name %d", i),
			"value": float64(i) * 1.5,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalContext := pipelines.NewPluginContext()
		globalContext.Set("bench_data", data)

		stepConfig := pipelines.StepConfig{
			Name:   "pdf_output",
			Plugin: "Output.pdf",
			Config: map[string]interface{}{
				"input":     "bench_data",
				"file_path": filepath.Join(tmpDir, fmt.Sprintf("bench_%d.pdf", i)),
			},
			Output: "result",
		}

		_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

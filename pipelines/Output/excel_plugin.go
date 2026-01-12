// Excel Output Plugin
// Writes data to Excel format (.xlsx)

package Output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

func cellName(col, row int) string {
	colStr := ""
	for col > 0 {
		col--
		colStr = string(rune('A'+col%26)) + colStr
		col /= 26
	}
	return fmt.Sprintf("%s%d", colStr, row)
}

// ExcelPlugin implements Excel output functionality
type ExcelPlugin struct {
	name    string
	version string
}

// NewExcelPlugin creates a new Excel output plugin instance
func NewExcelPlugin() *ExcelPlugin {
	return &ExcelPlugin{
		name:    "ExcelOutputPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep writes data to Excel format
func (p *ExcelPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	inputKey, ok := config["input"].(string)
	if !ok || inputKey == "" {
		return nil, fmt.Errorf("input key is required")
	}

	data, exists := globalContext.Get(inputKey)
	if !exists {
		return nil, fmt.Errorf("input data '%s' not found in context", inputKey)
	}

	filePath, _ := config["file_path"].(string)
	if filePath == "" {
		timestamp := time.Now().Format("20060102_150405")
		filePath = fmt.Sprintf("output_%s.xlsx", timestamp)
	}

	sheetName := "Sheet1"
	if sheet, ok := config["sheet_name"].(string); ok && sheet != "" {
		sheetName = sheet
	}

	withHeaders := true
	if headers, ok := config["with_headers"].(bool); ok {
		withHeaders = headers
	}

	f := excelize.NewFile()

	var err error
	var rowsWritten int

	switch dataVal := data.(type) {
	case []map[string]any:
		rowsWritten, err = p.writeRowsFromMaps(f, sheetName, dataVal, withHeaders)
	case []any:
		rowsWritten, err = p.writeRowsFromArray(f, sheetName, dataVal, withHeaders)
	case map[string]any:
		rowsWritten, err = p.writeRowsFromMap(f, sheetName, dataVal, withHeaders)
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to write Excel data: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := f.SaveAs(filePath); err != nil {
		return nil, fmt.Errorf("failed to save Excel file: %w", err)
	}

	result := map[string]any{
		"success":      true,
		"file_path":    filePath,
		"rows_written": rowsWritten,
		"sheet_name":   sheetName,
	}

	outputContext := pipelines.NewPluginContext()
	outputKey := stepConfig.Output
	if outputKey == "" {
		outputKey = "excel_output"
	}
	outputContext.Set(outputKey, result)

	return outputContext, nil
}

func (p *ExcelPlugin) writeRowsFromMaps(f *excelize.File, sheetName string, data []map[string]any, withHeaders bool) (int, error) {
	row := 1
	headers := make([]string, 0)

	if withHeaders && len(data) > 0 {
		for key := range data[0] {
			headers = append(headers, key)
		}
		for col, header := range headers {
			cell := cellName(col+1, row)
			f.SetCellValue(sheetName, cell, header)
		}
		row++
	}

	for _, item := range data {
		for col, header := range headers {
			cell := cellName(col+1, row)
			value := item[header]
			f.SetCellValue(sheetName, cell, p.formatValue(value))
		}
		row++
	}

	return row - 1, nil
}

func (p *ExcelPlugin) writeRowsFromArray(f *excelize.File, sheetName string, data []any, withHeaders bool) (int, error) {
	row := 1
	headers := make([]string, 0)

	if withHeaders {
		for i := range data {
			headers = append(headers, fmt.Sprintf("Column%d", i+1))
		}
		for col, header := range headers {
			cell := cellName(col+1, row)
			f.SetCellValue(sheetName, cell, header)
		}
		row++
	}

	for _, item := range data {
		switch itemVal := item.(type) {
		case map[string]any:
			for col, header := range headers {
				cell := cellName(col+1, row)
				value := itemVal[header]
				f.SetCellValue(sheetName, cell, p.formatValue(value))
			}
		case []any:
			for col, val := range itemVal {
				cell := cellName(col+1, row)
				f.SetCellValue(sheetName, cell, p.formatValue(val))
			}
		default:
			cell := cellName(1, row)
			f.SetCellValue(sheetName, cell, p.formatValue(item))
		}
		row++
	}

	return row - 1, nil
}

func (p *ExcelPlugin) writeRowsFromMap(f *excelize.File, sheetName string, data map[string]any, withHeaders bool) (int, error) {
	row := 1

	if withHeaders {
		f.SetCellValue(sheetName, "A1", "Key")
		f.SetCellValue(sheetName, "B1", "Value")
		row++
	}

	for key, value := range data {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), key)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), p.formatValue(value))
		row++
	}

	return row - 1, nil
}

func (p *ExcelPlugin) formatValue(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', 2, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 2, 32)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetPluginType returns the plugin type
func (p *ExcelPlugin) GetPluginType() string {
	return "Output"
}

// GetPluginName returns the plugin name
func (p *ExcelPlugin) GetPluginName() string {
	return "excel"
}

// ValidateConfig validates the plugin configuration
func (p *ExcelPlugin) ValidateConfig(config map[string]any) error {
	if config["input"] == nil {
		return fmt.Errorf("input is required")
	}
	if input, ok := config["input"].(string); !ok || input == "" {
		return fmt.Errorf("input must be a non-empty string")
	}
	return nil
}

// GetInputSchema returns the JSON Schema for plugin configuration
// NOTE: This only contains plugin-level settings. Step-level parameters like
// 'input', 'file_path', etc. should be defined in pipeline steps.
func (p *ExcelPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

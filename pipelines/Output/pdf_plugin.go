// PDF Report Output Plugin
// Generates PDF reports from data

package Output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/jung-kurt/gofpdf"
)

// PDFPlugin implements PDF report generation functionality
type PDFPlugin struct {
	name    string
	version string
}

// NewPDFPlugin creates a new PDF output plugin instance
func NewPDFPlugin() *PDFPlugin {
	return &PDFPlugin{
		name:    "PDFReportPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep generates a PDF report
func (p *PDFPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
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
		filePath = fmt.Sprintf("report_%s.pdf", timestamp)
	}

	title := "Report"
	if t, ok := config["title"].(string); ok && t != "" {
		title = t
	}

	pageSize := "A4"
	if ps, ok := config["page_size"].(string); ok && ps != "" {
		pageSize = ps
	}

	orientation := "P"
	if o, _ := config["orientation"].(string); o == "L" {
		orientation = "L"
	}

	pdf := gofpdf.New(orientation, "mm", pageSize, "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, title)
	pdf.Ln(15)

	pdf.SetFont("Arial", "", 10)
	generatedAt := fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05"))
	pdf.Cell(0, 8, generatedAt)
	pdf.Ln(10)

	var err error
	rowsWritten := 0

	switch dataVal := data.(type) {
	case []map[string]any:
		rowsWritten, err = p.writeTableFromMaps(pdf, dataVal)
	case []any:
		rowsWritten, err = p.writeTableFromArray(pdf, dataVal)
	case string:
		err = p.writeText(pdf, dataVal)
	default:
		err = p.writeDataAsText(pdf, dataVal)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to write PDF content: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return nil, fmt.Errorf("failed to save PDF file: %w", err)
	}

	result := map[string]any{
		"success":      true,
		"file_path":    filePath,
		"rows_written": rowsWritten,
		"title":        title,
	}

	outputContext := pipelines.NewPluginContext()
	outputKey := stepConfig.Output
	if outputKey == "" {
		outputKey = "pdf_output"
	}
	outputContext.Set(outputKey, result)

	return outputContext, nil
}

func (p *PDFPlugin) writeTableFromMaps(pdf *gofpdf.Fpdf, data []map[string]any) (int, error) {
	if len(data) == 0 {
		pdf.Cell(0, 10, "No data available")
		return 0, nil
	}

	headers := make([]string, 0)
	for key := range data[0] {
		headers = append(headers, key)
	}

	colWidth := 170.0 / float64(len(headers))

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(255, 200, 100)
	for _, header := range headers {
		pdf.CellFormat(colWidth, 8, p.truncateText(header, colWidth-2), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(245, 245, 245)

	row := 0
	for _, item := range data {
		if row%2 == 1 {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		for _, header := range headers {
			value := fmt.Sprintf("%v", item[header])
			pdf.CellFormat(colWidth, 7, p.truncateText(value, colWidth-2), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		row++
	}

	return len(data), nil
}

func (p *PDFPlugin) writeTableFromArray(pdf *gofpdf.Fpdf, data []any) (int, error) {
	if len(data) == 0 {
		pdf.Cell(0, 10, "No data available")
		return 0, nil
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(255, 200, 100)
	pdf.CellFormat(170, 8, "Item", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	row := 0
	for _, item := range data {
		if row%2 == 1 {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		value := fmt.Sprintf("%v", item)
		pdf.CellFormat(170, 7, p.truncateText(value, 168), "1", 0, "L", true, 0, "")
		pdf.Ln(-1)
		row++
	}

	return len(data), nil
}

func (p *PDFPlugin) writeText(pdf *gofpdf.Fpdf, text string) error {
	pdf.SetFont("Arial", "", 10)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		pdf.Cell(0, 6, line)
		pdf.Ln(-1)
	}
	return nil
}

func (p *PDFPlugin) writeDataAsText(pdf *gofpdf.Fpdf, data any) error {
	pdf.SetFont("Arial", "", 10)
	jsonBytes, err := formatJSON(data)
	if err != nil {
		return err
	}
	lines := strings.Split(string(jsonBytes), "\n")
	for _, line := range lines {
		pdf.Cell(0, 5, line)
		pdf.Ln(-1)
	}
	return nil
}

func (p *PDFPlugin) truncateText(text string, maxWidth float64) string {
	if len(text) <= 50 {
		return text
	}
	return text[:47] + "..."
}

func formatJSON(data any) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// GetPluginType returns the plugin type
func (p *PDFPlugin) GetPluginType() string {
	return "Output"
}

// GetPluginName returns the plugin name
func (p *PDFPlugin) GetPluginName() string {
	return "pdf"
}

// ValidateConfig validates the plugin configuration
func (p *PDFPlugin) ValidateConfig(config map[string]any) error {
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
// 'input', 'file_path', 'title', etc. should be defined in pipeline steps.
func (p *PDFPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

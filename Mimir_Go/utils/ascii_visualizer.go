package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// ASCIIVisualizer provides ASCII-based visualization of pipeline execution
type ASCIIVisualizer struct {
	showDetails bool
	maxWidth    int
}

// NewASCIIVisualizer creates a new ASCII visualizer
func NewASCIIVisualizer() *ASCIIVisualizer {
	return &ASCIIVisualizer{
		showDetails: true,
		maxWidth:    80,
	}
}

// SetShowDetails controls whether to show detailed information
func (av *ASCIIVisualizer) SetShowDetails(show bool) {
	av.showDetails = show
}

// SetMaxWidth sets the maximum width for the visualization
func (av *ASCIIVisualizer) SetMaxWidth(width int) {
	av.maxWidth = width
}

// VisualizePipeline generates an ASCII visualization of a pipeline configuration
func (av *ASCIIVisualizer) VisualizePipeline(config *PipelineConfig) string {
	var output strings.Builder

	// Header
	output.WriteString(av.createHeader("Pipeline Visualization"))
	output.WriteString("\n")

	// Pipeline info
	output.WriteString(fmt.Sprintf("Name: %s\n", config.Name))
	if config.Description != "" {
		output.WriteString(fmt.Sprintf("Description: %s\n", config.Description))
	}
	output.WriteString(fmt.Sprintf("Steps: %d\n", len(config.Steps)))
	output.WriteString("\n")

	// Steps visualization
	output.WriteString("Execution Flow:\n")
	output.WriteString(av.createPipelineFlow(config.Steps))

	return output.String()
}

// VisualizeExecution generates an ASCII visualization of pipeline execution status
func (av *ASCIIVisualizer) VisualizeExecution(result *PipelineExecutionResult, duration time.Duration) string {
	var output strings.Builder

	// Header
	status := "SUCCESS"
	if !result.Success {
		status = "FAILED"
	}
	output.WriteString(av.createHeader(fmt.Sprintf("Pipeline Execution - %s", status)))
	output.WriteString("\n")

	// Execution info
	output.WriteString(fmt.Sprintf("Duration: %v\n", duration))
	output.WriteString(fmt.Sprintf("Executed: %s\n", result.ExecutedAt))
	if !result.Success {
		output.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
	}
	output.WriteString("\n")

	// Context summary
	if len(result.Context) > 0 {
		output.WriteString("Context Summary:\n")
		output.WriteString(av.createContextSummary(result.Context))
	}

	return output.String()
}

// VisualizeSchedulerJobs generates an ASCII visualization of scheduled jobs
func (av *ASCIIVisualizer) VisualizeSchedulerJobs(jobs map[string]*ScheduledJob) string {
	var output strings.Builder

	output.WriteString(av.createHeader("Scheduled Jobs"))
	output.WriteString("\n")

	if len(jobs) == 0 {
		output.WriteString("No scheduled jobs\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Total Jobs: %d\n\n", len(jobs)))

	// Table header
	output.WriteString(av.createTableRow([]string{"ID", "Name", "Schedule", "Status", "Next Run"}, []int{15, 20, 15, 10, 20}))
	output.WriteString(av.createTableSeparator([]int{15, 20, 15, 10, 20}))

	// Job rows
	for _, job := range jobs {
		status := "Enabled"
		if !job.Enabled {
			status = "Disabled"
		}

		nextRun := "Never"
		if job.NextRun != nil {
			nextRun = job.NextRun.Format("15:04:05")
		}

		output.WriteString(av.createTableRow([]string{
			job.ID,
			av.truncateString(job.Name, 18),
			job.CronExpr,
			status,
			nextRun,
		}, []int{15, 20, 15, 10, 20}))
	}

	return output.String()
}

// VisualizePluginRegistry generates an ASCII visualization of available plugins
func (av *ASCIIVisualizer) VisualizePluginRegistry(registry *pipelines.PluginRegistry) string {
	var output strings.Builder

	output.WriteString(av.createHeader("Available Plugins"))
	output.WriteString("\n")

	pluginTypes := registry.ListPluginTypes()
	if len(pluginTypes) == 0 {
		output.WriteString("No plugins registered\n")
		return output.String()
	}

	for _, pluginType := range pluginTypes {
		plugins := registry.GetPluginsByType(pluginType)
		output.WriteString(fmt.Sprintf("📁 %s (%d plugins)\n", pluginType, len(plugins)))

		for pluginName := range plugins {
			output.WriteString(fmt.Sprintf("  🔧 %s\n", pluginName))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// createHeader creates a formatted header
func (av *ASCIIVisualizer) createHeader(title string) string {
	width := av.maxWidth
	if len(title)+4 > width {
		width = len(title) + 4
	}

	border := strings.Repeat("═", width-2)
	return fmt.Sprintf("╔%s╗\n║ %-*s ║\n╚%s╝", border, width-2, title, border)
}

// createPipelineFlow creates a visual representation of pipeline steps
func (av *ASCIIVisualizer) createPipelineFlow(steps []pipelines.StepConfig) string {
	var output strings.Builder

	for i, step := range steps {
		// Step box
		stepBox := fmt.Sprintf("┌─ %s ─┐", step.Name)
		output.WriteString(stepBox)
		output.WriteString("\n")

		// Plugin info
		output.WriteString(fmt.Sprintf("│ Plugin: %s │\n", step.Plugin))

		// Config preview (if enabled)
		if av.showDetails && len(step.Config) > 0 {
			output.WriteString("│ Config: " + av.createConfigPreview(step.Config) + " │\n")
		}

		// Output info
		if step.Output != "" {
			output.WriteString(fmt.Sprintf("│ Output: %s │\n", step.Output))
		}

		output.WriteString("└")
		if i < len(steps)-1 {
			output.WriteString("─────────┬─────────")
		} else {
			output.WriteString("────────────────────")
		}
		output.WriteString("┘\n")

		// Connection line
		if i < len(steps)-1 {
			output.WriteString("          │\n")
		}
	}

	return output.String()
}

// createContextSummary creates a summary of the execution context
func (av *ASCIIVisualizer) createContextSummary(context pipelines.PluginContext) string {
	var output strings.Builder

	for key, value := range context {
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > 50 {
			valueStr = valueStr[:47] + "..."
		}
		output.WriteString(fmt.Sprintf("  %s: %s\n", key, valueStr))
	}

	return output.String()
}

// createTableRow creates a formatted table row
func (av *ASCIIVisualizer) createTableRow(values []string, widths []int) string {
	var output strings.Builder

	for i, value := range values {
		if i > 0 {
			output.WriteString(" │ ")
		}
		width := widths[i]
		if len(value) > width {
			value = value[:width-3] + "..."
		}
		output.WriteString(fmt.Sprintf("%-*s", width, value))
	}

	output.WriteString("\n")
	return output.String()
}

// createTableSeparator creates a table separator line
func (av *ASCIIVisualizer) createTableSeparator(widths []int) string {
	var output strings.Builder

	for i, width := range widths {
		if i > 0 {
			output.WriteString("─┼─")
		}
		output.WriteString(strings.Repeat("─", width))
	}

	output.WriteString("\n")
	return output.String()
}

// createConfigPreview creates a preview of configuration parameters
func (av *ASCIIVisualizer) createConfigPreview(config map[string]interface{}) string {
	var parts []string
	count := 0

	for key, value := range config {
		if count >= 3 { // Limit to 3 parameters for preview
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
		count++
	}

	return strings.Join(parts, ", ")
}

// truncateString truncates a string to a maximum length
func (av *ASCIIVisualizer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Helper functions for quick visualization

// PrintPipeline prints a pipeline visualization to stdout
func PrintPipeline(config *PipelineConfig) {
	visualizer := NewASCIIVisualizer()
	fmt.Println(visualizer.VisualizePipeline(config))
}

// PrintExecution prints an execution visualization to stdout
func PrintExecution(result *PipelineExecutionResult, duration time.Duration) {
	visualizer := NewASCIIVisualizer()
	fmt.Println(visualizer.VisualizeExecution(result, duration))
}

// PrintSchedulerJobs prints scheduled jobs visualization to stdout
func PrintSchedulerJobs(jobs map[string]*ScheduledJob) {
	visualizer := NewASCIIVisualizer()
	fmt.Println(visualizer.VisualizeSchedulerJobs(jobs))
}

// PrintPluginRegistry prints plugin registry visualization to stdout
func PrintPluginRegistry(registry *pipelines.PluginRegistry) {
	visualizer := NewASCIIVisualizer()
	fmt.Println(visualizer.VisualizePluginRegistry(registry))
}

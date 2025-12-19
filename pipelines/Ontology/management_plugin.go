package ontology

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/google/uuid"
)

// ManagementPlugin handles ontology lifecycle operations
type ManagementPlugin struct {
	persistence *storage.PersistenceBackend
	tdb2Backend *knowledgegraph.TDB2Backend
	ontologyDir string
}

// NewManagementPlugin creates a new ontology management plugin
func NewManagementPlugin(persistence *storage.PersistenceBackend, tdb2Backend *knowledgegraph.TDB2Backend, ontologyDir string) *ManagementPlugin {
	return &ManagementPlugin{
		persistence: persistence,
		tdb2Backend: tdb2Backend,
		ontologyDir: ontologyDir,
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *ManagementPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	operation, ok := stepConfig.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation not specified in config")
	}

	switch operation {
	case "upload":
		return p.handleUpload(ctx, stepConfig, globalContext)
	case "validate":
		return p.handleValidate(ctx, stepConfig, globalContext)
	case "list":
		return p.handleList(ctx, stepConfig, globalContext)
	case "get":
		return p.handleGet(ctx, stepConfig, globalContext)
	case "delete":
		return p.handleDelete(ctx, stepConfig, globalContext)
	case "stats":
		return p.handleStats(ctx, stepConfig, globalContext)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *ManagementPlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *ManagementPlugin) GetPluginName() string {
	return "management"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *ManagementPlugin) ValidateConfig(config map[string]any) error {
	operation, ok := config["operation"].(string)
	if !ok {
		return fmt.Errorf("operation is required")
	}

	validOperations := []string{"upload", "validate", "list", "get", "delete", "stats"}
	valid := false
	for _, op := range validOperations {
		if operation == op {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}

// GetInputSchema returns the JSON Schema for ontology management operations
func (p *ManagementPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": "Manage ontology lifecycle operations including upload, validation, listing, retrieval, deletion, and statistics.",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation to perform",
				"enum":        []string{"upload", "validate", "list", "get", "delete", "stats"},
			},
			"ontology_id": map[string]any{
				"type":        "string",
				"description": "ID of the ontology (required for get, delete, stats operations)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Ontology name (required for upload)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Ontology description (optional for upload)",
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Ontology version (required for upload)",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Ontology format (turtle, rdfxml, ntriples, jsonld)",
				"enum":        []string{"turtle", "rdfxml", "ntriples", "jsonld"},
				"default":     "turtle",
			},
			"ontology_data": map[string]any{
				"type":        "string",
				"description": "Ontology data content (required for upload and validate)",
			},
			"created_by": map[string]any{
				"type":        "string",
				"description": "Creator username (optional for upload)",
			},
		},
		"required": []string{"operation"},
	}
}

// handleUpload handles ontology upload operation
func (p *ManagementPlugin) handleUpload(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Extract configuration
	name, _ := stepConfig.Config["name"].(string)
	description, _ := stepConfig.Config["description"].(string)
	version, _ := stepConfig.Config["version"].(string)
	format, _ := stepConfig.Config["format"].(string)
	ontologyData, _ := stepConfig.Config["ontology_data"].(string)
	createdBy, _ := stepConfig.Config["created_by"].(string)

	if name == "" || version == "" || ontologyData == "" {
		return nil, fmt.Errorf("name, version, and ontology_data are required")
	}

	if format == "" {
		format = "turtle"
	}

	// Validate ontology syntax
	validationResult := p.validateOntologyData(ontologyData, format)
	if !validationResult.Valid {
		return nil, fmt.Errorf("ontology validation failed: %v", validationResult.Errors)
	}

	// Generate unique ID
	ontologyID := uuid.New().String()

	// Create file path
	fileName := fmt.Sprintf("%s-%s.%s", sanitizeFilename(name), sanitizeFilename(version), getFileExtension(format))
	filePath := filepath.Join(p.ontologyDir, fileName)

	// Write ontology file
	if err := os.WriteFile(filePath, []byte(ontologyData), 0644); err != nil {
		return nil, fmt.Errorf("failed to write ontology file: %w", err)
	}

	// Generate TDB2 graph URI
	tdb2Graph := fmt.Sprintf("http://mimir-aip.io/ontology/%s/%s", sanitizeFilename(name), sanitizeFilename(version))

	// Create ontology metadata
	ontology := &storage.Ontology{
		ID:          ontologyID,
		Name:        name,
		Description: description,
		Version:     version,
		FilePath:    filePath,
		TDB2Graph:   tdb2Graph,
		Format:      format,
		Status:      "active",
		CreatedBy:   createdBy,
		Metadata:    "{}",
	}

	// Save to database
	if err := p.persistence.CreateOntology(ctx, ontology); err != nil {
		os.Remove(filePath) // Clean up file on error
		return nil, fmt.Errorf("failed to save ontology metadata: %w", err)
	}

	// Load ontology into TDB2
	if err := p.tdb2Backend.LoadOntology(ctx, tdb2Graph, ontologyData, format); err != nil {
		// Rollback: delete from database and remove file
		p.persistence.DeleteOntology(ctx, ontologyID)
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to load ontology into TDB2: %w", err)
	}

	// Set result in context
	result := pipelines.NewPluginContext()
	result.Set("ontology_id", ontologyID)
	result.Set("ontology_name", name)
	result.Set("ontology_version", version)
	result.Set("tdb2_graph", tdb2Graph)
	result.Set("status", "success")

	return result, nil
}

// handleValidate validates an ontology
func (p *ManagementPlugin) handleValidate(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyData, _ := stepConfig.Config["ontology_data"].(string)
	format, _ := stepConfig.Config["format"].(string)

	if format == "" {
		format = "turtle"
	}

	// Always validate, even if empty (validator will handle it)
	validationResult := p.validateOntologyData(ontologyData, format)

	result := pipelines.NewPluginContext()
	result.Set("valid", validationResult.Valid)
	result.Set("errors", validationResult.Errors)
	result.Set("warnings", validationResult.Warnings)

	return result, nil
}

// handleList lists all ontologies
func (p *ManagementPlugin) handleList(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	status, _ := stepConfig.Config["status"].(string)

	ontologies, err := p.persistence.ListOntologies(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list ontologies: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("ontologies", ontologies)
	result.Set("count", len(ontologies))

	return result, nil
}

// handleGet retrieves a single ontology
func (p *ManagementPlugin) handleGet(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)

	if ontologyID == "" {
		return nil, fmt.Errorf("ontology_id is required")
	}

	ontology, err := p.persistence.GetOntology(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	// Optionally load ontology content
	includeContent, _ := stepConfig.Config["include_content"].(bool)
	var content string
	if includeContent {
		contentBytes, err := os.ReadFile(ontology.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ontology file: %w", err)
		}
		content = string(contentBytes)
	}

	result := pipelines.NewPluginContext()
	result.Set("ontology", ontology)
	if includeContent {
		result.Set("content", content)
	}

	return result, nil
}

// handleDelete deletes an ontology
func (p *ManagementPlugin) handleDelete(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)

	if ontologyID == "" {
		return nil, fmt.Errorf("ontology_id is required")
	}

	// Get ontology metadata
	ontology, err := p.persistence.GetOntology(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	// Clear TDB2 graph
	if err := p.tdb2Backend.ClearGraph(ctx, ontology.TDB2Graph); err != nil {
		return nil, fmt.Errorf("failed to clear TDB2 graph: %w", err)
	}

	// Delete from database
	if err := p.persistence.DeleteOntology(ctx, ontologyID); err != nil {
		return nil, fmt.Errorf("failed to delete ontology metadata: %w", err)
	}

	// Delete file
	if err := os.Remove(ontology.FilePath); err != nil {
		// Log warning but don't fail (file might already be deleted)
		fmt.Printf("Warning: failed to delete ontology file %s: %v\n", ontology.FilePath, err)
	}

	result := pipelines.NewPluginContext()
	result.Set("status", "deleted")
	result.Set("ontology_id", ontologyID)

	return result, nil
}

// handleStats retrieves statistics for an ontology
func (p *ManagementPlugin) handleStats(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)

	if ontologyID == "" {
		return nil, fmt.Errorf("ontology_id is required")
	}

	// Get ontology metadata
	ontology, err := p.persistence.GetOntology(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	// Query TDB2 for graph statistics
	graphStats, err := p.tdb2Backend.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get graph stats: %w", err)
	}

	stats := &OntologyStats{
		OntologyID:   ontologyID,
		TotalTriples: graphStats.TotalTriples,
	}

	result := pipelines.NewPluginContext()
	result.Set("stats", stats)
	result.Set("ontology_name", ontology.Name)

	return result, nil
}

// validateOntologyData validates ontology syntax
func (p *ManagementPlugin) validateOntologyData(data, format string) ValidationResult {
	// Basic validation - check if data is not empty
	if strings.TrimSpace(data) == "" {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Severity: "error", Message: "Ontology data is empty"},
			},
		}
	}

	// Format-specific validation
	switch format {
	case "turtle", "ttl":
		return p.validateTurtle(data)
	case "rdfxml":
		return p.validateRDFXML(data)
	case "ntriples":
		return p.validateNTriples(data)
	case "jsonld":
		return p.validateJSONLD(data)
	default:
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Severity: "error", Message: fmt.Sprintf("Unknown format: %s", format)},
			},
		}
	}
}

// validateTurtle performs basic Turtle syntax validation
func (p *ManagementPlugin) validateTurtle(data string) ValidationResult {
	// Basic checks for Turtle format
	result := ValidationResult{Valid: true}

	// Check for common Turtle prefixes
	if !strings.Contains(data, "@prefix") && !strings.Contains(data, "PREFIX") {
		result.Warnings = append(result.Warnings, ValidationError{
			Severity: "warning",
			Message:  "No prefix declarations found",
		})
	}

	// Check for at least one triple with proper terminator
	// Look for lines that end with . (excluding prefix declarations)
	lines := strings.Split(data, "\n")
	hasValidTriple := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, comments, and prefix declarations
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@prefix") || strings.HasPrefix(line, "PREFIX") {
			continue
		}
		// Check if line ends with a period (proper triple terminator)
		if strings.HasSuffix(line, ".") {
			hasValidTriple = true
			break
		}
	}

	if !hasValidTriple {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Severity: "error",
			Message:  "No valid triples found (missing '.' statement terminator)",
		})
	}

	return result
}

// validateRDFXML performs basic RDF/XML syntax validation
func (p *ManagementPlugin) validateRDFXML(data string) ValidationResult {
	result := ValidationResult{Valid: true}

	if !strings.Contains(data, "<?xml") {
		result.Warnings = append(result.Warnings, ValidationError{
			Severity: "warning",
			Message:  "No XML declaration found",
		})
	}

	if !strings.Contains(data, "rdf:RDF") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Severity: "error",
			Message:  "No rdf:RDF root element found",
		})
	}

	return result
}

// validateNTriples performs basic N-Triples syntax validation
func (p *ManagementPlugin) validateNTriples(data string) ValidationResult {
	result := ValidationResult{Valid: true}

	lines := strings.Split(data, "\n")
	tripleCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ".") {
			tripleCount++
		}
	}

	if tripleCount == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Severity: "error",
			Message:  "No valid triples found",
		})
	}

	return result
}

// validateJSONLD performs basic JSON-LD syntax validation
func (p *ManagementPlugin) validateJSONLD(data string) ValidationResult {
	result := ValidationResult{Valid: true}

	if !strings.HasPrefix(strings.TrimSpace(data), "{") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Severity: "error",
			Message:  "Invalid JSON-LD: must be a JSON object",
		})
	}

	if !strings.Contains(data, "@context") {
		result.Warnings = append(result.Warnings, ValidationError{
			Severity: "warning",
			Message:  "No @context found in JSON-LD",
		})
	}

	return result
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		" ", "_",
	)
	return replacer.Replace(name)
}

// getFileExtension returns the file extension for a format
func getFileExtension(format string) string {
	switch format {
	case "turtle", "ttl":
		return "ttl"
	case "rdfxml":
		return "rdf"
	case "ntriples":
		return "nt"
	case "jsonld":
		return "jsonld"
	default:
		return "ttl"
	}
}

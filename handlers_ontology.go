package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	ontology "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	Storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// OntologyUploadRequest represents the request to upload an ontology
type OntologyUploadRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Version      string `json:"version"`
	Format       string `json:"format,omitempty"`
	OntologyData string `json:"ontology_data"`
	CreatedBy    string `json:"created_by,omitempty"`
}

// OntologyUploadResponse represents the response from uploading an ontology
type OntologyUploadResponse struct {
	OntologyID      string `json:"ontology_id"`
	OntologyName    string `json:"ontology_name"`
	OntologyVersion string `json:"ontology_version"`
	TDB2Graph       string `json:"tdb2_graph"`
	Status          string `json:"status"`
	Message         string `json:"message,omitempty"`
}

// SPARQLQueryRequest represents a SPARQL query request
type SPARQLQueryRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// ExtractionJobRequest represents a request to create an extraction job
type ExtractionJobRequest struct {
	OntologyID     string `json:"ontology_id"`
	JobName        string `json:"job_name,omitempty"`
	SourceType     string `json:"source_type"`
	ExtractionType string `json:"extraction_type,omitempty"`
	Data           any    `json:"data"`
}

// NLQueryRequest represents a natural language query request
type NLQueryRequest struct {
	Question   string `json:"question"`
	OntologyID string `json:"ontology_id,omitempty"`
}

// handleUploadOntology handles ontology upload requests
func (s *Server) handleUploadOntology(w http.ResponseWriter, r *http.Request) {
	// Check if ontology features are available
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	var req OntologyUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.Name == "" || req.Version == "" || req.OntologyData == "" {
		writeBadRequestResponse(w, "name, version, and ontology_data are required")
		return
	}

	// Get ontology management plugin
	plugin, err := s.registry.GetPlugin("Ontology", "management")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Ontology plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "upload_ontology",
		Plugin: "Ontology.management",
		Config: map[string]any{
			"operation":     "upload",
			"name":          req.Name,
			"description":   req.Description,
			"version":       req.Version,
			"format":        req.Format,
			"ontology_data": req.OntologyData,
			"created_by":    req.CreatedBy,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to upload ontology: %v", err))
		return
	}

	// Extract results
	ontologyID, _ := result.Get("ontology_id")
	ontologyName, _ := result.Get("ontology_name")
	ontologyVersion, _ := result.Get("ontology_version")
	tdb2Graph, _ := result.Get("tdb2_graph")
	status, _ := result.Get("status")

	response := OntologyUploadResponse{
		OntologyID:      fmt.Sprintf("%v", ontologyID),
		OntologyName:    fmt.Sprintf("%v", ontologyName),
		OntologyVersion: fmt.Sprintf("%v", ontologyVersion),
		TDB2Graph:       fmt.Sprintf("%v", tdb2Graph),
		Status:          fmt.Sprintf("%v", status),
		Message:         "Ontology uploaded successfully",
	}

	writeSuccessResponse(w, response)
}

// handleListOntologies handles requests to list all ontologies
func (s *Server) handleListOntologies(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	status := r.URL.Query().Get("status")

	ctx := context.Background()
	ontologies, err := s.persistence.ListOntologies(ctx, status)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list ontologies: %v", err))
		return
	}

	// Ensure we always return an array, never null
	if ontologies == nil {
		ontologies = []*Storage.Ontology{}
	}

	writeJSONResponse(w, http.StatusOK, ontologies)
}

// handleGetOntology handles requests to get a specific ontology
func (s *Server) handleGetOntology(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	ctx := context.Background()
	ontology, err := s.persistence.GetOntology(ctx, ontologyID)
	if err != nil {
		writeNotFoundResponse(w, fmt.Sprintf("Ontology not found: %v", err))
		return
	}

	// Check if content should be included
	includeContent := r.URL.Query().Get("include_content") == "true"

	response := map[string]any{
		"ontology": ontology,
	}

	if includeContent {
		contentBytes, err := os.ReadFile(ontology.FilePath)
		if err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to read ontology file: %v", err))
			return
		}
		response["content"] = string(contentBytes)
	}

	writeSuccessResponse(w, response)
}

// handleUpdateOntology handles requests to update an ontology
// Automatically creates a version if auto_version is enabled (default: true)
func (s *Server) handleUpdateOntology(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	var req struct {
		Name         string `json:"name,omitempty"`
		Description  string `json:"description,omitempty"`
		Version      string `json:"version,omitempty"`
		OntologyData string `json:"ontology_data,omitempty"`
		ModifiedBy   string `json:"modified_by,omitempty"`
		AutoVersion  *bool  `json:"auto_version,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get ontology management plugin
	plugin, err := s.registry.GetPlugin("Ontology", "management")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Ontology plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "update_ontology",
		Plugin: "Ontology.management",
		Config: map[string]any{
			"operation":     "update",
			"ontology_id":   ontologyID,
			"name":          req.Name,
			"description":   req.Description,
			"version":       req.Version,
			"ontology_data": req.OntologyData,
			"modified_by":   req.ModifiedBy,
		},
	}

	// Add auto_version setting if explicitly provided
	if req.AutoVersion != nil {
		stepConfig.Config["auto_version"] = *req.AutoVersion
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to update ontology: %v", err))
		return
	}

	// Extract results
	status, _ := result.Get("status")
	versionCreated, _ := result.Get("version_created")
	versionID, _ := result.Get("version_id")
	autoVersion, _ := result.Get("auto_version")

	response := map[string]any{
		"ontology_id":  ontologyID,
		"status":       status,
		"auto_version": autoVersion,
		"message":      "Ontology updated successfully",
	}

	if versionCreated != nil {
		response["version_created"] = versionCreated
		response["version_id"] = versionID
		response["auto_version_message"] = "Automatic version created"
	}

	writeSuccessResponse(w, response)
}

// handleDeleteOntology handles requests to delete an ontology
func (s *Server) handleDeleteOntology(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	// Get ontology management plugin
	plugin, err := s.registry.GetPlugin("Ontology", "management")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Ontology plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "delete_ontology",
		Plugin: "Ontology.management",
		Config: map[string]any{
			"operation":   "delete",
			"ontology_id": ontologyID,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to delete ontology: %v", err))
		return
	}

	status, _ := result.Get("status")

	response := map[string]any{
		"ontology_id": ontologyID,
		"status":      status,
		"message":     "Ontology deleted successfully",
	}

	writeSuccessResponse(w, response)
}

// handleValidateOntology handles requests to validate an ontology
func (s *Server) handleValidateOntology(w http.ResponseWriter, r *http.Request) {
	if s.registry == nil {
		writeInternalServerErrorResponse(w, "Plugin registry not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	// Validate from request body if no ID provided
	if ontologyID == "" {
		var req struct {
			OntologyData string `json:"ontology_data"`
			Format       string `json:"format,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		if req.OntologyData == "" {
			writeBadRequestResponse(w, "ontology_data is required")
			return
		}

		// Get ontology management plugin
		plugin, err := s.registry.GetPlugin("Ontology", "management")
		if err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Ontology plugin not available: %v", err))
			return
		}

		// Create step config
		stepConfig := pipelines.StepConfig{
			Name:   "validate_ontology",
			Plugin: "Ontology.management",
			Config: map[string]any{
				"operation":     "validate",
				"ontology_data": req.OntologyData,
				"format":        req.Format,
			},
		}

		// Execute plugin
		ctx := context.Background()
		globalContext := pipelines.NewPluginContext()
		result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		if err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Validation failed: %v", err))
			return
		}

		valid, _ := result.Get("valid")
		errors, _ := result.Get("errors")
		warnings, _ := result.Get("warnings")

		response := map[string]any{
			"valid":    valid,
			"errors":   errors,
			"warnings": warnings,
		}

		writeSuccessResponse(w, response)
		return
	}

	// Validate existing ontology
	response := map[string]any{
		"ontology_id": ontologyID,
		"message":     "Validation of existing ontologies not yet implemented",
	}

	writeSuccessResponse(w, response)
}

// handleSPARQLQuery handles SPARQL query requests with optional pagination
func (s *Server) handleSPARQLQuery(w http.ResponseWriter, r *http.Request) {
	if s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Knowledge graph features are not available")
		return
	}

	var req SPARQLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Query == "" {
		writeBadRequestResponse(w, "query is required")
		return
	}

	// Warn if query doesn't contain GRAPH clause
	queryUpper := strings.ToUpper(req.Query)
	if !strings.Contains(queryUpper, "GRAPH") {
		log := utils.GetLogger()
		log.Warn("User SPARQL query does not contain GRAPH clause - may return no results", utils.Component("sparql-query"))
	}

	// Apply pagination if requested
	query := req.Query
	if req.Limit > 0 {
		// Check if query already has LIMIT/OFFSET
		queryUpper := ""
		for _, ch := range query {
			if ch >= 'a' && ch <= 'z' {
				queryUpper += string(ch - 32)
			} else {
				queryUpper += string(ch)
			}
		}

		hasLimit := false
		hasOffset := false
		for i := 0; i < len(queryUpper)-5; i++ {
			if queryUpper[i:i+5] == "LIMIT" {
				hasLimit = true
			}
			if i < len(queryUpper)-6 && queryUpper[i:i+6] == "OFFSET" {
				hasOffset = true
			}
		}

		if !hasLimit {
			query = fmt.Sprintf("%s\nLIMIT %d", query, req.Limit)
		}
		if req.Offset > 0 && !hasOffset {
			query = fmt.Sprintf("%s\nOFFSET %d", query, req.Offset)
		}
	}

	ctx := context.Background()
	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Query failed: %v", err))
		return
	}

	// Add pagination metadata to response
	response := map[string]interface{}{
		"data":    result,
		"success": true,
	}

	if req.Limit > 0 {
		response["pagination"] = map[string]int{
			"limit":  req.Limit,
			"offset": req.Offset,
			"count":  len(result.Bindings),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleKnowledgeGraphStats handles requests for knowledge graph statistics
func (s *Server) handleKnowledgeGraphStats(w http.ResponseWriter, r *http.Request) {
	if s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Knowledge graph features are not available")
		return
	}

	ctx := context.Background()
	stats, err := s.tdb2Backend.Stats(ctx)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get stats: %v", err))
		return
	}

	writeSuccessResponse(w, stats)
}

// handleGetSubgraph handles requests to retrieve a subgraph for visualization
func (s *Server) handleGetSubgraph(w http.ResponseWriter, r *http.Request) {
	if s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Knowledge graph features are not available")
		return
	}

	rootURI := r.URL.Query().Get("root_uri")
	if rootURI == "" {
		writeBadRequestResponse(w, "root_uri query parameter is required")
		return
	}

	depth := 1
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		fmt.Sscanf(depthStr, "%d", &depth)
	}

	ctx := context.Background()
	viz, err := s.tdb2Backend.GetSubgraph(ctx, rootURI, depth)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get subgraph: %v", err))
		return
	}

	writeSuccessResponse(w, viz)
}

// handleOntologyStats handles requests for ontology-specific statistics
func (s *Server) handleOntologyStats(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	// Get ontology management plugin
	plugin, err := s.registry.GetPlugin("Ontology", "management")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Ontology plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "ontology_stats",
		Plugin: "Ontology.management",
		Config: map[string]any{
			"operation":   "stats",
			"ontology_id": ontologyID,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get stats: %v", err))
		return
	}

	stats, _ := result.Get("stats")
	ontologyName, _ := result.Get("ontology_name")

	response := map[string]any{
		"stats":         stats,
		"ontology_name": ontologyName,
	}

	writeSuccessResponse(w, response)
}

// handleExportOntology handles requests to export an ontology
func (s *Server) handleExportOntology(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "turtle"
	}

	ctx := context.Background()
	ontology, err := s.persistence.GetOntology(ctx, ontologyID)
	if err != nil {
		writeNotFoundResponse(w, fmt.Sprintf("Ontology not found: %v", err))
		return
	}

	// Read ontology content
	contentBytes, err := os.ReadFile(ontology.FilePath)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to read ontology file: %v", err))
		return
	}

	// Set appropriate content type
	contentType := "text/turtle"
	switch format {
	case "rdfxml":
		contentType = "application/rdf+xml"
	case "ntriples":
		contentType = "application/n-triples"
	case "jsonld":
		contentType = "application/ld+json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.%s\"", ontology.Name, ontology.Version, format))
	io.WriteString(w, string(contentBytes))
}

// handleCreateExtractionJob handles requests to create an entity extraction job
func (s *Server) handleCreateExtractionJob(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	var req ExtractionJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.OntologyID == "" || req.SourceType == "" || req.Data == nil {
		writeBadRequestResponse(w, "ontology_id, source_type, and data are required")
		return
	}

	// Get extraction plugin
	plugin, err := s.registry.GetPlugin("Ontology", "extraction")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Extraction plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "extract_entities",
		Plugin: "Ontology.extraction",
		Config: map[string]any{
			"operation":       "extract",
			"ontology_id":     req.OntologyID,
			"job_name":        req.JobName,
			"source_type":     req.SourceType,
			"extraction_type": req.ExtractionType,
			"data":            req.Data,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create extraction job: %v", err))
		return
	}

	// Extract results
	jobID, _ := result.Get("job_id")
	status, _ := result.Get("status")
	entitiesExtracted, _ := result.Get("entities_extracted")
	triplesGenerated, _ := result.Get("triples_generated")
	confidence, _ := result.Get("confidence")
	warnings, _ := result.Get("warnings")

	response := map[string]any{
		"job_id":             jobID,
		"status":             status,
		"entities_extracted": entitiesExtracted,
		"triples_generated":  triplesGenerated,
		"confidence":         confidence,
		"warnings":           warnings,
	}

	writeSuccessResponse(w, response)
}

// handleListExtractionJobs handles requests to list extraction jobs
func (s *Server) handleListExtractionJobs(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	// Get query parameters
	ontologyID := r.URL.Query().Get("ontology_id")
	status := r.URL.Query().Get("status")

	// Get extraction plugin
	plugin, err := s.registry.GetPlugin("Ontology", "extraction")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Extraction plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "list_extraction_jobs",
		Plugin: "Ontology.extraction",
		Config: map[string]any{
			"operation":   "list_jobs",
			"ontology_id": ontologyID,
			"status":      status,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list extraction jobs: %v", err))
		return
	}

	jobs, _ := result.Get("jobs")

	// Handle wrapped response (jobs might be {"value": [...]})
	var jobsList any = []map[string]any{}
	if jobsMap, ok := jobs.(map[string]any); ok {
		if val, exists := jobsMap["value"]; exists {
			jobsList = val
		} else {
			jobsList = jobs
		}
	} else {
		jobsList = jobs
	}

	// Ensure we return an array, not null
	if jobsList == nil {
		jobsList = []map[string]any{}
	}

	response := map[string]any{
		"jobs": jobsList,
	}

	writeSuccessResponse(w, response)
}

// handleGetExtractionJob handles requests to get a specific extraction job
func (s *Server) handleGetExtractionJob(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		writeBadRequestResponse(w, "job ID is required")
		return
	}

	// Get extraction plugin
	plugin, err := s.registry.GetPlugin("Ontology", "extraction")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Extraction plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "get_extraction_job",
		Plugin: "Ontology.extraction",
		Config: map[string]any{
			"operation": "get_job",
			"job_id":    jobID,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get extraction job: %v", err))
		return
	}

	job, _ := result.Get("job")
	entities, _ := result.Get("entities")

	response := map[string]any{
		"job":      job,
		"entities": entities,
	}

	writeSuccessResponse(w, response)
}

// handleNLQuery handles natural language query requests
func (s *Server) handleNLQuery(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil || s.tdb2Backend == nil {
		writeInternalServerErrorResponse(w, "Ontology features are not available")
		return
	}

	if s.llmClient == nil {
		writeInternalServerErrorResponse(w, "LLM client is not configured. Set OPENAI_API_KEY environment variable.")
		return
	}

	var req NLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Question == "" {
		writeBadRequestResponse(w, "question is required")
		return
	}

	// Get NL query plugin
	plugin, err := s.registry.GetPlugin("Ontology", "nl_query")
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("NL query plugin not available: %v", err))
		return
	}

	// Create step config
	stepConfig := pipelines.StepConfig{
		Name:   "nl_query",
		Plugin: "Ontology.nl_query",
		Config: map[string]any{
			"question":    req.Question,
			"ontology_id": req.OntologyID,
		},
	}

	// Execute plugin
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to execute NL query: %v", err))
		return
	}

	question, _ := result.Get("question")
	sparqlQuery, _ := result.Get("sparql_query")
	explanation, _ := result.Get("explanation")
	results, _ := result.Get("results")

	response := map[string]any{
		"question":     question,
		"sparql_query": sparqlQuery,
		"explanation":  explanation,
		"results":      results,
	}

	writeSuccessResponse(w, response)
}

// ==================== VERSIONING HANDLERS ====================

// CreateVersionRequest represents a request to create a new ontology version
type CreateVersionRequest struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	CreatedBy string `json:"created_by,omitempty"`
}

// handleCreateVersion creates a new version of an ontology
// POST /api/v1/ontology/:id/versions
//
// DEPRECATED: Manual version creation is deprecated. Versioning is now automatic
// when ontologies are updated (PUT /api/v1/ontology/:id). This endpoint is kept
// for backward compatibility but automatic versioning (enabled by default) is preferred.
// To disable automatic versioning for an ontology, set auto_version=false in the update request.
func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	var req CreateVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Version == "" {
		writeBadRequestResponse(w, "version is required")
		return
	}

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Check if auto-versioning is enabled - warn if it is
	autoVersionEnabled, err := versioningService.ShouldAutoVersion(ontologyID)
	if err == nil && autoVersionEnabled {
		// Return warning header
		w.Header().Set("X-Deprecation-Warning", "Manual versioning is deprecated. Automatic versioning is enabled for this ontology. Use PUT /api/v1/ontology/:id to trigger automatic versioning.")
	}

	// Create version
	version, err := versioningService.CreateVersion(ontologyID, req.Version, req.Changelog, req.CreatedBy)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create version: %v", err))
		return
	}

	response := map[string]any{
		"version":   version,
		"warning":   "Manual version creation is deprecated. Automatic versioning is now the default.",
		"migration": "Use PUT /api/v1/ontology/:id to update ontology and automatically create versions.",
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleListVersions lists all versions of an ontology
// GET /api/v1/ontology/:id/versions
func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Get versions
	versions, err := versioningService.GetVersions(ontologyID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get versions: %v", err))
		return
	}

	writeSuccessResponse(w, versions)
}

// handleGetVersion gets a specific version
// GET /api/v1/ontology/:id/versions/:vid
func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["vid"]

	// Parse version ID
	var vID int
	if _, err := fmt.Sscanf(versionID, "%d", &vID); err != nil {
		writeBadRequestResponse(w, "Invalid version ID")
		return
	}

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Get version
	version, err := versioningService.GetVersionByID(vID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get version: %v", err))
		return
	}

	// Get changes
	changes, err := versioningService.GetChanges(vID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get changes: %v", err))
		return
	}

	response := map[string]any{
		"version": version,
		"changes": changes,
	}

	writeSuccessResponse(w, response)
}

// handleCompareVersions compares two versions
// GET /api/v1/ontology/:id/versions/compare?v1=...&v2=...
func (s *Server) handleCompareVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	v1 := r.URL.Query().Get("v1")
	v2 := r.URL.Query().Get("v2")

	if v1 == "" || v2 == "" {
		writeBadRequestResponse(w, "Both v1 and v2 query parameters are required")
		return
	}

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Compare versions
	diff, err := versioningService.CompareVersions(ontologyID, v1, v2)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to compare versions: %v", err))
		return
	}

	writeSuccessResponse(w, diff)
}

// handleDeleteVersion deletes a version
// DELETE /api/v1/ontology/:id/versions/:vid
func (s *Server) handleDeleteVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["vid"]

	// Parse version ID
	var vID int
	if _, err := fmt.Sscanf(versionID, "%d", &vID); err != nil {
		writeBadRequestResponse(w, "Invalid version ID")
		return
	}

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Delete version
	if err := versioningService.DeleteVersion(vID); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to delete version: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSetAutoVersionSetting handles requests to enable/disable auto-versioning for an ontology
// PUT /api/v1/ontology/:id/auto-version
func (s *Server) handleSetAutoVersionSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get versioning service
	db := s.persistence.GetDB()
	versioningService := ontology.NewVersioningService(db)

	// Update setting
	if err := versioningService.SetAutoVersionSetting(ontologyID, req.Enabled); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to update auto-version setting: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"ontology_id":  ontologyID,
		"auto_version": req.Enabled,
		"message":      fmt.Sprintf("Auto-versioning %s for ontology", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]),
	})
}

// ========================================
// Drift Detection Handlers
// ========================================

// handleTriggerDriftDetection triggers drift detection for an ontology
// POST /api/v1/ontology/:id/drift/detect
func (s *Server) handleTriggerDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	// Parse request body
	var req struct {
		Source     string `json:"source"` // "extraction_job", "data", or "knowledge_graph"
		JobID      string `json:"job_id,omitempty"`
		Data       any    `json:"data,omitempty"`
		DataSource string `json:"data_source,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get database and clients
	db := s.persistence.GetDB()
	detector := ontology.NewDriftDetector(db, s.llmClient, s.tdb2Backend)

	var suggestionsCount int
	var err error

	// Execute drift detection based on source
	switch req.Source {
	case "extraction_job":
		if req.JobID == "" {
			writeBadRequestResponse(w, "job_id is required for extraction_job source")
			return
		}
		suggestionsCount, err = detector.DetectDriftFromExtractionJob(r.Context(), req.JobID)
	case "data":
		if req.Data == nil {
			writeBadRequestResponse(w, "data is required for data source")
			return
		}
		dataSource := req.DataSource
		if dataSource == "" {
			dataSource = "api_request"
		}
		suggestionsCount, err = detector.DetectDriftFromData(r.Context(), ontologyID, req.Data, dataSource)
	case "knowledge_graph":
		suggestionsCount, err = detector.MonitorKnowledgeGraphDrift(r.Context(), ontologyID)
	default:
		writeBadRequestResponse(w, fmt.Sprintf("Invalid source: %s (must be 'extraction_job', 'data', or 'knowledge_graph')", req.Source))
		return
	}

	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Drift detection failed: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message":               "Drift detection completed",
		"suggestions_generated": suggestionsCount,
		"ontology_id":           ontologyID,
	})
}

// handleGetDriftHistory retrieves drift detection history for an ontology
// GET /api/v1/ontology/:id/drift/history
func (s *Server) handleGetDriftHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	db := s.persistence.GetDB()

	query := `SELECT id, ontology_id, detection_type, data_source, suggestions_generated, status, 
	          started_at, completed_at, error_message 
	          FROM drift_detections WHERE ontology_id = ? ORDER BY started_at DESC`

	rows, err := db.QueryContext(r.Context(), query, ontologyID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to query drift history: %v", err))
		return
	}
	defer rows.Close()

	type DriftHistory struct {
		ID                   int        `json:"id"`
		OntologyID           string     `json:"ontology_id"`
		DetectionType        string     `json:"detection_type"`
		DataSource           string     `json:"data_source"`
		SuggestionsGenerated int        `json:"suggestions_generated"`
		Status               string     `json:"status"`
		StartedAt            time.Time  `json:"started_at"`
		CompletedAt          *time.Time `json:"completed_at,omitempty"`
		ErrorMessage         string     `json:"error_message,omitempty"`
	}

	history := make([]DriftHistory, 0)
	for rows.Next() {
		var h DriftHistory
		var completedAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(&h.ID, &h.OntologyID, &h.DetectionType, &h.DataSource,
			&h.SuggestionsGenerated, &h.Status, &h.StartedAt, &completedAt, &errorMessage)
		if err != nil {
			continue
		}

		if completedAt.Valid {
			h.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			h.ErrorMessage = errorMessage.String
		}

		history = append(history, h)
	}

	writeSuccessResponse(w, history)
}

// ========================================
// Suggestion Management Handlers
// ========================================

// handleListSuggestions retrieves suggestions for an ontology
// GET /api/v1/ontology/:id/suggestions
func (s *Server) handleListSuggestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	// Parse query parameters
	statusFilter := ontology.SuggestionStatus(r.URL.Query().Get("status"))

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// List suggestions
	suggestions, err := engine.ListSuggestions(r.Context(), ontologyID, statusFilter)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list suggestions: %v", err))
		return
	}

	writeSuccessResponse(w, suggestions)
}

// handleGetSuggestion retrieves a single suggestion
// GET /api/v1/ontology/:id/suggestions/:sid
func (s *Server) handleGetSuggestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suggestionID := vars["sid"]

	// Parse suggestion ID
	var sID int
	if _, err := fmt.Sscanf(suggestionID, "%d", &sID); err != nil {
		writeBadRequestResponse(w, "Invalid suggestion ID")
		return
	}

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// Get suggestion
	suggestion, err := engine.GetSuggestion(r.Context(), sID)
	if err != nil {
		if err.Error() == "suggestion not found" {
			writeNotFoundResponse(w, "Suggestion not found")
			return
		}
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get suggestion: %v", err))
		return
	}

	writeSuccessResponse(w, suggestion)
}

// handleApproveSuggestion approves a suggestion
// POST /api/v1/ontology/:id/suggestions/:sid/approve
func (s *Server) handleApproveSuggestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suggestionID := vars["sid"]

	// Parse suggestion ID
	var sID int
	if _, err := fmt.Sscanf(suggestionID, "%d", &sID); err != nil {
		writeBadRequestResponse(w, "Invalid suggestion ID")
		return
	}

	// Parse request body
	var req struct {
		ReviewedBy  string `json:"reviewed_by"`
		ReviewNotes string `json:"review_notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReviewedBy == "" {
		writeBadRequestResponse(w, "reviewed_by is required")
		return
	}

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// Approve suggestion
	if err := engine.ApproveSuggestion(r.Context(), sID, req.ReviewedBy, req.ReviewNotes); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to approve suggestion: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message":       "Suggestion approved",
		"suggestion_id": sID,
		"status":        "approved",
	})
}

// handleRejectSuggestion rejects a suggestion
// POST /api/v1/ontology/:id/suggestions/:sid/reject
func (s *Server) handleRejectSuggestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suggestionID := vars["sid"]

	// Parse suggestion ID
	var sID int
	if _, err := fmt.Sscanf(suggestionID, "%d", &sID); err != nil {
		writeBadRequestResponse(w, "Invalid suggestion ID")
		return
	}

	// Parse request body
	var req struct {
		ReviewedBy  string `json:"reviewed_by"`
		ReviewNotes string `json:"review_notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReviewedBy == "" {
		writeBadRequestResponse(w, "reviewed_by is required")
		return
	}

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// Reject suggestion
	if err := engine.RejectSuggestion(r.Context(), sID, req.ReviewedBy, req.ReviewNotes); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to reject suggestion: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message":       "Suggestion rejected",
		"suggestion_id": sID,
		"status":        "rejected",
	})
}

// handleApplySuggestion applies an approved suggestion to the ontology
// POST /api/v1/ontology/:id/suggestions/:sid/apply
func (s *Server) handleApplySuggestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suggestionID := vars["sid"]

	// Parse suggestion ID
	var sID int
	if _, err := fmt.Sscanf(suggestionID, "%d", &sID); err != nil {
		writeBadRequestResponse(w, "Invalid suggestion ID")
		return
	}

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// Apply suggestion
	if err := engine.ApplySuggestion(r.Context(), sID); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to apply suggestion: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message":       "Suggestion applied successfully",
		"suggestion_id": sID,
		"status":        "applied",
	})
}

// handleGetSuggestionSummary retrieves a summary of suggestions
// GET /api/v1/ontology/:id/suggestions/summary
func (s *Server) handleGetSuggestionSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	// Get suggestion engine
	db := s.persistence.GetDB()
	engine := ontology.NewSuggestionEngine(db, s.llmClient, s.tdb2Backend)

	// Get summary
	summary, err := engine.GenerateSuggestionSummary(r.Context(), ontologyID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to generate summary: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"ontology_id": ontologyID,
		"summary":     summary,
	})
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	ontology "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
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
	Query string `json:"query"`
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
func (s *Server) handleUpdateOntology(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology ID is required")
		return
	}

	var req OntologyUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// TODO: Implement ontology update logic
	response := map[string]any{
		"ontology_id": ontologyID,
		"status":      "updated",
		"message":     "Ontology update not yet implemented",
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

// handleSPARQLQuery handles SPARQL query requests
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

	ctx := context.Background()
	result, err := s.tdb2Backend.QuerySPARQL(ctx, req.Query)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Query failed: %v", err))
		return
	}

	writeSuccessResponse(w, result)
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

	response := map[string]any{
		"jobs": jobs,
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

	// Create version
	version, err := versioningService.CreateVersion(ontologyID, req.Version, req.Changelog, req.CreatedBy)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create version: %v", err))
		return
	}

	writeSuccessResponse(w, version)
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
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

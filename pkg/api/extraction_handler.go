package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

// ExtractionHandler handles entity extraction HTTP requests
type ExtractionHandler struct {
	extractionService *extraction.Service
	ontologyService   *ontology.Service
}

// NewExtractionHandler creates a new extraction handler
func NewExtractionHandler(extractionService *extraction.Service, ontologyService *ontology.Service) *ExtractionHandler {
	return &ExtractionHandler{
		extractionService: extractionService,
		ontologyService:   ontologyService,
	}
}

// HandleExtractAndGenerate handles POST /api/extraction/generate-ontology
// Extracts entities from storage and generates an ontology
func (h *ExtractionHandler) HandleExtractAndGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OntologyExtractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Extract entities from storage
	extractionResult, err := h.extractionService.ExtractFromStorage(
		req.ProjectID,
		req.StorageIDs,
		req.IncludeStructured,
		req.IncludeUnstructured,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract entities: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate ontology from extraction results
	ontology, err := h.ontologyService.GenerateFromExtraction(req.ProjectID, req.OntologyName, extractionResult)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate ontology: %v", err), http.StatusInternalServerError)
		return
	}

	// Return ontology
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ontology": ontology,
		"extraction_summary": map[string]interface{}{
			"entities_count":      len(extractionResult.Entities),
			"relationships_count": len(extractionResult.Relationships),
		},
	})
}

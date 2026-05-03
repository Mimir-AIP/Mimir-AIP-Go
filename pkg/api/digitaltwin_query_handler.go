package api

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
)

// handleDigitalTwinQuery handles POST /api/digital-twins/{id}/query
func (h *DigitalTwinHandler) handleDigitalTwinQuery(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.QueryForProject(projectIDFromRequest(r), twinID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute query: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDigitalTwinPredict handles POST /api/digital-twins/{id}/predict
func (h *DigitalTwinHandler) handleDigitalTwinPredict(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if it's a batch prediction request
	var rawReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Check if inputs field exists (batch request)
	if inputs, ok := rawReq["inputs"]; ok && inputs != nil {
		// Batch prediction
		reqBytes, _ := json.Marshal(rawReq)
		var batchReq models.BatchPredictionRequest
		if err := json.Unmarshal(reqBytes, &batchReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid batch prediction request: %v", err), http.StatusBadRequest)
			return
		}

		if err := batchReq.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		predictions, err := h.service.BatchPredictForProject(projectIDFromRequest(r), twinID, &batchReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to run batch predictions: %v", err), digitalTwinErrorStatus(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predictions)
	} else {
		// Single prediction
		reqBytes, _ := json.Marshal(rawReq)
		var predReq models.PredictionRequest
		if err := json.Unmarshal(reqBytes, &predReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid prediction request: %v", err), http.StatusBadRequest)
			return
		}

		if err := predReq.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		prediction, err := h.service.PredictForProject(projectIDFromRequest(r), twinID, &predReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to run prediction: %v", err), digitalTwinErrorStatus(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prediction)
	}
}

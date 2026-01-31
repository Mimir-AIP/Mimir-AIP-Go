package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// File-Based Ingestion Tests
// ============================================================================

// TestContinuousIngestion_FileBasedAutomation tests automation with file-based input
// that simulates file watcher triggering
func TestContinuousIngestion_FileBasedAutomation(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create test CSV file
	csvFile := filepath.Join(tmpDir, "sales_data.csv")
	csvContent := `transaction_id,customer_name,amount,currency,date,timestamp
TXN-001,Alice Smith,150.00,USD,2026-01-30,2026-01-30T10:00:00Z
TXN-002,Bob Johnson,230.50,USD,2026-01-30,2026-01-30T10:05:00Z
TXN-003,Charlie Brown,89.99,EUR,2026-01-30,2026-01-30T10:10:00Z
TXN-004,Diana Prince,450.00,USD,2026-01-30,2026-01-30T10:15:00Z
TXN-005,Eve Anderson,125.75,GBP,2026-01-30,2026-01-30T10:20:00Z`

	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	var pipelineID string

	// Create pipeline for file processing
	t.Run("Create file-based pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":        "automated-sales-ingestion",
				"description": "Automatically process sales data from CSV files",
				"enabled":     true,
				"tags":        []string{"sales", "csv", "automated"},
			},
			"config": map[string]any{
				"name":    "automated-sales-ingestion",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "read-csv",
						"plugin": "Input.csv",
						"config": map[string]any{
							"file_path":   csvFile,
							"has_headers": true,
							"delimiter":   ",",
						},
						"output": "raw_sales_data",
					},
					{
						"name":   "filter-usd-transactions",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "raw_sales_data",
							"operation": "filter",
							"field":     "currency",
							"op":        "==",
							"value":     "USD",
						},
						"output": "usd_transactions",
					},
					{
						"name":   "select-relevant-fields",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "usd_transactions",
							"operation": "select",
							"fields":    []string{"transaction_id", "customer_name", "amount", "timestamp"},
						},
						"output": "processed_transactions",
					},
					{
						"name":   "validate-transactions",
						"plugin": "Data_Processing.validate",
						"config": map[string]any{
							"input": "processed_transactions",
							"rules": map[string]any{
								"required": []string{"transaction_id", "customer_name", "amount"},
								"types": map[string]any{
									"amount": "number",
								},
							},
						},
						"output": "validated_transactions",
					},
					{
						"name":   "aggregate-by-customer",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "validated_transactions",
							"operation": "aggregate",
							"group_by":  []string{"customer_name"},
							"aggregations": []map[string]any{
								{"field": "amount", "op": "sum", "as": "total_amount"},
								{"field": "amount", "op": "count", "as": "transaction_count"},
							},
						},
						"output": "customer_summary",
					},
				},
			},
		}

		body, _ := json.Marshal(pipelineDef)
		req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		require.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		pipeline, ok := response["pipeline"].(map[string]any)
		require.True(t, ok)

		pipelineID = pipeline["id"].(string)
	})

	// Execute the pipeline
	t.Run("Execute file-based pipeline", func(t *testing.T) {
		require.NotEmpty(t, pipelineID)

		execReq := PipelineExecutionRequest{
			PipelineID: pipelineID,
		}

		body, _ := json.Marshal(execReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/execute", pipelineID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PipelineExecutionResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
	})
}

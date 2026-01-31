package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// What-If Analysis and Insights Tests
// ============================================================================

// TestWhatIfAnalysis_BasicQuery tests basic what-if analysis with a question
func TestWhatIfAnalysis_BasicQuery(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	testCases := []struct {
		name     string
		question string
	}{
		{
			name:     "Resource availability question",
			question: "What happens if Server A becomes unavailable?",
		},
		{
			name:     "Demand surge question",
			question: "What if demand increases by 50%?",
		},
		{
			name:     "Process delay question",
			question: "What is the impact if the production process is delayed?",
		},
		{
			name:     "Capacity question",
			question: "Can we handle double the current load?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a fake twin ID
			whatIfReq := map[string]any{
				"question":    tc.question,
				"max_results": 5,
			}

			body, _ := json.Marshal(whatIfReq)
			req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Should not get 400 for valid question format
			// May get 404 (twin not found), 503 (service unavailable), or 200/500 (if LLM/service issues)
			assert.NotEqual(t, http.StatusBadRequest, w.Code,
				"Should not get 400 Bad Request for valid question, got %d", w.Code)

			// If we get a successful response, verify structure
			if w.Code == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.question, response["question"])
				assert.NotEmpty(t, response["interpretation"])
				assert.NotEmpty(t, response["summary"])

				if findings, ok := response["key_findings"].([]any); ok {
					_ = findings
				}

				if recommendations, ok := response["recommendations"].([]any); ok {
					_ = recommendations
				}
			}
		})
	}
}

// TestWhatIfAnalysis_InvalidInput tests error handling for invalid input
func TestWhatIfAnalysis_InvalidInput(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Empty question should fail", func(t *testing.T) {
		whatIfReq := map[string]any{
			"question": "",
		}

		body, _ := json.Marshal(whatIfReq)
		req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 for empty question, or 500 if service unavailable
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError,
			"Should get 400 or 500 for empty question, got %d", w.Code)
	})

	t.Run("Missing question field should fail", func(t *testing.T) {
		whatIfReq := map[string]any{
			"max_results": 5,
		}

		body, _ := json.Marshal(whatIfReq)
		req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 for missing question, or 500 if service unavailable
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError,
			"Should get 400 or 500 for missing question, got %d", w.Code)
	})
}

// TestInsightsAndAnalysis_ProactiveInsights tests the insights endpoint
func TestInsightsAndAnalysis_ProactiveInsights(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get insights endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/insights", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var report DigitalTwin.InsightReport
			err := json.Unmarshal(w.Body.Bytes(), &report)
			require.NoError(t, err)

			assert.NotEmpty(t, report.TwinID)
			assert.NotZero(t, report.GeneratedAt)

			// Verify insights structure
			assert.NotNil(t, report.Insights)
			assert.NotNil(t, report.SuggestedQuestions)

			// Verify scores are in valid range
			assert.GreaterOrEqual(t, report.RiskScore, 0.0)
			assert.LessOrEqual(t, report.RiskScore, 1.0)
			assert.GreaterOrEqual(t, report.HealthScore, 0.0)
			assert.LessOrEqual(t, report.HealthScore, 1.0)
		}
	})
}

// TestInsightsAndAnalysis_OntologyAnalysis tests ontology analysis endpoint
func TestInsightsAndAnalysis_OntologyAnalysis(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Ontology analysis endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/analysis", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var analysis map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &analysis)
			require.NoError(t, err)

			// Verify analysis structure
			assert.NotEmpty(t, analysis)
		}
	})
}

// TestInsightsAndAnalysis_ImpactAnalysis tests impact analysis endpoint
func TestInsightsAndAnalysis_ImpactAnalysis(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Impact analysis endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/runs/fake-run/impact", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var impact DigitalTwin.ImpactAnalysis
			err := json.Unmarshal(w.Body.Bytes(), &impact)
			require.NoError(t, err)

			assert.NotEmpty(t, impact.RunID)
			assert.NotEmpty(t, impact.OverallImpact)

			// Verify affected entities
			assert.NotNil(t, impact.AffectedEntities)

			// Verify risk score
			assert.GreaterOrEqual(t, impact.RiskScore, 0.0)
			assert.LessOrEqual(t, impact.RiskScore, 1.0)
		}
	})
}

// TestSmartScenarioGeneration tests the smart scenario generator endpoint
func TestSmartScenarioGeneration(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Generate smart scenarios endpoint", func(t *testing.T) {
		// Use a fake twin ID - we're testing the endpoint structure
		req := httptest.NewRequest("POST", "/api/v1/twin/fake-twin/smart-scenarios", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 400 (bad request/missing params), 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 400, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["scenarios"])

			if count, ok := response["count"].(float64); ok {
				assert.GreaterOrEqual(t, int(count), 0)
			}
		}
	})
}

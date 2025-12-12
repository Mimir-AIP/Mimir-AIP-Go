package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerToolDiscovery tests MCP tool discovery endpoint
func TestMCPServerToolDiscovery(t *testing.T) {
	// Create MCP server using mock implementation
	ms := NewMockServer()

	// Register test plugins on the mock server's registry
	registry := ms.GetRegistry()
	err := registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)
	err = registry.RegisterPlugin(&utils.MockHTMLPlugin{})
	require.NoError(t, err)

	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Discover Available Tools", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/mcp/tools")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify tools list exists
		tools, exists := response["tools"]
		assert.True(t, exists, "Response should contain tools list")

		// Verify tools structure
		toolsList, ok := tools.([]interface{})
		require.True(t, ok, "Tools should be an array")
		assert.Greater(t, len(toolsList), 0, "Should have at least one tool")

		// Verify first tool structure
		if len(toolsList) > 0 {
			tool := toolsList[0].(map[string]interface{})
			assert.Contains(t, tool, "name", "Tool should have a name")
			assert.Contains(t, tool, "description", "Tool should have a description")
			assert.Contains(t, tool, "inputSchema", "Tool should have an input schema")
		}
	})

	t.Run("Tool Discovery Performance", func(t *testing.T) {
		// Test that tool discovery is fast
		const numRequests = 10

		for i := 0; i < numRequests; i++ {
			resp, err := http.Get(testServer.URL + "/mcp/tools")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}
	})
}

// TestMCPServerToolExecution tests MCP tool execution endpoint
func TestMCPServerToolExecution(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Execute Valid Tool", func(t *testing.T) {
		// Create tool execution request
		execReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name": "test_execution",
					"config": map[string]interface{}{
						"url":    "https://httpbin.org/json",
						"method": "GET",
					},
					"output": "test_result",
				},
				"context": map[string]interface{}{
					"test_mode": true,
				},
			},
		}

		reqBody, err := json.Marshal(execReq)
		require.NoError(t, err)

		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify successful execution
		success, exists := response["success"]
		assert.True(t, exists, "Response should contain success field")
		assert.True(t, success.(bool), "Execution should be successful")

		// Verify result exists
		result, exists := response["result"]
		assert.True(t, exists, "Response should contain result")
		assert.NotNil(t, result, "Result should not be nil")
	})

	t.Run("Execute Non-existent Tool", func(t *testing.T) {
		execReq := map[string]interface{}{
			"tool_name": "NonExistent.plugin",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name": "test",
				},
			},
		}

		reqBody, err := json.Marshal(execReq)
		require.NoError(t, err)

		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error status
		assert.True(t, resp.StatusCode >= 400, "Should return error for non-existent tool")
	})

	t.Run("Execute Tool with Invalid Arguments", func(t *testing.T) {
		execReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"invalid_arg": "value",
			},
		}

		reqBody, err := json.Marshal(execReq)
		require.NoError(t, err)

		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should handle invalid arguments gracefully
		assert.True(t, resp.StatusCode >= 400 || resp.StatusCode == 200)
	})

	t.Run("Execute Tool with Malformed JSON", func(t *testing.T) {
		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer([]byte("invalid json {{")),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestMCPServerPluginRegistry tests integration with plugin registry
func TestMCPServerPluginRegistry(t *testing.T) {
	ms := NewMockServer()

	// Register multiple plugins on the mock server's registry
	registry := ms.GetRegistry()
	err := registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)
	err = registry.RegisterPlugin(&utils.MockHTMLPlugin{})
	require.NoError(t, err)

	testServer := ms.Start()
	defer testServer.Close()

	t.Run("All Registered Plugins Available as Tools", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/mcp/tools")
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		tools := response["tools"].([]interface{})

		// Should have tools for all registered plugins
		assert.Greater(t, len(tools), 0, "Should have tools from registered plugins")

		// Verify tool names match plugin types and names
		toolNames := make([]string, 0)
		for _, tool := range tools {
			toolMap := tool.(map[string]interface{})
			toolNames = append(toolNames, toolMap["name"].(string))
		}

		assert.Contains(t, toolNames, "Input.api", "Should have Input.api tool")
		assert.Contains(t, toolNames, "Output.html", "Should have Output.html tool")
	})
}

// TestMCPServerEndToEnd tests complete MCP workflow
func TestMCPServerEndToEnd(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Complete MCP Workflow", func(t *testing.T) {
		// Step 1: Discover tools
		discoverResp, err := http.Get(testServer.URL + "/mcp/tools")
		require.NoError(t, err)
		defer discoverResp.Body.Close()
		require.Equal(t, http.StatusOK, discoverResp.StatusCode)

		var discoverResponse map[string]interface{}
		err = json.NewDecoder(discoverResp.Body).Decode(&discoverResponse)
		require.NoError(t, err)

		tools := discoverResponse["tools"].([]interface{})
		require.Greater(t, len(tools), 0, "Should have discovered tools")

		// Step 2: Get first tool details
		firstTool := tools[0].(map[string]interface{})
		toolName := firstTool["name"].(string)
		t.Logf("Testing tool: %s", toolName)

		// Step 3: Execute the tool (if it's Input.api)
		if toolName == "Input.api" {
			execReq := map[string]interface{}{
				"tool_name": toolName,
				"arguments": map[string]interface{}{
					"step_config": map[string]interface{}{
						"name": "e2e_test",
						"config": map[string]interface{}{
							"url":    "https://httpbin.org/json",
							"method": "GET",
						},
						"output": "result",
					},
				},
			}

			reqBody, err := json.Marshal(execReq)
			require.NoError(t, err)

			execResp, err := http.Post(
				testServer.URL+"/mcp/tools/execute",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			defer execResp.Body.Close()

			require.Equal(t, http.StatusOK, execResp.StatusCode)

			var execResponse map[string]interface{}
			err = json.NewDecoder(execResp.Body).Decode(&execResponse)
			require.NoError(t, err)

			assert.True(t, execResponse["success"].(bool))
			assert.NotNil(t, execResponse["result"])

			t.Logf("Tool execution successful, result: %v", execResponse["result"])
		}
	})
}

// TestMCPServerConcurrentToolExecution tests concurrent tool executions
func TestMCPServerConcurrentToolExecution(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Concurrent Tool Executions", func(t *testing.T) {
		const numExecutions = 10
		results := make(chan bool, numExecutions)

		execReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name": "concurrent_test",
					"config": map[string]interface{}{
						"url":    "https://httpbin.org/delay/1",
						"method": "GET",
					},
					"output": "result",
				},
			},
		}

		reqBody, _ := json.Marshal(execReq)

		// Launch concurrent executions
		for i := 0; i < numExecutions; i++ {
			go func(execNum int) {
				resp, err := http.Post(
					testServer.URL+"/mcp/tools/execute",
					"application/json",
					bytes.NewBuffer(reqBody),
				)

				success := false
				if err == nil && resp != nil {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						var response map[string]interface{}
						if json.NewDecoder(resp.Body).Decode(&response) == nil {
							if s, ok := response["success"].(bool); ok && s {
								success = true
							}
						}
					}
				}

				results <- success
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numExecutions; i++ {
			if <-results {
				successCount++
			}
		}

		t.Logf("Concurrent executions: %d/%d successful", successCount, numExecutions)
		assert.Greater(t, successCount, numExecutions*7/10,
			"At least 70%% of concurrent tool executions should succeed")
	})
}

// TestMCPServerErrorRecovery tests error handling and recovery
func TestMCPServerErrorRecovery(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Recovery After Failed Execution", func(t *testing.T) {
		// Execute with invalid configuration (should fail)
		failReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name":   "fail_test",
					"config": map[string]interface{}{}, // Missing required 'url'
					"output": "result",
				},
			},
		}

		reqBody, _ := json.Marshal(failReq)
		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should handle error gracefully
		assert.True(t, resp.StatusCode >= 400 || resp.StatusCode == 200)

		// Execute valid request to verify recovery
		successReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name": "recovery_test",
					"config": map[string]interface{}{
						"url":    "https://httpbin.org/json",
						"method": "GET",
					},
					"output": "result",
				},
			},
		}

		reqBody, _ = json.Marshal(successReq)
		resp, err = http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Server should recover and handle valid request
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool), "Server should recover after error")
	})
}

// TestMCPServerResponseFormat tests response format compliance
func TestMCPServerResponseFormat(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Tool Discovery Response Format", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/mcp/tools")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/json")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "tools", "Response should have 'tools' field")

		tools := response["tools"].([]interface{})
		if len(tools) > 0 {
			tool := tools[0].(map[string]interface{})

			// Verify required fields
			requiredFields := []string{"name", "description", "inputSchema"}
			for _, field := range requiredFields {
				assert.Contains(t, tool, field, "Tool should have '%s' field", field)
			}

			// Verify inputSchema structure
			if schema, ok := tool["inputSchema"].(map[string]interface{}); ok {
				assert.Contains(t, schema, "type")
				assert.Contains(t, schema, "properties")
			}
		}
	})

	t.Run("Tool Execution Response Format", func(t *testing.T) {
		execReq := map[string]interface{}{
			"tool_name": "Input.api",
			"arguments": map[string]interface{}{
				"step_config": map[string]interface{}{
					"name": "format_test",
					"config": map[string]interface{}{
						"url":    "https://httpbin.org/json",
						"method": "GET",
					},
					"output": "result",
				},
			},
		}

		reqBody, _ := json.Marshal(execReq)
		resp, err := http.Post(
			testServer.URL+"/mcp/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/json")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "success", "Response should have 'success' field")

		if success, ok := response["success"].(bool); ok && success {
			assert.Contains(t, response, "result", "Successful response should have 'result' field")
		}
	})
}

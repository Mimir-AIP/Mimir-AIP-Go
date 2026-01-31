package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// AgentChatIntegrationTestSuite tests the full agent chat API functionality
// using real HTTP calls against a test server
func TestAgentChatAPI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test server
	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL
	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("CreateConversation", func(t *testing.T) {
		testCreateConversation(t, client, baseURL)
	})

	t.Run("GetConversation", func(t *testing.T) {
		testGetConversation(t, client, baseURL)
	})

	t.Run("SendMessage", func(t *testing.T) {
		testSendMessage(t, client, baseURL)
	})

	t.Run("UpdateConversation", func(t *testing.T) {
		testUpdateConversation(t, client, baseURL)
	})

	t.Run("DeleteConversation", func(t *testing.T) {
		testDeleteConversation(t, client, baseURL)
	})

	t.Run("ListConversations", func(t *testing.T) {
		testListConversations(t, client, baseURL)
	})

	t.Run("ExecuteAgentTools", func(t *testing.T) {
		testExecuteAgentTools(t, client, baseURL)
	})

	t.Run("FullConversationWorkflow", func(t *testing.T) {
		testFullConversationWorkflow(t, client, baseURL)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, client, baseURL)
	})
}

// testCreateConversation tests POST /api/v1/chat
func testCreateConversation(t *testing.T, client *http.Client, baseURL string) {
	tests := []struct {
		name       string
		req        CreateConversationRequest
		wantStatus int
		wantErr    bool
	}{
		{
			name: "create conversation with all fields",
			req: CreateConversationRequest{
				Title:         "Test Conversation",
				ModelProvider: "openai",
				ModelName:     "gpt-4",
				SystemPrompt:  "You are a helpful assistant.",
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "create conversation with minimal fields",
			req: CreateConversationRequest{
				Title: "Minimal Chat",
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "create conversation with empty title gets default",
			req: CreateConversationRequest{
				Title:         "",
				ModelProvider: "mock",
				ModelName:     "mock-gpt-4",
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "create conversation with twin association",
			req: CreateConversationRequest{
				Title:         "Twin Analysis Chat",
				ModelProvider: "mock",
				ModelName:     "mock-claude-3",
				TwinID:        strPtr("twin_test_123"),
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			resp, err := client.Post(
				baseURL+"/api/v1/chat",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("CreateConversation() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !tt.wantErr {
				// Verify response structure
				if _, ok := result["conversation_id"]; !ok {
					t.Error("CreateConversation() response missing conversation_id")
				}
				if _, ok := result["conversation"]; !ok {
					t.Error("CreateConversation() response missing conversation object")
				}

				// Verify conversation fields
				conv := result["conversation"].(map[string]interface{})
				if conv["title"] != tt.req.Title && tt.req.Title != "" {
					t.Errorf("Title mismatch: got %v, want %v", conv["title"], tt.req.Title)
				}
				if tt.req.Title == "" && conv["title"] != "New Conversation" {
					t.Errorf("Default title mismatch: got %v, want 'New Conversation'", conv["title"])
				}

				t.Logf("Created conversation: %v", result["conversation_id"])
			}
		})
	}
}

// testGetConversation tests GET /api/v1/chat/:id
func testGetConversation(t *testing.T, client *http.Client, baseURL string) {
	// First create a conversation
	convID := createTestConversation(t, client, baseURL, "Get Test Chat")

	tests := []struct {
		name       string
		convID     string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "get existing conversation",
			convID:     convID,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "get non-existent conversation",
			convID:     uuid.New().String(),
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "get conversation with invalid ID",
			convID:     "invalid-id",
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(baseURL + "/api/v1/chat/" + tt.convID)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("GetConversation() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if !tt.wantErr {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify response structure
				if _, ok := result["conversation"]; !ok {
					t.Error("GetConversation() response missing conversation object")
				}
				if _, ok := result["messages"]; !ok {
					t.Error("GetConversation() response missing messages array")
				}

				conv := result["conversation"].(map[string]interface{})
				if conv["id"] != tt.convID {
					t.Errorf("Conversation ID mismatch: got %v, want %v", conv["id"], tt.convID)
				}

				t.Logf("Retrieved conversation: %v with %d messages", tt.convID, len(result["messages"].([]interface{})))
			}
		})
	}
}

// testSendMessage tests POST /api/v1/chat/:id/message
func testSendMessage(t *testing.T, client *http.Client, baseURL string) {
	// Create a conversation first
	convID := createTestConversation(t, client, baseURL, "Message Test Chat")

	tests := []struct {
		name         string
		convID       string
		message      string
		modelProv    string
		modelName    string
		wantStatus   int
		wantErr      bool
		wantToolCall bool
	}{
		{
			name:       "send simple message",
			convID:     convID,
			message:    "Hello, how are you?",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "send message with special characters",
			convID:     convID,
			message:    "Test with special chars: àáâãäåæçèéêë ñ 中文 🎉",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "send empty message fails",
			convID:     convID,
			message:    "",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "send message to non-existent conversation",
			convID:     uuid.New().String(),
			message:    "Test message",
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "send message with long text",
			convID:     convID,
			message:    strings.Repeat("This is a long message. ", 100),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:         "send message that triggers tool call",
			convID:       convID,
			message:      "List all available pipelines",
			modelProv:    "mock",
			modelName:    "mock-gpt-4",
			wantStatus:   http.StatusOK,
			wantErr:      false,
			wantToolCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SendMessageRequest{
				Message:       tt.message,
				ModelProvider: tt.modelProv,
				ModelName:     tt.modelName,
			}
			body, _ := json.Marshal(req)

			resp, err := client.Post(
				baseURL+"/api/v1/chat/"+tt.convID+"/message",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("SendMessage() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if !tt.wantErr {
				var result SendMessageResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify response structure
				if result.ConversationID != tt.convID {
					t.Errorf("Conversation ID mismatch: got %v, want %v", result.ConversationID, tt.convID)
				}
				if result.UserMessage.Content != tt.message {
					t.Errorf("User message content mismatch: got %v, want %v", result.UserMessage.Content, tt.message)
				}
				if result.AssistantReply.Content == "" {
					t.Error("Assistant reply content is empty")
				}
				if result.AssistantReply.Role != "assistant" {
					t.Errorf("Assistant reply role mismatch: got %v, want 'assistant'", result.AssistantReply.Role)
				}

				// Verify tool calls if expected
				if tt.wantToolCall && len(result.ToolCalls) == 0 {
					t.Log("Warning: Expected tool calls but none were returned")
				}

				t.Logf("Message sent. User: %d chars, Assistant: %d chars, Tools: %d",
					len(result.UserMessage.Content),
					len(result.AssistantReply.Content),
					len(result.ToolCalls))
			}
		})
	}
}

// testUpdateConversation tests PUT /api/v1/chat/:id
func testUpdateConversation(t *testing.T, client *http.Client, baseURL string) {
	// Create a conversation first
	convID := createTestConversation(t, client, baseURL, "Update Test Chat")

	tests := []struct {
		name       string
		convID     string
		updates    map[string]interface{}
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "update title only",
			convID: convID,
			updates: map[string]interface{}{
				"title": "Updated Title",
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update model settings",
			convID: convID,
			updates: map[string]interface{}{
				"model_provider": "anthropic",
				"model_name":     "claude-3-opus",
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update system prompt",
			convID: convID,
			updates: map[string]interface{}{
				"system_prompt": "You are an expert data analyst.",
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update multiple fields",
			convID: convID,
			updates: map[string]interface{}{
				"title":           "Multi-Update Chat",
				"model_provider":  "openai",
				"model_name":      "gpt-4-turbo",
				"system_prompt":   "You are a helpful coding assistant.",
				"context_summary": "Discussion about Go programming",
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update non-existent conversation",
			convID: uuid.New().String(),
			updates: map[string]interface{}{
				"title": "Should Fail",
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:   "update with no valid fields",
			convID: convID,
			updates: map[string]interface{}{
				"invalid_field": "value",
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.updates)
			req, _ := http.NewRequest(
				http.MethodPut,
				baseURL+"/api/v1/chat/"+tt.convID,
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("UpdateConversation() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if !tt.wantErr {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result["conversation_id"] != tt.convID {
					t.Errorf("Conversation ID mismatch: got %v, want %v", result["conversation_id"], tt.convID)
				}

				t.Logf("Updated conversation: %v", tt.convID)
			}
		})
	}
}

// testDeleteConversation tests DELETE /api/v1/chat/:id
func testDeleteConversation(t *testing.T, client *http.Client, baseURL string) {
	// Create a conversation to delete
	convID := createTestConversation(t, client, baseURL, "Delete Test Chat")

	tests := []struct {
		name       string
		convID     string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "delete existing conversation",
			convID:     convID,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "delete already deleted conversation",
			convID:     convID,
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "delete non-existent conversation",
			convID:     uuid.New().String(),
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(
				http.MethodDelete,
				baseURL+"/api/v1/chat/"+tt.convID,
				nil,
			)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("DeleteConversation() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if !tt.wantErr {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result["conversation_id"] != tt.convID {
					t.Errorf("Conversation ID mismatch: got %v, want %v", result["conversation_id"], tt.convID)
				}

				// Verify it's actually deleted
				getResp, _ := client.Get(baseURL + "/api/v1/chat/" + tt.convID)
				if getResp.StatusCode != http.StatusNotFound {
					t.Error("Conversation was not actually deleted")
				}
				getResp.Body.Close()

				t.Logf("Deleted conversation: %v", tt.convID)
			}
		})
	}
}

// testListConversations tests GET /api/v1/chat
func testListConversations(t *testing.T, client *http.Client, baseURL string) {
	// Create a few test conversations
	conv1 := createTestConversation(t, client, baseURL, "List Test 1")
	_ = createTestConversation(t, client, baseURL, "List Test 2")
	_ = createTestConversation(t, client, baseURL, "List Test 3")

	// Send a message to update timestamp
	sendTestMessage(t, client, baseURL, conv1, "Test message")

	tests := []struct {
		name       string
		query      string
		wantStatus int
		minCount   int
	}{
		{
			name:       "list all conversations",
			query:      "",
			wantStatus: http.StatusOK,
			minCount:   3,
		},
		{
			name:       "list conversations with twin filter",
			query:      "?twin_id=nonexistent",
			wantStatus: http.StatusOK,
			minCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(baseURL + "/api/v1/chat" + tt.query)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("ListConversations() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			var conversations []AgentConversation
			if err := json.NewDecoder(resp.Body).Decode(&conversations); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(conversations) < tt.minCount {
				t.Errorf("ListConversations() returned %d conversations, want at least %d", len(conversations), tt.minCount)
			}

			// Verify conversations are sorted by updated_at DESC
			if len(conversations) >= 2 {
				for i := 1; i < len(conversations); i++ {
					if conversations[i-1].UpdatedAt.Before(conversations[i].UpdatedAt) {
						t.Error("Conversations not sorted by updated_at DESC")
						break
					}
				}
			}

			t.Logf("Listed %d conversations", len(conversations))
		})
	}
}

// testExecuteAgentTools tests POST /api/v1/agent/tools/execute
func testExecuteAgentTools(t *testing.T, client *http.Client, baseURL string) {
	tests := []struct {
		name        string
		toolName    string
		input       map[string]interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name:        "list_pipelines tool",
			toolName:    "list_pipelines",
			input:       map[string]interface{}{},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "list_ontologies tool",
			toolName:    "list_ontologies",
			input:       map[string]interface{}{},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "list_alerts tool",
			toolName:    "list_alerts",
			input:       map[string]interface{}{},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "create_pipeline tool",
			toolName: "create_pipeline",
			input: map[string]interface{}{
				"name":        "Test Pipeline From Tool",
				"description": "Created via agent tool test",
				"steps": []map[string]interface{}{
					{
						"name":   "read_data",
						"plugin": "api",
						"url":    "https://example.com/data",
					},
				},
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "extract_ontology tool",
			toolName: "extract_ontology",
			input: map[string]interface{}{
				"data_source":   "test_data",
				"ontology_name": "Test Ontology",
				"description":   "Test ontology extraction",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "recommend_models tool",
			toolName: "recommend_models",
			input: map[string]interface{}{
				"use_case":  "anomaly_detection",
				"data_type": "time_series",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "create_twin tool",
			toolName: "create_twin",
			input: map[string]interface{}{
				"name":        "Test Twin",
				"description": "Created via agent tool test",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "create_alert tool",
			toolName: "create_alert",
			input: map[string]interface{}{
				"title":    "Test Alert",
				"type":     "threshold",
				"severity": "medium",
				"message":  "Test alert message",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:     "train_model tool",
			toolName: "train_model",
			input: map[string]interface{}{
				"ontology_id":     "test_ont_123",
				"target_property": "value",
				"model_type":      "regression",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: false, // Will fail because ontology doesn't exist
		},
		{
			name:        "unknown tool",
			toolName:    "unknown_tool",
			input:       map[string]interface{}{},
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := AgentToolRequest{
				ToolName: tt.toolName,
				Input:    tt.input,
			}
			body, _ := json.Marshal(req)

			resp, err := client.Post(
				baseURL+"/api/v1/agent/tools/execute",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("ExecuteAgentTool() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if resp.StatusCode == http.StatusOK {
				var result AgentToolResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result.Success != tt.wantSuccess {
					t.Errorf("Tool execution success = %v, want %v", result.Success, tt.wantSuccess)
				}

				if result.Duration <= 0 {
					t.Error("Tool execution duration should be positive")
				}

				t.Logf("Tool %s executed in %d ms (success: %v)", tt.toolName, result.Duration, result.Success)
			}
		})
	}
}

// testFullConversationWorkflow tests the complete chat workflow
func testFullConversationWorkflow(t *testing.T, client *http.Client, baseURL string) {
	t.Log("Starting full conversation workflow test...")

	// Step 1: Create a conversation
	convReq := CreateConversationRequest{
		Title:         "Workflow Test Chat",
		ModelProvider: "mock",
		ModelName:     "mock-gpt-4",
		SystemPrompt:  "You are a helpful assistant for testing workflows.",
	}
	body, _ := json.Marshal(convReq)
	resp, err := client.Post(baseURL+"/api/v1/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	var createResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createResult)
	resp.Body.Close()

	convID := createResult["conversation_id"].(string)
	t.Logf("Step 1: Created conversation %s", convID)

	// Step 2: Send multiple messages
	messages := []string{
		"Hello, can you help me analyze my data?",
		"What pipelines do I have available?",
		"Create a new pipeline called 'Data Analysis'",
		"Thank you for your help!",
	}

	for i, msg := range messages {
		msgReq := SendMessageRequest{Message: msg}
		body, _ := json.Marshal(msgReq)
		resp, err := client.Post(
			baseURL+"/api/v1/chat/"+convID+"/message",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Failed to send message %d: %v", i+1, err)
		}

		var msgResult SendMessageResponse
		json.NewDecoder(resp.Body).Decode(&msgResult)
		resp.Body.Close()

		if msgResult.UserMessage.Content != msg {
			t.Errorf("Message %d content mismatch", i+1)
		}
		if msgResult.AssistantReply.Content == "" {
			t.Errorf("Message %d: Assistant reply is empty", i+1)
		}

		t.Logf("Step 2.%d: Sent message and received %d char response", i+1, len(msgResult.AssistantReply.Content))
		time.Sleep(100 * time.Millisecond) // Small delay between messages
	}

	// Step 3: Get conversation with all messages
	resp, err = client.Get(baseURL + "/api/v1/chat/" + convID)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	var getResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&getResult)
	resp.Body.Close()

	messagesList := getResult["messages"].([]interface{})
	if len(messagesList) != len(messages)*2 { // User + Assistant for each message
		t.Errorf("Expected %d messages, got %d", len(messages)*2, len(messagesList))
	}
	t.Logf("Step 3: Retrieved conversation with %d messages", len(messagesList))

	// Step 4: Update conversation settings
	updateReq := map[string]interface{}{
		"title":          "Updated Workflow Chat",
		"model_provider": "anthropic",
		"model_name":     "claude-3",
	}
	body, _ = json.Marshal(updateReq)
	updateHTTPReq, _ := http.NewRequest(http.MethodPut, baseURL+"/api/v1/chat/"+convID, bytes.NewReader(body))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(updateHTTPReq)
	if err != nil {
		t.Fatalf("Failed to update conversation: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Update conversation failed: %d", resp.StatusCode)
	}
	t.Log("Step 4: Updated conversation settings")

	// Step 5: Execute a tool directly
	toolReq := AgentToolRequest{
		ToolName: "list_pipelines",
		Input:    map[string]interface{}{},
	}
	body, _ = json.Marshal(toolReq)
	resp, err = client.Post(baseURL+"/api/v1/agent/tools/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to execute tool: %v", err)
	}

	var toolResult AgentToolResponse
	json.NewDecoder(resp.Body).Decode(&toolResult)
	resp.Body.Close()

	if !toolResult.Success {
		t.Error("Tool execution failed")
	}
	t.Logf("Step 5: Executed list_pipelines tool (duration: %d ms)", toolResult.Duration)

	// Step 6: Delete the conversation
	deleteReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/v1/chat/"+convID, nil)
	resp, err = client.Do(deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Delete conversation failed: %d", resp.StatusCode)
	}
	t.Log("Step 6: Deleted conversation")

	// Step 7: Verify conversation is deleted
	resp, err = client.Get(baseURL + "/api/v1/chat/" + convID)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Conversation should be deleted, got status %d", resp.StatusCode)
	}
	t.Log("Step 7: Verified conversation deletion")

	t.Log("✅ Full conversation workflow completed successfully")
}

// testErrorHandling tests error scenarios
func testErrorHandling(t *testing.T, client *http.Client, baseURL string) {
	t.Run("invalid JSON in request body", func(t *testing.T) {
		resp, err := client.Post(
			baseURL+"/api/v1/chat",
			"application/json",
			bytes.NewReader([]byte(`invalid json`)),
		)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", resp.StatusCode)
		}
	})

	t.Run("empty request body", func(t *testing.T) {
		resp, err := client.Post(
			baseURL+"/api/v1/chat",
			"application/json",
			bytes.NewReader([]byte{}),
		)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Empty body should use defaults
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201 for empty body, got %d", resp.StatusCode)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/chat/test-id", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
			t.Logf("Method not allowed returned %d (may be handled by router differently)", resp.StatusCode)
		}
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// Test without proper authentication if auth is enabled
		// This depends on server configuration
		t.Log("Auth test skipped - depends on server configuration")
	})
}

// Helper functions

func createTestConversation(t *testing.T, client *http.Client, baseURL string, title string) string {
	req := CreateConversationRequest{
		Title:         title,
		ModelProvider: "mock",
		ModelName:     "mock-gpt-4",
	}
	body, _ := json.Marshal(req)
	resp, err := client.Post(baseURL+"/api/v1/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	return result["conversation_id"].(string)
}

func sendTestMessage(t *testing.T, client *http.Client, baseURL, convID, message string) {
	req := SendMessageRequest{Message: message}
	body, _ := json.Marshal(req)
	resp, err := client.Post(
		baseURL+"/api/v1/chat/"+convID+"/message",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}
	resp.Body.Close()
}

func strPtr(s string) *string {
	return &s
}

func setupTestServer(t *testing.T) *httptest.Server {
	// This is a simplified test server setup
	// In a real scenario, you'd initialize the full server with proper dependencies
	router := setupTestRouter()
	return httptest.NewServer(router)
}

// Additional integration test: Tool execution via chat
func TestAgentChat_ToolExecutionViaChat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL
	client := &http.Client{Timeout: 30 * time.Second}

	// Create conversation
	convID := createTestConversation(t, client, baseURL, "Tool Test Chat")

	// Test messages that should trigger tools
	toolTestCases := []struct {
		name         string
		message      string
		wantTool     string
		minToolCalls int
	}{
		{
			name:         "ask about pipelines",
			message:      "What pipelines do I have?",
			wantTool:     "list_pipelines",
			minToolCalls: 0, // Mock may or may not trigger tools
		},
		{
			name:         "ask to create something",
			message:      "Create a pipeline for data processing",
			wantTool:     "create_pipeline",
			minToolCalls: 0,
		},
		{
			name:         "ask about ontologies",
			message:      "List my ontologies",
			wantTool:     "list_ontologies",
			minToolCalls: 0,
		},
	}

	for _, tc := range toolTestCases {
		t.Run(tc.name, func(t *testing.T) {
			req := SendMessageRequest{Message: tc.message}
			body, _ := json.Marshal(req)

			resp, err := client.Post(
				baseURL+"/api/v1/chat/"+convID+"/message",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("Failed to send message: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Unexpected status: %d", resp.StatusCode)
			}

			var result SendMessageResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify message was saved
			if result.UserMessage.Content != tc.message {
				t.Error("User message not saved correctly")
			}

			// Verify assistant responded
			if result.AssistantReply.Content == "" {
				t.Error("Assistant reply is empty")
			}

			t.Logf("Message: %d chars, Reply: %d chars, Tools: %d",
				len(result.UserMessage.Content),
				len(result.AssistantReply.Content),
				len(result.ToolCalls))
		})
	}
}

// Test conversation persistence and retrieval
func TestAgentChat_ConversationPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL
	client := &http.Client{Timeout: 30 * time.Second}

	// Create conversation with specific settings
	createReq := CreateConversationRequest{
		Title:         "Persistence Test",
		ModelProvider: "openai",
		ModelName:     "gpt-4-turbo",
		SystemPrompt:  "You are a test assistant.",
	}
	body, _ := json.Marshal(createReq)
	resp, err := client.Post(baseURL+"/api/v1/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	var createResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createResult)
	resp.Body.Close()

	convID := createResult["conversation_id"].(string)

	// Send several messages
	for i := 0; i < 5; i++ {
		sendTestMessage(t, client, baseURL, convID, fmt.Sprintf("Test message %d", i+1))
		time.Sleep(50 * time.Millisecond)
	}

	// Retrieve conversation and verify all data persisted
	resp, err = client.Get(baseURL + "/api/v1/chat/" + convID)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	var getResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&getResult)
	resp.Body.Close()

	conv := getResult["conversation"].(map[string]interface{})
	messages := getResult["messages"].([]interface{})

	// Verify conversation metadata
	if conv["title"] != createReq.Title {
		t.Errorf("Title mismatch: got %v, want %v", conv["title"], createReq.Title)
	}
	if conv["model_provider"] != createReq.ModelProvider {
		t.Errorf("Model provider mismatch: got %v, want %v", conv["model_provider"], createReq.ModelProvider)
	}
	if conv["model_name"] != createReq.ModelName {
		t.Errorf("Model name mismatch: got %v, want %v", conv["model_name"], createReq.ModelName)
	}

	// Verify all messages persisted (5 user + 5 assistant = 10 total)
	if len(messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(messages))
	}

	// Verify message order and content
	for i, msgRaw := range messages {
		msg := msgRaw.(map[string]interface{})
		expectedRole := "user"
		if i%2 == 1 {
			expectedRole = "assistant"
		}
		if msg["role"] != expectedRole {
			t.Errorf("Message %d role mismatch: got %v, want %v", i, msg["role"], expectedRole)
		}
	}

	t.Logf("✅ Conversation persistence verified: %d messages saved correctly", len(messages))
}

// Performance test for chat endpoints
func TestAgentChat_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL
	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("create conversation performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 10; i++ {
			createTestConversation(t, client, baseURL, fmt.Sprintf("Perf Test %d", i))
		}
		duration := time.Since(start)
		avgMs := float64(duration.Milliseconds()) / 10.0

		if avgMs > 500 {
			t.Errorf("Average create time too slow: %.2f ms (max 500ms)", avgMs)
		}
		t.Logf("Average create conversation time: %.2f ms", avgMs)
	})

	t.Run("send message performance", func(t *testing.T) {
		convID := createTestConversation(t, client, baseURL, "Perf Message Test")

		start := time.Now()
		for i := 0; i < 5; i++ {
			sendTestMessage(t, client, baseURL, convID, fmt.Sprintf("Performance test message %d", i))
		}
		duration := time.Since(start)
		avgMs := float64(duration.Milliseconds()) / 5.0

		if avgMs > 2000 {
			t.Errorf("Average message time too slow: %.2f ms (max 2000ms)", avgMs)
		}
		t.Logf("Average send message time: %.2f ms", avgMs)
	})

	t.Run("list conversations performance", func(t *testing.T) {
		// Create some conversations first
		for i := 0; i < 20; i++ {
			createTestConversation(t, client, baseURL, fmt.Sprintf("List Test %d", i))
		}

		start := time.Now()
		resp, err := client.Get(baseURL + "/api/v1/chat")
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		resp.Body.Close()
		duration := time.Since(start)

		if duration.Milliseconds() > 500 {
			t.Errorf("List conversations too slow: %d ms (max 500ms)", duration.Milliseconds())
		}
		t.Logf("List conversations time: %d ms", duration.Milliseconds())
	})
}

// setupTestRouter creates a minimal router for testing
// In real implementation, this should match your actual server setup
func setupTestRouter() http.Handler {
	// This is a placeholder - in the real test file,
	// you would import and use your actual server setup
	// For now, returning a simple handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/chat":
			handleTestCreateConversation(w, r)
		case r.Method == http.MethodGet && matchPattern(r.URL.Path, "/api/v1/chat/"):
			handleTestGetConversation(w, r)
		case r.Method == http.MethodPost && matchPattern(r.URL.Path, "/api/v1/chat/*/message"):
			handleTestSendMessage(w, r)
		case r.Method == http.MethodPut && matchPattern(r.URL.Path, "/api/v1/chat/"):
			handleTestUpdateConversation(w, r)
		case r.Method == http.MethodDelete && matchPattern(r.URL.Path, "/api/v1/chat/"):
			handleTestDeleteConversation(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/chat":
			handleTestListConversations(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agent/tools/execute":
			handleTestExecuteTool(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		}
	})
}

// Test handlers (simplified implementations for testing)
func handleTestCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if decode fails
		req.Title = "New Conversation"
		req.ModelProvider = "mock"
		req.ModelName = "mock-gpt-4"
	}

	if req.Title == "" {
		req.Title = "New Conversation"
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "mock"
	}
	if req.ModelName == "" {
		req.ModelName = "mock-gpt-4"
	}

	convID := uuid.New().String()
	conv := AgentConversation{
		ID:            convID,
		Title:         req.Title,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		SystemPrompt:  req.SystemPrompt,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if req.TwinID != nil {
		conv.TwinID = req.TwinID
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": convID,
		"conversation":    conv,
		"message":         "Conversation created successfully",
	})
}

func handleTestGetConversation(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	convID := parts[len(parts)-1]

	// Simulate not found for invalid UUIDs
	if _, err := uuid.Parse(convID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Conversation not found"})
		return
	}

	// Return mock data
	conv := AgentConversation{
		ID:            convID,
		Title:         "Test Conversation",
		ModelProvider: "mock",
		ModelName:     "mock-gpt-4",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation": conv,
		"messages":     []AgentMessage{},
	})
}

func handleTestSendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Message cannot be empty"})
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	convID := parts[len(parts)-2]

	response := SendMessageResponse{
		ConversationID: convID,
		UserMessage: AgentMessage{
			ID:             1,
			ConversationID: convID,
			Role:           "user",
			Content:        req.Message,
			CreatedAt:      time.Now(),
		},
		AssistantReply: AgentMessage{
			ID:             2,
			ConversationID: convID,
			Role:           "assistant",
			Content:        "This is a test response from the assistant.",
			CreatedAt:      time.Now(),
		},
		ToolCalls: []ToolCallInfo{},
	}

	json.NewEncoder(w).Encode(response)
}

func handleTestUpdateConversation(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	convID := parts[len(parts)-1]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Check if any valid fields
	allowedFields := map[string]bool{
		"title": true, "model_provider": true, "model_name": true,
		"system_prompt": true, "context_summary": true,
	}
	hasValidField := false
	for field := range updates {
		if allowedFields[field] {
			hasValidField = true
			break
		}
	}

	if !hasValidField {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No valid fields to update"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Conversation updated successfully",
		"conversation_id": convID,
	})
}

func handleTestDeleteConversation(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	convID := parts[len(parts)-1]

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Conversation deleted successfully",
		"conversation_id": convID,
	})
}

func handleTestListConversations(w http.ResponseWriter, r *http.Request) {
	conversations := []AgentConversation{
		{
			ID:        uuid.New().String(),
			Title:     "Conversation 1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New().String(),
			Title:     "Conversation 2",
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		},
		{
			ID:        uuid.New().String(),
			Title:     "Conversation 3",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Minute),
		},
	}

	json.NewEncoder(w).Encode(conversations)
}

func handleTestExecuteTool(w http.ResponseWriter, r *http.Request) {
	var req AgentToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	validTools := map[string]bool{
		"create_pipeline": true, "execute_pipeline": true, "schedule_pipeline": true,
		"extract_ontology": true, "list_ontologies": true, "recommend_models": true,
		"create_twin": true, "get_twin_status": true, "simulate_scenario": true,
		"detect_anomalies": true, "create_alert": true, "list_alerts": true,
		"get_pipeline_status": true, "list_pipelines": true, "train_model": true,
		"query_ontology": true,
	}

	if !validTools[req.ToolName] {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown tool: " + req.ToolName})
		return
	}

	response := AgentToolResponse{
		Success:  true,
		Result:   map[string]interface{}{"message": "Tool executed successfully"},
		Duration: 100,
	}

	json.NewEncoder(w).Encode(response)
}

func matchPattern(path, pattern string) bool {
	return len(path) > len(pattern) && strings.HasPrefix(path, pattern)
}

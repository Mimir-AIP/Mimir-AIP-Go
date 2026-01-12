package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// AgentConversation represents a chat conversation
type AgentConversation struct {
	ID             string    `json:"id"`
	TwinID         *string   `json:"twin_id,omitempty"`
	Title          string    `json:"title"`
	ModelProvider  string    `json:"model_provider"`
	ModelName      string    `json:"model_name"`
	SystemPrompt   string    `json:"system_prompt,omitempty"`
	ContextSummary string    `json:"context_summary,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	MessageCount   int       `json:"message_count,omitempty"`
}

// AgentMessage represents a single message in a conversation
type AgentMessage struct {
	ID             int             `json:"id"`
	ConversationID string          `json:"conversation_id"`
	Role           string          `json:"role"` // "user", "assistant", "system", "tool"
	Content        string          `json:"content"`
	ToolCalls      json.RawMessage `json:"tool_calls,omitempty"`
	ToolResults    json.RawMessage `json:"tool_results,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// CreateConversationRequest represents a request to create a new conversation
type CreateConversationRequest struct {
	TwinID        *string `json:"twin_id,omitempty"`
	Title         string  `json:"title"`
	ModelProvider string  `json:"model_provider"`
	ModelName     string  `json:"model_name"`
	SystemPrompt  string  `json:"system_prompt,omitempty"`
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Message       string `json:"message"`
	ModelProvider string `json:"model_provider,omitempty"`
	ModelName     string `json:"model_name,omitempty"`
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	ConversationID string         `json:"conversation_id"`
	UserMessage    AgentMessage   `json:"user_message"`
	AssistantReply AgentMessage   `json:"assistant_reply"`
	ToolCalls      []ToolCallInfo `json:"tool_calls,omitempty"`
}

// ToolCallInfo represents information about a tool call
type ToolCallInfo struct {
	ID       string          `json:"id"`
	ToolName string          `json:"tool_name"`
	Input    json.RawMessage `json:"input"`
	Output   json.RawMessage `json:"output"`
	Duration int64           `json:"duration_ms"`
}

// handleListConversations lists all conversations
func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Get optional twin_id filter
	twinID := r.URL.Query().Get("twin_id")

	query := `
		SELECT c.id, c.twin_id, c.title, c.model_provider, c.model_name, 
		       c.system_prompt, c.context_summary, c.created_at, c.updated_at,
		       COUNT(m.id) as message_count
		FROM agent_conversations c
		LEFT JOIN agent_messages m ON c.id = m.conversation_id
	`

	var args []interface{}
	if twinID != "" {
		query += " WHERE c.twin_id = ?"
		args = append(args, twinID)
	}

	query += " GROUP BY c.id ORDER BY c.updated_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to query conversations: "+err.Error())
		return
	}
	defer rows.Close()

	conversations := []AgentConversation{}
	for rows.Next() {
		var conv AgentConversation
		var twinIDPtr *string
		var systemPrompt, contextSummary *string

		err := rows.Scan(
			&conv.ID, &twinIDPtr, &conv.Title, &conv.ModelProvider, &conv.ModelName,
			&systemPrompt, &contextSummary, &conv.CreatedAt, &conv.UpdatedAt, &conv.MessageCount,
		)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to scan conversation: "+err.Error())
			return
		}

		conv.TwinID = twinIDPtr
		if systemPrompt != nil {
			conv.SystemPrompt = *systemPrompt
		}
		if contextSummary != nil {
			conv.ContextSummary = *contextSummary
		}

		conversations = append(conversations, conv)
	}

	writeJSONResponse(w, http.StatusOK, conversations)
}

// handleGetConversation gets a specific conversation with messages
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]

	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Get conversation
	var conv AgentConversation
	var twinIDPtr *string
	var systemPrompt, contextSummary *string

	err := db.QueryRow(`
		SELECT id, twin_id, title, model_provider, model_name, system_prompt, 
		       context_summary, created_at, updated_at
		FROM agent_conversations WHERE id = ?
	`, conversationID).Scan(
		&conv.ID, &twinIDPtr, &conv.Title, &conv.ModelProvider, &conv.ModelName,
		&systemPrompt, &contextSummary, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return
	}

	conv.TwinID = twinIDPtr
	if systemPrompt != nil {
		conv.SystemPrompt = *systemPrompt
	}
	if contextSummary != nil {
		conv.ContextSummary = *contextSummary
	}

	// Get messages
	rows, err := db.Query(`
		SELECT id, conversation_id, role, content, tool_calls, tool_results, metadata, created_at
		FROM agent_messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to query messages: "+err.Error())
		return
	}
	defer rows.Close()

	messages := []AgentMessage{}
	for rows.Next() {
		var msg AgentMessage
		var toolCalls, toolResults, metadata *string

		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&toolCalls, &toolResults, &metadata, &msg.CreatedAt,
		)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to scan message: "+err.Error())
			return
		}

		if toolCalls != nil {
			msg.ToolCalls = json.RawMessage(*toolCalls)
		}
		if toolResults != nil {
			msg.ToolResults = json.RawMessage(*toolResults)
		}
		if metadata != nil {
			msg.Metadata = json.RawMessage(*metadata)
		}

		messages = append(messages, msg)
	}

	response := map[string]interface{}{
		"conversation": conv,
		"messages":     messages,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleCreateConversation creates a new conversation
func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate
	if req.Title == "" {
		req.Title = "New Conversation"
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "openai"
	}
	if req.ModelName == "" {
		req.ModelName = "gpt-4"
	}

	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	conversationID := uuid.New().String()
	now := time.Now()

	_, err := db.Exec(`
		INSERT INTO agent_conversations (id, twin_id, title, model_provider, model_name, system_prompt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, conversationID, req.TwinID, req.Title, req.ModelProvider, req.ModelName, req.SystemPrompt, now, now)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create conversation: "+err.Error())
		return
	}

	conv := AgentConversation{
		ID:            conversationID,
		TwinID:        req.TwinID,
		Title:         req.Title,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		SystemPrompt:  req.SystemPrompt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"conversation_id": conversationID,
		"conversation":    conv,
		"message":         "Conversation created successfully",
	})
}

// handleSendMessage sends a message and gets a response
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Message == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Message cannot be empty")
		return
	}

	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Verify conversation exists
	var conv AgentConversation
	err := db.QueryRow("SELECT id, model_provider, model_name FROM agent_conversations WHERE id = ?", conversationID).
		Scan(&conv.ID, &conv.ModelProvider, &conv.ModelName)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Use provided model or conversation default
	modelProvider := req.ModelProvider
	if modelProvider == "" {
		modelProvider = conv.ModelProvider
	}
	modelName := req.ModelName
	if modelName == "" {
		modelName = conv.ModelName
	}

	// Save user message
	userMsgResult, err := db.Exec(`
		INSERT INTO agent_messages (conversation_id, role, content, created_at)
		VALUES (?, ?, ?, ?)
	`, conversationID, "user", req.Message, time.Now())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to save user message: "+err.Error())
		return
	}

	userMsgID, _ := userMsgResult.LastInsertId()

	// Get conversation history for context
	ctx := context.Background()
	messages, err := s.getConversationHistory(ctx, conversationID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get conversation history: "+err.Error())
		return
	}

	// Call LLM with conversation context
	assistantReply, toolCalls, err := s.callLLMWithTools(ctx, modelProvider, modelName, messages)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get LLM response: "+err.Error())
		return
	}

	// Save assistant message
	var toolCallsJSON, toolResultsJSON *string
	var toolCallsRawMsg json.RawMessage
	if len(toolCalls) > 0 {
		tc, _ := json.Marshal(toolCalls)
		tcStr := string(tc)
		toolCallsJSON = &tcStr
		toolCallsRawMsg = tc

		// Extract tool results
		tr, _ := json.Marshal(toolCalls)
		trStr := string(tr)
		toolResultsJSON = &trStr
	}

	assistantMsgResult, err := db.Exec(`
		INSERT INTO agent_messages (conversation_id, role, content, tool_calls, tool_results, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, conversationID, "assistant", assistantReply, toolCallsJSON, toolResultsJSON, time.Now())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to save assistant message: "+err.Error())
		return
	}

	assistantMsgID, _ := assistantMsgResult.LastInsertId()

	// Update conversation timestamp
	_, _ = db.Exec("UPDATE agent_conversations SET updated_at = ? WHERE id = ?", time.Now(), conversationID)

	response := SendMessageResponse{
		ConversationID: conversationID,
		UserMessage: AgentMessage{
			ID:             int(userMsgID),
			ConversationID: conversationID,
			Role:           "user",
			Content:        req.Message,
			CreatedAt:      time.Now(),
		},
		AssistantReply: AgentMessage{
			ID:             int(assistantMsgID),
			ConversationID: conversationID,
			Role:           "assistant",
			Content:        assistantReply,
			ToolCalls:      toolCallsRawMsg,
			CreatedAt:      time.Now(),
		},
		ToolCalls: toolCalls,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleDeleteConversation deletes a conversation
func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]

	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	result, err := db.Exec("DELETE FROM agent_conversations WHERE id = ?", conversationID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete conversation: "+err.Error())
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":         "Conversation deleted successfully",
		"conversation_id": conversationID,
	})
}

// handleUpdateConversation updates conversation metadata
func (s *Server) handleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	db := s.persistence.GetDB()
	if db == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Build dynamic update query
	allowedFields := map[string]bool{
		"title":           true,
		"model_provider":  true,
		"model_name":      true,
		"system_prompt":   true,
		"context_summary": true,
	}

	setClauses := []string{}
	args := []interface{}{}

	for field, value := range updates {
		if allowedFields[field] {
			setClauses = append(setClauses, field+" = ?")
			args = append(args, value)
		}
	}

	if len(setClauses) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "No valid fields to update")
		return
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, conversationID)

	query := "UPDATE agent_conversations SET " + setClauses[0]
	for i := 1; i < len(setClauses); i++ {
		query += ", " + setClauses[i]
	}
	query += " WHERE id = ?"

	result, err := db.Exec(query, args...)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update conversation: "+err.Error())
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":         "Conversation updated successfully",
		"conversation_id": conversationID,
	})
}

// getConversationHistory retrieves all messages for context
func (s *Server) getConversationHistory(ctx context.Context, conversationID string) ([]map[string]string, error) {
	db := s.persistence.GetDB()
	if db == nil {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT role, content FROM agent_messages 
		WHERE conversation_id = ? 
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []map[string]string{}
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		messages = append(messages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	return messages, nil
}

// callLLMWithTools calls the LLM and executes any tool calls
func (s *Server) callLLMWithTools(ctx context.Context, provider, model string, messages []map[string]string) (string, []ToolCallInfo, error) {
	// Get the appropriate LLM client for the provider
	var llmClient AI.LLMClient

	if provider == "mock" {
		// Create mock client with specified model (supports mock-gpt-4, mock-claude-3, etc.)
		llmClient = AI.NewIntelligentMockLLMClientWithModel(model)
	} else if client, ok := s.llmClients[AI.LLMProvider(provider)]; ok {
		llmClient = client
	} else {
		// Provider not available, fall back to mock
		log.Printf("LLM provider '%s' not available, falling back to mock", provider)
		llmClient = AI.NewIntelligentMockLLMClientWithModel(model)
	}

	// Convert messages to LLMMessage format
	llmMessages := make([]AI.LLMMessage, len(messages))
	for i, msg := range messages {
		llmMessages[i] = AI.LLMMessage{
			Role:    msg["role"],
			Content: msg["content"],
		}
	}

	// Build the LLM request
	request := AI.LLMRequest{
		Messages: llmMessages,
		Model:    model,
	}

	// Call the LLM
	response, err := llmClient.Complete(ctx, request)
	if err != nil {
		// If the provider's LLM fails, fall back to mock for demo purposes
		log.Printf("LLM call failed for provider '%s': %v, falling back to mock", provider, err)
		mockClient := AI.NewIntelligentMockLLMClientWithModel(model)
		response, err = mockClient.Complete(ctx, request)
		if err != nil {
			return "", nil, fmt.Errorf("LLM call failed: %w", err)
		}
	}

	// Execute tool calls via MCP server and collect results
	toolCalls := []ToolCallInfo{}
	for _, tc := range response.ToolCalls {
		startTime := time.Now()
		inputJSON, _ := json.Marshal(tc.Arguments)

		// Execute tool via MCP server
		output, err := s.executeToolViaMCP(ctx, tc.Name, tc.Arguments)
		duration := time.Since(startTime).Milliseconds()

		if err != nil {
			// Tool execution failed, but don't fail the whole request
			outputJSON, _ := json.Marshal(map[string]any{
				"status": "error",
				"error":  err.Error(),
			})
			toolCalls = append(toolCalls, ToolCallInfo{
				ID:       tc.ID,
				ToolName: tc.Name,
				Input:    json.RawMessage(inputJSON),
				Output:   json.RawMessage(outputJSON),
				Duration: duration,
			})
		} else {
			outputJSON, _ := json.Marshal(output)
			toolCalls = append(toolCalls, ToolCallInfo{
				ID:       tc.ID,
				ToolName: tc.Name,
				Input:    json.RawMessage(inputJSON),
				Output:   json.RawMessage(outputJSON),
				Duration: duration,
			})
		}
	}

	return response.Content, toolCalls, nil
}

// executeToolViaMCP executes a tool by calling the MCP server
func (s *Server) executeToolViaMCP(ctx context.Context, toolName string, arguments map[string]any) (map[string]any, error) {
	// Call the MCP server's tool execution endpoint
	requestBody := map[string]any{
		"name":      toolName,
		"arguments": arguments,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool request: %w", err)
	}

	// Create HTTP request to MCP server
	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:8080/mcp/tools/execute", bytes.NewReader(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create tool request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tool execution failed with status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tool response: %w", err)
	}

	return result, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

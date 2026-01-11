package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// AlertAction represents an automated action when an alert is triggered
type AlertAction struct {
	ID         int
	Name       string
	RuleID     string
	AlertType  string
	Severity   string
	ActionType string // 'execute_pipeline', 'send_email', 'webhook'
	Config     string // JSON configuration
	IsEnabled  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AlertActionConfig represents parsed configuration for an action
type AlertActionConfig struct {
	// For execute_pipeline action
	PipelineFile string         `json:"pipeline_file,omitempty"`
	PipelineName string         `json:"pipeline_name,omitempty"`
	Context      map[string]any `json:"context,omitempty"`

	// For send_email action
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Body    string   `json:"body,omitempty"`

	// For webhook action
	WebhookURL string         `json:"webhook_url,omitempty"`
	Headers    map[string]any `json:"headers,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// AlertActionExecutor handles execution of alert actions
type AlertActionExecutor struct {
	db       *sql.DB
	registry *pipelines.PluginRegistry
	logger   *Logger
}

// NewAlertActionExecutor creates a new alert action executor
func NewAlertActionExecutor(db *sql.DB, registry *pipelines.PluginRegistry) *AlertActionExecutor {
	return &AlertActionExecutor{
		db:       db,
		registry: registry,
		logger:   GetLogger(),
	}
}

// HandleAnomalyDetected handles anomaly.detected events
func (e *AlertActionExecutor) HandleAnomalyDetected(event Event) error {
	e.logger.Info("Handling anomaly detected event",
		String("event_type", event.Type),
		String("source", event.Source))

	// Extract alert info from event payload
	alertID, ok := event.Payload["alert_id"]
	if !ok {
		e.logger.Warn("Anomaly event missing alert_id")
		return nil // Don't fail the event handler
	}

	alertType, _ := event.Payload["alert_type"].(string)
	severity, _ := event.Payload["severity"].(string)
	ruleID, _ := event.Payload["rule_id"].(string)

	e.logger.Info("Processing alert actions",
		String("alert_id", fmt.Sprintf("%v", alertID)),
		String("alert_type", alertType),
		String("severity", severity))

	// Get matching alert actions
	actions, err := e.GetMatchingActions(ruleID, alertType, severity)
	if err != nil {
		e.logger.Error("Failed to get matching actions", err)
		return fmt.Errorf("failed to get matching actions: %w", err)
	}

	if len(actions) == 0 {
		e.logger.Debug("No matching actions found for alert")
		return nil
	}

	// Execute each action
	for _, action := range actions {
		if err := e.ExecuteAction(&action, alertID, event.Payload); err != nil {
			e.logger.Error("Failed to execute alert action", err,
				String("action_name", action.Name),
				String("action_type", action.ActionType))
			// Continue with other actions even if one fails
			continue
		}
	}

	return nil
}

// GetMatchingActions retrieves alert actions that match the criteria
func (e *AlertActionExecutor) GetMatchingActions(ruleID, alertType, severity string) ([]AlertAction, error) {
	query := `
		SELECT id, name, rule_id, alert_type, severity, action_type, config, is_enabled, created_at, updated_at
		FROM alert_actions
		WHERE is_enabled = 1
		  AND (rule_id IS NULL OR rule_id = ?)
		  AND (alert_type IS NULL OR alert_type = ?)
		  AND (severity IS NULL OR severity = ?)
		ORDER BY id
	`

	rows, err := e.db.Query(query, ruleID, alertType, severity)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert actions: %w", err)
	}
	defer rows.Close()

	var actions []AlertAction
	for rows.Next() {
		var action AlertAction
		var ruleIDNull, alertTypeNull, severityNull sql.NullString

		err := rows.Scan(
			&action.ID,
			&action.Name,
			&ruleIDNull,
			&alertTypeNull,
			&severityNull,
			&action.ActionType,
			&action.Config,
			&action.IsEnabled,
			&action.CreatedAt,
			&action.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert action: %w", err)
		}

		if ruleIDNull.Valid {
			action.RuleID = ruleIDNull.String
		}
		if alertTypeNull.Valid {
			action.AlertType = alertTypeNull.String
		}
		if severityNull.Valid {
			action.Severity = severityNull.String
		}

		actions = append(actions, action)
	}

	return actions, rows.Err()
}

// ExecuteAction executes a single alert action
func (e *AlertActionExecutor) ExecuteAction(action *AlertAction, alertID interface{}, eventPayload map[string]any) error {
	startTime := time.Now()

	// Parse configuration
	var config AlertActionConfig
	if err := json.Unmarshal([]byte(action.Config), &config); err != nil {
		return e.recordExecution(action.ID, alertID, "failed", err.Error(), nil)
	}

	var result map[string]any
	var err error

	// Execute based on action type
	switch action.ActionType {
	case "execute_pipeline":
		result, err = e.executePipeline(&config, eventPayload)
	case "send_email":
		result, err = e.sendEmail(&config, eventPayload)
	case "webhook":
		result, err = e.callWebhook(&config, eventPayload)
	default:
		err = fmt.Errorf("unknown action type: %s", action.ActionType)
	}

	// Record execution
	status := "success"
	errorMsg := ""
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	duration := time.Since(startTime)
	if result == nil {
		result = make(map[string]any)
	}
	result["duration_ms"] = duration.Milliseconds()

	return e.recordExecution(action.ID, alertID, status, errorMsg, result)
}

// executePipeline executes a pipeline as an alert action
func (e *AlertActionExecutor) executePipeline(config *AlertActionConfig, eventPayload map[string]any) (map[string]any, error) {
	pipelineFile := config.PipelineFile
	if pipelineFile == "" && config.PipelineName != "" {
		pipelineFile = fmt.Sprintf("pipelines/%s.yaml", config.PipelineName)
	}

	if pipelineFile == "" {
		return nil, fmt.Errorf("pipeline_file or pipeline_name is required")
	}

	e.logger.Info("Executing pipeline action",
		String("pipeline_file", pipelineFile))

	// Parse pipeline
	pipelineConfig, err := ParsePipeline(pipelineFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline: %w", err)
	}

	// Merge context from config and event payload
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()

	// Add event payload to context
	for k, v := range eventPayload {
		globalContext.Set(k, v)
	}

	// Add configured context (overrides event payload)
	for k, v := range config.Context {
		globalContext.Set(k, v)
	}

	// Execute pipeline
	result, err := ExecutePipeline(ctx, pipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("pipeline failed: %s", result.Error)
	}

	return map[string]any{
		"success":      result.Success,
		"pipeline":     pipelineFile,
		"executed_at":  result.ExecutedAt,
	}, nil
}

// sendEmail sends an email alert action
func (e *AlertActionExecutor) sendEmail(config *AlertActionConfig, eventPayload map[string]any) (map[string]any, error) {
	if len(config.To) == 0 {
		return nil, fmt.Errorf("email recipients (to) are required")
	}

	// Prepare email content with event data
	subject := config.Subject
	if subject == "" {
		subject = fmt.Sprintf("Alert: %v", eventPayload["alert_type"])
	}

	body := config.Body
	if body == "" {
		// Default body with event details
		body = fmt.Sprintf(`
Alert Notification

Type: %v
Severity: %v
Message: %v
Value: %v
Threshold: %v

Timestamp: %v
`,
			eventPayload["alert_type"],
			eventPayload["severity"],
			eventPayload["message"],
			eventPayload["value"],
			eventPayload["threshold"],
			time.Now().Format(time.RFC3339),
		)
	}

	e.logger.Info("Sending email alert",
		Int("recipients", len(config.To)),
		String("subject", subject))

	// Get email sender (will be implemented in next step)
	sender := GetEmailSender()
	if sender == nil {
		return nil, fmt.Errorf("email sender not configured")
	}

	err := sender.Send(config.To, subject, body)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	return map[string]any{
		"recipients": config.To,
		"subject":    subject,
		"sent_at":    time.Now().Format(time.RFC3339),
	}, nil
}

// callWebhook calls a webhook as an alert action
func (e *AlertActionExecutor) callWebhook(config *AlertActionConfig, eventPayload map[string]any) (map[string]any, error) {
	if config.WebhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}

	e.logger.Info("Calling webhook",
		String("url", config.WebhookURL))

	// Merge event payload into webhook payload
	payload := make(map[string]any)
	for k, v := range eventPayload {
		payload[k] = v
	}
	for k, v := range config.Payload {
		payload[k] = v
	}

	// TODO: Implement actual HTTP webhook call
	// For now, just log it
	e.logger.Info("Webhook payload prepared", String("url", config.WebhookURL))

	return map[string]any{
		"webhook_url": config.WebhookURL,
		"called_at":   time.Now().Format(time.RFC3339),
	}, nil
}

// recordExecution records an alert action execution in the database
func (e *AlertActionExecutor) recordExecution(actionID int, alertID interface{}, status string, errorMsg string, result map[string]any) error {
	resultJSON := "{}"
	if result != nil {
		bytes, err := json.Marshal(result)
		if err == nil {
			resultJSON = string(bytes)
		}
	}

	query := `
		INSERT INTO alert_action_executions (action_id, alert_id, status, error_message, result, completed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := e.db.Exec(query, actionID, alertID, status, errorMsg, resultJSON, time.Now())
	if err != nil {
		e.logger.Error("Failed to record action execution", err)
		return fmt.Errorf("failed to record execution: %w", err)
	}

	return nil
}

// InitializeAlertActionExecutor sets up the alert action executor
// This should be called during application startup
func InitializeAlertActionExecutor(db *sql.DB, registry *pipelines.PluginRegistry) {
	executor := NewAlertActionExecutor(db, registry)

	// Subscribe to anomaly detection events
	GetEventBus().Subscribe(EventAnomalyDetected, executor.HandleAnomalyDetected)

	GetLogger().Info("Alert action executor initialized")
}

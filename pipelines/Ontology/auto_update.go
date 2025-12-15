package ontology

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
)

// AutoUpdatePolicy defines rules for automatically applying suggestions
type AutoUpdatePolicy struct {
	ID                  int              `json:"id"`
	OntologyID          string           `json:"ontology_id"`
	Enabled             bool             `json:"enabled"`
	AutoApplyClasses    bool             `json:"auto_apply_classes"`    // Auto-apply add_class suggestions
	AutoApplyProperties bool             `json:"auto_apply_properties"` // Auto-apply add_property suggestions
	AutoApplyModify     bool             `json:"auto_apply_modify"`     // Auto-apply modify suggestions
	MaxRiskLevel        RiskLevel        `json:"max_risk_level"`        // Maximum risk level to auto-apply
	MinConfidence       float64          `json:"min_confidence"`        // Minimum confidence threshold (0.0-1.0)
	RequireApproval     []SuggestionType `json:"require_approval"`      // Types that always require manual approval
	NotificationEmail   string           `json:"notification_email"`    // Email for notifications
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// AutoUpdateEngine manages automated ontology evolution
type AutoUpdateEngine struct {
	db               *sql.DB
	llmClient        AI.LLMClient
	tdb2Backend      *knowledgegraph.TDB2Backend
	suggestionEngine *SuggestionEngine
}

// NewAutoUpdateEngine creates a new auto-update engine
func NewAutoUpdateEngine(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *AutoUpdateEngine {
	return &AutoUpdateEngine{
		db:               db,
		llmClient:        llmClient,
		tdb2Backend:      tdb2Backend,
		suggestionEngine: NewSuggestionEngine(db, llmClient, tdb2Backend),
	}
}

// ProcessPendingSuggestions processes all pending suggestions according to policy
func (a *AutoUpdateEngine) ProcessPendingSuggestions(ctx context.Context, ontologyID string) (*AutoUpdateResult, error) {
	// Get policy
	policy, err := a.getPolicy(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if !policy.Enabled {
		return &AutoUpdateResult{
			Message: "Auto-update is disabled for this ontology",
		}, nil
	}

	// Get pending suggestions
	suggestions, err := a.suggestionEngine.ListSuggestions(ctx, ontologyID, SuggestionPending)
	if err != nil {
		return nil, fmt.Errorf("failed to list suggestions: %w", err)
	}

	result := &AutoUpdateResult{
		OntologyID:       ontologyID,
		TotalSuggestions: len(suggestions),
		Processed:        make([]SuggestionAction, 0),
		Skipped:          make([]SuggestionAction, 0),
	}

	// Process each suggestion
	for _, suggestion := range suggestions {
		action := a.evaluateSuggestion(ctx, suggestion, policy)

		switch action.Action {
		case "auto_apply":
			// Auto-approve and apply
			err := a.suggestionEngine.ApproveSuggestion(ctx, suggestion.ID, "auto-update-engine", action.Reason)
			if err != nil {
				action.Action = "error"
				action.Error = err.Error()
				result.Skipped = append(result.Skipped, action)
				continue
			}

			err = a.suggestionEngine.ApplySuggestion(ctx, suggestion.ID)
			if err != nil {
				action.Action = "error"
				action.Error = err.Error()
				result.Skipped = append(result.Skipped, action)
				continue
			}

			result.AutoApplied++
			result.Processed = append(result.Processed, action)

		case "auto_reject":
			// Auto-reject
			err := a.suggestionEngine.RejectSuggestion(ctx, suggestion.ID, "auto-update-engine", action.Reason)
			if err != nil {
				action.Action = "error"
				action.Error = err.Error()
				result.Skipped = append(result.Skipped, action)
				continue
			}

			result.AutoRejected++
			result.Processed = append(result.Processed, action)

		case "require_approval":
			result.RequireManual++
			result.Skipped = append(result.Skipped, action)

		default:
			result.Skipped = append(result.Skipped, action)
		}
	}

	// Send notification if configured
	if policy.NotificationEmail != "" && (result.AutoApplied > 0 || result.AutoRejected > 0) {
		a.sendNotification(ctx, policy, result)
	}

	return result, nil
}

// evaluateSuggestion evaluates a single suggestion against policy
func (a *AutoUpdateEngine) evaluateSuggestion(ctx context.Context, suggestion OntologySuggestion, policy *AutoUpdatePolicy) SuggestionAction {
	action := SuggestionAction{
		SuggestionID: suggestion.ID,
		Type:         suggestion.SuggestionType,
		EntityURI:    suggestion.EntityURI,
		Confidence:   suggestion.Confidence,
		RiskLevel:    suggestion.RiskLevel,
	}

	// Check if type requires manual approval
	for _, reqType := range policy.RequireApproval {
		if suggestion.SuggestionType == reqType {
			action.Action = "require_approval"
			action.Reason = fmt.Sprintf("Type %s requires manual approval per policy", suggestion.SuggestionType)
			return action
		}
	}

	// Check risk level
	if a.compareRiskLevel(suggestion.RiskLevel, policy.MaxRiskLevel) > 0 {
		action.Action = "require_approval"
		action.Reason = fmt.Sprintf("Risk level %s exceeds policy maximum %s", suggestion.RiskLevel, policy.MaxRiskLevel)
		return action
	}

	// Check confidence threshold
	if suggestion.Confidence < policy.MinConfidence {
		action.Action = "auto_reject"
		action.Reason = fmt.Sprintf("Confidence %.2f below threshold %.2f", suggestion.Confidence, policy.MinConfidence)
		return action
	}

	// Check type-specific policy
	switch suggestion.SuggestionType {
	case SuggestionAddClass:
		if !policy.AutoApplyClasses {
			action.Action = "require_approval"
			action.Reason = "Auto-apply classes is disabled"
			return action
		}

	case SuggestionAddProperty:
		if !policy.AutoApplyProperties {
			action.Action = "require_approval"
			action.Reason = "Auto-apply properties is disabled"
			return action
		}

	case SuggestionModifyClass, SuggestionModifyProperty:
		if !policy.AutoApplyModify {
			action.Action = "require_approval"
			action.Reason = "Auto-apply modifications is disabled"
			return action
		}

	case SuggestionDeprecate:
		// Deprecation always requires manual approval
		action.Action = "require_approval"
		action.Reason = "Deprecation requires manual approval"
		return action
	}

	// All checks passed - auto-apply
	action.Action = "auto_apply"
	action.Reason = fmt.Sprintf("Meets policy criteria: confidence %.2f >= %.2f, risk %s <= %s",
		suggestion.Confidence, policy.MinConfidence, suggestion.RiskLevel, policy.MaxRiskLevel)
	return action
}

// compareRiskLevel compares two risk levels (-1 = less, 0 = equal, 1 = greater)
func (a *AutoUpdateEngine) compareRiskLevel(level1, level2 RiskLevel) int {
	levels := map[RiskLevel]int{
		RiskLevelLow:      1,
		RiskLevelMedium:   2,
		RiskLevelHigh:     3,
		RiskLevelCritical: 4,
	}

	score1 := levels[level1]
	score2 := levels[level2]

	if score1 < score2 {
		return -1
	} else if score1 > score2 {
		return 1
	}
	return 0
}

// getPolicy retrieves the auto-update policy for an ontology
func (a *AutoUpdateEngine) getPolicy(ctx context.Context, ontologyID string) (*AutoUpdatePolicy, error) {
	query := `SELECT id, ontology_id, enabled, auto_apply_classes, auto_apply_properties, 
	          auto_apply_modify, max_risk_level, min_confidence, require_approval, 
	          notification_email, created_at, updated_at 
	          FROM auto_update_policies WHERE ontology_id = ?`

	var policy AutoUpdatePolicy
	var requireApprovalJSON sql.NullString
	var notificationEmail sql.NullString

	err := a.db.QueryRowContext(ctx, query, ontologyID).Scan(
		&policy.ID,
		&policy.OntologyID,
		&policy.Enabled,
		&policy.AutoApplyClasses,
		&policy.AutoApplyProperties,
		&policy.AutoApplyModify,
		&policy.MaxRiskLevel,
		&policy.MinConfidence,
		&requireApprovalJSON,
		&notificationEmail,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default policy
		return &AutoUpdatePolicy{
			OntologyID:          ontologyID,
			Enabled:             false, // Disabled by default for safety
			AutoApplyClasses:    false,
			AutoApplyProperties: false,
			AutoApplyModify:     false,
			MaxRiskLevel:        RiskLevelLow,
			MinConfidence:       0.8,
			RequireApproval:     []SuggestionType{SuggestionDeprecate},
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}, nil
	}

	if err != nil {
		return nil, err
	}

	if requireApprovalJSON.Valid {
		json.Unmarshal([]byte(requireApprovalJSON.String), &policy.RequireApproval)
	}
	if notificationEmail.Valid {
		policy.NotificationEmail = notificationEmail.String
	}

	return &policy, nil
}

// CreateOrUpdatePolicy creates or updates an auto-update policy
func (a *AutoUpdateEngine) CreateOrUpdatePolicy(ctx context.Context, policy *AutoUpdatePolicy) error {
	requireApprovalJSON, _ := json.Marshal(policy.RequireApproval)

	// Check if policy exists
	var exists bool
	err := a.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM auto_update_policies WHERE ontology_id = ?)",
		policy.OntologyID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update
		query := `UPDATE auto_update_policies 
		          SET enabled = ?, auto_apply_classes = ?, auto_apply_properties = ?, 
		              auto_apply_modify = ?, max_risk_level = ?, min_confidence = ?, 
		              require_approval = ?, notification_email = ?, updated_at = ? 
		          WHERE ontology_id = ?`
		_, err = a.db.ExecContext(ctx, query,
			policy.Enabled,
			policy.AutoApplyClasses,
			policy.AutoApplyProperties,
			policy.AutoApplyModify,
			policy.MaxRiskLevel,
			policy.MinConfidence,
			string(requireApprovalJSON),
			policy.NotificationEmail,
			time.Now(),
			policy.OntologyID,
		)
	} else {
		// Insert
		query := `INSERT INTO auto_update_policies 
		          (ontology_id, enabled, auto_apply_classes, auto_apply_properties, 
		           auto_apply_modify, max_risk_level, min_confidence, require_approval, 
		           notification_email, created_at, updated_at) 
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = a.db.ExecContext(ctx, query,
			policy.OntologyID,
			policy.Enabled,
			policy.AutoApplyClasses,
			policy.AutoApplyProperties,
			policy.AutoApplyModify,
			policy.MaxRiskLevel,
			policy.MinConfidence,
			string(requireApprovalJSON),
			policy.NotificationEmail,
			time.Now(),
			time.Now(),
		)
	}

	return err
}

// sendNotification sends a notification about auto-update actions
func (a *AutoUpdateEngine) sendNotification(ctx context.Context, policy *AutoUpdatePolicy, result *AutoUpdateResult) {
	// In a real implementation, this would send an email
	// For now, just log
	fmt.Printf("Auto-update notification for ontology %s: %d auto-applied, %d auto-rejected, %d require manual review\n",
		policy.OntologyID, result.AutoApplied, result.AutoRejected, result.RequireManual)
}

// AutoUpdateResult represents the result of an auto-update run
type AutoUpdateResult struct {
	OntologyID       string             `json:"ontology_id"`
	TotalSuggestions int                `json:"total_suggestions"`
	AutoApplied      int                `json:"auto_applied"`
	AutoRejected     int                `json:"auto_rejected"`
	RequireManual    int                `json:"require_manual"`
	Processed        []SuggestionAction `json:"processed"`
	Skipped          []SuggestionAction `json:"skipped"`
	Message          string             `json:"message,omitempty"`
}

// SuggestionAction represents an action taken on a suggestion
type SuggestionAction struct {
	SuggestionID int            `json:"suggestion_id"`
	Type         SuggestionType `json:"type"`
	EntityURI    string         `json:"entity_uri"`
	Action       string         `json:"action"` // auto_apply, auto_reject, require_approval, error
	Reason       string         `json:"reason"`
	Confidence   float64        `json:"confidence"`
	RiskLevel    RiskLevel      `json:"risk_level"`
	Error        string         `json:"error,omitempty"`
}

// AutoUpdatePlugin implements BasePlugin for auto-update operations
type AutoUpdatePlugin struct {
	engine *AutoUpdateEngine
}

// NewAutoUpdatePlugin creates a new auto-update plugin
func NewAutoUpdatePlugin(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *AutoUpdatePlugin {
	return &AutoUpdatePlugin{
		engine: NewAutoUpdateEngine(db, llmClient, tdb2Backend),
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *AutoUpdatePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	operation, ok := stepConfig.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation not specified in config")
	}

	resultContext := pipelines.NewPluginContext()

	switch operation {
	case "process":
		ontologyID, _ := stepConfig.Config["ontology_id"].(string)

		result, err := p.engine.ProcessPendingSuggestions(ctx, ontologyID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("result", result)

	case "get_policy":
		ontologyID, _ := stepConfig.Config["ontology_id"].(string)

		policy, err := p.engine.getPolicy(ctx, ontologyID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("policy", policy)

	case "update_policy":
		policyData, ok := stepConfig.Config["policy"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("policy data is required")
		}

		policyJSON, _ := json.Marshal(policyData)
		var policy AutoUpdatePolicy
		if err := json.Unmarshal(policyJSON, &policy); err != nil {
			return nil, fmt.Errorf("invalid policy format: %w", err)
		}

		if err := p.engine.CreateOrUpdatePolicy(ctx, &policy); err != nil {
			return nil, err
		}
		resultContext.Set("status", "updated")

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return resultContext, nil
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *AutoUpdatePlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *AutoUpdatePlugin) GetPluginName() string {
	return "auto_update"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *AutoUpdatePlugin) ValidateConfig(config map[string]any) error {
	operation, ok := config["operation"].(string)
	if !ok {
		return fmt.Errorf("operation is required")
	}

	validOperations := []string{"process", "get_policy", "update_policy"}
	valid := false
	for _, op := range validOperations {
		if operation == op {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}
